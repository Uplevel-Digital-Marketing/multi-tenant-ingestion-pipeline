package integration

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"cloud.google.com/go/spanner"

	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
	"github.com/home-renovators/ingestion-pipeline/internal/callrail"
	"github.com/home-renovators/ingestion-pipeline/internal/spanner"
	"github.com/home-renovators/ingestion-pipeline/internal/auth"
)

// CallRailWebhookIntegrationTestSuite tests the complete CallRail webhook integration
type CallRailWebhookIntegrationTestSuite struct {
	suite.Suite
	ctx             context.Context
	spannerClient   *spanner.Client
	spannerDB       *spannerdb.DB
	callrailClient  *callrail.Client
	authService     *auth.Service
	config          *config.Config
	testTenantID    string
	testWebhookKey  string
}

func (suite *CallRailWebhookIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testTenantID = "tenant_integration_test"
	suite.testWebhookKey = "test_webhook_secret_key_12345"

	// Setup test configuration
	suite.config = &config.Config{
		ProjectID:         "test-project",
		SpannerInstance:   "test-instance",
		SpannerDatabase:   "test-database",
		CallRailBaseURL:   "https://api.callrail.com/v3",
		VertexAIProject:   "test-vertex-project",
		VertexAILocation:  "us-central1",
		VertexAIModel:     "gemini-2.0-flash-exp",
		SpeechToTextModel: "chirp-3",
		SpeechLanguage:    "en-US",
		EnableDiarization: true,
	}

	// Initialize Spanner client for testing
	var err error
	suite.spannerClient, err = spanner.NewClient(suite.ctx,
		fmt.Sprintf("projects/%s/instances/%s/databases/%s",
			suite.config.ProjectID, suite.config.SpannerInstance, suite.config.SpannerDatabase))
	require.NoError(suite.T(), err)

	// Initialize database service
	suite.spannerDB = spannerdb.NewDB(suite.spannerClient)

	// Initialize CallRail client
	suite.callrailClient = callrail.NewClient()

	// Initialize auth service
	suite.authService = auth.NewService(suite.spannerDB)

	// Setup test tenant
	suite.setupTestTenant()
}

func (suite *CallRailWebhookIntegrationTestSuite) TearDownSuite() {
	// Cleanup test data
	suite.cleanupTestTenant()

	if suite.spannerClient != nil {
		suite.spannerClient.Close()
	}
}

func (suite *CallRailWebhookIntegrationTestSuite) SetupTest() {
	// Clean up any existing test data before each test
	suite.cleanupTestCalls()
}

// setupTestTenant creates a test tenant in the database
func (suite *CallRailWebhookIntegrationTestSuite) setupTestTenant() {
	workflowConfig := models.WorkflowConfig{
		CommunicationDetection: models.CommunicationDetectionConfig{
			Enabled: true,
			PhoneProcessing: models.PhoneProcessingConfig{
				TranscribeAudio:    true,
				ExtractDetails:     true,
				SentimentAnalysis:  true,
				SpeakerDiarization: true,
			},
		},
		Validation: models.ValidationConfig{
			SpamDetection: models.SpamDetectionConfig{
				Enabled:             true,
				ConfidenceThreshold: 50,
				MLModel:            "gemini-2.0-flash-exp",
			},
		},
		ServiceArea: models.ServiceAreaConfig{
			Enabled:          true,
			ValidationMethod: "geocode",
			AllowedAreas:     []string{"90210", "90211", "90212"},
			BufferMiles:      25,
		},
		CRMIntegration: models.CRMIntegrationConfig{
			Enabled:  true,
			Provider: "hubspot",
			FieldMapping: map[string]string{
				"name":         "firstname",
				"phone":        "phone",
				"lead_score":   "hs_lead_score",
				"project_type": "custom_project_type",
			},
			PushImmediately: true,
		},
		EmailNotifications: models.EmailNotificationsConfig{
			Enabled:    true,
			Recipients: []string{"test@example.com"},
			Conditions: models.EmailConditionsConfig{
				MinLeadScore: 30,
			},
		},
	}

	configJSON, err := json.Marshal(workflowConfig)
	require.NoError(suite.T(), err)

	office := &models.Office{
		TenantID:          suite.testTenantID,
		OfficeID:         "office_test_001",
		CallRailCompanyID: "CR123456789",
		CallRailAPIKey:    "test_api_key",
		WorkflowConfig:    string(configJSON),
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err = suite.spannerDB.CreateOffice(suite.ctx, office)
	require.NoError(suite.T(), err)
}

// cleanupTestTenant removes test tenant data
func (suite *CallRailWebhookIntegrationTestSuite) cleanupTestTenant() {
	// Delete test office
	suite.spannerDB.DeleteOffice(suite.ctx, suite.testTenantID, "office_test_001")
}

// cleanupTestCalls removes test call data
func (suite *CallRailWebhookIntegrationTestSuite) cleanupTestCalls() {
	// Delete test requests for this tenant
	suite.spannerDB.DeleteRequestsByTenant(suite.ctx, suite.testTenantID)
}

// TestCallRailWebhookHMACVerification tests HMAC signature verification
func (suite *CallRailWebhookIntegrationTestSuite) TestCallRailWebhookHMACVerification() {
	// Test case 1: Valid HMAC signature
	suite.T().Run("ValidHMACSignature", func(t *testing.T) {
		payload := models.CallRailWebhook{
			CallID:                "CAL123456789",
			AccountID:             "AC987654321",
			CompanyID:             "CR123456789",
			CallerID:              "+15551234567",
			CalledNumber:          "+15559876543",
			Duration:              "180",
			StartTime:             time.Now().Add(-5 * time.Minute),
			EndTime:               time.Now().Add(-2 * time.Minute),
			Direction:             "inbound",
			RecordingURL:          "https://api.callrail.com/v3/a/AC987654321/calls/CAL123456789/recording.json",
			Answered:              true,
			FirstCall:             true,
			Value:                 "0",
			BusinessPhoneNumber:   "+15559876543",
			CustomerName:          "John Doe",
			CustomerPhoneNumber:   "+15551234567",
			CustomerCity:          "Los Angeles",
			CustomerState:         "CA",
			CustomerCountry:       "US",
			LeadStatus:            "good_lead",
			TenantID:              suite.testTenantID,
			CallRailCompanyID:     "CR123456789",
		}

		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		// Generate valid HMAC signature
		validSignature := suite.generateHMACSignature(payloadBytes, suite.testWebhookKey)

		req := httptest.NewRequest("POST", "/api/v1/callrail/webhook", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CallRail-Signature", "sha256="+validSignature)

		// Verify signature
		isValid := suite.authService.VerifyCallRailSignature(payloadBytes, validSignature, suite.testWebhookKey)
		assert.True(t, isValid, "Valid HMAC signature should be verified successfully")
	})

	// Test case 2: Invalid HMAC signature
	suite.T().Run("InvalidHMACSignature", func(t *testing.T) {
		payload := models.CallRailWebhook{
			CallID:    "CAL123456789",
			CompanyID: "CR123456789",
		}

		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		invalidSignature := "invalid_signature_12345"

		// Verify signature should fail
		isValid := suite.authService.VerifyCallRailSignature(payloadBytes, invalidSignature, suite.testWebhookKey)
		assert.False(t, isValid, "Invalid HMAC signature should fail verification")
	})

	// Test case 3: Missing signature
	suite.T().Run("MissingSignature", func(t *testing.T) {
		payload := models.CallRailWebhook{
			CallID:    "CAL123456789",
			CompanyID: "CR123456789",
		}

		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		// Verify with empty signature
		isValid := suite.authService.VerifyCallRailSignature(payloadBytes, "", suite.testWebhookKey)
		assert.False(t, isValid, "Missing signature should fail verification")
	})
}

// TestCallRailTenantIsolation tests multi-tenant data isolation
func (suite *CallRailWebhookIntegrationTestSuite) TestCallRailTenantIsolation() {
	// Create additional test tenant
	tenant2ID := "tenant_isolation_test_2"
	suite.setupAdditionalTenant(tenant2ID)
	defer suite.cleanupAdditionalTenant(tenant2ID)

	// Test case 1: Tenant 1 webhook should only access tenant 1 data
	suite.T().Run("TenantDataIsolation", func(t *testing.T) {
		// Create request for tenant 1
		request1 := &models.Request{
			RequestID:         models.NewRequestID(),
			TenantID:          suite.testTenantID,
			Source:            "callrail_webhook",
			RequestType:       "call",
			Status:            "processed",
			Data:              `{"call_id":"CAL_TENANT1_001"}`,
			CallID:            stringPtr("CAL_TENANT1_001"),
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := suite.spannerDB.CreateRequest(suite.ctx, request1)
		require.NoError(t, err)

		// Create request for tenant 2
		request2 := &models.Request{
			RequestID:         models.NewRequestID(),
			TenantID:          tenant2ID,
			Source:            "callrail_webhook",
			RequestType:       "call",
			Status:            "processed",
			Data:              `{"call_id":"CAL_TENANT2_001"}`,
			CallID:            stringPtr("CAL_TENANT2_001"),
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err = suite.spannerDB.CreateRequest(suite.ctx, request2)
		require.NoError(t, err)

		// Verify tenant 1 can only access its own data
		tenant1Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, suite.testTenantID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(tenant1Requests))
		assert.Equal(t, "CAL_TENANT1_001", *tenant1Requests[0].CallID)

		// Verify tenant 2 can only access its own data
		tenant2Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, tenant2ID)
		require.NoError(t, err)
		assert.Equal(t, 1, len(tenant2Requests))
		assert.Equal(t, "CAL_TENANT2_001", *tenant2Requests[0].CallID)

		// Verify cross-tenant access is blocked
		crossTenantRequest, err := suite.spannerDB.GetRequestByID(suite.ctx, request2.RequestID, suite.testTenantID)
		assert.Error(t, err, "Cross-tenant access should be blocked")
		assert.Nil(t, crossTenantRequest)
	})
}

// TestCallRailWebhookEndToEnd tests complete webhook processing pipeline
func (suite *CallRailWebhookIntegrationTestSuite) TestCallRailWebhookEndToEnd() {
	suite.T().Run("CompleteCallProcessingWorkflow", func(t *testing.T) {
		// Arrange: Create realistic CallRail webhook payload
		payload := models.CallRailWebhook{
			CallID:                "CAL_E2E_12345",
			AccountID:             "AC_E2E_98765",
			CompanyID:             "CR123456789", // Matches test tenant
			CallerID:              "+15551234567",
			CalledNumber:          "+15559876543",
			Duration:              "240", // 4 minutes
			StartTime:             time.Now().Add(-10 * time.Minute),
			EndTime:               time.Now().Add(-6 * time.Minute),
			Direction:             "inbound",
			RecordingURL:          "https://api.callrail.com/v3/a/AC_E2E_98765/calls/CAL_E2E_12345/recording.json",
			Answered:              true,
			FirstCall:             true,
			Value:                 "150.00",
			GoodCall:              boolPtr(true),
			Tags:                  []string{"kitchen_remodel", "high_value", "qualified_lead"},
			Note:                  "Customer interested in complete kitchen renovation",
			BusinessPhoneNumber:   "+15559876543",
			CustomerName:          "Sarah Johnson",
			CustomerPhoneNumber:   "+15551234567",
			CustomerCity:          "Beverly Hills",
			CustomerState:         "CA",
			CustomerCountry:       "US",
			LeadStatus:            "good_lead",
			TenantID:              suite.testTenantID,
			CallRailCompanyID:     "CR123456789",
		}

		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		// Generate HMAC signature
		signature := suite.generateHMACSignature(payloadBytes, suite.testWebhookKey)

		// Act: Process webhook
		req := httptest.NewRequest("POST", "/api/v1/callrail/webhook", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CallRail-Signature", "sha256="+signature)

		// This would normally be handled by the actual webhook handler
		// For integration test, we'll simulate the complete flow

		// Step 1: Verify HMAC signature
		isValidSignature := suite.authService.VerifyCallRailSignature(payloadBytes, signature, suite.testWebhookKey)
		assert.True(t, isValidSignature, "HMAC signature verification should pass")

		// Step 2: Resolve tenant
		office, err := suite.spannerDB.GetOfficeByCallRailCompanyID(suite.ctx, payload.CallRailCompanyID)
		require.NoError(t, err)
		assert.Equal(t, suite.testTenantID, office.TenantID)

		// Step 3: Create request record
		requestID := models.NewRequestID()
		request := &models.Request{
			RequestID:         requestID,
			TenantID:          suite.testTenantID,
			Source:            "callrail_webhook",
			RequestType:       "call",
			Status:            "processing",
			Data:              string(payloadBytes),
			CallID:            &payload.CallID,
			RecordingURL:      &payload.RecordingURL,
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err = suite.spannerDB.CreateRequest(suite.ctx, request)
		require.NoError(t, err)

		// Step 4: Create call recording record
		recordingID := models.NewRecordingID()
		callRecording := &models.CallRecording{
			RecordingID:         recordingID,
			TenantID:            suite.testTenantID,
			CallID:              payload.CallID,
			StorageURL:          fmt.Sprintf("gs://test-audio-storage/%s/%s.mp3", suite.testTenantID, payload.CallID),
			TranscriptionStatus: "pending",
			CreatedAt:           time.Now(),
		}

		err = suite.spannerDB.CreateCallRecording(suite.ctx, callRecording)
		require.NoError(t, err)

		// Step 5: Create webhook event record
		eventID := models.NewEventID()
		webhookEvent := &models.WebhookEvent{
			EventID:          eventID,
			WebhookSource:    "callrail",
			CallID:           &payload.CallID,
			ProcessingStatus: "completed",
			CreatedAt:        time.Now(),
		}

		err = suite.spannerDB.CreateWebhookEvent(suite.ctx, webhookEvent)
		require.NoError(t, err)

		// Assert: Verify all records were created successfully

		// Verify request record
		storedRequest, err := suite.spannerDB.GetRequestByID(suite.ctx, requestID, suite.testTenantID)
		require.NoError(t, err)
		assert.Equal(t, "callrail_webhook", storedRequest.Source)
		assert.Equal(t, payload.CallID, *storedRequest.CallID)

		// Verify call recording record
		storedRecording, err := suite.spannerDB.GetCallRecording(suite.ctx, suite.testTenantID, payload.CallID)
		require.NoError(t, err)
		assert.Equal(t, recordingID, storedRecording.RecordingID)
		assert.Equal(t, "pending", storedRecording.TranscriptionStatus)

		// Verify webhook event record
		storedEvent, err := suite.spannerDB.GetWebhookEvent(suite.ctx, eventID)
		require.NoError(t, err)
		assert.Equal(t, "callrail", storedEvent.WebhookSource)
		assert.Equal(t, payload.CallID, *storedEvent.CallID)

		// Verify tenant isolation - other tenants cannot access this data
		_, err = suite.spannerDB.GetRequestByID(suite.ctx, requestID, "different_tenant")
		assert.Error(t, err, "Cross-tenant access should be blocked")
	})
}

// TestCallRailWebhookPerformance tests webhook processing latency (<200ms requirement)
func (suite *CallRailWebhookIntegrationTestSuite) TestCallRailWebhookPerformance() {
	suite.T().Run("WebhookLatencyUnder200ms", func(t *testing.T) {
		payload := models.CallRailWebhook{
			CallID:            "CAL_PERF_TEST",
			CompanyID:         "CR123456789",
			CallerID:          "+15551234567",
			Duration:          "120",
			TenantID:          suite.testTenantID,
			CallRailCompanyID: "CR123456789",
		}

		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		// Measure processing time
		startTime := time.Now()

		// Simulate webhook processing steps
		// Step 1: HMAC verification
		signature := suite.generateHMACSignature(payloadBytes, suite.testWebhookKey)
		isValid := suite.authService.VerifyCallRailSignature(payloadBytes, signature, suite.testWebhookKey)
		require.True(t, isValid)

		// Step 2: Tenant resolution
		office, err := suite.spannerDB.GetOfficeByCallRailCompanyID(suite.ctx, payload.CallRailCompanyID)
		require.NoError(t, err)

		// Step 3: Create database records
		requestID := models.NewRequestID()
		request := &models.Request{
			RequestID:         requestID,
			TenantID:          office.TenantID,
			Source:            "callrail_webhook",
			RequestType:       "call",
			Status:            "processing",
			Data:              string(payloadBytes),
			CallID:            &payload.CallID,
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err = suite.spannerDB.CreateRequest(suite.ctx, request)
		require.NoError(t, err)

		processingTime := time.Since(startTime)

		// Assert performance requirement
		assert.True(t, processingTime < 200*time.Millisecond,
			"Webhook processing should complete within 200ms, got %v", processingTime)

		suite.T().Logf("Webhook processing time: %v", processingTime)
	})
}

// TestCallRailWebhookErrorHandling tests various error scenarios
func (suite *CallRailWebhookIntegrationTestSuite) TestCallRailWebhookErrorHandling() {
	// Test case 1: Unknown tenant/company ID
	suite.T().Run("UnknownTenantError", func(t *testing.T) {
		payload := models.CallRailWebhook{
			CallID:            "CAL_UNKNOWN_TENANT",
			CompanyID:         "CR_UNKNOWN_123",
			CallRailCompanyID: "CR_UNKNOWN_123",
		}

		// Should fail to resolve tenant
		office, err := suite.spannerDB.GetOfficeByCallRailCompanyID(suite.ctx, payload.CallRailCompanyID)
		assert.Error(t, err)
		assert.Nil(t, office)
	})

	// Test case 2: Malformed JSON payload
	suite.T().Run("MalformedJSONPayload", func(t *testing.T) {
		malformedJSON := `{"call_id": "CAL123", "invalid_json": }`

		var payload models.CallRailWebhook
		err := json.Unmarshal([]byte(malformedJSON), &payload)
		assert.Error(t, err, "Malformed JSON should cause unmarshal error")
	})

	// Test case 3: Database connection failure simulation
	suite.T().Run("DatabaseErrorHandling", func(t *testing.T) {
		// This would typically involve closing the database connection
		// and verifying graceful error handling
		// For this test, we'll verify error propagation

		invalidRequest := &models.Request{
			RequestID: "", // Invalid empty request ID
			TenantID:  suite.testTenantID,
			Source:    "callrail_webhook",
		}

		err := suite.spannerDB.CreateRequest(suite.ctx, invalidRequest)
		assert.Error(t, err, "Invalid request should cause database error")
	})
}

// Helper functions

func (suite *CallRailWebhookIntegrationTestSuite) generateHMACSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func (suite *CallRailWebhookIntegrationTestSuite) setupAdditionalTenant(tenantID string) {
	workflowConfig := models.WorkflowConfig{
		CommunicationDetection: models.CommunicationDetectionConfig{
			Enabled: true,
			PhoneProcessing: models.PhoneProcessingConfig{
				TranscribeAudio: true,
			},
		},
	}

	configJSON, _ := json.Marshal(workflowConfig)

	office := &models.Office{
		TenantID:          tenantID,
		OfficeID:         "office_test_002",
		CallRailCompanyID: "CR987654321",
		CallRailAPIKey:    "test_api_key_2",
		WorkflowConfig:    string(configJSON),
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	suite.spannerDB.CreateOffice(suite.ctx, office)
}

func (suite *CallRailWebhookIntegrationTestSuite) cleanupAdditionalTenant(tenantID string) {
	suite.spannerDB.DeleteOffice(suite.ctx, tenantID, "office_test_002")
	suite.spannerDB.DeleteRequestsByTenant(suite.ctx, tenantID)
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

// Run the test suite
func TestCallRailWebhookIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CallRailWebhookIntegrationTestSuite))
}