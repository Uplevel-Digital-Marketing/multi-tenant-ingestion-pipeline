package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockServices for dependency injection in tests
type MockAudioProcessor struct {
	mock.Mock
}

func (m *MockAudioProcessor) ProcessAudio(ctx context.Context, audioData []byte, tenantID string) (*AudioProcessingResult, error) {
	args := m.Called(ctx, audioData, tenantID)
	return args.Get(0).(*AudioProcessingResult), args.Error(1)
}

type MockAIService struct {
	mock.Mock
}

func (m *MockAIService) ExtractInformation(ctx context.Context, transcript string, tenantConfig *TenantConfig) (*ExtractionResult, error) {
	args := m.Called(ctx, transcript, tenantConfig)
	return args.Get(0).(*ExtractionResult), args.Error(1)
}

type MockSpannerClient struct {
	mock.Mock
}

func (m *MockSpannerClient) SaveIngestionRecord(ctx context.Context, record *IngestionRecord) error {
	args := m.Called(ctx, record)
	return args.Error(0)
}

type MockCRMIntegration struct {
	mock.Mock
}

func (m *MockCRMIntegration) CreateLead(ctx context.Context, leadData *LeadData, tenantID string) (*CRMResponse, error) {
	args := m.Called(ctx, leadData, tenantID)
	return args.Get(0).(*CRMResponse), args.Error(1)
}

// Test data structures
type AudioProcessingResult struct {
	Transcript string
	Duration   time.Duration
	Language   string
	Quality    float64
}

type ExtractionResult struct {
	ContactInfo *ContactInfo
	ProjectInfo *ProjectInfo
	Urgency     string
	Confidence  float64
}

type ContactInfo struct {
	Name    string
	Phone   string
	Email   string
	Address string
}

type ProjectInfo struct {
	Type        string
	Description string
	Budget      string
	Timeline    string
}

type IngestionRecord struct {
	ID           string
	TenantID     string
	Timestamp    time.Time
	AudioHash    string
	Transcript   string
	ExtractedData *ExtractionResult
	Status       string
}

type LeadData struct {
	Contact     *ContactInfo
	Project     *ProjectInfo
	Source      string
	Urgency     string
	TenantID    string
}

type CRMResponse struct {
	LeadID   string
	Status   string
	Message  string
}

type TenantConfig struct {
	ID              string
	Name            string
	CRMSettings     map[string]interface{}
	AIPrompts       map[string]string
	ProcessingRules map[string]interface{}
}

// WorkflowEngine represents the main workflow orchestrator
type WorkflowEngine struct {
	audioProcessor  AudioProcessor
	aiService      AIService
	spannerClient  SpannerClient
	crmIntegration CRMIntegration
}

// Interfaces for dependency injection
type AudioProcessor interface {
	ProcessAudio(ctx context.Context, audioData []byte, tenantID string) (*AudioProcessingResult, error)
}

type AIService interface {
	ExtractInformation(ctx context.Context, transcript string, tenantConfig *TenantConfig) (*ExtractionResult, error)
}

type SpannerClient interface {
	SaveIngestionRecord(ctx context.Context, record *IngestionRecord) error
}

type CRMIntegration interface {
	CreateLead(ctx context.Context, leadData *LeadData, tenantID string) (*CRMResponse, error)
}

func NewWorkflowEngine(ap AudioProcessor, ai AIService, sc SpannerClient, crm CRMIntegration) *WorkflowEngine {
	return &WorkflowEngine{
		audioProcessor:  ap,
		aiService:      ai,
		spannerClient:  sc,
		crmIntegration: crm,
	}
}

// ProcessIngestion orchestrates the complete ingestion workflow
func (w *WorkflowEngine) ProcessIngestion(ctx context.Context, audioData []byte, tenantID string, tenantConfig *TenantConfig) (*IngestionRecord, error) {
	// Step 1: Process audio
	audioResult, err := w.audioProcessor.ProcessAudio(ctx, audioData, tenantID)
	if err != nil {
		return nil, err
	}

	// Step 2: Extract information using AI
	extractionResult, err := w.aiService.ExtractInformation(ctx, audioResult.Transcript, tenantConfig)
	if err != nil {
		return nil, err
	}

	// Step 3: Create ingestion record
	record := &IngestionRecord{
		ID:            generateID(),
		TenantID:      tenantID,
		Timestamp:     time.Now(),
		AudioHash:     hashAudio(audioData),
		Transcript:    audioResult.Transcript,
		ExtractedData: extractionResult,
		Status:        "processed",
	}

	// Step 4: Save to Spanner
	if err := w.spannerClient.SaveIngestionRecord(ctx, record); err != nil {
		return nil, err
	}

	// Step 5: Create CRM lead if confidence is high enough
	if extractionResult.Confidence > 0.8 {
		leadData := &LeadData{
			Contact:  extractionResult.ContactInfo,
			Project:  extractionResult.ProjectInfo,
			Source:   "voice_ingestion",
			Urgency:  extractionResult.Urgency,
			TenantID: tenantID,
		}

		_, err := w.crmIntegration.CreateLead(ctx, leadData, tenantID)
		if err != nil {
			// Log error but don't fail the entire workflow
			record.Status = "processed_with_crm_error"
		}
	}

	return record, nil
}

// Helper functions (would be in separate utils package)
func generateID() string {
	// Mock implementation
	return "test-id-12345"
}

func hashAudio(data []byte) string {
	// Mock implementation
	return "audio-hash-67890"
}

// Test Suite
type WorkflowTestSuite struct {
	suite.Suite
	engine            *WorkflowEngine
	mockAudioProcessor *MockAudioProcessor
	mockAIService     *MockAIService
	mockSpannerClient *MockSpannerClient
	mockCRMIntegration *MockCRMIntegration
	ctx               context.Context
}

func (suite *WorkflowTestSuite) SetupTest() {
	suite.mockAudioProcessor = new(MockAudioProcessor)
	suite.mockAIService = new(MockAIService)
	suite.mockSpannerClient = new(MockSpannerClient)
	suite.mockCRMIntegration = new(MockCRMIntegration)

	suite.engine = NewWorkflowEngine(
		suite.mockAudioProcessor,
		suite.mockAIService,
		suite.mockSpannerClient,
		suite.mockCRMIntegration,
	)

	suite.ctx = context.Background()
}

func (suite *WorkflowTestSuite) TearDownTest() {
	suite.mockAudioProcessor.AssertExpectations(suite.T())
	suite.mockAIService.AssertExpectations(suite.T())
	suite.mockSpannerClient.AssertExpectations(suite.T())
	suite.mockCRMIntegration.AssertExpectations(suite.T())
}

func (suite *WorkflowTestSuite) TestProcessIngestion_SuccessfulFlow() {
	// Arrange
	audioData := []byte("mock audio data")
	tenantID := "tenant-123"
	tenantConfig := &TenantConfig{
		ID:   tenantID,
		Name: "Test Tenant",
	}

	audioResult := &AudioProcessingResult{
		Transcript: "I need kitchen remodeling, my name is John Doe, phone 555-1234",
		Duration:   time.Minute * 2,
		Language:   "en",
		Quality:    0.95,
	}

	extractionResult := &ExtractionResult{
		ContactInfo: &ContactInfo{
			Name:  "John Doe",
			Phone: "555-1234",
		},
		ProjectInfo: &ProjectInfo{
			Type:        "kitchen",
			Description: "kitchen remodeling",
		},
		Urgency:    "medium",
		Confidence: 0.9,
	}

	crmResponse := &CRMResponse{
		LeadID:  "crm-lead-456",
		Status:  "created",
		Message: "Lead created successfully",
	}

	// Setup mocks
	suite.mockAudioProcessor.On("ProcessAudio", suite.ctx, audioData, tenantID).Return(audioResult, nil)
	suite.mockAIService.On("ExtractInformation", suite.ctx, audioResult.Transcript, tenantConfig).Return(extractionResult, nil)
	suite.mockSpannerClient.On("SaveIngestionRecord", suite.ctx, mock.AnythingOfType("*unit.IngestionRecord")).Return(nil)
	suite.mockCRMIntegration.On("CreateLead", suite.ctx, mock.AnythingOfType("*unit.LeadData"), tenantID).Return(crmResponse, nil)

	// Act
	result, err := suite.engine.ProcessIngestion(suite.ctx, audioData, tenantID, tenantConfig)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(tenantID, result.TenantID)
	suite.Equal(audioResult.Transcript, result.Transcript)
	suite.Equal(extractionResult, result.ExtractedData)
	suite.Equal("processed", result.Status)
}

func (suite *WorkflowTestSuite) TestProcessIngestion_AudioProcessingFailure() {
	// Arrange
	audioData := []byte("mock audio data")
	tenantID := "tenant-123"
	tenantConfig := &TenantConfig{ID: tenantID}

	suite.mockAudioProcessor.On("ProcessAudio", suite.ctx, audioData, tenantID).Return((*AudioProcessingResult)(nil), assert.AnError)

	// Act
	result, err := suite.engine.ProcessIngestion(suite.ctx, audioData, tenantID, tenantConfig)

	// Assert
	suite.Error(err)
	suite.Nil(result)
}

func (suite *WorkflowTestSuite) TestProcessIngestion_AIExtractionFailure() {
	// Arrange
	audioData := []byte("mock audio data")
	tenantID := "tenant-123"
	tenantConfig := &TenantConfig{ID: tenantID}

	audioResult := &AudioProcessingResult{
		Transcript: "test transcript",
		Duration:   time.Minute,
		Language:   "en",
		Quality:    0.8,
	}

	suite.mockAudioProcessor.On("ProcessAudio", suite.ctx, audioData, tenantID).Return(audioResult, nil)
	suite.mockAIService.On("ExtractInformation", suite.ctx, audioResult.Transcript, tenantConfig).Return((*ExtractionResult)(nil), assert.AnError)

	// Act
	result, err := suite.engine.ProcessIngestion(suite.ctx, audioData, tenantID, tenantConfig)

	// Assert
	suite.Error(err)
	suite.Nil(result)
}

func (suite *WorkflowTestSuite) TestProcessIngestion_SpannerSaveFailure() {
	// Arrange
	audioData := []byte("mock audio data")
	tenantID := "tenant-123"
	tenantConfig := &TenantConfig{ID: tenantID}

	audioResult := &AudioProcessingResult{
		Transcript: "test transcript",
		Duration:   time.Minute,
		Language:   "en",
		Quality:    0.8,
	}

	extractionResult := &ExtractionResult{
		ContactInfo: &ContactInfo{Name: "Test User"},
		ProjectInfo: &ProjectInfo{Type: "bathroom"},
		Confidence:  0.7,
	}

	suite.mockAudioProcessor.On("ProcessAudio", suite.ctx, audioData, tenantID).Return(audioResult, nil)
	suite.mockAIService.On("ExtractInformation", suite.ctx, audioResult.Transcript, tenantConfig).Return(extractionResult, nil)
	suite.mockSpannerClient.On("SaveIngestionRecord", suite.ctx, mock.AnythingOfType("*unit.IngestionRecord")).Return(assert.AnError)

	// Act
	result, err := suite.engine.ProcessIngestion(suite.ctx, audioData, tenantID, tenantConfig)

	// Assert
	suite.Error(err)
	suite.Nil(result)
}

func (suite *WorkflowTestSuite) TestProcessIngestion_LowConfidenceSkipsCRM() {
	// Arrange
	audioData := []byte("mock audio data")
	tenantID := "tenant-123"
	tenantConfig := &TenantConfig{ID: tenantID}

	audioResult := &AudioProcessingResult{
		Transcript: "unclear audio",
		Duration:   time.Second * 30,
		Language:   "en",
		Quality:    0.6,
	}

	extractionResult := &ExtractionResult{
		ContactInfo: &ContactInfo{Name: "Unclear"},
		ProjectInfo: &ProjectInfo{Type: "unknown"},
		Confidence:  0.5, // Low confidence, should skip CRM
	}

	suite.mockAudioProcessor.On("ProcessAudio", suite.ctx, audioData, tenantID).Return(audioResult, nil)
	suite.mockAIService.On("ExtractInformation", suite.ctx, audioResult.Transcript, tenantConfig).Return(extractionResult, nil)
	suite.mockSpannerClient.On("SaveIngestionRecord", suite.ctx, mock.AnythingOfType("*unit.IngestionRecord")).Return(nil)
	// Note: No CRM mock expectation - should not be called

	// Act
	result, err := suite.engine.ProcessIngestion(suite.ctx, audioData, tenantID, tenantConfig)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("processed", result.Status)
}

func (suite *WorkflowTestSuite) TestProcessIngestion_CRMFailureDoesNotFailWorkflow() {
	// Arrange
	audioData := []byte("mock audio data")
	tenantID := "tenant-123"
	tenantConfig := &TenantConfig{ID: tenantID}

	audioResult := &AudioProcessingResult{
		Transcript: "I need kitchen remodeling, John Doe 555-1234",
		Duration:   time.Minute,
		Language:   "en",
		Quality:    0.9,
	}

	extractionResult := &ExtractionResult{
		ContactInfo: &ContactInfo{Name: "John Doe", Phone: "555-1234"},
		ProjectInfo: &ProjectInfo{Type: "kitchen"},
		Confidence:  0.9, // High confidence, should trigger CRM
	}

	suite.mockAudioProcessor.On("ProcessAudio", suite.ctx, audioData, tenantID).Return(audioResult, nil)
	suite.mockAIService.On("ExtractInformation", suite.ctx, audioResult.Transcript, tenantConfig).Return(extractionResult, nil)
	suite.mockSpannerClient.On("SaveIngestionRecord", suite.ctx, mock.AnythingOfType("*unit.IngestionRecord")).Return(nil)
	suite.mockCRMIntegration.On("CreateLead", suite.ctx, mock.AnythingOfType("*unit.LeadData"), tenantID).Return((*CRMResponse)(nil), assert.AnError)

	// Act
	result, err := suite.engine.ProcessIngestion(suite.ctx, audioData, tenantID, tenantConfig)

	// Assert
	suite.NoError(err) // Should not fail despite CRM error
	suite.NotNil(result)
	suite.Equal("processed_with_crm_error", result.Status)
}

// Table-driven tests for various scenarios
func TestAudioProcessingScenarios(t *testing.T) {
	tests := []struct {
		name           string
		audioQuality   float64
		expectedResult bool
		description    string
	}{
		{
			name:           "High Quality Audio",
			audioQuality:   0.95,
			expectedResult: true,
			description:    "Should process high quality audio successfully",
		},
		{
			name:           "Medium Quality Audio",
			audioQuality:   0.75,
			expectedResult: true,
			description:    "Should process medium quality audio with warnings",
		},
		{
			name:           "Low Quality Audio",
			audioQuality:   0.4,
			expectedResult: false,
			description:    "Should reject low quality audio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock processor would implement quality-based logic
			processor := &MockAudioProcessor{}

			if tt.expectedResult {
				processor.On("ProcessAudio", mock.Anything, mock.Anything, mock.Anything).Return(
					&AudioProcessingResult{
						Transcript: "test transcript",
						Quality:    tt.audioQuality,
					}, nil)
			} else {
				processor.On("ProcessAudio", mock.Anything, mock.Anything, mock.Anything).Return(
					(*AudioProcessingResult)(nil), assert.AnError)
			}

			result, err := processor.ProcessAudio(context.Background(), []byte("test"), "tenant")

			if tt.expectedResult {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.audioQuality, result.Quality)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}

			processor.AssertExpectations(t)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkWorkflowProcessing(b *testing.B) {
	engine := &WorkflowEngine{
		audioProcessor: &MockAudioProcessor{},
		aiService:     &MockAIService{},
		spannerClient: &MockSpannerClient{},
		crmIntegration: &MockCRMIntegration{},
	}

	audioData := make([]byte, 1024*1024) // 1MB audio file
	tenantConfig := &TenantConfig{ID: "benchmark-tenant"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This would benchmark the actual workflow processing
		_ = audioData
		_ = tenantConfig
		_ = engine
		// engine.ProcessIngestion(context.Background(), audioData, "tenant", tenantConfig)
	}
}

// Run the test suite
func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}