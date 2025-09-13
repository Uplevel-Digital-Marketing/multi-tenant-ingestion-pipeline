package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

// Service handles AI operations including Speech-to-Text and Vertex AI
type Service struct {
	speechClient *speech.Client
	aiClient     *aiplatform.PredictionClient
	config       *config.Config
}

// NewService creates a new AI service
func NewService(ctx context.Context, cfg *config.Config) (*Service, error) {
	speechClient, err := speech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech client: %w", err)
	}

	aiClient, err := aiplatform.NewPredictionClient(ctx)
	if err != nil {
		speechClient.Close()
		return nil, fmt.Errorf("failed to create AI platform client: %w", err)
	}

	return &Service{
		speechClient: speechClient,
		aiClient:     aiClient,
		config:       cfg,
	}, nil
}

// Close closes the AI service clients
func (s *Service) Close() error {
	if err := s.speechClient.Close(); err != nil {
		return err
	}
	return s.aiClient.Close()
}

// TranscribeAudio transcribes audio using Speech-to-Text API with Chirp 3
func (s *Service) TranscribeAudio(ctx context.Context, audioFileURL string) (*models.TranscriptionResult, error) {
	config := &speechpb.RecognitionConfig{
		Encoding:                   speechpb.RecognitionConfig_MP3,
		SampleRateHertz:           8000, // Typical for phone calls
		LanguageCode:              s.config.SpeechLanguage,
		EnableAutomaticPunctuation: true,
		EnableWordTimeOffsets:      true,
		EnableWordConfidence:       true,
		Model:                      s.config.SpeechToTextModel, // "chirp-3"
		UseEnhanced:               true,
		DiarizationConfig: &speechpb.SpeakerDiarizationConfig{
			EnableSpeakerDiarization: s.config.EnableDiarization,
			MinSpeakerCount:         1,
			MaxSpeakerCount:         2, // Typical for customer service calls
		},
	}

	audio := &speechpb.RecognitionAudio{
		AudioSource: &speechpb.RecognitionAudio_Uri{
			Uri: audioFileURL,
		},
	}

	req := &speechpb.LongRunningRecognizeRequest{
		Config: config,
		Audio:  audio,
	}

	// Start long-running recognition
	op, err := s.speechClient.LongRunningRecognize(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start transcription: %w", err)
	}

	// Wait for the operation to complete
	resp, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Process the results
	return s.processTranscriptionResults(resp), nil
}

// processTranscriptionResults processes Speech-to-Text API results
func (s *Service) processTranscriptionResults(resp *speechpb.LongRunningRecognizeResponse) *models.TranscriptionResult {
	result := &models.TranscriptionResult{
		SpeakerDiarization: []models.SpeakerSegment{},
		WordDetails:        []models.WordDetail{},
	}

	var transcriptBuilder strings.Builder
	var totalConfidence float32
	wordCount := 0

	for _, res := range resp.Results {
		if len(res.Alternatives) == 0 {
			continue
		}

		alt := res.Alternatives[0]
		transcriptBuilder.WriteString(alt.Transcript)
		transcriptBuilder.WriteString(" ")

		totalConfidence += alt.Confidence
		wordCount++

		// Process speaker diarization
		if len(alt.Words) > 0 {
			currentSpeaker := alt.Words[0].SpeakerTag
			segmentStart := alt.Words[0].StartTime
			var segmentText strings.Builder

			for _, word := range alt.Words {
				if word.SpeakerTag != currentSpeaker {
					// Speaker changed, save current segment
					result.SpeakerDiarization = append(result.SpeakerDiarization, models.SpeakerSegment{
						Speaker:   int(currentSpeaker),
						StartTime: formatDuration(segmentStart),
						EndTime:   formatDuration(word.StartTime),
						Text:      strings.TrimSpace(segmentText.String()),
					})

					// Start new segment
					currentSpeaker = word.SpeakerTag
					segmentStart = word.StartTime
					segmentText.Reset()
				}

				segmentText.WriteString(word.Word)
				segmentText.WriteString(" ")

				// Add word details
				result.WordDetails = append(result.WordDetails, models.WordDetail{
					Word:       word.Word,
					StartTime:  formatDuration(word.StartTime),
					EndTime:    formatDuration(word.EndTime),
					Confidence: word.Confidence,
				})
			}

			// Add final segment
			if segmentText.Len() > 0 {
				lastWord := alt.Words[len(alt.Words)-1]
				result.SpeakerDiarization = append(result.SpeakerDiarization, models.SpeakerSegment{
					Speaker:   int(currentSpeaker),
					StartTime: formatDuration(segmentStart),
					EndTime:   formatDuration(lastWord.EndTime),
					Text:      strings.TrimSpace(segmentText.String()),
				})
			}
		}
	}

	result.Transcript = strings.TrimSpace(transcriptBuilder.String())
	if wordCount > 0 {
		result.Confidence = totalConfidence / float32(wordCount)
	}

	// Count unique speakers
	speakerSet := make(map[int]bool)
	for _, segment := range result.SpeakerDiarization {
		speakerSet[segment.Speaker] = true
	}
	result.SpeakerCount = len(speakerSet)

	return result
}

// AnalyzeCallContent analyzes call content using Gemini 2.5 Flash
func (s *Service) AnalyzeCallContent(ctx context.Context, transcription string, callDetails models.CallDetails) (*models.CallAnalysis, error) {
	prompt := s.buildAnalysisPrompt(transcription, callDetails)

	// Prepare the request for Vertex AI
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s",
		s.config.VertexAIProject, s.config.VertexAILocation, s.config.VertexAIModel)

	// Create the prediction request
	instances, err := s.createGeminiInstances(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create instances: %w", err)
	}

	parameters, err := s.createGeminiParameters()
	if err != nil {
		return nil, fmt.Errorf("failed to create parameters: %w", err)
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:   endpoint,
		Instances:  instances,
		Parameters: parameters,
	}

	// Make the prediction
	resp, err := s.aiClient.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("prediction failed: %w", err)
	}

	// Parse the response
	analysis, err := s.parseGeminiResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return analysis, nil
}

// buildAnalysisPrompt creates the analysis prompt for Gemini
func (s *Service) buildAnalysisPrompt(transcription string, callDetails models.CallDetails) string {
	return fmt.Sprintf(`
Analyze this phone call transcription for a home remodeling company:

TRANSCRIPT: %s

CALL METADATA:
- Customer Name: %s
- Customer Phone: %s
- Customer Location: %s, %s
- Call Duration: %d seconds
- Source: %s
- Tags: %s
- Lead Status: %s

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
		transcription,
		callDetails.CustomerName,
		callDetails.CustomerPhoneNumber,
		callDetails.CustomerCity,
		callDetails.CustomerState,
		callDetails.Duration,
		callDetails.Source,
		strings.Join(callDetails.Tags, ", "),
		callDetails.LeadStatus)
}

// createGeminiInstances creates instances for Gemini prediction
func (s *Service) createGeminiInstances(prompt string) ([]*structpb.Value, error) {
	instance := map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			"temperature":     0.2,
			"maxOutputTokens": 1024,
			"topP":            0.8,
			"topK":            40,
		},
	}

	instanceValue, err := structpb.NewValue(instance)
	if err != nil {
		return nil, err
	}

	return []*structpb.Value{instanceValue}, nil
}

// createGeminiParameters creates parameters for Gemini prediction
func (s *Service) createGeminiParameters() (*structpb.Value, error) {
	parameters := map[string]interface{}{
		"temperature":     0.2,
		"maxOutputTokens": 1024,
		"topP":            0.8,
		"topK":            40,
	}

	return structpb.NewValue(parameters)
}

// parseGeminiResponse parses the Gemini response into CallAnalysis
func (s *Service) parseGeminiResponse(resp *aiplatformpb.PredictResponse) (*models.CallAnalysis, error) {
	if len(resp.Predictions) == 0 {
		return nil, fmt.Errorf("no predictions returned")
	}

	prediction := resp.Predictions[0]

	// Extract the content from the prediction
	predictionMap := prediction.GetStructValue().AsMap()

	content, ok := predictionMap["content"]
	if !ok {
		return nil, fmt.Errorf("no content in prediction")
	}

	contentStr, ok := content.(string)
	if !ok {
		return nil, fmt.Errorf("content is not a string")
	}

	// Parse the JSON response
	var analysis models.CallAnalysis
	if err := json.Unmarshal([]byte(contentStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &analysis, nil
}

// DetectSpam analyzes content for spam likelihood
func (s *Service) DetectSpam(ctx context.Context, transcription string, callDetails models.CallDetails) (float64, error) {
	prompt := fmt.Sprintf(`
Analyze this phone call for spam likelihood:

TRANSCRIPT: %s
CALLER: %s
PHONE: %s
DURATION: %d seconds

Evaluate for spam indicators:
- Robotic or scripted speech patterns
- Generic sales pitches
- Suspicious caller behavior
- Short call duration with generic content
- Known spam phone patterns

Return ONLY a number between 0-100 representing spam likelihood percentage.
`, transcription, callDetails.CustomerName, callDetails.CustomerPhoneNumber, callDetails.Duration)

	// Use similar Gemini prediction logic
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s",
		s.config.VertexAIProject, s.config.VertexAILocation, s.config.VertexAIModel)

	instances, err := s.createGeminiInstances(prompt)
	if err != nil {
		return 0, fmt.Errorf("failed to create instances: %w", err)
	}

	parameters, err := s.createGeminiParameters()
	if err != nil {
		return 0, fmt.Errorf("failed to create parameters: %w", err)
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:   endpoint,
		Instances:  instances,
		Parameters: parameters,
	}

	resp, err := s.aiClient.Predict(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("spam detection failed: %w", err)
	}

	// Parse spam likelihood from response
	if len(resp.Predictions) == 0 {
		return 0, fmt.Errorf("no spam predictions returned")
	}

	prediction := resp.Predictions[0]
	predictionMap := prediction.GetStructValue().AsMap()

	content, ok := predictionMap["content"]
	if !ok {
		return 0, fmt.Errorf("no content in spam prediction")
	}

	contentStr, ok := content.(string)
	if !ok {
		return 0, fmt.Errorf("spam content is not a string")
	}

	var spamLikelihood float64
	if _, err := fmt.Sscanf(strings.TrimSpace(contentStr), "%f", &spamLikelihood); err != nil {
		return 0, fmt.Errorf("failed to parse spam likelihood: %w", err)
	}

	return spamLikelihood, nil
}

// formatDuration formats a protobuf duration to string
func formatDuration(duration *durationpb.Duration) string {
	if duration == nil {
		return "0.0s"
	}

	seconds := duration.Seconds
	nanos := duration.Nanos

	totalSeconds := float64(seconds) + float64(nanos)/1e9
	return fmt.Sprintf("%.1fs", totalSeconds)
}