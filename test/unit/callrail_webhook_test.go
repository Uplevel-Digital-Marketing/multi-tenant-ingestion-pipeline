package unit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CallRailWebhookTestSuite tests the CallRail webhook integration components
type CallRailWebhookTestSuite struct {
	suite.Suite
	webhookHandler   *CallRailWebhookHandler
	mockProcessor    *MockCallProcessor
	mockValidator    *MockWebhookValidator
	mockAudioClient  *MockAudioDownloader
	ctx              context.Context
}

// CallRail webhook payload structures
type CallRailWebhookPayload struct {
	CallID         string                 `json:"call_id"`
	CompanyID      string                 `json:"company_id"`
	AccountID      string                 `json:"account_id"`
	PhoneNumber    string                 `json:"phone_number"`
	CallerID       string                 `json:"caller_id"`
	Duration       int                    `json:"duration"` // seconds
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Direction      string                 `json:"direction"` // inbound/outbound
	RecordingURL   string                 `json:"recording_url,omitempty"`
	Transcription  string                 `json:"transcription,omitempty"`
	CallStatus     string                 `json:"call_status"` // completed, abandoned, etc.
	TrackingNumber string                 `json:"tracking_number"`
	CustomFields   map[string]interface{} `json:"custom_fields,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	// Tenant identification from CallRail configuration
	TenantMapping  map[string]string      `json:"tenant_mapping,omitempty"`
}

type CallRailWebhookResponse struct {
	Status      string    `json:"status"`
	IngestionID string    `json:"ingestion_id,omitempty"`
	Message     string    `json:"message"`
	ProcessedAt time.Time `json:"processed_at"`
	TenantID    string    `json:"tenant_id"`
}

type CallProcessingResult struct {
	IngestionID     string                 `json:"ingestion_id"`
	TenantID        string                 `json:"tenant_id"`
	CallID          string                 `json:"call_id"`
	AudioURL        string                 `json:"audio_url"`
	AudioSize       int64                  `json:"audio_size_bytes"`
	ProcessingStage string                 `json:"processing_stage"`
	ExtractedData   map[string]interface{} `json:"extracted_data,omitempty"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Success         bool                   `json:"success"`
	Error           string                 `json:"error,omitempty"`
}

// Mock implementations
type MockCallProcessor struct {
	mock.Mock
}

func (m *MockCallProcessor) ProcessCall(ctx context.Context, payload *CallRailWebhookPayload) (*CallProcessingResult, error) {
	args := m.Called(ctx, payload)
	return args.Get(0).(*CallProcessingResult), args.Error(1)
}

type MockWebhookValidator struct {
	mock.Mock
}

func (m *MockWebhookValidator) ValidateWebhook(r *http.Request) (*CallRailWebhookPayload, error) {
	args := m.Called(r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CallRailWebhookPayload), args.Error(1)
}

func (m *MockWebhookValidator) ValidateSignature(payload []byte, signature string) bool {
	args := m.Called(payload, signature)
	return args.Bool(0)
}

type MockAudioDownloader struct {
	mock.Mock
}

func (m *MockAudioDownloader) DownloadAudio(ctx context.Context, url string) ([]byte, error) {
	args := m.Called(ctx, url)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockAudioDownloader) GetAudioMetadata(url string) (*AudioMetadata, error) {
	args := m.Called(url)
	return args.Get(0).(*AudioMetadata), args.Error(1)
}

type AudioMetadata struct {
	Duration    time.Duration `json:"duration"`
	SampleRate  int           `json:"sample_rate"`
	Channels    int           `json:"channels"`
	Bitrate     int           `json:"bitrate"`
	Format      string        `json:"format"`
	Size        int64         `json:"size_bytes"`
}

// CallRailWebhookHandler handles incoming CallRail webhooks
type CallRailWebhookHandler struct {
	processor      CallProcessor
	validator      WebhookValidator
	audioClient    AudioDownloader
	tenantResolver TenantResolver
	rateLimiter    RateLimiter
}

// Interfaces for dependency injection
type CallProcessor interface {
	ProcessCall(ctx context.Context, payload *CallRailWebhookPayload) (*CallProcessingResult, error)
}

type WebhookValidator interface {
	ValidateWebhook(r *http.Request) (*CallRailWebhookPayload, error)
	ValidateSignature(payload []byte, signature string) bool
}

type AudioDownloader interface {
	DownloadAudio(ctx context.Context, url string) ([]byte, error)
	GetAudioMetadata(url string) (*AudioMetadata, error)
}

type TenantResolver interface {
	ResolveTenant(payload *CallRailWebhookPayload) (string, error)
}

type RateLimiter interface {
	Allow(tenantID string) bool
	GetLimit(tenantID string) int
}

type MockTenantResolver struct {
	mock.Mock
}

func (m *MockTenantResolver) ResolveTenant(payload *CallRailWebhookPayload) (string, error) {
	args := m.Called(payload)
	return args.String(0), args.Error(1)
}

type MockRateLimiter struct {
	mock.Mock
}

func (m *MockRateLimiter) Allow(tenantID string) bool {
	args := m.Called(tenantID)
	return args.Bool(0)
}

func (m *MockRateLimiter) GetLimit(tenantID string) int {
	args := m.Called(tenantID)
	return args.Int(0)
}

func NewCallRailWebhookHandler(processor CallProcessor, validator WebhookValidator,
	audioClient AudioDownloader, tenantResolver TenantResolver, rateLimiter RateLimiter) *CallRailWebhookHandler {
	return &CallRailWebhookHandler{
		processor:      processor,
		validator:      validator,
		audioClient:    audioClient,
		tenantResolver: tenantResolver,
		rateLimiter:    rateLimiter,
	}
}

// HandleWebhook processes incoming CallRail webhook requests
func (h *CallRailWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Step 1: Validate webhook payload and signature
	payload, err := h.validator.ValidateWebhook(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid webhook: %v", err), http.StatusBadRequest)
		return
	}

	// Step 2: Resolve tenant from payload
	tenantID, err := h.tenantResolver.ResolveTenant(payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Tenant resolution failed: %v", err), http.StatusBadRequest)
		return
	}

	// Step 3: Check rate limits
	if !h.rateLimiter.Allow(tenantID) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Step 4: Process the call asynchronously
	result, err := h.processor.ProcessCall(ctx, payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 5: Return success response
	response := &CallRailWebhookResponse{
		Status:      "accepted",
		IngestionID: result.IngestionID,
		Message:     "Call processing initiated successfully",
		ProcessedAt: time.Now(),
		TenantID:    tenantID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	// Log processing time for performance monitoring
	processingTime := time.Since(startTime)
	if processingTime > 200*time.Millisecond {
		// Log slow processing warning
		fmt.Printf("WARN: Slow webhook processing: %v for tenant %s\n", processingTime, tenantID)
	}
}

func (suite *CallRailWebhookTestSuite) SetupTest() {
	suite.mockProcessor = new(MockCallProcessor)
	suite.mockValidator = new(MockWebhookValidator)
	suite.mockAudioClient = new(MockAudioDownloader)
	mockTenantResolver := new(MockTenantResolver)
	mockRateLimiter := new(MockRateLimiter)

	suite.webhookHandler = NewCallRailWebhookHandler(
		suite.mockProcessor,
		suite.mockValidator,
		suite.mockAudioClient,
		mockTenantResolver,
		mockRateLimiter,
	)

	suite.ctx = context.Background()
}

func (suite *CallRailWebhookTestSuite) TearDownTest() {
	suite.mockProcessor.AssertExpectations(suite.T())
	suite.mockValidator.AssertExpectations(suite.T())
	suite.mockAudioClient.AssertExpectations(suite.T())
}

func (suite *CallRailWebhookTestSuite) TestValidCallRailWebhook_Success() {
	// Arrange
	payload := &CallRailWebhookPayload{
		CallID:         "call-12345",
		CompanyID:      "company-67890",
		AccountID:      "account-11111",
		PhoneNumber:    "+15551234567",
		CallerID:       "+15559876543",
		Duration:       180, // 3 minutes
		StartTime:      time.Now().Add(-5 * time.Minute),
		EndTime:        time.Now().Add(-2 * time.Minute),
		Direction:      "inbound",
		RecordingURL:   "https://callrail.com/recordings/call-12345.wav",
		CallStatus:     "completed",
		TrackingNumber: "+15551111111",
		CustomFields: map[string]interface{}{
			"lead_source": "google_ads",
			"campaign_id": "camp-98765",
		},
		Tags: []string{"home_remodeling", "kitchen"},
		TenantMapping: map[string]string{
			"company_id": "tenant-home-pros",
		},
	}

	expectedResult := &CallProcessingResult{
		IngestionID:     "ing_call_12345_" + fmt.Sprintf("%d", time.Now().Unix()),
		TenantID:        "tenant-home-pros",
		CallID:          "call-12345",
		AudioURL:        "https://callrail.com/recordings/call-12345.wav",
		AudioSize:       1024 * 1024, // 1MB
		ProcessingStage: "initiated",
		Success:         true,
		ProcessingTime:  50 * time.Millisecond,
	}

	// Create HTTP request
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", "valid-signature")

	// Setup mocks
	suite.mockValidator.On("ValidateWebhook", req).Return(payload, nil)

	// Mock tenant resolution
	mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
	mockTenantResolver.On("ResolveTenant", payload).Return("tenant-home-pros", nil)

	// Mock rate limiter
	mockRateLimiter := suite.webhookHandler.rateLimiter.(*MockRateLimiter)
	mockRateLimiter.On("Allow", "tenant-home-pros").Return(true)

	suite.mockProcessor.On("ProcessCall", mock.AnythingOfType("*context.emptyCtx"), payload).Return(expectedResult, nil)

	// Act
	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, rr.Code)

	var response CallRailWebhookResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "accepted", response.Status)
	assert.Equal(suite.T(), expectedResult.IngestionID, response.IngestionID)
	assert.Equal(suite.T(), "tenant-home-pros", response.TenantID)
	assert.NotEmpty(suite.T(), response.Message)
}

func (suite *CallRailWebhookTestSuite) TestWebhookValidationFailure() {
	// Arrange
	req := httptest.NewRequest("POST", "/webhook/callrail", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	suite.mockValidator.On("ValidateWebhook", req).Return((*CallRailWebhookPayload)(nil), fmt.Errorf("invalid payload"))

	// Act
	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)
	assert.Contains(suite.T(), rr.Body.String(), "Invalid webhook")
}

func (suite *CallRailWebhookTestSuite) TestTenantResolutionFailure() {
	// Arrange
	payload := &CallRailWebhookPayload{
		CallID:    "call-12345",
		CompanyID: "unknown-company",
	}

	req := httptest.NewRequest("POST", "/webhook/callrail", nil)
	suite.mockValidator.On("ValidateWebhook", req).Return(payload, nil)

	mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
	mockTenantResolver.On("ResolveTenant", payload).Return("", fmt.Errorf("tenant not found"))

	// Act
	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, rr.Code)
	assert.Contains(suite.T(), rr.Body.String(), "Tenant resolution failed")
}

func (suite *CallRailWebhookTestSuite) TestRateLimitExceeded() {
	// Arrange
	payload := &CallRailWebhookPayload{
		CallID:    "call-12345",
		CompanyID: "company-67890",
	}

	req := httptest.NewRequest("POST", "/webhook/callrail", nil)
	suite.mockValidator.On("ValidateWebhook", req).Return(payload, nil)

	mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
	mockTenantResolver.On("ResolveTenant", payload).Return("tenant-rate-limited", nil)

	mockRateLimiter := suite.webhookHandler.rateLimiter.(*MockRateLimiter)
	mockRateLimiter.On("Allow", "tenant-rate-limited").Return(false)

	// Act
	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert
	assert.Equal(suite.T(), http.StatusTooManyRequests, rr.Code)
	assert.Contains(suite.T(), rr.Body.String(), "Rate limit exceeded")
}

func (suite *CallRailWebhookTestSuite) TestCallProcessingFailure() {
	// Arrange
	payload := &CallRailWebhookPayload{
		CallID:       "call-12345",
		CompanyID:    "company-67890",
		RecordingURL: "https://callrail.com/recordings/call-12345.wav",
	}

	req := httptest.NewRequest("POST", "/webhook/callrail", nil)
	suite.mockValidator.On("ValidateWebhook", req).Return(payload, nil)

	mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
	mockTenantResolver.On("ResolveTenant", payload).Return("tenant-processing-error", nil)

	mockRateLimiter := suite.webhookHandler.rateLimiter.(*MockRateLimiter)
	mockRateLimiter.On("Allow", "tenant-processing-error").Return(true)

	suite.mockProcessor.On("ProcessCall", mock.AnythingOfType("*context.emptyCtx"), payload).
		Return((*CallProcessingResult)(nil), fmt.Errorf("audio download failed"))

	// Act
	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)

	// Assert
	assert.Equal(suite.T(), http.StatusInternalServerError, rr.Code)
	assert.Contains(suite.T(), rr.Body.String(), "Processing failed")
}

func (suite *CallRailWebhookTestSuite) TestMultipleTenantWebhooks_Concurrent() {
	// Test concurrent webhook processing for different tenants
	tenants := []struct {
		tenantID  string
		companyID string
		callID    string
	}{
		{"tenant-1", "company-1", "call-1"},
		{"tenant-2", "company-2", "call-2"},
		{"tenant-3", "company-3", "call-3"},
	}

	var responses []*httptest.ResponseRecorder
	done := make(chan bool, len(tenants))

	for _, tenant := range tenants {
		go func(t struct{ tenantID, companyID, callID string }) {
			payload := &CallRailWebhookPayload{
				CallID:    t.callID,
				CompanyID: t.companyID,
				Duration:  120,
			}

			req := httptest.NewRequest("POST", "/webhook/callrail", nil)

			suite.mockValidator.On("ValidateWebhook", req).Return(payload, nil).Once()

			mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
			mockTenantResolver.On("ResolveTenant", payload).Return(t.tenantID, nil).Once()

			mockRateLimiter := suite.webhookHandler.rateLimiter.(*MockRateLimiter)
			mockRateLimiter.On("Allow", t.tenantID).Return(true).Once()

			expectedResult := &CallProcessingResult{
				IngestionID: "ing_" + t.callID,
				TenantID:    t.tenantID,
				CallID:      t.callID,
				Success:     true,
			}
			suite.mockProcessor.On("ProcessCall", mock.AnythingOfType("*context.emptyCtx"), payload).
				Return(expectedResult, nil).Once()

			rr := httptest.NewRecorder()
			suite.webhookHandler.HandleWebhook(rr, req)
			responses = append(responses, rr)

			done <- true
		}(tenant)
	}

	// Wait for all requests to complete
	for i := 0; i < len(tenants); i++ {
		<-done
	}

	// Verify all requests succeeded
	for i, rr := range responses {
		assert.Equal(suite.T(), http.StatusOK, rr.Code, "Request %d should succeed", i)

		var response CallRailWebhookResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "accepted", response.Status)
	}
}

// Table-driven tests for various CallRail scenarios
func (suite *CallRailWebhookTestSuite) TestCallRailWebhookScenarios() {
	testCases := []struct {
		name           string
		payload        *CallRailWebhookPayload
		expectedStatus int
		description    string
	}{
		{
			name: "Inbound_Completed_Call",
			payload: &CallRailWebhookPayload{
				CallID:         "call-inbound-1",
				CompanyID:      "company-1",
				Direction:      "inbound",
				CallStatus:     "completed",
				Duration:       240,
				RecordingURL:   "https://callrail.com/recordings/call-inbound-1.wav",
				TrackingNumber: "+15551111111",
			},
			expectedStatus: http.StatusOK,
			description:    "Standard inbound completed call should be processed",
		},
		{
			name: "Outbound_Call",
			payload: &CallRailWebhookPayload{
				CallID:     "call-outbound-1",
				CompanyID:  "company-1",
				Direction:  "outbound",
				CallStatus: "completed",
				Duration:   180,
			},
			expectedStatus: http.StatusOK,
			description:    "Outbound calls should be processed",
		},
		{
			name: "Abandoned_Call",
			payload: &CallRailWebhookPayload{
				CallID:     "call-abandoned-1",
				CompanyID:  "company-1",
				Direction:  "inbound",
				CallStatus: "abandoned",
				Duration:   15, // Short duration
			},
			expectedStatus: http.StatusOK,
			description:    "Abandoned calls should still be processed for analytics",
		},
		{
			name: "Call_With_Custom_Fields",
			payload: &CallRailWebhookPayload{
				CallID:    "call-custom-1",
				CompanyID: "company-1",
				Duration:  300,
				CustomFields: map[string]interface{}{
					"lead_source":    "facebook_ads",
					"campaign_name":  "Kitchen Remodel 2024",
					"project_type":   "kitchen",
					"budget_range":   "$20000-$40000",
					"urgency_level":  "high",
				},
				Tags: []string{"kitchen", "urgent", "qualified_lead"},
			},
			expectedStatus: http.StatusOK,
			description:    "Calls with rich custom fields should be processed",
		},
		{
			name: "Call_With_Transcription",
			payload: &CallRailWebhookPayload{
				CallID:        "call-transcribed-1",
				CompanyID:     "company-1",
				Duration:      450,
				RecordingURL:  "https://callrail.com/recordings/call-transcribed-1.wav",
				Transcription: "Hi, I'm interested in a complete kitchen remodeling. My name is Sarah Johnson, and my phone number is 555-9876. I'd like to schedule a consultation.",
			},
			expectedStatus: http.StatusOK,
			description:    "Calls with pre-existing transcription should be processed",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook/callrail", nil)

			suite.mockValidator.On("ValidateWebhook", req).Return(tc.payload, nil).Once()

			mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
			mockTenantResolver.On("ResolveTenant", tc.payload).Return("test-tenant", nil).Once()

			mockRateLimiter := suite.webhookHandler.rateLimiter.(*MockRateLimiter)
			mockRateLimiter.On("Allow", "test-tenant").Return(true).Once()

			expectedResult := &CallProcessingResult{
				IngestionID: "ing_" + tc.payload.CallID,
				TenantID:    "test-tenant",
				CallID:      tc.payload.CallID,
				Success:     true,
			}
			suite.mockProcessor.On("ProcessCall", mock.AnythingOfType("*context.emptyCtx"), tc.payload).
				Return(expectedResult, nil).Once()

			rr := httptest.NewRecorder()
			suite.webhookHandler.HandleWebhook(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, tc.description)
		})
	}
}

// Performance test for webhook latency requirement (<200ms)
func (suite *CallRailWebhookTestSuite) TestWebhookLatencyPerformance() {
	payload := &CallRailWebhookPayload{
		CallID:    "perf-test-call",
		CompanyID: "perf-test-company",
		Duration:  120,
	}

	// Setup mocks for fast processing
	req := httptest.NewRequest("POST", "/webhook/callrail", nil)
	suite.mockValidator.On("ValidateWebhook", req).Return(payload, nil)

	mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
	mockTenantResolver.On("ResolveTenant", payload).Return("perf-tenant", nil)

	mockRateLimiter := suite.webhookHandler.rateLimiter.(*MockRateLimiter)
	mockRateLimiter.On("Allow", "perf-tenant").Return(true)

	expectedResult := &CallProcessingResult{
		IngestionID: "ing_perf_test",
		TenantID:    "perf-tenant",
		CallID:      "perf-test-call",
		Success:     true,
	}
	suite.mockProcessor.On("ProcessCall", mock.AnythingOfType("*context.emptyCtx"), payload).
		Return(expectedResult, nil)

	// Measure latency
	startTime := time.Now()
	rr := httptest.NewRecorder()
	suite.webhookHandler.HandleWebhook(rr, req)
	latency := time.Since(startTime)

	// Assert performance target
	assert.Equal(suite.T(), http.StatusOK, rr.Code)
	assert.True(suite.T(), latency < 200*time.Millisecond,
		"Webhook processing should complete within 200ms, got %v", latency)

	suite.T().Logf("Webhook latency: %v", latency)
}

// Benchmark webhook processing performance
func BenchmarkCallRailWebhookProcessing(b *testing.B) {
	suite := &CallRailWebhookTestSuite{}
	suite.SetupTest()

	payload := &CallRailWebhookPayload{
		CallID:    "benchmark-call",
		CompanyID: "benchmark-company",
		Duration:  120,
	}

	req := httptest.NewRequest("POST", "/webhook/callrail", nil)

	// Setup mocks
	suite.mockValidator.On("ValidateWebhook", req).Return(payload, nil)
	mockTenantResolver := suite.webhookHandler.tenantResolver.(*MockTenantResolver)
	mockTenantResolver.On("ResolveTenant", payload).Return("benchmark-tenant", nil)
	mockRateLimiter := suite.webhookHandler.rateLimiter.(*MockRateLimiter)
	mockRateLimiter.On("Allow", "benchmark-tenant").Return(true)

	expectedResult := &CallProcessingResult{
		IngestionID: "ing_benchmark",
		TenantID:    "benchmark-tenant",
		Success:     true,
	}
	suite.mockProcessor.On("ProcessCall", mock.AnythingOfType("*context.emptyCtx"), payload).
		Return(expectedResult, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		suite.webhookHandler.HandleWebhook(rr, req)
	}
}

// Run the test suite
func TestCallRailWebhookTestSuite(t *testing.T) {
	suite.Run(t, new(CallRailWebhookTestSuite))
}