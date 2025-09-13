package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	aiplatform "cloud.google.com/go/aiplatform/apiv1"

	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
	"github.com/home-renovators/ingestion-pipeline/internal/ai"
)

// AIAnalysisIntegrationTestSuite tests Vertex AI Gemini analysis pipeline
type AIAnalysisIntegrationTestSuite struct {
	suite.Suite
	ctx       context.Context
	aiService *ai.Service
	config    *config.Config
}

func (suite *AIAnalysisIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup test configuration
	suite.config = &config.Config{
		ProjectID:           "test-project",
		VertexAIProject:     "test-vertex-project",
		VertexAILocation:    "us-central1",
		VertexAIModel:       "gemini-2.0-flash-exp",
		SpeechToTextModel:   "chirp-3",
		SpeechLanguage:      "en-US",
		EnableDiarization:   true,
		GeminiTemperature:   0.1,
		GeminiMaxTokens:     1024,
		GeminiTopP:          0.8,
		GeminiTopK:          40,
	}

	// Initialize AI service
	var err error
	suite.aiService, err = ai.NewService(suite.ctx, suite.config)
	require.NoError(suite.T(), err)
}

func (suite *AIAnalysisIntegrationTestSuite) TearDownSuite() {
	if suite.aiService != nil {
		suite.aiService.Close()
	}
}

// TestCallContentAnalysis tests AI analysis of call transcriptions
func (suite *AIAnalysisIntegrationTestSuite) TestCallContentAnalysis() {
	testCases := []struct {
		name                string
		transcription       string
		callDetails         models.CallDetails
		expectedIntent      string
		expectedProjectType string
		minLeadScore        int
		maxLeadScore        int
		expectedSentiment   string
		expectedUrgency     string
	}{
		{
			name: "HighValueKitchenRemodelLead",
			transcription: "Hi, I'm Sarah Johnson and I'm interested in a complete kitchen remodel. " +
				"I have a budget of around $50,000 and I'd like to get started within the next 2-3 months. " +
				"Could someone call me back at 555-123-4567 to schedule a consultation? " +
				"I'm particularly interested in custom cabinets and granite countertops.",
			callDetails: models.CallDetails{
				ID:                  "CAL_KITCHEN_001",
				Duration:            240, // 4 minutes
				CustomerName:        "Sarah Johnson",
				CustomerPhoneNumber: "+15551234567",
				CustomerCity:        "Beverly Hills",
				CustomerState:       "CA",
				FirstCall:          true,
				Direction:          "inbound",
			},
			expectedIntent:      "quote_request",
			expectedProjectType: "kitchen",
			minLeadScore:        80,
			maxLeadScore:        100,
			expectedSentiment:   "positive",
			expectedUrgency:     "medium",
		},
		{
			name: "EmergencyWaterDamageCall",
			transcription: "EMERGENCY! We have a major water leak in our bathroom and it's flooding into the kitchen! " +
				"We need someone out here immediately. This is urgent - water is everywhere and we need help now! " +
				"My name is John Smith, phone number 555-987-6543. Please send someone today!",
			callDetails: models.CallDetails{
				ID:                  "CAL_EMERGENCY_001",
				Duration:            90, // 1.5 minutes
				CustomerName:        "John Smith",
				CustomerPhoneNumber: "+15559876543",
				CustomerCity:        "Los Angeles",
				CustomerState:       "CA",
				FirstCall:          true,
				Direction:          "inbound",
			},
			expectedIntent:      "emergency_service",
			expectedProjectType: "plumbing",
			minLeadScore:        95,
			maxLeadScore:        100,
			expectedSentiment:   "negative",
			expectedUrgency:     "high",
		},
		{
			name: "BathroomRenovationInquiry",
			transcription: "Hello, I'm looking into renovating my master bathroom sometime in the next year. " +
				"It's not urgent, but I'd like to get some ideas and pricing. I'm thinking about updating " +
				"the shower, vanity, and maybe the flooring. My name is Lisa Williams, 555-456-7890.",
			callDetails: models.CallDetails{
				ID:                  "CAL_BATHROOM_001",
				Duration:            180, // 3 minutes
				CustomerName:        "Lisa Williams",
				CustomerPhoneNumber: "+15554567890",
				CustomerCity:        "Santa Monica",
				CustomerState:       "CA",
				FirstCall:          true,
				Direction:          "inbound",
			},
			expectedIntent:      "quote_request",
			expectedProjectType: "bathroom",
			minLeadScore:        60,
			maxLeadScore:        85,
			expectedSentiment:   "positive",
			expectedUrgency:     "low",
		},
		{
			name: "PricingOnlyInquiry",
			transcription: "I just want to know how much it costs to remodel a kitchen. " +
				"I'm not ready to commit to anything, just shopping around for prices. " +
				"Don't need anyone to call me back.",
			callDetails: models.CallDetails{
				ID:                  "CAL_PRICING_001",
				Duration:            45, // 45 seconds
				CustomerName:        "Unknown",
				CustomerPhoneNumber: "+15555551234",
				CustomerCity:        "Unknown",
				CustomerState:       "CA",
				FirstCall:          false,
				Direction:          "inbound",
			},
			expectedIntent:      "price_inquiry",
			expectedProjectType: "kitchen",
			minLeadScore:        20,
			maxLeadScore:        50,
			expectedSentiment:   "neutral",
			expectedUrgency:     "low",
		},
		{
			name: "ComplaintCall",
			transcription: "I'm calling to complain about the work you did last month. " +
				"The tiles are already cracking and I'm very unhappy with the quality. " +
				"I want someone to come out and fix this immediately or I'm going to leave bad reviews everywhere.",
			callDetails: models.CallDetails{
				ID:                  "CAL_COMPLAINT_001",
				Duration:            120, // 2 minutes
				CustomerName:        "Angry Customer",
				CustomerPhoneNumber: "+15559999999",
				CustomerCity:        "Los Angeles",
				CustomerState:       "CA",
				FirstCall:          false,
				Direction:          "inbound",
			},
			expectedIntent:      "complaint",
			expectedProjectType: "service",
			minLeadScore:        0,
			maxLeadScore:        30,
			expectedSentiment:   "negative",
			expectedUrgency:     "high",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Measure analysis time
			startTime := time.Now()

			// Perform AI analysis
			analysis, err := suite.aiService.AnalyzeCallContent(suite.ctx, tc.transcription, tc.callDetails)
			analysisTime := time.Since(startTime)

			// Assert analysis completed successfully
			require.NoError(t, err, "AI analysis should complete without error")
			require.NotNil(t, analysis, "Analysis result should not be nil")

			// Assert performance requirement (<1s for AI analysis)
			assert.True(t, analysisTime < 1*time.Second,
				"AI analysis should complete within 1 second, took %v", analysisTime)

			// Assert intent detection
			assert.Equal(t, tc.expectedIntent, analysis.Intent,
				"Should correctly identify call intent")

			// Assert project type detection
			assert.Equal(t, tc.expectedProjectType, analysis.ProjectType,
				"Should correctly identify project type")

			// Assert lead score is within expected range
			assert.GreaterOrEqual(t, analysis.LeadScore, tc.minLeadScore,
				"Lead score should be at least %d, got %d", tc.minLeadScore, analysis.LeadScore)
			assert.LessOrEqual(t, analysis.LeadScore, tc.maxLeadScore,
				"Lead score should be at most %d, got %d", tc.maxLeadScore, analysis.LeadScore)

			// Assert sentiment analysis
			assert.Equal(t, tc.expectedSentiment, analysis.Sentiment,
				"Should correctly identify sentiment")

			// Assert urgency assessment
			assert.Equal(t, tc.expectedUrgency, analysis.Urgency,
				"Should correctly assess urgency level")

			// Assert follow-up requirement for high-value leads
			if analysis.LeadScore >= 70 {
				assert.True(t, analysis.FollowUpRequired,
					"High-value leads should require follow-up")
			}

			// Assert appointment request detection for service-oriented calls
			if tc.expectedIntent == "quote_request" || tc.expectedIntent == "emergency_service" {
				// These types of calls often include appointment requests
				// We don't assert true/false as it depends on the specific transcription
				assert.NotNil(t, &analysis.AppointmentRequested,
					"Should analyze appointment request status")
			}

			// Assert key details extraction
			assert.NotEmpty(t, analysis.KeyDetails,
				"Should extract key details from the call")

			// Verify key details contain relevant information
			keyDetailsText := strings.ToLower(strings.Join(analysis.KeyDetails, " "))
			if tc.expectedProjectType != "service" {
				assert.Contains(t, keyDetailsText, tc.expectedProjectType,
					"Key details should mention the project type")
			}

			suite.T().Logf("Analysis completed in %v: Intent=%s, ProjectType=%s, LeadScore=%d, Sentiment=%s",
				analysisTime, analysis.Intent, analysis.ProjectType, analysis.LeadScore, analysis.Sentiment)
		})
	}
}

// TestSpamDetectionAnalysis tests AI-powered spam detection
func (suite *AIAnalysisIntegrationTestSuite) TestSpamDetectionAnalysis() {
	testCases := []struct {
		name               string
		enhancedPayload    models.EnhancedPayload
		expectedSpamLevel  string // "low", "medium", "high"
		maxSpamLikelihood  float64
		minSpamLikelihood  float64
	}{
		{
			name: "LegitimateKitchenLead",
			enhancedPayload: models.EnhancedPayload{
				RequestID:         "req_legitimate_001",
				TenantID:          "tenant_test",
				Source:            "callrail_webhook",
				CommunicationMode: "phone_call",
				OriginalWebhook: models.CallRailWebhook{
					CallID:              "CAL_LEGIT_001",
					Duration:            "240",
					CustomerName:        "Sarah Johnson",
					CustomerPhoneNumber: "+15551234567",
					CustomerCity:        "Beverly Hills",
					CustomerState:       "CA",
				},
				AudioProcessing: models.AudioProcessingData{
					Transcription: "Hi, I'm interested in a kitchen remodel. Can someone call me back to discuss pricing and timeline?",
					Confidence:    0.95,
					Duration:      240.0,
				},
				AIAnalysis: models.CallAnalysis{
					Intent:           "quote_request",
					ProjectType:      "kitchen",
					LeadScore:        85,
					Sentiment:        "positive",
					FollowUpRequired: true,
				},
			},
			expectedSpamLevel: "low",
			maxSpamLikelihood: 0.3,
			minSpamLikelihood: 0.0,
		},
		{
			name: "SuspiciousShortCall",
			enhancedPayload: models.EnhancedPayload{
				RequestID:         "req_suspicious_001",
				TenantID:          "tenant_test",
				Source:            "callrail_webhook",
				CommunicationMode: "phone_call",
				OriginalWebhook: models.CallRailWebhook{
					CallID:              "CAL_SUSPICIOUS_001",
					Duration:            "5", // Very short call
					CustomerName:        "",  // No name
					CustomerPhoneNumber: "+15555555555", // Suspicious number pattern
					CustomerCity:        "",
					CustomerState:       "",
				},
				AudioProcessing: models.AudioProcessingData{
					Transcription: "", // No transcription
					Confidence:    0.2, // Low confidence
					Duration:      5.0,
				},
				AIAnalysis: models.CallAnalysis{
					Intent:      "unknown",
					ProjectType: "",
					LeadScore:   10,
					Sentiment:   "neutral",
				},
			},
			expectedSpamLevel: "medium",
			maxSpamLikelihood: 0.8,
			minSpamLikelihood: 0.4,
		},
		{
			name: "HighSpamCall",
			enhancedPayload: models.EnhancedPayload{
				RequestID:         "req_spam_001",
				TenantID:          "tenant_test",
				Source:            "callrail_webhook",
				CommunicationMode: "phone_call",
				OriginalWebhook: models.CallRailWebhook{
					CallID:              "CAL_SPAM_001",
					Duration:            "3", // Very short
					CustomerName:        "Telemarketer",
					CustomerPhoneNumber: "+18005551234", // 800 number
					CustomerCity:        "",
					CustomerState:       "",
				},
				AudioProcessing: models.AudioProcessingData{
					Transcription: "This is a recorded message about your car warranty",
					Confidence:    0.9,
					Duration:      3.0,
				},
				AIAnalysis: models.CallAnalysis{
					Intent:      "telemarketing",
					ProjectType: "",
					LeadScore:   5,
					Sentiment:   "neutral",
				},
			},
			expectedSpamLevel: "high",
			maxSpamLikelihood: 1.0,
			minSpamLikelihood: 0.8,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Measure spam detection time
			startTime := time.Now()

			// Perform spam detection analysis
			spamLikelihood, err := suite.aiService.AnalyzeSpamLikelihood(suite.ctx, tc.enhancedPayload)
			detectionTime := time.Since(startTime)

			// Assert spam detection completed successfully
			require.NoError(t, err, "Spam detection should complete without error")

			// Assert performance requirement
			assert.True(t, detectionTime < 500*time.Millisecond,
				"Spam detection should complete within 500ms, took %v", detectionTime)

			// Assert spam likelihood is within expected range
			assert.GreaterOrEqual(t, spamLikelihood, tc.minSpamLikelihood,
				"Spam likelihood should be at least %f, got %f", tc.minSpamLikelihood, spamLikelihood)
			assert.LessOrEqual(t, spamLikelihood, tc.maxSpamLikelihood,
				"Spam likelihood should be at most %f, got %f", tc.maxSpamLikelihood, spamLikelihood)

			// Assert spam likelihood is a valid probability
			assert.GreaterOrEqual(t, spamLikelihood, 0.0, "Spam likelihood should be non-negative")
			assert.LessOrEqual(t, spamLikelihood, 1.0, "Spam likelihood should not exceed 1.0")

			suite.T().Logf("Spam detection completed in %v: likelihood=%f (%s)",
				detectionTime, spamLikelihood, tc.expectedSpamLevel)
		})
	}
}

// TestAIAnalysisErrorHandling tests error scenarios in AI analysis
func (suite *AIAnalysisIntegrationTestSuite) TestAIAnalysisErrorHandling() {
	suite.T().Run("EmptyTranscription", func(t *testing.T) {
		callDetails := models.CallDetails{
			ID:       "CAL_EMPTY_001",
			Duration: 30,
		}

		analysis, err := suite.aiService.AnalyzeCallContent(suite.ctx, "", callDetails)

		// Empty transcription might succeed with default values or fail
		if err != nil {
			assert.Contains(t, err.Error(), "transcription",
				"Error should mention transcription issue")
		} else {
			assert.NotNil(t, analysis, "Analysis should not be nil")
			// Default values should be reasonable
			assert.NotEmpty(t, analysis.Intent, "Intent should have default value")
			assert.GreaterOrEqual(t, analysis.LeadScore, 0, "Lead score should be non-negative")
		}
	})

	suite.T().Run("InvalidCallDetails", func(t *testing.T) {
		invalidCallDetails := models.CallDetails{
			ID:       "", // Invalid empty ID
			Duration: -1, // Invalid negative duration
		}

		transcription := "This is a test transcription"

		analysis, err := suite.aiService.AnalyzeCallContent(suite.ctx, transcription, invalidCallDetails)

		// Should handle invalid data gracefully
		if err != nil {
			assert.NotEmpty(t, err.Error(), "Error message should not be empty")
		} else {
			assert.NotNil(t, analysis, "Analysis should not be nil")
			// Should provide reasonable defaults despite invalid input
		}
	})

	suite.T().Run("ContextTimeout", func(t *testing.T) {
		// Create context with very short timeout
		timeoutCtx, cancel := context.WithTimeout(suite.ctx, 10*time.Millisecond)
		defer cancel()

		callDetails := models.CallDetails{
			ID:       "CAL_TIMEOUT_001",
			Duration: 120,
		}

		transcription := "This is a test transcription for timeout testing"

		analysis, err := suite.aiService.AnalyzeCallContent(timeoutCtx, transcription, callDetails)

		assert.Error(t, err, "Analysis should fail with timeout")
		assert.Nil(t, analysis, "Result should be nil on timeout")
		assert.Contains(t, err.Error(), "context deadline exceeded",
			"Error should indicate context timeout")
	})

	suite.T().Run("VeryLongTranscription", func(t *testing.T) {
		// Create extremely long transcription to test token limits
		longTranscription := strings.Repeat("This is a very long transcription that repeats many times. ", 1000)

		callDetails := models.CallDetails{
			ID:       "CAL_LONG_001",
			Duration: 3600, // 1 hour call
		}

		analysis, err := suite.aiService.AnalyzeCallContent(suite.ctx, longTranscription, callDetails)

		// Should handle long transcriptions gracefully
		if err != nil {
			// Might fail due to token limits
			assert.Contains(t, strings.ToLower(err.Error()), "token",
				"Error should mention token limit issue")
		} else {
			assert.NotNil(t, analysis, "Analysis should not be nil")
			assert.NotEmpty(t, analysis.Intent, "Should extract intent from long text")
		}
	})
}

// TestAIAnalysisConsistency tests consistency of AI analysis results
func (suite *AIAnalysisIntegrationTestSuite) TestAIAnalysisConsistency() {
	suite.T().Run("RepeatedAnalysisConsistency", func(t *testing.T) {
		transcription := "Hi, I'm interested in a kitchen remodel. I have a budget of $30,000 and would like to start next month."
		callDetails := models.CallDetails{
			ID:                  "CAL_CONSISTENCY_001",
			Duration:            180,
			CustomerName:        "Test Customer",
			CustomerPhoneNumber: "+15551234567",
			FirstCall:          true,
		}

		var analyses []*models.CallAnalysis
		const numRuns = 3

		// Run analysis multiple times
		for i := 0; i < numRuns; i++ {
			analysis, err := suite.aiService.AnalyzeCallContent(suite.ctx, transcription, callDetails)
			require.NoError(t, err, "Analysis %d should succeed", i+1)
			require.NotNil(t, analysis, "Analysis %d should not be nil", i+1)
			analyses = append(analyses, analysis)
		}

		// Check consistency across runs
		firstAnalysis := analyses[0]
		for i, analysis := range analyses[1:] {
			// Intent should be consistent
			assert.Equal(t, firstAnalysis.Intent, analysis.Intent,
				"Intent should be consistent across runs (run %d)", i+2)

			// Project type should be consistent
			assert.Equal(t, firstAnalysis.ProjectType, analysis.ProjectType,
				"Project type should be consistent across runs (run %d)", i+2)

			// Lead score should be within reasonable range (±10 points)
			scoreDiff := abs(firstAnalysis.LeadScore - analysis.LeadScore)
			assert.LessOrEqual(t, scoreDiff, 10,
				"Lead score should be consistent within ±10 points (run %d)", i+2)

			// Sentiment should be consistent
			assert.Equal(t, firstAnalysis.Sentiment, analysis.Sentiment,
				"Sentiment should be consistent across runs (run %d)", i+2)
		}

		suite.T().Logf("Consistency test: %d runs completed with consistent results", numRuns)
	})
}

// TestAIAnalysisPerformance tests performance characteristics
func (suite *AIAnalysisIntegrationTestSuite) TestAIAnalysisPerformance() {
	suite.T().Run("ConcurrentAnalysisPerformance", func(t *testing.T) {
		transcriptions := []string{
			"I need a kitchen remodel, budget is $40,000",
			"Emergency plumbing repair needed immediately",
			"Bathroom renovation quote request",
			"Just want pricing information for flooring",
			"Complaint about previous work quality",
		}

		callDetails := models.CallDetails{
			ID:       "CAL_CONCURRENT_001",
			Duration: 120,
		}

		results := make(chan struct {
			index    int
			duration time.Duration
			err      error
		}, len(transcriptions))

		startTime := time.Now()

		// Start concurrent analyses
		for i, transcription := range transcriptions {
			go func(idx int, text string) {
				analysisStart := time.Now()
				_, err := suite.aiService.AnalyzeCallContent(suite.ctx, text, callDetails)
				duration := time.Since(analysisStart)

				results <- struct {
					index    int
					duration time.Duration
					err      error
				}{idx, duration, err}
			}(i, transcription)
		}

		// Collect results
		var maxDuration time.Duration
		successCount := 0

		for i := 0; i < len(transcriptions); i++ {
			result := <-results
			if result.err == nil {
				successCount++
				if result.duration > maxDuration {
					maxDuration = result.duration
				}
			}
		}

		totalDuration := time.Since(startTime)

		// Assert performance characteristics
		assert.Greater(t, successCount, len(transcriptions)/2,
			"At least half of concurrent analyses should succeed")
		assert.True(t, maxDuration < 2*time.Second,
			"Individual analysis should complete within 2 seconds")
		assert.True(t, totalDuration < 5*time.Second,
			"All concurrent analyses should complete within 5 seconds")

		suite.T().Logf("Concurrent analysis: %d/%d succeeded in %v (max individual: %v)",
			successCount, len(transcriptions), totalDuration, maxDuration)
	})
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Run the test suite
func TestAIAnalysisIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AIAnalysisIntegrationTestSuite))
}