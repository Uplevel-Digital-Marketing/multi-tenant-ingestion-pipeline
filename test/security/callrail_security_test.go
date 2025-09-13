package security

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CallRailSecurityTestSuite tests security aspects of CallRail webhook integration
type CallRailSecurityTestSuite struct {
	suite.Suite
	server              *httptest.Server
	webhookHandler      *SecureCallRailWebhookHandler
	ctx                 context.Context
	webhookSecret       string
	validTenantMappings map[string]string // company_id -> tenant_id
	rateLimiters        map[string]*RateLimiter
}

type SecureCallRailWebhookHandler struct {
	webhookSecret     string
	tenantMappings    map[string]string
	rateLimiters      map[string]*RateLimiter
	auditLogger       AuditLogger
	encryption        EncryptionService
	validator         WebhookValidator
}

type CallRailWebhookPayload struct {
	CallID         string                 `json:"call_id"`
	CompanyID      string                 `json:"company_id"`
	AccountID      string                 `json:"account_id"`
	PhoneNumber    string                 `json:"phone_number"`
	CallerID       string                 `json:"caller_id"`
	Duration       int                    `json:"duration"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Direction      string                 `json:"direction"`
	RecordingURL   string                 `json:"recording_url,omitempty"`
	Transcription  string                 `json:"transcription,omitempty"`
	CallStatus     string                 `json:"call_status"`
	TrackingNumber string                 `json:"tracking_number"`
	CustomFields   map[string]interface{} `json:"custom_fields,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
}

type SecurityAuditEvent struct {
	EventID          string                 `json:"event_id"`
	EventType        string                 `json:"event_type"`
	TenantID         string                 `json:"tenant_id"`
	CompanyID        string                 `json:"company_id"`
	CallID           string                 `json:"call_id"`
	SourceIP         string                 `json:"source_ip"`
	UserAgent        string                 `json:"user_agent"`
	Timestamp        time.Time              `json:"timestamp"`
	Success          bool                   `json:"success"`
	SecurityViolation string                `json:"security_violation,omitempty"`
	Details          map[string]interface{} `json:"details"`
}

type RateLimiter struct {
	tenantID        string
	requestsPerMin  int
	currentRequests int
	windowStart     time.Time
	violations      int
}

type AuditLogger interface {
	LogWebhookEvent(event *SecurityAuditEvent)
	LogSecurityViolation(violation *SecurityViolation)
}

type EncryptionService interface {
	EncryptSensitiveData(data string) (string, error)
	DecryptSensitiveData(encryptedData string) (string, error)
	ValidateDataIntegrity(data string, signature string) bool
}

type WebhookValidator interface {
	ValidateSignature(payload []byte, signature string, secret string) bool
	ValidatePayloadStructure(payload *CallRailWebhookPayload) error
	SanitizePayload(payload *CallRailWebhookPayload) *CallRailWebhookPayload
}

type SecurityViolation struct {
	ViolationType string                 `json:"violation_type"`
	Severity      string                 `json:"severity"`
	Description   string                 `json:"description"`
	TenantID      string                 `json:"tenant_id"`
	CompanyID     string                 `json:"company_id"`
	SourceIP      string                 `json:"source_ip"`
	Details       map[string]interface{} `json:"details"`
	Timestamp     time.Time              `json:"timestamp"`
}

// Mock implementations for testing
type MockAuditLogger struct {
	events    []*SecurityAuditEvent
	violations []*SecurityViolation
}

func (m *MockAuditLogger) LogWebhookEvent(event *SecurityAuditEvent) {
	m.events = append(m.events, event)
}

func (m *MockAuditLogger) LogSecurityViolation(violation *SecurityViolation) {
	m.violations = append(m.violations, violation)
}

func (m *MockAuditLogger) GetEvents() []*SecurityAuditEvent {
	return m.events
}

func (m *MockAuditLogger) GetViolations() []*SecurityViolation {
	return m.violations
}

func (m *MockAuditLogger) Reset() {
	m.events = []*SecurityAuditEvent{}
	m.violations = []*SecurityViolation{}
}

type MockEncryptionService struct{}

func (m *MockEncryptionService) EncryptSensitiveData(data string) (string, error) {
	// Mock encryption - just base64 encode for testing
	return fmt.Sprintf("encrypted_%s", data), nil
}

func (m *MockEncryptionService) DecryptSensitiveData(encryptedData string) (string, error) {
	// Mock decryption
	if strings.HasPrefix(encryptedData, "encrypted_") {
		return strings.TrimPrefix(encryptedData, "encrypted_"), nil
	}
	return "", fmt.Errorf("invalid encrypted data")
}

func (m *MockEncryptionService) ValidateDataIntegrity(data string, signature string) bool {
	// Mock data integrity validation
	return signature == fmt.Sprintf("integrity_%s", data)
}

type MockWebhookValidator struct{}

func (m *MockWebhookValidator) ValidateSignature(payload []byte, signature string, secret string) bool {
	// Implement HMAC-SHA256 signature validation
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (m *MockWebhookValidator) ValidatePayloadStructure(payload *CallRailWebhookPayload) error {
	if payload.CallID == "" {
		return fmt.Errorf("call_id is required")
	}
	if payload.CompanyID == "" {
		return fmt.Errorf("company_id is required")
	}
	if payload.Duration < 0 {
		return fmt.Errorf("duration cannot be negative")
	}
	return nil
}

func (m *MockWebhookValidator) SanitizePayload(payload *CallRailWebhookPayload) *CallRailWebhookPayload {
	// Create a copy and sanitize sensitive fields
	sanitized := *payload

	// Mask phone numbers for logging
	if len(sanitized.PhoneNumber) > 4 {
		sanitized.PhoneNumber = "***-***-" + sanitized.PhoneNumber[len(sanitized.PhoneNumber)-4:]
	}
	if len(sanitized.CallerID) > 4 {
		sanitized.CallerID = "***-***-" + sanitized.CallerID[len(sanitized.CallerID)-4:]
	}

	// Remove potentially sensitive custom fields
	if sanitized.CustomFields != nil {
		safeCopy := make(map[string]interface{})
		for k, v := range sanitized.CustomFields {
			// Only copy non-sensitive fields
			if !isSensitiveField(k) {
				safeCopy[k] = v
			}
		}
		sanitized.CustomFields = safeCopy
	}

	return &sanitized
}

func isSensitiveField(fieldName string) bool {
	sensitiveFields := []string{
		"ssn", "social_security", "credit_card", "password",
		"personal_id", "driver_license", "bank_account",
	}

	fieldLower := strings.ToLower(fieldName)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}

func NewRateLimiter(tenantID string, requestsPerMin int) *RateLimiter {
	return &RateLimiter{
		tenantID:        tenantID,
		requestsPerMin:  requestsPerMin,
		currentRequests: 0,
		windowStart:     time.Now(),
		violations:      0,
	}
}

func (rl *RateLimiter) AllowRequest() bool {
	now := time.Now()

	// Reset window if minute has passed
	if now.Sub(rl.windowStart) >= time.Minute {
		rl.currentRequests = 0
		rl.windowStart = now
	}

	if rl.currentRequests >= rl.requestsPerMin {
		rl.violations++
		return false
	}

	rl.currentRequests++
	return true
}

func (rl *RateLimiter) GetViolationCount() int {
	return rl.violations
}

func NewSecureCallRailWebhookHandler(secret string, tenantMappings map[string]string,
	rateLimiters map[string]*RateLimiter, auditLogger AuditLogger,
	encryption EncryptionService, validator WebhookValidator) *SecureCallRailWebhookHandler {

	return &SecureCallRailWebhookHandler{
		webhookSecret:  secret,
		tenantMappings: tenantMappings,
		rateLimiters:   rateLimiters,
		auditLogger:    auditLogger,
		encryption:     encryption,
		validator:      validator,
	}
}

func (h *SecureCallRailWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	sourceIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// Step 1: Read and validate request body
	body, err := readRequestBody(r)
	if err != nil {
		h.logSecurityViolation("invalid_request_body", "medium", "Failed to read request body", "", "", sourceIP, map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Step 2: Validate webhook signature
	signature := r.Header.Get("X-CallRail-Signature")
	if signature == "" {
		h.logSecurityViolation("missing_signature", "high", "Missing webhook signature", "", "", sourceIP, nil)
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	if !h.validator.ValidateSignature(body, signature, h.webhookSecret) {
		h.logSecurityViolation("invalid_signature", "critical", "Invalid webhook signature", "", "", sourceIP, map[string]interface{}{
			"signature": signature,
		})
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Step 3: Parse and validate payload
	var payload CallRailWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logSecurityViolation("invalid_json", "medium", "Invalid JSON payload", "", "", sourceIP, map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.validator.ValidatePayloadStructure(&payload); err != nil {
		h.logSecurityViolation("invalid_payload_structure", "medium", "Invalid payload structure", "", payload.CompanyID, sourceIP, map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid payload structure", http.StatusBadRequest)
		return
	}

	// Step 4: Resolve tenant and validate authorization
	tenantID, authorized := h.resolveTenant(payload.CompanyID)
	if !authorized {
		h.logSecurityViolation("unauthorized_company", "high", "Unauthorized company ID", "", payload.CompanyID, sourceIP, map[string]interface{}{
			"company_id": payload.CompanyID,
		})
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Step 5: Check rate limits
	if rateLimiter, exists := h.rateLimiters[tenantID]; exists {
		if !rateLimiter.AllowRequest() {
			h.logSecurityViolation("rate_limit_exceeded", "medium", "Rate limit exceeded", tenantID, payload.CompanyID, sourceIP, map[string]interface{}{
				"violation_count": rateLimiter.GetViolationCount(),
			})
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	// Step 6: Sanitize sensitive data for logging
	sanitizedPayload := h.validator.SanitizePayload(&payload)

	// Step 7: Process webhook (simulate processing)
	ingestionID := fmt.Sprintf("ing_%s_%d", payload.CallID, time.Now().Unix())

	// Step 8: Log successful webhook event
	h.logWebhookEvent(&SecurityAuditEvent{
		EventID:   fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		EventType: "callrail_webhook_processed",
		TenantID:  tenantID,
		CompanyID: payload.CompanyID,
		CallID:    payload.CallID,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Timestamp: time.Now(),
		Success:   true,
		Details: map[string]interface{}{
			"ingestion_id":     ingestionID,
			"call_duration":    payload.Duration,
			"call_direction":   payload.Direction,
			"processing_time":  time.Since(startTime).Milliseconds(),
			"sanitized_payload": sanitizedPayload,
		},
	})

	// Step 9: Return success response
	response := map[string]interface{}{
		"status":       "accepted",
		"ingestion_id": ingestionID,
		"tenant_id":    tenantID,
		"call_id":      payload.CallID,
		"processed_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *SecureCallRailWebhookHandler) resolveTenant(companyID string) (string, bool) {
	tenantID, exists := h.tenantMappings[companyID]
	return tenantID, exists
}

func (h *SecureCallRailWebhookHandler) logWebhookEvent(event *SecurityAuditEvent) {
	h.auditLogger.LogWebhookEvent(event)
}

func (h *SecureCallRailWebhookHandler) logSecurityViolation(violationType, severity, description, tenantID, companyID, sourceIP string, details map[string]interface{}) {
	violation := &SecurityViolation{
		ViolationType: violationType,
		Severity:      severity,
		Description:   description,
		TenantID:      tenantID,
		CompanyID:     companyID,
		SourceIP:      sourceIP,
		Details:       details,
		Timestamp:     time.Now(),
	}
	h.auditLogger.LogSecurityViolation(violation)
}

// Helper functions
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

func readRequestBody(r *http.Request) ([]byte, error) {
	// Limit request body size to prevent DoS attacks
	const maxBodySize = 1024 * 1024 // 1MB

	r.Body = http.MaxBytesReader(nil, r.Body, maxBodySize)
	defer r.Body.Close()

	body := make([]byte, 0, r.ContentLength)
	buffer := make([]byte, 4096)

	for {
		n, err := r.Body.Read(buffer)
		if n > 0 {
			body = append(body, buffer[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}
	}

	return body, nil
}

func (suite *CallRailSecurityTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.webhookSecret = "test-webhook-secret-12345"

	// Setup valid tenant mappings
	suite.validTenantMappings = map[string]string{
		"company-home-pros":        "tenant-home-remodeling-pros",
		"company-kitchen-experts":  "tenant-kitchen-specialists",
		"company-emergency-repair": "tenant-emergency-services",
	}

	// Setup rate limiters for each tenant
	suite.rateLimiters = map[string]*RateLimiter{
		"tenant-home-remodeling-pros": NewRateLimiter("tenant-home-remodeling-pros", 100), // 100 req/min
		"tenant-kitchen-specialists":  NewRateLimiter("tenant-kitchen-specialists", 50),   // 50 req/min
		"tenant-emergency-services":   NewRateLimiter("tenant-emergency-services", 200),   // 200 req/min (emergency)
	}

	// Setup secure webhook handler
	auditLogger := &MockAuditLogger{}
	encryption := &MockEncryptionService{}
	validator := &MockWebhookValidator{}

	suite.webhookHandler = NewSecureCallRailWebhookHandler(
		suite.webhookSecret,
		suite.validTenantMappings,
		suite.rateLimiters,
		auditLogger,
		encryption,
		validator,
	)

	// Setup test server
	suite.setupTestServer()
}

func (suite *CallRailSecurityTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *CallRailSecurityTestSuite) SetupTest() {
	// Reset audit logger before each test
	if auditLogger, ok := suite.webhookHandler.auditLogger.(*MockAuditLogger); ok {
		auditLogger.Reset()
	}
}

func (suite *CallRailSecurityTestSuite) setupTestServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", suite.webhookHandler.HandleWebhook)
	suite.server = httptest.NewServer(mux)
}

// Test Cases

func (suite *CallRailSecurityTestSuite) TestValidWebhookWithProperSignature() {
	// Test that properly signed webhooks are accepted
	payload := CallRailWebhookPayload{
		CallID:    "secure-test-call-1",
		CompanyID: "company-home-pros",
		Duration:  180,
		Direction: "inbound",
	}

	body, _ := json.Marshal(payload)
	signature := suite.generateValidSignature(body)

	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", signature)
	req.Header.Set("X-Real-IP", "203.0.113.1")

	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert successful processing
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "accepted", response["status"])
	assert.NotEmpty(suite.T(), response["ingestion_id"])
	assert.Equal(suite.T(), "tenant-home-remodeling-pros", response["tenant_id"])

	// Verify audit logging
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	events := auditLogger.GetEvents()
	assert.Len(suite.T(), events, 1)
	assert.Equal(suite.T(), "callrail_webhook_processed", events[0].EventType)
	assert.True(suite.T(), events[0].Success)
}

func (suite *CallRailSecurityTestSuite) TestInvalidSignatureRejection() {
	// Test that webhooks with invalid signatures are rejected
	payload := CallRailWebhookPayload{
		CallID:    "invalid-signature-call",
		CompanyID: "company-home-pros",
		Duration:  180,
	}

	body, _ := json.Marshal(payload)
	invalidSignature := "invalid-signature-12345"

	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", invalidSignature)
	req.Header.Set("X-Real-IP", "203.0.113.100")

	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert rejection
	assert.Equal(suite.T(), http.StatusUnauthorized, rr.Code)

	// Verify security violation logging
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	violations := auditLogger.GetViolations()
	assert.Len(suite.T(), violations, 1)
	assert.Equal(suite.T(), "invalid_signature", violations[0].ViolationType)
	assert.Equal(suite.T(), "critical", violations[0].Severity)
}

func (suite *CallRailSecurityTestSuite) TestMissingSignatureRejection() {
	// Test that webhooks without signatures are rejected
	payload := CallRailWebhookPayload{
		CallID:    "no-signature-call",
		CompanyID: "company-home-pros",
		Duration:  180,
	}

	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Intentionally not setting X-CallRail-Signature header

	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert rejection
	assert.Equal(suite.T(), http.StatusUnauthorized, rr.Code)

	// Verify security violation logging
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	violations := auditLogger.GetViolations()
	assert.Len(suite.T(), violations, 1)
	assert.Equal(suite.T(), "missing_signature", violations[0].ViolationType)
	assert.Equal(suite.T(), "high", violations[0].Severity)
}

func (suite *CallRailSecurityTestSuite) TestUnauthorizedCompanyRejection() {
	// Test that webhooks from unauthorized companies are rejected
	payload := CallRailWebhookPayload{
		CallID:    "unauthorized-company-call",
		CompanyID: "company-unauthorized",
		Duration:  180,
	}

	body, _ := json.Marshal(payload)
	signature := suite.generateValidSignature(body)

	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", signature)
	req.Header.Set("X-Real-IP", "203.0.113.200")

	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert rejection
	assert.Equal(suite.T(), http.StatusForbidden, rr.Code)

	// Verify security violation logging
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	violations := auditLogger.GetViolations()
	assert.Len(suite.T(), violations, 1)
	assert.Equal(suite.T(), "unauthorized_company", violations[0].ViolationType)
	assert.Equal(suite.T(), "high", violations[0].Severity)
	assert.Equal(suite.T(), "company-unauthorized", violations[0].CompanyID)
}

func (suite *CallRailSecurityTestSuite) TestRateLimitingEnforcement() {
	// Test that rate limiting is properly enforced
	tenantID := "tenant-kitchen-specialists"
	companyID := "company-kitchen-experts"

	// This tenant has a limit of 50 requests per minute
	rateLimiter := suite.rateLimiters[tenantID]

	successCount := 0
	rateLimitedCount := 0

	// Send requests up to and beyond the rate limit
	for i := 0; i < 60; i++ {
		payload := CallRailWebhookPayload{
			CallID:    fmt.Sprintf("rate-limit-call-%d", i),
			CompanyID: companyID,
			Duration:  60,
		}

		body, _ := json.Marshal(payload)
		signature := suite.generateValidSignature(body)

		req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CallRail-Signature", signature)

		rr := httptest.NewRecorder()
		suite.webhookHandler.HandleWebhook(rr, req)

		if rr.Code == http.StatusOK {
			successCount++
		} else if rr.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should accept up to limit and reject excess
	assert.Equal(suite.T(), 50, successCount, "Should accept requests up to rate limit")
	assert.Equal(suite.T(), 10, rateLimitedCount, "Should reject requests exceeding rate limit")
	assert.True(suite.T(), rateLimiter.GetViolationCount() > 0, "Should record rate limit violations")

	// Verify security violation logging
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	violations := auditLogger.GetViolations()
	rateLimitViolations := 0
	for _, violation := range violations {
		if violation.ViolationType == "rate_limit_exceeded" {
			rateLimitViolations++
		}
	}
	assert.Equal(suite.T(), 10, rateLimitViolations, "Should log rate limit violations")
}

func (suite *CallRailSecurityTestSuite) TestPayloadValidationAndSanitization() {
	// Test payload validation and sanitization of sensitive data
	payload := CallRailWebhookPayload{
		CallID:      "sensitive-data-call",
		CompanyID:   "company-home-pros",
		PhoneNumber: "+15551234567",
		CallerID:    "+15559876543",
		Duration:    300,
		CustomFields: map[string]interface{}{
			"customer_name":    "John Doe",
			"project_type":     "kitchen",
			"ssn":              "123-45-6789", // Sensitive field
			"credit_card":      "4111-1111-1111-1111", // Sensitive field
			"budget_range":     "$30,000-$50,000",
		},
	}

	body, _ := json.Marshal(payload)
	signature := suite.generateValidSignature(body)

	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", signature)

	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert successful processing
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	// Verify audit logging with sanitized data
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	events := auditLogger.GetEvents()
	require.Len(suite.T(), events, 1)

	// Check that sensitive data was sanitized in the logs
	sanitizedPayload := events[0].Details["sanitized_payload"].(*CallRailWebhookPayload)

	// Phone numbers should be masked
	assert.True(suite.T(), strings.Contains(sanitizedPayload.PhoneNumber, "***"))
	assert.True(suite.T(), strings.Contains(sanitizedPayload.CallerID, "***"))

	// Sensitive custom fields should be removed
	_, hasSsn := sanitizedPayload.CustomFields["ssn"]
	_, hasCreditCard := sanitizedPayload.CustomFields["credit_card"]
	assert.False(suite.T(), hasSsn, "SSN should be removed from sanitized payload")
	assert.False(suite.T(), hasCreditCard, "Credit card should be removed from sanitized payload")

	// Non-sensitive fields should remain
	assert.Equal(suite.T(), "John Doe", sanitizedPayload.CustomFields["customer_name"])
	assert.Equal(suite.T(), "kitchen", sanitizedPayload.CustomFields["project_type"])
}

func (suite *CallRailSecurityTestSuite) TestInvalidJSONPayloadRejection() {
	// Test rejection of malformed JSON payloads
	invalidJSON := []byte(`{"call_id": "invalid-json-call", "company_id": "company-home-pros", "duration": invalid}`)
	signature := suite.generateValidSignature(invalidJSON)

	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", signature)

	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert rejection
	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)

	// Verify security violation logging
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	violations := auditLogger.GetViolations()
	assert.Len(suite.T(), violations, 1)
	assert.Equal(suite.T(), "invalid_json", violations[0].ViolationType)
}

func (suite *CallRailSecurityTestSuite) TestPayloadStructureValidation() {
	// Test validation of required fields
	invalidPayloads := []struct {
		name    string
		payload CallRailWebhookPayload
	}{
		{
			name: "Missing CallID",
			payload: CallRailWebhookPayload{
				CompanyID: "company-home-pros",
				Duration:  180,
			},
		},
		{
			name: "Missing CompanyID",
			payload: CallRailWebhookPayload{
				CallID:   "missing-company-call",
				Duration: 180,
			},
		},
		{
			name: "Negative Duration",
			payload: CallRailWebhookPayload{
				CallID:    "negative-duration-call",
				CompanyID: "company-home-pros",
				Duration:  -1,
			},
		},
	}

	for _, testCase := range invalidPayloads {
		suite.T().Run(testCase.name, func(t *testing.T) {
			body, _ := json.Marshal(testCase.payload)
			signature := suite.generateValidSignature(body)

			req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CallRail-Signature", signature)

			rr := httptest.NewRecorder()
			suite.webhookHandler.HandleWebhook(rr, req)

			// Assert rejection
			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func (suite *CallRailSecurityTestSuite) TestIPAddressLogging() {
	// Test that source IP addresses are properly logged for security auditing
	testCases := []struct {
		name       string
		headers    map[string]string
		expectedIP string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 198.51.100.1",
			},
			expectedIP: "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.2",
			},
			expectedIP: "203.0.113.2",
		},
	}

	for _, testCase := range testCases {
		suite.T().Run(testCase.name, func(t *testing.T) {
			// Reset audit logger
			auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
			auditLogger.Reset()

			payload := CallRailWebhookPayload{
				CallID:    "ip-logging-call",
				CompanyID: "company-home-pros",
				Duration:  180,
			}

			body, _ := json.Marshal(payload)
			signature := suite.generateValidSignature(body)

			req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CallRail-Signature", signature)

			// Set test headers
			for key, value := range testCase.headers {
				req.Header.Set(key, value)
			}

			rr := httptest.NewRecorder()
			suite.webhookHandler.HandleWebhook(rr, req)

			// Assert successful processing
			assert.Equal(t, http.StatusOK, rr.Code)

			// Verify IP address logging
			events := auditLogger.GetEvents()
			require.Len(t, events, 1)
			assert.Equal(t, testCase.expectedIP, events[0].SourceIP)
		})
	}
}

func (suite *CallRailSecurityTestSuite) TestConcurrentSecurityValidation() {
	// Test security validation under concurrent load
	const numConcurrent = 20
	const requestsPerGoroutine = 10

	results := make(chan bool, numConcurrent*requestsPerGoroutine)

	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			for j := 0; j < requestsPerGoroutine; j++ {
				payload := CallRailWebhookPayload{
					CallID:    fmt.Sprintf("concurrent-security-call-%d-%d", index, j),
					CompanyID: "company-home-pros",
					Duration:  120,
				}

				body, _ := json.Marshal(payload)
				signature := suite.generateValidSignature(body)

				req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-CallRail-Signature", signature)

				rr := httptest.NewRecorder()
				suite.webhookHandler.HandleWebhook(rr, req)

				results <- rr.Code == http.StatusOK
			}
		}(i)
	}

	// Collect results
	successCount := 0
	totalRequests := numConcurrent * requestsPerGoroutine

	for i := 0; i < totalRequests; i++ {
		if <-results {
			successCount++
		}
	}

	// All properly signed requests should succeed
	assert.Equal(suite.T(), totalRequests, successCount, "All valid concurrent requests should succeed")

	// Verify audit logging
	auditLogger := suite.webhookHandler.auditLogger.(*MockAuditLogger)
	events := auditLogger.GetEvents()
	assert.Len(suite.T(), events, totalRequests, "All requests should be audited")

	// Verify no security violations
	violations := auditLogger.GetViolations()
	assert.Len(suite.T(), violations, 0, "No security violations should occur for valid requests")
}

func (suite *CallRailSecurityTestSuite) TestLargePayloadHandling() {
	// Test handling of unusually large payloads (potential DoS attack)
	largeCustomFields := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeCustomFields[fmt.Sprintf("field_%d", i)] = strings.Repeat("data", 100)
	}

	payload := CallRailWebhookPayload{
		CallID:       "large-payload-call",
		CompanyID:    "company-home-pros",
		Duration:     180,
		CustomFields: largeCustomFields,
	}

	body, _ := json.Marshal(payload)

	// This should be over 1MB and trigger the size limit
	suite.T().Logf("Payload size: %d bytes", len(body))

	signature := suite.generateValidSignature(body)

	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", signature)

	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Large payloads should be rejected
	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)
}

// Helper methods

func (suite *CallRailSecurityTestSuite) generateValidSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(suite.webhookSecret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Performance test for security validation
func (suite *CallRailSecurityTestSuite) TestSecurityValidationPerformance() {
	// Test that security validation doesn't significantly impact performance
	const numRequests = 100

	payload := CallRailWebhookPayload{
		CallID:    "performance-test-call",
		CompanyID: "company-home-pros",
		Duration:  180,
	}

	body, _ := json.Marshal(payload)
	signature := suite.generateValidSignature(body)

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CallRail-Signature", signature)

		rr := httptest.NewRecorder()
		suite.webhookHandler.HandleWebhook(rr, req)

		assert.Equal(suite.T(), http.StatusOK, rr.Code)
	}

	totalTime := time.Since(startTime)
	averageTime := totalTime / numRequests

	suite.T().Logf("Security validation performance: %d requests in %v (avg: %v per request)",
		numRequests, totalTime, averageTime)

	// Security validation should not add significant overhead
	assert.True(suite.T(), averageTime < 50*time.Millisecond,
		"Security validation should complete within 50ms per request")
}

// Run the test suite
func TestCallRailSecurityTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CallRail security tests in short mode")
	}

	suite.Run(t, new(CallRailSecurityTestSuite))
}