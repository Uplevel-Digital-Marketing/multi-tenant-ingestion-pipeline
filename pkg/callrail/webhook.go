package callrail

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

// WebhookProcessor handles CallRail webhook processing
type WebhookProcessor struct {
	webhookSecret string
}

// NewWebhookProcessor creates a new CallRail webhook processor
func NewWebhookProcessor(webhookSecret string) *WebhookProcessor {
	return &WebhookProcessor{
		webhookSecret: webhookSecret,
	}
}

// WebhookValidationResult contains the result of webhook validation
type WebhookValidationResult struct {
	Valid         bool   `json:"valid"`
	Error         string `json:"error,omitempty"`
	PayloadSize   int    `json:"payload_size"`
	Signature     string `json:"signature"`
	ComputedHash  string `json:"computed_hash,omitempty"`
}

// WebhookProcessingOptions contains options for webhook processing
type WebhookProcessingOptions struct {
	ValidateSignature bool `json:"validate_signature"`
	MaxPayloadSize    int  `json:"max_payload_size"` // in bytes
	RequiredFields    []string `json:"required_fields"`
}

// DefaultProcessingOptions returns default processing options
func DefaultProcessingOptions() *WebhookProcessingOptions {
	return &WebhookProcessingOptions{
		ValidateSignature: true,
		MaxPayloadSize:    1024 * 1024, // 1MB
		RequiredFields:    []string{"call_id", "tenant_id", "callrail_company_id"},
	}
}

// ValidateWebhook validates the incoming webhook signature and payload
func (w *WebhookProcessor) ValidateWebhook(payload []byte, signature string, options *WebhookProcessingOptions) *WebhookValidationResult {
	result := &WebhookValidationResult{
		PayloadSize: len(payload),
		Signature:   signature,
	}

	// Check payload size
	if options != nil && options.MaxPayloadSize > 0 && len(payload) > options.MaxPayloadSize {
		result.Error = fmt.Sprintf("payload too large: %d bytes (max: %d)", len(payload), options.MaxPayloadSize)
		return result
	}

	// Validate signature if required
	if options == nil || options.ValidateSignature {
		if err := w.verifySignature(payload, signature); err != nil {
			result.Error = fmt.Sprintf("signature validation failed: %v", err)
			return result
		}
	}

	result.Valid = true
	return result
}

// ParseWebhook parses and validates the webhook payload
func (w *WebhookProcessor) ParseWebhook(payload []byte, options *WebhookProcessingOptions) (*models.CallRailWebhook, error) {
	var webhook models.CallRailWebhook
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("failed to parse JSON payload: %w", err)
	}

	// Validate required fields
	if options != nil && len(options.RequiredFields) > 0 {
		if err := w.validateRequiredFields(&webhook, options.RequiredFields); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return &webhook, nil
}

// ProcessWebhook is a convenience method that validates and parses the webhook
func (w *WebhookProcessor) ProcessWebhook(payload []byte, signature string, options *WebhookProcessingOptions) (*models.CallRailWebhook, *WebhookValidationResult, error) {
	if options == nil {
		options = DefaultProcessingOptions()
	}

	// Validate webhook
	validationResult := w.ValidateWebhook(payload, signature, options)
	if !validationResult.Valid {
		return nil, validationResult, fmt.Errorf("webhook validation failed: %s", validationResult.Error)
	}

	// Parse webhook
	webhook, err := w.ParseWebhook(payload, options)
	if err != nil {
		return nil, validationResult, err
	}

	return webhook, validationResult, nil
}

// verifySignature verifies the HMAC signature of the webhook payload
func (w *WebhookProcessor) verifySignature(payload []byte, signature string) error {
	if signature == "" {
		return fmt.Errorf("missing signature header")
	}

	// CallRail typically sends signatures in the format "sha256=<hash>"
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("invalid signature format, expected sha256= prefix")
	}

	expectedHash := strings.TrimPrefix(signature, "sha256=")

	// Compute HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(w.webhookSecret))
	mac.Write(payload)
	computedHash := hex.EncodeToString(mac.Sum(nil))

	// Compare hashes using constant time comparison to prevent timing attacks
	if !hmac.Equal([]byte(expectedHash), []byte(computedHash)) {
		return fmt.Errorf("signature mismatch: expected %s, computed %s", expectedHash, computedHash)
	}

	return nil
}

// validateRequiredFields checks that required fields are present and not empty
func (w *WebhookProcessor) validateRequiredFields(webhook *models.CallRailWebhook, requiredFields []string) error {
	for _, field := range requiredFields {
		switch field {
		case "call_id":
			if webhook.CallID == "" {
				return fmt.Errorf("missing required field: call_id")
			}
		case "tenant_id":
			if webhook.TenantID == "" {
				return fmt.Errorf("missing required field: tenant_id")
			}
		case "callrail_company_id":
			if webhook.CallRailCompanyID == "" {
				return fmt.Errorf("missing required field: callrail_company_id")
			}
		case "account_id":
			if webhook.AccountID == "" {
				return fmt.Errorf("missing required field: account_id")
			}
		case "caller_id":
			if webhook.CallerID == "" {
				return fmt.Errorf("missing required field: caller_id")
			}
		case "customer_phone_number":
			if webhook.CustomerPhoneNumber == "" {
				return fmt.Errorf("missing required field: customer_phone_number")
			}
		default:
			// Custom field validation can be added here
		}
	}
	return nil
}

// WebhookMetrics contains metrics about webhook processing
type WebhookMetrics struct {
	TotalReceived      int64             `json:"total_received"`
	TotalProcessed     int64             `json:"total_processed"`
	TotalFailed        int64             `json:"total_failed"`
	SignatureFailures  int64             `json:"signature_failures"`
	ParseFailures      int64             `json:"parse_failures"`
	ValidationFailures int64             `json:"validation_failures"`
	AverageProcessingTime time.Duration  `json:"average_processing_time"`
	LastProcessed      time.Time         `json:"last_processed"`
	ErrorCounts        map[string]int64  `json:"error_counts"`
}

// WebhookMetricsCollector collects metrics about webhook processing
type WebhookMetricsCollector struct {
	metrics    *WebhookMetrics
	startTimes map[string]time.Time
}

// NewWebhookMetricsCollector creates a new metrics collector
func NewWebhookMetricsCollector() *WebhookMetricsCollector {
	return &WebhookMetricsCollector{
		metrics: &WebhookMetrics{
			ErrorCounts: make(map[string]int64),
		},
		startTimes: make(map[string]time.Time),
	}
}

// StartProcessing marks the start of webhook processing
func (m *WebhookMetricsCollector) StartProcessing(webhookID string) {
	m.startTimes[webhookID] = time.Now()
	m.metrics.TotalReceived++
}

// EndProcessing marks the end of webhook processing
func (m *WebhookMetricsCollector) EndProcessing(webhookID string, success bool, errorType string) {
	startTime, exists := m.startTimes[webhookID]
	if exists {
		processingTime := time.Since(startTime)
		// Update average processing time (simple moving average)
		if m.metrics.TotalProcessed > 0 {
			m.metrics.AverageProcessingTime = (m.metrics.AverageProcessingTime*time.Duration(m.metrics.TotalProcessed) + processingTime) / time.Duration(m.metrics.TotalProcessed+1)
		} else {
			m.metrics.AverageProcessingTime = processingTime
		}
		delete(m.startTimes, webhookID)
	}

	if success {
		m.metrics.TotalProcessed++
		m.metrics.LastProcessed = time.Now()
	} else {
		m.metrics.TotalFailed++
		switch errorType {
		case "signature":
			m.metrics.SignatureFailures++
		case "parse":
			m.metrics.ParseFailures++
		case "validation":
			m.metrics.ValidationFailures++
		}
		m.metrics.ErrorCounts[errorType]++
	}
}

// GetMetrics returns the current metrics
func (m *WebhookMetricsCollector) GetMetrics() *WebhookMetrics {
	return m.metrics
}

// ResetMetrics resets all metrics to zero
func (m *WebhookMetricsCollector) ResetMetrics() {
	m.metrics = &WebhookMetrics{
		ErrorCounts: make(map[string]int64),
	}
	m.startTimes = make(map[string]time.Time)
}

// WebhookHandler provides HTTP handler functionality for webhooks
type WebhookHandler struct {
	processor        *WebhookProcessor
	metricsCollector *WebhookMetricsCollector
	options          *WebhookProcessingOptions
}

// NewWebhookHandler creates a new webhook HTTP handler
func NewWebhookHandler(webhookSecret string, options *WebhookProcessingOptions) *WebhookHandler {
	if options == nil {
		options = DefaultProcessingOptions()
	}

	return &WebhookHandler{
		processor:        NewWebhookProcessor(webhookSecret),
		metricsCollector: NewWebhookMetricsCollector(),
		options:          options,
	}
}

// HandleWebhook is an HTTP handler function for CallRail webhooks
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request, onWebhook func(*models.CallRailWebhook) error) {
	webhookID := fmt.Sprintf("webhook_%d", time.Now().UnixNano())
	h.metricsCollector.StartProcessing(webhookID)

	// Read the request body
	payload := make([]byte, r.ContentLength)
	_, err := r.Body.Read(payload)
	if err != nil {
		h.metricsCollector.EndProcessing(webhookID, false, "read_error")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Get signature from header
	signature := r.Header.Get("X-CallRail-Signature")
	if signature == "" {
		signature = r.Header.Get("x-callrail-signature") // Try lowercase
	}

	// Process webhook
	webhook, validationResult, err := h.processor.ProcessWebhook(payload, signature, h.options)
	if err != nil {
		var errorType string
		if !validationResult.Valid {
			errorType = "signature"
		} else {
			errorType = "parse"
		}
		h.metricsCollector.EndProcessing(webhookID, false, errorType)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Call the webhook handler function
	if err := onWebhook(webhook); err != nil {
		h.metricsCollector.EndProcessing(webhookID, false, "processing_error")
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	h.metricsCollector.EndProcessing(webhookID, true, "")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "success",
		"webhook_id": webhookID,
		"call_id":    webhook.CallID,
		"tenant_id":  webhook.TenantID,
	})
}

// GetMetricsHandler returns an HTTP handler for webhook metrics
func (h *WebhookHandler) GetMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(h.metricsCollector.GetMetrics())
	}
}