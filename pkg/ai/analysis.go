package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

// AnalysisService handles AI-powered content analysis using Vertex AI
type AnalysisService struct {
	aiClient *aiplatform.PredictionClient
	config   *AnalysisConfig
}

// AnalysisConfig contains configuration for AI analysis
type AnalysisConfig struct {
	ProjectID    string  `json:"project_id"`
	Location     string  `json:"location"`
	Model        string  `json:"model"` // e.g., "gemini-2.5-flash"
	Temperature  float32 `json:"temperature"`
	MaxTokens    int32   `json:"max_tokens"`
	TopP         float32 `json:"top_p"`
	TopK         int32   `json:"top_k"`
}

// DefaultAnalysisConfig returns default configuration for Gemini 2.5 Flash
func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		Model:       "gemini-2.5-flash",
		Temperature: 0.2,
		MaxTokens:   1024,
		TopP:        0.8,
		TopK:        40,
	}
}

// NewAnalysisService creates a new AI analysis service
func NewAnalysisService(ctx context.Context, config *AnalysisConfig) (*AnalysisService, error) {
	if config == nil {
		config = DefaultAnalysisConfig()
	}

	aiClient, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI platform client: %w", err)
	}

	return &AnalysisService{
		aiClient: aiClient,
		config:   config,
	}, nil
}

// Close closes the analysis service
func (as *AnalysisService) Close() error {
	return as.aiClient.Close()
}

// AnalysisRequest contains parameters for content analysis
type AnalysisRequest struct {
	CallID         string                `json:"call_id"`
	TenantID       string                `json:"tenant_id"`
	Transcription  string                `json:"transcription"`
	CallDetails    *models.CallDetails   `json:"call_details,omitempty"`
	AnalysisType   AnalysisType          `json:"analysis_type"`
	CustomPrompt   string                `json:"custom_prompt,omitempty"`
	CustomConfig   *AnalysisConfig       `json:"custom_config,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Priority       AnalysisPriority      `json:"priority"`
}

// AnalysisType defines the type of analysis to perform
type AnalysisType string

const (
	AnalysisTypeContent   AnalysisType = "content_analysis"
	AnalysisTypeSpam      AnalysisType = "spam_detection"
	AnalysisTypeSentiment AnalysisType = "sentiment_analysis"
	AnalysisTypeIntent    AnalysisType = "intent_classification"
	AnalysisTypeLeadScore AnalysisType = "lead_scoring"
	AnalysisTypeCustom    AnalysisType = "custom"
)

// AnalysisPriority defines the priority of analysis requests
type AnalysisPriority string

const (
	PriorityLow    AnalysisPriority = "low"
	PriorityNormal AnalysisPriority = "normal"
	PriorityHigh   AnalysisPriority = "high"
	PriorityUrgent AnalysisPriority = "urgent"
)

// AnalysisResponse contains the full response from analysis
type AnalysisResponse struct {
	Request      *AnalysisRequest       `json:"request"`
	CallAnalysis *models.CallAnalysis   `json:"call_analysis,omitempty"`
	SpamResult   *SpamAnalysisResult    `json:"spam_result,omitempty"`
	SentimentResult *SentimentAnalysisResult `json:"sentiment_result,omitempty"`
	CustomResult map[string]interface{} `json:"custom_result,omitempty"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Metadata     AnalysisMetadata       `json:"metadata"`
}

// SpamAnalysisResult contains spam detection results
type SpamAnalysisResult struct {
	SpamLikelihood float64           `json:"spam_likelihood"` // 0-100
	Confidence     float64           `json:"confidence"`      // 0-1
	Indicators     []string          `json:"indicators"`
	Reasoning      string            `json:"reasoning"`
	Details        map[string]interface{} `json:"details"`
}

// SentimentAnalysisResult contains sentiment analysis results
type SentimentAnalysisResult struct {
	Sentiment           string            `json:"sentiment"` // positive, neutral, negative
	Confidence          float64           `json:"confidence"` // 0-1
	EmotionalTone       string            `json:"emotional_tone"`
	CustomerSatisfaction string           `json:"customer_satisfaction"` // high, medium, low
	KeyEmotions         []string          `json:"key_emotions"`
	Details             map[string]interface{} `json:"details"`
}

// AnalysisMetadata contains metadata about the analysis process
type AnalysisMetadata struct {
	ModelUsed       string             `json:"model_used"`
	InputTokens     int64              `json:"input_tokens"`
	OutputTokens    int64              `json:"output_tokens"`
	ProcessingTime  time.Duration      `json:"processing_time"`
	PromptTemplate  string             `json:"prompt_template"`
	Parameters      map[string]interface{} `json:"parameters"`
}

// AnalyzeContent performs comprehensive content analysis on call transcription
func (as *AnalysisService) AnalyzeContent(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
	startTime := time.Now()

	response := &AnalysisResponse{
		Request:   req,
		StartTime: startTime,
		Metadata: AnalysisMetadata{
			ModelUsed: as.config.Model,
		},
	}

	// Use custom config if provided
	config := as.config
	if req.CustomConfig != nil {
		config = req.CustomConfig
	}

	var prompt string
	switch req.AnalysisType {
	case AnalysisTypeContent:
		prompt = as.buildContentAnalysisPrompt(req.Transcription, req.CallDetails)
	case AnalysisTypeSpam:
		prompt = as.buildSpamDetectionPrompt(req.Transcription, req.CallDetails)
	case AnalysisTypeSentiment:
		prompt = as.buildSentimentAnalysisPrompt(req.Transcription, req.CallDetails)
	case AnalysisTypeIntent:
		prompt = as.buildIntentClassificationPrompt(req.Transcription, req.CallDetails)
	case AnalysisTypeLeadScore:
		prompt = as.buildLeadScoringPrompt(req.Transcription, req.CallDetails)
	case AnalysisTypeCustom:
		if req.CustomPrompt == "" {
			return nil, fmt.Errorf("custom prompt required for custom analysis type")
		}
		prompt = req.CustomPrompt
	default:
		return nil, fmt.Errorf("unsupported analysis type: %s", req.AnalysisType)
	}

	response.Metadata.PromptTemplate = prompt
	response.Metadata.InputTokens = int64(len(prompt))

	// Create prediction request
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s",
		config.ProjectID, config.Location, config.Model)

	instances, err := as.createGeminiInstances(prompt, config)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("failed to create instances: %v", err)
		response.EndTime = time.Now()
		response.Duration = time.Since(startTime)
		return response, err
	}

	parameters, err := as.createGeminiParameters(config)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("failed to create parameters: %v", err)
		response.EndTime = time.Now()
		response.Duration = time.Since(startTime)
		return response, err
	}

	predictionReq := &aiplatformpb.PredictRequest{
		Endpoint:   endpoint,
		Instances:  instances,
		Parameters: parameters,
	}

	// Make the prediction
	resp, err := as.aiClient.Predict(ctx, predictionReq)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("prediction failed: %v", err)
		response.EndTime = time.Now()
		response.Duration = time.Since(startTime)
		return response, err
	}

	// Parse the response based on analysis type
	err = as.parseAnalysisResponse(resp, req.AnalysisType, response)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("failed to parse response: %v", err)
		response.EndTime = time.Now()
		response.Duration = time.Since(startTime)
		return response, err
	}

	response.Success = true
	response.EndTime = time.Now()
	response.Duration = time.Since(startTime)
	response.Metadata.ProcessingTime = response.Duration

	return response, nil
}

// AnalyzeBatch processes multiple analysis requests in batch
func (as *AnalysisService) AnalyzeBatch(ctx context.Context, requests []*AnalysisRequest) ([]*AnalysisResponse, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no analysis requests provided")
	}

	responses := make([]*AnalysisResponse, len(requests))

	// Process requests concurrently with a semaphore to limit concurrency
	semaphore := make(chan struct{}, 3) // Allow up to 3 concurrent analyses

	type result struct {
		index    int
		response *AnalysisResponse
		err      error
	}

	resultChan := make(chan result, len(requests))

	for i, req := range requests {
		go func(index int, request *AnalysisRequest) {
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			resp, err := as.AnalyzeContent(ctx, request)
			resultChan <- result{index: index, response: resp, err: err}
		}(i, req)
	}

	// Collect results
	for i := 0; i < len(requests); i++ {
		res := <-resultChan
		responses[res.index] = res.response
		if res.err != nil && res.response != nil {
			res.response.Success = false
			res.response.Error = res.err.Error()
		}
	}

	return responses, nil
}

// buildContentAnalysisPrompt creates the prompt for comprehensive content analysis
func (as *AnalysisService) buildContentAnalysisPrompt(transcription string, callDetails *models.CallDetails) string {
	var callInfo string
	if callDetails != nil {
		callInfo = fmt.Sprintf(`
CALL METADATA:
- Customer Name: %s
- Customer Phone: %s
- Customer Location: %s, %s
- Call Duration: %d seconds
- Source: %s
- Tags: %s
- Lead Status: %s`,
			callDetails.CustomerName,
			callDetails.CustomerPhoneNumber,
			callDetails.CustomerCity,
			callDetails.CustomerState,
			callDetails.Duration,
			callDetails.Source,
			strings.Join(callDetails.Tags, ", "),
			callDetails.LeadStatus)
	}

	return fmt.Sprintf(`
Analyze this phone call transcription for a home remodeling company:

TRANSCRIPT: %s
%s

Extract the following information in JSON format:
{
  "intent": "quote_request|information_seeking|appointment_booking|complaint|follow_up|other",
  "project_type": "kitchen|bathroom|whole_home|addition|flooring|roofing|windows|doors|other",
  "timeline": "immediate|1-3_months|3-6_months|6+_months|unknown",
  "budget_indicator": "high|medium|low|unknown",
  "sentiment": "positive|neutral|negative",
  "lead_score": 1-100,
  "urgency": "high|medium|low",
  "appointment_requested": true|false,
  "follow_up_required": true|false,
  "key_details": ["detail1", "detail2", "detail3"]
}

Consider these factors for lead scoring:
- Project type complexity (kitchen/bathroom = higher score)
- Customer engagement level
- Timeline urgency
- Budget indicators
- Location within service area
- Quality of conversation

Respond with ONLY the JSON object, no additional text.`,
		transcription, callInfo)
}

// buildSpamDetectionPrompt creates the prompt for spam detection
func (as *AnalysisService) buildSpamDetectionPrompt(transcription string, callDetails *models.CallDetails) string {
	var callInfo string
	if callDetails != nil {
		callInfo = fmt.Sprintf(`
CALLER: %s
PHONE: %s
DURATION: %d seconds`,
			callDetails.CustomerName,
			callDetails.CustomerPhoneNumber,
			callDetails.Duration)
	}

	return fmt.Sprintf(`
Analyze this phone call for spam likelihood:

TRANSCRIPT: %s
%s

Evaluate for spam indicators:
- Robotic or scripted speech patterns
- Generic sales pitches
- Suspicious caller behavior
- Short call duration with generic content
- Known spam phone patterns
- Telemarketing characteristics

Return ONLY a JSON object:
{
  "spam_likelihood": 0-100,
  "confidence": 0.0-1.0,
  "indicators": ["list", "of", "spam", "indicators"],
  "reasoning": "brief explanation of spam assessment"
}`, transcription, callInfo)
}

// buildSentimentAnalysisPrompt creates the prompt for sentiment analysis
func (as *AnalysisService) buildSentimentAnalysisPrompt(transcription string, callDetails *models.CallDetails) string {
	return fmt.Sprintf(`
Analyze the sentiment of this call transcription:

TRANSCRIPT: %s

Return ONLY a JSON object with:
{
  "sentiment": "positive|neutral|negative",
  "confidence": 0.0-1.0,
  "emotional_tone": "string describing emotional tone",
  "customer_satisfaction": "high|medium|low",
  "key_emotions": ["emotion1", "emotion2"]
}`, transcription)
}

// buildIntentClassificationPrompt creates the prompt for intent classification
func (as *AnalysisService) buildIntentClassificationPrompt(transcription string, callDetails *models.CallDetails) string {
	return fmt.Sprintf(`
Classify the primary intent of this customer call:

TRANSCRIPT: %s

Return ONLY a JSON object:
{
  "primary_intent": "quote_request|information_seeking|appointment_booking|complaint|follow_up|emergency|other",
  "confidence": 0.0-1.0,
  "secondary_intents": ["list", "of", "secondary", "intents"],
  "reasoning": "brief explanation of intent classification"
}`, transcription)
}

// buildLeadScoringPrompt creates the prompt for lead scoring
func (as *AnalysisService) buildLeadScoringPrompt(transcription string, callDetails *models.CallDetails) string {
	var callInfo string
	if callDetails != nil {
		callInfo = fmt.Sprintf(`
CALL CONTEXT:
- Duration: %d seconds
- Customer Location: %s, %s
- Source: %s`,
			callDetails.Duration,
			callDetails.CustomerCity,
			callDetails.CustomerState,
			callDetails.Source)
	}

	return fmt.Sprintf(`
Score this lead based on the call transcription:

TRANSCRIPT: %s
%s

Evaluate based on:
- Project complexity and value potential
- Customer readiness to buy
- Timeline urgency
- Budget capability indicators
- Engagement quality

Return ONLY a JSON object:
{
  "lead_score": 1-100,
  "confidence": 0.0-1.0,
  "scoring_factors": {
    "project_complexity": 1-10,
    "buying_readiness": 1-10,
    "timeline_urgency": 1-10,
    "budget_capability": 1-10,
    "engagement_quality": 1-10
  },
  "reasoning": "explanation of score"
}`, transcription, callInfo)
}

// createGeminiInstances creates instances for Gemini prediction
func (as *AnalysisService) createGeminiInstances(prompt string, config *AnalysisConfig) ([]*structpb.Value, error) {
	instance := map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			"temperature":     config.Temperature,
			"maxOutputTokens": config.MaxTokens,
			"topP":           config.TopP,
			"topK":           config.TopK,
		},
	}

	instanceValue, err := structpb.NewValue(instance)
	if err != nil {
		return nil, err
	}

	return []*structpb.Value{instanceValue}, nil
}

// createGeminiParameters creates parameters for Gemini prediction
func (as *AnalysisService) createGeminiParameters(config *AnalysisConfig) (*structpb.Value, error) {
	parameters := map[string]interface{}{
		"temperature":     config.Temperature,
		"maxOutputTokens": config.MaxTokens,
		"topP":           config.TopP,
		"topK":           config.TopK,
	}

	return structpb.NewValue(parameters)
}

// parseAnalysisResponse parses the Gemini response based on analysis type
func (as *AnalysisService) parseAnalysisResponse(resp *aiplatformpb.PredictResponse, analysisType AnalysisType, response *AnalysisResponse) error {
	if len(resp.Predictions) == 0 {
		return fmt.Errorf("no predictions returned")
	}

	prediction := resp.Predictions[0]
	predictionMap := prediction.GetStructValue().AsMap()

	content, ok := predictionMap["content"]
	if !ok {
		return fmt.Errorf("no content in prediction")
	}

	contentStr, ok := content.(string)
	if !ok {
		return fmt.Errorf("content is not a string")
	}

	response.Metadata.OutputTokens = int64(len(contentStr))

	switch analysisType {
	case AnalysisTypeContent:
		var analysis models.CallAnalysis
		if err := json.Unmarshal([]byte(contentStr), &analysis); err != nil {
			return fmt.Errorf("failed to parse content analysis response: %w", err)
		}
		response.CallAnalysis = &analysis

	case AnalysisTypeSpam:
		var spamResult SpamAnalysisResult
		if err := json.Unmarshal([]byte(contentStr), &spamResult); err != nil {
			return fmt.Errorf("failed to parse spam detection response: %w", err)
		}
		response.SpamResult = &spamResult

	case AnalysisTypeSentiment:
		var sentimentResult SentimentAnalysisResult
		if err := json.Unmarshal([]byte(contentStr), &sentimentResult); err != nil {
			return fmt.Errorf("failed to parse sentiment analysis response: %w", err)
		}
		response.SentimentResult = &sentimentResult

	case AnalysisTypeCustom:
		// For custom analysis, try to parse as generic JSON
		var customResult map[string]interface{}
		if err := json.Unmarshal([]byte(contentStr), &customResult); err != nil {
			// If JSON parsing fails, store as raw string
			customResult = map[string]interface{}{
				"raw_response": contentStr,
			}
		}
		response.CustomResult = customResult

	default:
		// For other types, store the raw content
		response.CustomResult = map[string]interface{}{
			"raw_response": contentStr,
		}
	}

	return nil
}