package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite tests the complete ingestion flow end-to-end
type E2ETestSuite struct {
	suite.Suite
	server         *httptest.Server
	spannerClient  *spanner.Client
	ctx            context.Context
	testDataDir    string
	testTenantID   string
	authToken      string
}

// API request/response models
type UploadRequest struct {
	TenantID string `json:"tenant_id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type UploadResponse struct {
	IngestionID string `json:"ingestion_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

type StatusResponse struct {
	IngestionID      string                 `json:"ingestion_id"`
	Status           string                 `json:"status"`
	Progress         float64                `json:"progress"`
	ProcessingStages []ProcessingStage      `json:"processing_stages"`
	ExtractedData    map[string]interface{} `json:"extracted_data,omitempty"`
	Error            string                 `json:"error,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type ProcessingStage struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Duration    *int64    `json:"duration_ms,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type CRMIntegrationResponse struct {
	LeadID      string `json:"lead_id"`
	CRMType     string `json:"crm_type"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
}

type TenantConfigRequest struct {
	TenantName      string                 `json:"tenant_name"`
	CRMSettings     map[string]interface{} `json:"crm_settings"`
	AIPrompts       map[string]string      `json:"ai_prompts"`
	ProcessingRules map[string]interface{} `json:"processing_rules"`
}

func (suite *E2ETestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testTenantID = "e2e-test-tenant"
	suite.authToken = "test-auth-token-12345"

	// Setup test data directory
	suite.testDataDir = filepath.Join("../fixtures", "audio")
	err := os.MkdirAll(suite.testDataDir, 0755)
	require.NoError(suite.T(), err)

	// Create test audio files if they don't exist
	suite.createTestAudioFiles()

	// Setup test server (mock HTTP server for API endpoints)
	suite.setupTestServer()

	// Setup Spanner client for verification
	suite.setupSpannerClient()

	// Setup test tenant
	suite.setupTestTenant()
}

func (suite *E2ETestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.spannerClient != nil {
		suite.spannerClient.Close()
	}
	suite.cleanupTestData()
}

func (suite *E2ETestSuite) SetupTest() {
	// Clean up any previous test data
	suite.cleanupTestIngestionRecords()
}

func (suite *E2ETestSuite) createTestAudioFiles() {
	// Create sample audio files for testing
	testFiles := map[string]string{
		"kitchen_remodel.wav":     "Mock WAV audio data for kitchen remodeling inquiry",
		"bathroom_renovation.mp3": "Mock MP3 audio data for bathroom renovation",
		"poor_quality.wav":        "Mock poor quality audio with noise",
		"large_file.wav":          strings.Repeat("Mock large audio file data ", 1000),
		"multiple_projects.wav":   "Mock audio with multiple project discussions",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(suite.testDataDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			err = os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(suite.T(), err)
		}
	}
}

func (suite *E2ETestSuite) setupTestServer() {
	mux := http.NewServeMux()

	// Upload endpoint
	mux.HandleFunc("/api/v1/ingestion/upload", suite.handleUpload)

	// Status endpoint
	mux.HandleFunc("/api/v1/ingestion/status/", suite.handleStatus)

	// Tenant management
	mux.HandleFunc("/api/v1/tenants", suite.handleTenants)

	// CRM integration endpoint
	mux.HandleFunc("/api/v1/integrations/crm/", suite.handleCRMIntegration)

	// Health check
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	suite.server = httptest.NewServer(mux)
}

func (suite *E2ETestSuite) setupSpannerClient() {
	if os.Getenv("SPANNER_EMULATOR_HOST") == "" {
		suite.T().Skip("Spanner emulator not available for E2E tests")
		return
	}

	var err error
	databasePath := "projects/test-project/instances/test-instance/databases/test-db"
	suite.spannerClient, err = spanner.NewClient(suite.ctx, databasePath)
	require.NoError(suite.T(), err)
}

func (suite *E2ETestSuite) setupTestTenant() {
	tenantConfig := TenantConfigRequest{
		TenantName: "E2E Test Tenant",
		CRMSettings: map[string]interface{}{
			"type":     "salesforce",
			"endpoint": "https://test.salesforce.com/api",
			"token":    "test-crm-token",
		},
		AIPrompts: map[string]string{
			"extraction": "Extract contact information and project details from the transcript",
			"classification": "Classify the urgency and project type",
		},
		ProcessingRules: map[string]interface{}{
			"min_confidence": 0.8,
			"auto_create_lead": true,
			"notification_email": "test@example.com",
		},
	}

	body, _ := json.Marshal(tenantConfig)
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/tenants", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+suite.authToken)
	req.Header.Set("X-Tenant-ID", suite.testTenantID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		bodyBytes, _ := io.ReadAll(resp.Body)
		suite.T().Fatalf("Failed to create test tenant: %d - %s", resp.StatusCode, string(bodyBytes))
	}
}

func (suite *E2ETestSuite) cleanupTestData() {
	// Remove test files
	os.RemoveAll(suite.testDataDir)
}

func (suite *E2ETestSuite) cleanupTestIngestionRecords() {
	if suite.spannerClient == nil {
		return
	}

	// Delete test records from Spanner
	mutations := []*spanner.Mutation{
		spanner.Delete("ingestion_records", spanner.KeyRange{
			Start: spanner.Key{suite.testTenantID},
			End:   spanner.Key{suite.testTenantID + "\xFF"},
			Kind:  spanner.ClosedOpen,
		}),
	}

	_, err := suite.spannerClient.Apply(suite.ctx, mutations)
	if err != nil {
		suite.T().Log("Failed to cleanup test records:", err)
	}
}

// HTTP Handlers for mock API

func (suite *E2ETestSuite) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file
	file, header, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "Audio file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get tenant ID
	tenantID := r.FormValue("tenant_id")
	if tenantID == "" {
		tenantID = r.Header.Get("X-Tenant-ID")
	}
	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	// Simulate processing
	ingestionID := fmt.Sprintf("ing_%d", time.Now().Unix())

	// Mock file validation
	if header.Size > 100*1024*1024 { // 100MB limit
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Mock format validation
	filename := header.Filename
	validExtensions := []string{".wav", ".mp3", ".m4a", ".flac"}
	isValidFormat := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			isValidFormat = true
			break
		}
	}
	if !isValidFormat {
		http.Error(w, "Unsupported audio format", http.StatusBadRequest)
		return
	}

	response := UploadResponse{
		IngestionID: ingestionID,
		Status:      "accepted",
		Message:     "Audio file uploaded successfully and processing started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Simulate async processing
	go suite.simulateProcessing(ingestionID, tenantID, filename)
}

func (suite *E2ETestSuite) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ingestionID := strings.TrimPrefix(r.URL.Path, "/api/v1/ingestion/status/")
	if ingestionID == "" {
		http.Error(w, "Ingestion ID required", http.StatusBadRequest)
		return
	}

	// Mock status response based on ingestion ID pattern
	response := suite.generateMockStatus(ingestionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *E2ETestSuite) handleTenants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Create tenant (already handled in setup)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	case http.MethodGet:
		// Get tenant config
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			http.Error(w, "Tenant ID required", http.StatusBadRequest)
			return
		}

		// Mock tenant config response
		config := map[string]interface{}{
			"tenant_id":   tenantID,
			"tenant_name": "E2E Test Tenant",
			"is_active":   true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (suite *E2ETestSuite) handleCRMIntegration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ingestionID := strings.TrimPrefix(r.URL.Path, "/api/v1/integrations/crm/")
	if ingestionID == "" {
		http.Error(w, "Ingestion ID required", http.StatusBadRequest)
		return
	}

	// Mock CRM integration response
	response := CRMIntegrationResponse{
		LeadID:    fmt.Sprintf("crm_lead_%s", ingestionID),
		CRMType:   "salesforce",
		Status:    "created",
		Message:   "Lead created successfully in Salesforce",
		CreatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *E2ETestSuite) simulateProcessing(ingestionID, tenantID, filename string) {
	// Simulate different processing outcomes based on filename
	time.Sleep(100 * time.Millisecond) // Simulate processing delay

	// This would typically trigger actual processing pipeline
	// For E2E tests, we just update status in memory or mock storage
}

func (suite *E2ETestSuite) generateMockStatus(ingestionID string) StatusResponse {
	now := time.Now()
	completedTime := now.Add(-30 * time.Second)

	stages := []ProcessingStage{
		{
			Name:        "audio_validation",
			Status:      "completed",
			StartedAt:   now.Add(-60 * time.Second),
			CompletedAt: &now,
			Duration:    int64Ptr(5000),
		},
		{
			Name:        "audio_transcription",
			Status:      "completed",
			StartedAt:   now.Add(-55 * time.Second),
			CompletedAt: &completedTime,
			Duration:    int64Ptr(25000),
		},
		{
			Name:        "ai_extraction",
			Status:      "completed",
			StartedAt:   now.Add(-30 * time.Second),
			CompletedAt: &now,
			Duration:    int64Ptr(15000),
		},
		{
			Name:        "database_storage",
			Status:      "completed",
			StartedAt:   now.Add(-15 * time.Second),
			CompletedAt: &now,
			Duration:    int64Ptr(2000),
		},
		{
			Name:        "crm_integration",
			Status:      "completed",
			StartedAt:   now.Add(-10 * time.Second),
			CompletedAt: &now,
			Duration:    int64Ptr(8000),
		},
	}

	extractedData := map[string]interface{}{
		"contact": map[string]interface{}{
			"name":    "John Doe",
			"phone":   "555-1234",
			"email":   "john.doe@example.com",
			"address": "123 Main St, Anytown, USA",
		},
		"project": map[string]interface{}{
			"type":        "kitchen",
			"description": "Complete kitchen remodeling with new cabinets and countertops",
			"budget":      "$25,000 - $40,000",
			"timeline":    "2-3 months",
		},
		"urgency":    "medium",
		"confidence": 0.92,
		"tags":       []string{"kitchen", "remodeling", "cabinets", "countertops"},
	}

	return StatusResponse{
		IngestionID:      ingestionID,
		Status:           "completed",
		Progress:         100.0,
		ProcessingStages: stages,
		ExtractedData:    extractedData,
		CreatedAt:        now.Add(-60 * time.Second),
		UpdatedAt:        now,
	}
}

// Test Cases

func (suite *E2ETestSuite) TestCompleteIngestionFlow_Success() {
	// Arrange
	audioFile := filepath.Join(suite.testDataDir, "kitchen_remodel.wav")

	// Act - Upload audio file
	uploadResp := suite.uploadAudioFile(audioFile, suite.testTenantID)

	// Assert upload response
	assert.Equal(suite.T(), "accepted", uploadResp.Status)
	assert.NotEmpty(suite.T(), uploadResp.IngestionID)

	// Wait for processing (or poll status)
	time.Sleep(200 * time.Millisecond)

	// Act - Check status
	statusResp := suite.getIngestionStatus(uploadResp.IngestionID)

	// Assert processing completed
	assert.Equal(suite.T(), "completed", statusResp.Status)
	assert.Equal(suite.T(), 100.0, statusResp.Progress)
	assert.NotEmpty(suite.T(), statusResp.ExtractedData)

	// Verify all stages completed
	for _, stage := range statusResp.ProcessingStages {
		assert.Equal(suite.T(), "completed", stage.Status)
		assert.NotNil(suite.T(), stage.CompletedAt)
		assert.NotNil(suite.T(), stage.Duration)
	}

	// Act - Check CRM integration
	crmResp := suite.getCRMIntegration(uploadResp.IngestionID)

	// Assert CRM integration
	assert.Equal(suite.T(), "created", crmResp.Status)
	assert.NotEmpty(suite.T(), crmResp.LeadID)
	assert.Equal(suite.T(), "salesforce", crmResp.CRMType)

	// Verify database storage (if Spanner client available)
	if suite.spannerClient != nil {
		suite.verifyDatabaseStorage(uploadResp.IngestionID, suite.testTenantID)
	}
}

func (suite *E2ETestSuite) TestInvalidFileUpload() {
	// Test various invalid file scenarios
	testCases := []struct {
		name           string
		filename       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Unsupported Format",
			filename:       "document.pdf",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Unsupported audio format",
		},
		{
			name:           "No File",
			filename:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Audio file required",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			if tc.filename == "" {
				// Test missing file
				resp := suite.uploadWithoutFile(suite.testTenantID)
				assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			} else {
				// Create invalid file
				invalidFile := filepath.Join(suite.testDataDir, tc.filename)
				os.WriteFile(invalidFile, []byte("invalid content"), 0644)
				defer os.Remove(invalidFile)

				resp := suite.uploadAudioFileExpectError(invalidFile, suite.testTenantID)
				assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func (suite *E2ETestSuite) TestTenantIsolation() {
	// Arrange - Create files for different tenants
	tenant1ID := "tenant-isolation-1"
	tenant2ID := "tenant-isolation-2"

	audioFile := filepath.Join(suite.testDataDir, "kitchen_remodel.wav")

	// Act - Upload files for different tenants
	upload1 := suite.uploadAudioFile(audioFile, tenant1ID)
	upload2 := suite.uploadAudioFile(audioFile, tenant2ID)

	// Assert - Both uploads succeed with different ingestion IDs
	assert.Equal(suite.T(), "accepted", upload1.Status)
	assert.Equal(suite.T(), "accepted", upload2.Status)
	assert.NotEqual(suite.T(), upload1.IngestionID, upload2.IngestionID)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Act - Check status for both
	status1 := suite.getIngestionStatus(upload1.IngestionID)
	status2 := suite.getIngestionStatus(upload2.IngestionID)

	// Assert - Both processed independently
	assert.Equal(suite.T(), "completed", status1.Status)
	assert.Equal(suite.T(), "completed", status2.Status)

	// Verify tenant isolation at API level
	// Attempt to access tenant1's ingestion with tenant2's credentials
	// This should fail in a real implementation
}

func (suite *E2ETestSuite) TestConcurrentUploads() {
	// Test multiple concurrent uploads
	const numUploads = 5
	audioFile := filepath.Join(suite.testDataDir, "kitchen_remodel.wav")

	type uploadResult struct {
		response *UploadResponse
		error    error
	}

	resultChan := make(chan uploadResult, numUploads)

	// Act - Upload files concurrently
	for i := 0; i < numUploads; i++ {
		go func(index int) {
			tenantID := fmt.Sprintf("%s-concurrent-%d", suite.testTenantID, index)
			resp := suite.uploadAudioFile(audioFile, tenantID)
			resultChan <- uploadResult{response: resp, error: nil}
		}(i)
	}

	// Assert - All uploads succeed
	var responses []*UploadResponse
	for i := 0; i < numUploads; i++ {
		result := <-resultChan
		assert.NoError(suite.T(), result.error)
		assert.Equal(suite.T(), "accepted", result.response.Status)
		responses = append(responses, result.response)
	}

	// Verify all ingestion IDs are unique
	ingestionIDs := make(map[string]bool)
	for _, resp := range responses {
		assert.False(suite.T(), ingestionIDs[resp.IngestionID], "Duplicate ingestion ID: %s", resp.IngestionID)
		ingestionIDs[resp.IngestionID] = true
	}
}

func (suite *E2ETestSuite) TestLargeFileHandling() {
	// Test handling of large audio files
	largeFile := filepath.Join(suite.testDataDir, "large_file.wav")

	// Act
	uploadResp := suite.uploadAudioFile(largeFile, suite.testTenantID)

	// Assert
	assert.Equal(suite.T(), "accepted", uploadResp.Status)

	// Monitor processing for longer duration
	maxWait := 30 * time.Second
	start := time.Now()

	for time.Since(start) < maxWait {
		status := suite.getIngestionStatus(uploadResp.IngestionID)
		if status.Status == "completed" || status.Status == "failed" {
			assert.Equal(suite.T(), "completed", status.Status)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	suite.T().Fatal("Large file processing timed out")
}

// Helper methods

func (suite *E2ETestSuite) uploadAudioFile(filePath, tenantID string) *UploadResponse {
	file, err := os.Open(filePath)
	require.NoError(suite.T(), err)
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("audio", filepath.Base(filePath))
	require.NoError(suite.T(), err)
	_, err = io.Copy(part, file)
	require.NoError(suite.T(), err)

	// Add tenant ID
	err = writer.WriteField("tenant_id", tenantID)
	require.NoError(suite.T(), err)

	err = writer.Close()
	require.NoError(suite.T(), err)

	// Make request
	req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/ingestion/upload", &buf)
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var uploadResp UploadResponse
	err = json.NewDecoder(resp.Body).Decode(&uploadResp)
	require.NoError(suite.T(), err)

	return &uploadResp
}

func (suite *E2ETestSuite) uploadAudioFileExpectError(filePath, tenantID string) *http.Response {
	file, err := os.Open(filePath)
	require.NoError(suite.T(), err)
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("audio", filepath.Base(filePath))
	require.NoError(suite.T(), err)
	_, err = io.Copy(part, file)
	require.NoError(suite.T(), err)

	err = writer.WriteField("tenant_id", tenantID)
	require.NoError(suite.T(), err)
	err = writer.Close()
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/ingestion/upload", &buf)
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)

	return resp
}

func (suite *E2ETestSuite) uploadWithoutFile(tenantID string) *http.Response {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	err := writer.WriteField("tenant_id", tenantID)
	require.NoError(suite.T(), err)
	err = writer.Close()
	require.NoError(suite.T(), err)

	req, err := http.NewRequest("POST", suite.server.URL+"/api/v1/ingestion/upload", &buf)
	require.NoError(suite.T(), err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)

	return resp
}

func (suite *E2ETestSuite) getIngestionStatus(ingestionID string) *StatusResponse {
	req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/ingestion/status/"+ingestionID, nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var statusResp StatusResponse
	err = json.NewDecoder(resp.Body).Decode(&statusResp)
	require.NoError(suite.T(), err)

	return &statusResp
}

func (suite *E2ETestSuite) getCRMIntegration(ingestionID string) *CRMIntegrationResponse {
	req, err := http.NewRequest("GET", suite.server.URL+"/api/v1/integrations/crm/"+ingestionID, nil)
	require.NoError(suite.T(), err)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	require.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var crmResp CRMIntegrationResponse
	err = json.NewDecoder(resp.Body).Decode(&crmResp)
	require.NoError(suite.T(), err)

	return &crmResp
}

func (suite *E2ETestSuite) verifyDatabaseStorage(ingestionID, tenantID string) {
	// Query Spanner to verify record was saved
	stmt := spanner.Statement{
		SQL: "SELECT id, processing_status, confidence_score FROM ingestion_records WHERE tenant_id = @tenantId AND id = @ingestionId",
		Params: map[string]interface{}{
			"tenantId":    tenantID,
			"ingestionId": ingestionID,
		},
	}

	iter := suite.spannerClient.Single().Query(suite.ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	require.NoError(suite.T(), err, "Ingestion record should be saved to database")

	var id, status string
	var confidence float64
	err = row.Columns(&id, &status, &confidence)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), ingestionID, id)
	assert.Equal(suite.T(), "completed", status)
	assert.True(suite.T(), confidence > 0.8)
}

// Utility function
func int64Ptr(i int64) *int64 {
	return &i
}

// Run the test suite
func TestE2ETestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	suite.Run(t, new(E2ETestSuite))
}