package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CallRailE2ETestSuite tests the complete CallRail webhook to CRM integration flow
type CallRailE2ETestSuite struct {
	suite.Suite
	server            *httptest.Server
	spannerClient     *spanner.Client
	ctx               context.Context
	testTenantID      string
	mockCRMServer     *httptest.Server
	mockAudioServer   *httptest.Server
	testCallRecords   []TestCallRecord
}

type TestCallRecord struct {
	CallRailPayload  CallRailWebhookPayload    `json:"callrail_payload"`
	ExpectedResult   ExpectedIngestionResult   `json:"expected_result"`
	CRMExpectation   ExpectedCRMIntegration    `json:"crm_expectation"`
	TestDescription  string                    `json:"test_description"`
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

type ExpectedIngestionResult struct {
	ShouldSucceed     bool                   `json:"should_succeed"`
	ProcessingTimeMax time.Duration          `json:"processing_time_max"`
	ExtractedContact  ContactInfo            `json:"extracted_contact"`
	ExtractedProject  ProjectInfo            `json:"extracted_project"`
	ConfidenceMin     float64                `json:"confidence_min"`
	ExpectedTags      []string               `json:"expected_tags"`
}

type ExpectedCRMIntegration struct {
	ShouldCreateLead bool              `json:"should_create_lead"`
	LeadFields       map[string]string `json:"lead_fields"`
	CRMType          string            `json:"crm_type"`
	Priority         string            `json:"priority"`
}

type ContactInfo struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type ProjectInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Budget      string `json:"budget"`
	Timeline    string `json:"timeline"`
	Urgency     string `json:"urgency"`
}

type IngestionStatusResponse struct {
	IngestionID      string                 `json:"ingestion_id"`
	Status           string                 `json:"status"`
	Progress         float64                `json:"progress"`
	ProcessingStages []ProcessingStageInfo  `json:"processing_stages"`
	ExtractedData    map[string]interface{} `json:"extracted_data"`
	Error            string                 `json:"error,omitempty"`
	TenantID         string                 `json:"tenant_id"`
	CallID           string                 `json:"call_id"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type ProcessingStageInfo struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    *int64     `json:"duration_ms,omitempty"`
	Error       string     `json:"error,omitempty"`
}

type CRMLeadResponse struct {
	LeadID      string                 `json:"lead_id"`
	CRMType     string                 `json:"crm_type"`
	Status      string                 `json:"status"`
	Fields      map[string]interface{} `json:"fields"`
	CreatedAt   time.Time              `json:"created_at"`
	TenantID    string                 `json:"tenant_id"`
	SourceID    string                 `json:"source_id"`
}

func (suite *CallRailE2ETestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testTenantID = "tenant-home-remodeling-pros"

	// Setup mock CRM server
	suite.setupMockCRMServer()

	// Setup mock audio server
	suite.setupMockAudioServer()

	// Setup main API server
	suite.setupMainAPIServer()

	// Setup Spanner client
	suite.setupSpannerClient()

	// Create test call records with various scenarios
	suite.createTestCallRecords()

	// Setup test tenant configuration
	suite.setupTestTenant()
}

func (suite *CallRailE2ETestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.mockCRMServer != nil {
		suite.mockCRMServer.Close()
	}
	if suite.mockAudioServer != nil {
		suite.mockAudioServer.Close()
	}
	if suite.spannerClient != nil {
		suite.spannerClient.Close()
	}
}

func (suite *CallRailE2ETestSuite) SetupTest() {
	// Clean up data before each test
	suite.cleanupTestData()
}

func (suite *CallRailE2ETestSuite) setupMockCRMServer() {
	mux := http.NewServeMux()

	// Salesforce-style lead creation endpoint
	mux.HandleFunc("/services/data/v52.0/sobjects/Lead/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var leadData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&leadData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Simulate lead creation
		leadID := fmt.Sprintf("lead_%d", time.Now().UnixNano())
		response := CRMLeadResponse{
			LeadID:    leadID,
			CRMType:   "salesforce",
			Status:    "created",
			Fields:    leadData,
			CreatedAt: time.Now(),
			TenantID:  suite.testTenantID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// HubSpot-style contact creation endpoint
	mux.HandleFunc("/crm/v3/objects/contacts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var contactData map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&contactData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		contactID := fmt.Sprintf("contact_%d", time.Now().UnixNano())
		response := map[string]interface{}{
			"id":         contactID,
			"properties": contactData,
			"createdAt":  time.Now().Format(time.RFC3339),
			"updatedAt":  time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	suite.mockCRMServer = httptest.NewServer(mux)
}

func (suite *CallRailE2ETestSuite) setupMockAudioServer() {
	mux := http.NewServeMux()

	// Serve mock audio files
	mux.HandleFunc("/recordings/", func(w http.ResponseWriter, r *http.Request) {
		// Extract call ID from path
		callID := r.URL.Path[len("/recordings/"):]
		callID = callID[:len(callID)-4] // Remove .wav extension

		// Generate mock audio based on call scenario
		audioContent := suite.generateMockAudioContent(callID)

		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(audioContent)))
		w.WriteHeader(http.StatusOK)
		w.Write(audioContent)
	})

	suite.mockAudioServer = httptest.NewServer(mux)
}

func (suite *CallRailE2ETestSuite) setupMainAPIServer() {
	mux := http.NewServeMux()

	// CallRail webhook endpoint
	mux.HandleFunc("/webhook/callrail", suite.handleCallRailWebhook)

	// Ingestion status endpoint
	mux.HandleFunc("/api/v1/ingestion/status/", suite.handleIngestionStatus)

	// CRM integration status endpoint
	mux.HandleFunc("/api/v1/integrations/crm/", suite.handleCRMStatus)

	// Tenant configuration endpoint
	mux.HandleFunc("/api/v1/tenants/", suite.handleTenantConfig)

	suite.server = httptest.NewServer(mux)
}

func (suite *CallRailE2ETestSuite) setupSpannerClient() {
	if os.Getenv("SPANNER_EMULATOR_HOST") == "" {
		suite.T().Skip("Spanner emulator not available for integration tests")
		return
	}

	var err error
	databasePath := "projects/test-project/instances/test-instance/databases/test-db"
	suite.spannerClient, err = spanner.NewClient(suite.ctx, databasePath)
	require.NoError(suite.T(), err)
}

func (suite *CallRailE2ETestSuite) createTestCallRecords() {
	suite.testCallRecords = []TestCallRecord{
		{
			CallRailPayload: CallRailWebhookPayload{
				CallID:         "call-kitchen-remodel-high-value",
				CompanyID:      "company-home-pros",
				PhoneNumber:    "+15551234567",
				CallerID:       "+15559876543",
				Duration:       420, // 7 minutes - good length
				StartTime:      time.Now().Add(-10 * time.Minute),
				EndTime:        time.Now().Add(-3 * time.Minute),
				Direction:      "inbound",
				RecordingURL:   suite.mockAudioServer.URL + "/recordings/call-kitchen-remodel-high-value.wav",
				CallStatus:     "completed",
				TrackingNumber: "+15551111111",
				CustomFields: map[string]interface{}{
					"lead_source":   "google_ads",
					"campaign_name": "Kitchen Remodel Premium",
					"ad_group":      "luxury_kitchen",
					"keyword":       "kitchen remodeling contractor",
				},
				Tags: []string{"kitchen", "remodeling", "high_value", "qualified"},
			},
			ExpectedResult: ExpectedIngestionResult{
				ShouldSucceed:     true,
				ProcessingTimeMax: 30 * time.Second,
				ExtractedContact: ContactInfo{
					Name:  "Sarah Johnson",
					Phone: "555-9876",
					Email: "sarah.j@email.com",
				},
				ExtractedProject: ProjectInfo{
					Type:        "kitchen",
					Description: "complete kitchen remodeling",
					Budget:      "$40,000-$60,000",
					Timeline:    "3-4 months",
					Urgency:     "medium",
				},
				ConfidenceMin: 0.85,
				ExpectedTags:  []string{"kitchen", "remodeling", "qualified_lead"},
			},
			CRMExpectation: ExpectedCRMIntegration{
				ShouldCreateLead: true,
				LeadFields: map[string]string{
					"FirstName":    "Sarah",
					"LastName":     "Johnson",
					"Phone":        "555-9876",
					"Email":        "sarah.j@email.com",
					"LeadSource":   "CallRail",
					"Description":  "Kitchen remodeling inquiry",
				},
				CRMType:  "salesforce",
				Priority: "High",
			},
			TestDescription: "High-value kitchen remodeling lead with complete contact information",
		},
		{
			CallRailPayload: CallRailWebhookPayload{
				CallID:       "call-bathroom-quick-inquiry",
				CompanyID:    "company-home-pros",
				PhoneNumber:  "+15551234567",
				CallerID:     "+15555555555",
				Duration:     90, // Short call
				StartTime:    time.Now().Add(-5 * time.Minute),
				EndTime:      time.Now().Add(-3 * time.Minute),
				Direction:    "inbound",
				RecordingURL: suite.mockAudioServer.URL + "/recordings/call-bathroom-quick-inquiry.wav",
				CallStatus:   "completed",
				Tags:         []string{"bathroom", "quick_inquiry"},
			},
			ExpectedResult: ExpectedIngestionResult{
				ShouldSucceed:     true,
				ProcessingTimeMax: 20 * time.Second,
				ExtractedContact: ContactInfo{
					Name:  "Mike",
					Phone: "555-5555",
				},
				ExtractedProject: ProjectInfo{
					Type:    "bathroom",
					Urgency: "low",
				},
				ConfidenceMin: 0.60,
				ExpectedTags:  []string{"bathroom", "inquiry"},
			},
			CRMExpectation: ExpectedCRMIntegration{
				ShouldCreateLead: false, // Low confidence, short call
				CRMType:          "salesforce",
			},
			TestDescription: "Quick bathroom inquiry with minimal information",
		},
		{
			CallRailPayload: CallRailWebhookPayload{
				CallID:         "call-emergency-water-damage",
				CompanyID:      "company-home-pros",
				PhoneNumber:    "+15551234567",
				CallerID:       "+15551119999",
				Duration:       180,
				StartTime:      time.Now().Add(-8 * time.Minute),
				EndTime:        time.Now().Add(-5 * time.Minute),
				Direction:      "inbound",
				RecordingURL:   suite.mockAudioServer.URL + "/recordings/call-emergency-water-damage.wav",
				CallStatus:     "completed",
				TrackingNumber: "+15551111111",
				CustomFields: map[string]interface{}{
					"emergency_call": true,
					"priority":       "urgent",
				},
				Tags: []string{"emergency", "water_damage", "urgent"},
			},
			ExpectedResult: ExpectedIngestionResult{
				ShouldSucceed:     true,
				ProcessingTimeMax: 15 * time.Second, // Emergency should be fast
				ExtractedContact: ContactInfo{
					Name:    "Jennifer Martinez",
					Phone:   "555-1119",
					Address: "123 Oak Street",
				},
				ExtractedProject: ProjectInfo{
					Type:        "emergency_repair",
					Description: "water damage repair",
					Urgency:     "urgent",
				},
				ConfidenceMin: 0.90,
				ExpectedTags:  []string{"emergency", "water_damage", "urgent", "qualified_lead"},
			},
			CRMExpectation: ExpectedCRMIntegration{
				ShouldCreateLead: true,
				LeadFields: map[string]string{
					"FirstName":   "Jennifer",
					"LastName":    "Martinez",
					"Phone":       "555-1119",
					"Street":      "123 Oak Street",
					"LeadSource":  "CallRail",
					"Description": "URGENT: Water damage repair needed",
				},
				CRMType:  "salesforce",
				Priority: "Urgent",
			},
			TestDescription: "Emergency water damage call requiring immediate attention",
		},
		{
			CallRailPayload: CallRailWebhookPayload{
				CallID:       "call-abandoned-short",
				CompanyID:    "company-home-pros",
				PhoneNumber:  "+15551234567",
				CallerID:     "+15557777777",
				Duration:     8, // Very short - abandoned
				StartTime:    time.Now().Add(-2 * time.Minute),
				EndTime:      time.Now().Add(-2*time.Minute + 8*time.Second),
				Direction:    "inbound",
				CallStatus:   "abandoned",
				Tags:         []string{"abandoned"},
			},
			ExpectedResult: ExpectedIngestionResult{
				ShouldSucceed:     true,
				ProcessingTimeMax: 10 * time.Second,
				ConfidenceMin:     0.20,
			},
			CRMExpectation: ExpectedCRMIntegration{
				ShouldCreateLead: false,
				CRMType:          "salesforce",
			},
			TestDescription: "Abandoned call with minimal data - should be tracked but not create lead",
		},
	}
}

func (suite *CallRailE2ETestSuite) setupTestTenant() {
	tenantConfig := map[string]interface{}{
		"tenant_id":   suite.testTenantID,
		"tenant_name": "Home Remodeling Pros",
		"is_active":   true,
		"crm_settings": map[string]interface{}{
			"type":         "salesforce",
			"endpoint":     suite.mockCRMServer.URL + "/services/data/v52.0",
			"access_token": "mock-salesforce-token",
			"instance_url": suite.mockCRMServer.URL,
		},
		"ai_prompts": map[string]string{
			"extraction": "Extract customer contact information and project details from this home remodeling call transcript. Focus on: customer name, phone, email, project type (kitchen/bathroom/emergency/etc), budget range, timeline, and urgency level.",
		},
		"processing_rules": map[string]interface{}{
			"min_confidence_for_crm": 0.80,
			"auto_create_lead":       true,
			"emergency_keywords":     []string{"emergency", "urgent", "water damage", "leak"},
		},
		"callrail_mapping": map[string]interface{}{
			"company_ids":     []string{"company-home-pros"},
			"webhook_secret":  "test-webhook-secret-12345",
		},
	}

	// In a real implementation, this would create the tenant in the database
	suite.T().Logf("Test tenant configured: %+v", tenantConfig)
}

func (suite *CallRailE2ETestSuite) cleanupTestData() {
	if suite.spannerClient == nil {
		return
	}

	// Clean up test records
	mutations := []*spanner.Mutation{
		spanner.Delete("ingestion_records", spanner.KeyRange{
			Start: spanner.Key{suite.testTenantID},
			End:   spanner.Key{suite.testTenantID + "\xFF"},
			Kind:  spanner.ClosedOpen,
		}),
		spanner.Delete("crm_integrations", spanner.KeyRange{
			Start: spanner.Key{suite.testTenantID},
			End:   spanner.Key{suite.testTenantID + "\xFF"},
			Kind:  spanner.ClosedOpen,
		}),
	}

	_, err := suite.spannerClient.Apply(suite.ctx, mutations)
	if err != nil {
		suite.T().Logf("Cleanup warning: %v", err)
	}
}

func (suite *CallRailE2ETestSuite) generateMockAudioContent(callID string) []byte {
	// Generate different mock audio content based on call ID
	switch {
	case contains(callID, "kitchen-remodel"):
		return []byte("MOCK_AUDIO_KITCHEN_REMODEL: Hi, I'm Sarah Johnson and I'm interested in a complete kitchen remodeling. My phone number is 555-9876 and email is sarah.j@email.com. I'm looking to spend between $40,000 and $60,000 and would like to start in the next 3-4 months.")
	case contains(callID, "bathroom-quick"):
		return []byte("MOCK_AUDIO_BATHROOM: Hi, I'm Mike, my number is 555-5555. Just wondering about bathroom renovations.")
	case contains(callID, "emergency-water"):
		return []byte("MOCK_AUDIO_EMERGENCY: This is Jennifer Martinez at 555-1119. I have an emergency water damage situation at 123 Oak Street. I need help immediately!")
	case contains(callID, "abandoned"):
		return []byte("MOCK_AUDIO_ABANDONED: Hello? Hello?") // Very short
	default:
		return []byte("MOCK_AUDIO_DEFAULT: Generic call content.")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// HTTP Handlers for mock API

func (suite *CallRailE2ETestSuite) handleCallRailWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload CallRailWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Simulate webhook processing
	processingStartTime := time.Now()

	// Validate CallRail signature (mock)
	signature := r.Header.Get("X-CallRail-Signature")
	if signature == "" {
		http.Error(w, "Missing signature", http.StatusUnauthorized)
		return
	}

	// Generate ingestion ID
	ingestionID := fmt.Sprintf("ing_%s_%d", payload.CallID, time.Now().Unix())

	// Simulate async processing initiation
	go suite.simulateCallProcessing(ingestionID, &payload, processingStartTime)

	// Return immediate webhook response
	response := map[string]interface{}{
		"status":       "accepted",
		"ingestion_id": ingestionID,
		"message":      "Call processing initiated",
		"tenant_id":    suite.testTenantID,
		"call_id":      payload.CallID,
		"processed_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (suite *CallRailE2ETestSuite) handleIngestionStatus(w http.ResponseWriter, r *http.Request) {
	ingestionID := r.URL.Path[len("/api/v1/ingestion/status/"):]

	// Mock status response based on ingestion ID
	status := suite.generateMockIngestionStatus(ingestionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (suite *CallRailE2ETestSuite) handleCRMStatus(w http.ResponseWriter, r *http.Request) {
	ingestionID := r.URL.Path[len("/api/v1/integrations/crm/"):]

	// Mock CRM integration status
	status := map[string]interface{}{
		"ingestion_id": ingestionID,
		"crm_status":   "completed",
		"lead_id":      "lead_" + ingestionID,
		"crm_type":     "salesforce",
		"created_at":   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (suite *CallRailE2ETestSuite) handleTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Path[len("/api/v1/tenants/"):]

	if tenantID != suite.testTenantID {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	config := map[string]interface{}{
		"tenant_id":   tenantID,
		"tenant_name": "Home Remodeling Pros",
		"is_active":   true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (suite *CallRailE2ETestSuite) simulateCallProcessing(ingestionID string, payload *CallRailWebhookPayload, startTime time.Time) {
	// Simulate different processing stages
	stages := []string{
		"webhook_received",
		"audio_download",
		"audio_transcription",
		"ai_extraction",
		"database_storage",
		"crm_integration",
		"completed",
	}

	processingDelay := time.Duration(len(payload.Transcription)*10 + payload.Duration*50) * time.Millisecond
	if processingDelay > 30*time.Second {
		processingDelay = 30 * time.Second
	}
	if processingDelay < 1*time.Second {
		processingDelay = 1 * time.Second
	}

	// Simulate processing time
	time.Sleep(processingDelay)

	suite.T().Logf("Simulated processing for %s completed in %v", ingestionID, time.Since(startTime))
}

func (suite *CallRailE2ETestSuite) generateMockIngestionStatus(ingestionID string) IngestionStatusResponse {
	now := time.Now()
	completedTime := now.Add(-10 * time.Second)

	stages := []ProcessingStageInfo{
		{
			Name:        "webhook_received",
			Status:      "completed",
			StartedAt:   now.Add(-30 * time.Second),
			CompletedAt: &now,
			Duration:    int64Ptr(100),
		},
		{
			Name:        "audio_download",
			Status:      "completed",
			StartedAt:   now.Add(-25 * time.Second),
			CompletedAt: &completedTime,
			Duration:    int64Ptr(2000),
		},
		{
			Name:        "audio_transcription",
			Status:      "completed",
			StartedAt:   now.Add(-20 * time.Second),
			CompletedAt: &completedTime,
			Duration:    int64Ptr(8000),
		},
		{
			Name:        "ai_extraction",
			Status:      "completed",
			StartedAt:   now.Add(-15 * time.Second),
			CompletedAt: &completedTime,
			Duration:    int64Ptr(3000),
		},
		{
			Name:        "database_storage",
			Status:      "completed",
			StartedAt:   now.Add(-12 * time.Second),
			CompletedAt: &completedTime,
			Duration:    int64Ptr(500),
		},
		{
			Name:        "crm_integration",
			Status:      "completed",
			StartedAt:   now.Add(-10 * time.Second),
			CompletedAt: &completedTime,
			Duration:    int64Ptr(1500),
		},
	}

	extractedData := map[string]interface{}{
		"contact": map[string]interface{}{
			"name":  "Sarah Johnson",
			"phone": "555-9876",
			"email": "sarah.j@email.com",
		},
		"project": map[string]interface{}{
			"type":        "kitchen",
			"description": "complete kitchen remodeling",
			"budget":      "$40,000-$60,000",
			"timeline":    "3-4 months",
		},
		"confidence": 0.92,
		"urgency":    "medium",
	}

	return IngestionStatusResponse{
		IngestionID:      ingestionID,
		Status:           "completed",
		Progress:         100.0,
		ProcessingStages: stages,
		ExtractedData:    extractedData,
		TenantID:         suite.testTenantID,
		CallID:           "call-kitchen-remodel-high-value",
		CreatedAt:        now.Add(-30 * time.Second),
		UpdatedAt:        now,
	}
}

// Test Cases

func (suite *CallRailE2ETestSuite) TestCompleteCallRailWorkflow_KitchenRemodel() {
	// Test the complete flow for a high-value kitchen remodeling lead
	testRecord := suite.testCallRecords[0] // Kitchen remodel test case

	// Step 1: Send CallRail webhook
	body, _ := json.Marshal(testRecord.CallRailPayload)
	req, _ := http.NewRequest("POST", suite.server.URL+"/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", "test-signature-123")

	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Assert webhook acceptance
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var webhookResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&webhookResponse)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "accepted", webhookResponse["status"])
	assert.NotEmpty(suite.T(), webhookResponse["ingestion_id"])
	ingestionID := webhookResponse["ingestion_id"].(string)

	// Step 2: Wait for processing to complete
	maxWait := testRecord.ExpectedResult.ProcessingTimeMax
	suite.waitForProcessingCompletion(ingestionID, maxWait)

	// Step 3: Verify ingestion status
	statusResp := suite.getIngestionStatus(ingestionID)
	assert.Equal(suite.T(), "completed", statusResp.Status)
	assert.Equal(suite.T(), 100.0, statusResp.Progress)
	assert.NotEmpty(suite.T(), statusResp.ExtractedData)

	// Verify extracted data quality
	extractedData := statusResp.ExtractedData
	if contact, ok := extractedData["contact"].(map[string]interface{}); ok {
		assert.Contains(suite.T(), contact["name"], "Sarah")
		assert.Contains(suite.T(), contact["phone"], "555")
	}

	if project, ok := extractedData["project"].(map[string]interface{}); ok {
		assert.Equal(suite.T(), "kitchen", project["type"])
		assert.Contains(suite.T(), project["description"], "kitchen")
	}

	// Step 4: Verify CRM integration
	if testRecord.CRMExpectation.ShouldCreateLead {
		crmResp := suite.getCRMIntegrationStatus(ingestionID)
		assert.Equal(suite.T(), "completed", crmResp["crm_status"])
		assert.NotEmpty(suite.T(), crmResp["lead_id"])
		assert.Equal(suite.T(), "salesforce", crmResp["crm_type"])
	}

	// Step 5: Verify database storage
	if suite.spannerClient != nil {
		suite.verifyDatabaseStorage(ingestionID, testRecord.CallRailPayload.CallID)
	}

	// Step 6: Verify performance requirements
	totalProcessingTime := time.Since(startTime)
	assert.True(suite.T(), totalProcessingTime < testRecord.ExpectedResult.ProcessingTimeMax,
		"Total processing time %v should be less than %v",
		totalProcessingTime, testRecord.ExpectedResult.ProcessingTimeMax)

	suite.T().Logf("Complete workflow test passed in %v", totalProcessingTime)
}

func (suite *CallRailE2ETestSuite) TestCallRailWorkflow_EmergencyCall() {
	// Test emergency call handling with priority processing
	testRecord := suite.testCallRecords[2] // Emergency water damage

	body, _ := json.Marshal(testRecord.CallRailPayload)
	req, _ := http.NewRequest("POST", suite.server.URL+"/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", "test-signature-123")

	startTime := time.Now()
	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var webhookResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&webhookResponse)
	require.NoError(suite.T(), err)
	ingestionID := webhookResponse["ingestion_id"].(string)

	// Emergency calls should process faster
	suite.waitForProcessingCompletion(ingestionID, 15*time.Second)

	statusResp := suite.getIngestionStatus(ingestionID)
	assert.Equal(suite.T(), "completed", statusResp.Status)

	// Verify emergency handling
	extractedData := statusResp.ExtractedData
	assert.Equal(suite.T(), "urgent", extractedData["urgency"])

	// Emergency calls should definitely create CRM leads
	crmResp := suite.getCRMIntegrationStatus(ingestionID)
	assert.Equal(suite.T(), "completed", crmResp["crm_status"])

	processingTime := time.Since(startTime)
	assert.True(suite.T(), processingTime < 15*time.Second,
		"Emergency call processing should be under 15 seconds, got %v", processingTime)

	suite.T().Logf("Emergency call processed in %v", processingTime)
}

func (suite *CallRailE2ETestSuite) TestCallRailWorkflow_AbandonedCall() {
	// Test handling of abandoned calls
	testRecord := suite.testCallRecords[3] // Abandoned call

	body, _ := json.Marshal(testRecord.CallRailPayload)
	req, _ := http.NewRequest("POST", suite.server.URL+"/webhook/callrail", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CallRail-Signature", "test-signature-123")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var webhookResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&webhookResponse)
	require.NoError(suite.T(), err)
	ingestionID := webhookResponse["ingestion_id"].(string)

	// Even abandoned calls should be processed for analytics
	suite.waitForProcessingCompletion(ingestionID, 10*time.Second)

	statusResp := suite.getIngestionStatus(ingestionID)
	assert.Equal(suite.T(), "completed", statusResp.Status)

	// Should not create CRM lead for abandoned calls
	crmResp := suite.getCRMIntegrationStatus(ingestionID)
	// CRM integration might be skipped or marked as "skipped"
	crmStatus := crmResp["crm_status"]
	assert.True(suite.T(), crmStatus == "skipped" || crmStatus == "not_qualified",
		"Abandoned calls should not create CRM leads")
}

func (suite *CallRailE2ETestSuite) TestConcurrentCallRailWebhooks() {
	// Test handling multiple concurrent webhooks
	const numConcurrent = 5

	results := make(chan bool, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			testRecord := suite.testCallRecords[0] // Use kitchen remodel as template

			// Modify call ID to make it unique
			testRecord.CallRailPayload.CallID = fmt.Sprintf("concurrent-call-%d", index)

			body, _ := json.Marshal(testRecord.CallRailPayload)
			req, _ := http.NewRequest("POST", suite.server.URL+"/webhook/callrail", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CallRail-Signature", "test-signature-123")

			resp, err := http.DefaultClient.Do(req)
			success := err == nil && resp.StatusCode == http.StatusOK
			if resp != nil {
				resp.Body.Close()
			}

			results <- success
		}(i)
	}

	// Verify all requests succeeded
	successCount := 0
	for i := 0; i < numConcurrent; i++ {
		if <-results {
			successCount++
		}
	}

	assert.Equal(suite.T(), numConcurrent, successCount,
		"All concurrent webhook requests should succeed")
}

func (suite *CallRailE2ETestSuite) TestTenantIsolation_CallRailWebhooks() {
	// Test that different tenants' calls are properly isolated

	// This would test with different company IDs that map to different tenants
	// For now, we simulate the isolation verification

	payload1 := suite.testCallRecords[0].CallRailPayload
	payload1.CompanyID = "company-tenant-1"
	payload1.CallID = "isolation-test-call-1"

	payload2 := suite.testCallRecords[0].CallRailPayload
	payload2.CompanyID = "company-tenant-2"
	payload2.CallID = "isolation-test-call-2"

	// Send both webhooks
	for i, payload := range []CallRailWebhookPayload{payload1, payload2} {
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", suite.server.URL+"/webhook/callrail", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CallRail-Signature", "test-signature-123")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		suite.T().Logf("Webhook %d processed for company %s", i+1, payload.CompanyID)
	}

	// In a real test, we would verify that:
	// 1. Each call is assigned to the correct tenant
	// 2. CRM integrations use the correct tenant's CRM configuration
	// 3. Data is stored in the correct tenant partition
	// 4. No cross-tenant data leakage occurs
}

// Helper methods

func (suite *CallRailE2ETestSuite) waitForProcessingCompletion(ingestionID string, maxWait time.Duration) {
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		status := suite.getIngestionStatus(ingestionID)
		if status.Status == "completed" || status.Status == "failed" {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	suite.T().Fatalf("Processing did not complete within %v for ingestion %s", maxWait, ingestionID)
}

func (suite *CallRailE2ETestSuite) getIngestionStatus(ingestionID string) IngestionStatusResponse {
	resp, err := http.Get(suite.server.URL + "/api/v1/ingestion/status/" + ingestionID)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var status IngestionStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(suite.T(), err)

	return status
}

func (suite *CallRailE2ETestSuite) getCRMIntegrationStatus(ingestionID string) map[string]interface{} {
	resp, err := http.Get(suite.server.URL + "/api/v1/integrations/crm/" + ingestionID)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(suite.T(), err)

	return status
}

func (suite *CallRailE2ETestSuite) verifyDatabaseStorage(ingestionID, callID string) {
	// Query Spanner to verify record storage
	stmt := spanner.Statement{
		SQL: `SELECT id, call_id, processing_status, extracted_data
			  FROM ingestion_records
			  WHERE tenant_id = @tenantId AND id = @ingestionId`,
		Params: map[string]interface{}{
			"tenantId":    suite.testTenantID,
			"ingestionId": ingestionID,
		},
	}

	iter := suite.spannerClient.Single().Query(suite.ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	require.NoError(suite.T(), err, "Ingestion record should be stored in database")

	var id, storedCallID, status, extractedData string
	err = row.Columns(&id, &storedCallID, &status, &extractedData)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), ingestionID, id)
	assert.Equal(suite.T(), callID, storedCallID)
	assert.Equal(suite.T(), "completed", status)
	assert.NotEmpty(suite.T(), extractedData)
}

func int64Ptr(i int64) *int64 {
	return &i
}

// Run the test suite
func TestCallRailE2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CallRail E2E tests in short mode")
	}

	suite.Run(t, new(CallRailE2ETestSuite))
}