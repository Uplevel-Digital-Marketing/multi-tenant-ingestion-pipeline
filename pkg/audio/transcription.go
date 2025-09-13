package audio

import (
	"context"
	"fmt"
	"strings"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

// TranscriptionService handles audio transcription using Google Speech-to-Text
type TranscriptionService struct {
	speechClient *speech.Client
	config       *TranscriptionConfig
}

// TranscriptionConfig contains configuration for the transcription service
type TranscriptionConfig struct {
	ProjectID            string  `json:"project_id"`
	Location             string  `json:"location"`
	Model                string  `json:"model"` // e.g., "chirp-3"
	LanguageCode         string  `json:"language_code"`
	SampleRateHertz      int32   `json:"sample_rate_hertz"`
	EnableDiarization    bool    `json:"enable_diarization"`
	EnablePunctuation    bool    `json:"enable_punctuation"`
	EnableWordTimestamp  bool    `json:"enable_word_timestamp"`
	EnableWordConfidence bool    `json:"enable_word_confidence"`
	MinSpeakerCount      int32   `json:"min_speaker_count"`
	MaxSpeakerCount      int32   `json:"max_speaker_count"`
	AudioEncoding        speechpb.RecognitionConfig_AudioEncoding `json:"audio_encoding"`
	UseEnhanced          bool    `json:"use_enhanced"`
}

// DefaultTranscriptionConfig returns default configuration for phone calls
func DefaultTranscriptionConfig() *TranscriptionConfig {
	return &TranscriptionConfig{
		Model:                "chirp-3",
		LanguageCode:         "en-US",
		SampleRateHertz:      8000, // Typical for phone calls
		EnableDiarization:    true,
		EnablePunctuation:    true,
		EnableWordTimestamp:  true,
		EnableWordConfidence: true,
		MinSpeakerCount:      1,
		MaxSpeakerCount:      2, // Customer and agent
		AudioEncoding:        speechpb.RecognitionConfig_MP3,
		UseEnhanced:          true,
	}
}

// NewTranscriptionService creates a new transcription service
func NewTranscriptionService(ctx context.Context, config *TranscriptionConfig) (*TranscriptionService, error) {
	if config == nil {
		config = DefaultTranscriptionConfig()
	}

	speechClient, err := speech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech client: %w", err)
	}

	return &TranscriptionService{
		speechClient: speechClient,
		config:       config,
	}, nil
}

// Close closes the transcription service
func (ts *TranscriptionService) Close() error {
	return ts.speechClient.Close()
}

// TranscriptionRequest contains the parameters for a transcription request
type TranscriptionRequest struct {
	AudioURI         string                     `json:"audio_uri"`
	CallID           string                     `json:"call_id"`
	TenantID         string                     `json:"tenant_id"`
	CustomConfig     *TranscriptionConfig       `json:"custom_config,omitempty"`
	Metadata         map[string]string          `json:"metadata,omitempty"`
	Priority         TranscriptionPriority      `json:"priority"`
	Timeout          time.Duration              `json:"timeout"`
}

// TranscriptionPriority defines the priority of transcription requests
type TranscriptionPriority string

const (
	PriorityLow    TranscriptionPriority = "low"
	PriorityNormal TranscriptionPriority = "normal"
	PriorityHigh   TranscriptionPriority = "high"
	PriorityUrgent TranscriptionPriority = "urgent"
)

// TranscriptionResponse contains the full response from transcription
type TranscriptionResponse struct {
	Request      *TranscriptionRequest      `json:"request"`
	Result       *models.TranscriptionResult `json:"result"`
	Success      bool                       `json:"success"`
	Error        string                     `json:"error,omitempty"`
	StartTime    time.Time                  `json:"start_time"`
	EndTime      time.Time                  `json:"end_time"`
	Duration     time.Duration              `json:"duration"`
	Metadata     map[string]interface{}     `json:"metadata"`
}

// TranscribeAudio transcribes audio from a URI using Speech-to-Text API
func (ts *TranscriptionService) TranscribeAudio(ctx context.Context, req *TranscriptionRequest) (*TranscriptionResponse, error) {
	startTime := time.Now()

	response := &TranscriptionResponse{
		Request:   req,
		StartTime: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// Use custom config if provided, otherwise use service default
	config := ts.config
	if req.CustomConfig != nil {
		config = req.CustomConfig
	}

	// Set timeout if specified
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// Create recognition config
	recognitionConfig := &speechpb.RecognitionConfig{
		Encoding:                   config.AudioEncoding,
		SampleRateHertz:           config.SampleRateHertz,
		LanguageCode:              config.LanguageCode,
		EnableAutomaticPunctuation: config.EnablePunctuation,
		EnableWordTimeOffsets:      config.EnableWordTimestamp,
		EnableWordConfidence:       config.EnableWordConfidence,
		Model:                      config.Model,
		UseEnhanced:               config.UseEnhanced,
	}

	// Configure speaker diarization if enabled
	if config.EnableDiarization {
		recognitionConfig.DiarizationConfig = &speechpb.SpeakerDiarizationConfig{
			EnableSpeakerDiarization: true,
			MinSpeakerCount:         config.MinSpeakerCount,
			MaxSpeakerCount:         config.MaxSpeakerCount,
		}
	}

	// Create audio source
	audio := &speechpb.RecognitionAudio{
		AudioSource: &speechpb.RecognitionAudio_Uri{
			Uri: req.AudioURI,
		},
	}

	// Create long running recognition request
	longRunningReq := &speechpb.LongRunningRecognizeRequest{
		Config: recognitionConfig,
		Audio:  audio,
	}

	// Start long-running recognition
	operation, err := ts.speechClient.LongRunningRecognize(ctx, longRunningReq)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("failed to start transcription: %v", err)
		response.EndTime = time.Now()
		response.Duration = time.Since(startTime)
		return response, err
	}

	// Store operation metadata
	response.Metadata["operation_name"] = operation.Name()

	// Wait for the operation to complete
	speechResponse, err := operation.Wait(ctx)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("transcription failed: %v", err)
		response.EndTime = time.Now()
		response.Duration = time.Since(startTime)
		return response, err
	}

	// Process the results
	result := ts.processTranscriptionResults(speechResponse, config)

	response.Result = result
	response.Success = true
	response.EndTime = time.Now()
	response.Duration = time.Since(startTime)
	response.Metadata["total_results"] = len(speechResponse.Results)

	return response, nil
}

// TranscribeBatch transcribes multiple audio files in batch
func (ts *TranscriptionService) TranscribeBatch(ctx context.Context, requests []*TranscriptionRequest) ([]*TranscriptionResponse, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no transcription requests provided")
	}

	responses := make([]*TranscriptionResponse, len(requests))

	// Process requests concurrently with a semaphore to limit concurrency
	semaphore := make(chan struct{}, 5) // Allow up to 5 concurrent transcriptions

	type result struct {
		index    int
		response *TranscriptionResponse
		err      error
	}

	resultChan := make(chan result, len(requests))

	for i, req := range requests {
		go func(index int, request *TranscriptionRequest) {
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			resp, err := ts.TranscribeAudio(ctx, request)
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

// processTranscriptionResults processes Speech-to-Text API results
func (ts *TranscriptionService) processTranscriptionResults(resp *speechpb.LongRunningRecognizeResponse, config *TranscriptionConfig) *models.TranscriptionResult {
	result := &models.TranscriptionResult{
		SpeakerDiarization: []models.SpeakerSegment{},
		WordDetails:        []models.WordDetail{},
	}

	var transcriptBuilder strings.Builder
	var totalConfidence float32
	var totalDuration float64
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

		// Calculate duration from word timings
		if len(alt.Words) > 0 {
			firstWord := alt.Words[0]
			lastWord := alt.Words[len(alt.Words)-1]
			if firstWord.StartTime != nil && lastWord.EndTime != nil {
				segmentDuration := lastWord.EndTime.Seconds + float64(lastWord.EndTime.Nanos)/1e9 -
								 (firstWord.StartTime.Seconds + float64(firstWord.StartTime.Nanos)/1e9)
				if segmentDuration > totalDuration {
					totalDuration = segmentDuration
				}
			}
		}

		// Process speaker diarization if enabled
		if config.EnableDiarization && len(alt.Words) > 0 {
			currentSpeaker := alt.Words[0].SpeakerTag
			segmentStart := alt.Words[0].StartTime
			var segmentText strings.Builder

			for _, word := range alt.Words {
				// Check for speaker change
				if word.SpeakerTag != currentSpeaker && segmentText.Len() > 0 {
					// Save current segment
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

				// Add word details if enabled
				if config.EnableWordTimestamp {
					result.WordDetails = append(result.WordDetails, models.WordDetail{
						Word:       word.Word,
						StartTime:  formatDuration(word.StartTime),
						EndTime:    formatDuration(word.EndTime),
						Confidence: word.Confidence,
					})
				}
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

	// Set final results
	result.Transcript = strings.TrimSpace(transcriptBuilder.String())
	result.Duration = totalDuration

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

// TranscriptionStatus represents the status of a transcription job
type TranscriptionStatus struct {
	JobID       string                 `json:"job_id"`
	Status      string                 `json:"status"` // pending, processing, completed, failed
	Progress    float64               `json:"progress"` // 0.0 to 1.0
	StartTime   time.Time             `json:"start_time"`
	EndTime     *time.Time            `json:"end_time,omitempty"`
	Error       string                `json:"error,omitempty"`
	Result      *models.TranscriptionResult `json:"result,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TranscriptionJob represents a transcription job for tracking
type TranscriptionJob struct {
	JobID    string                `json:"job_id"`
	Request  *TranscriptionRequest `json:"request"`
	Status   *TranscriptionStatus  `json:"status"`
	Response *TranscriptionResponse `json:"response,omitempty"`
}

// JobManager manages transcription jobs
type JobManager struct {
	jobs map[string]*TranscriptionJob
}

// NewJobManager creates a new job manager
func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*TranscriptionJob),
	}
}

// CreateJob creates a new transcription job
func (jm *JobManager) CreateJob(request *TranscriptionRequest) *TranscriptionJob {
	jobID := fmt.Sprintf("transcription_%s_%d", request.CallID, time.Now().UnixNano())

	job := &TranscriptionJob{
		JobID:   jobID,
		Request: request,
		Status: &TranscriptionStatus{
			JobID:     jobID,
			Status:    "pending",
			Progress:  0.0,
			StartTime: time.Now(),
			Metadata:  make(map[string]interface{}),
		},
	}

	jm.jobs[jobID] = job
	return job
}

// UpdateJobStatus updates the status of a transcription job
func (jm *JobManager) UpdateJobStatus(jobID string, status string, progress float64, error string) {
	if job, exists := jm.jobs[jobID]; exists {
		job.Status.Status = status
		job.Status.Progress = progress
		if error != "" {
			job.Status.Error = error
		}
		if status == "completed" || status == "failed" {
			now := time.Now()
			job.Status.EndTime = &now
		}
	}
}

// CompleteJob marks a job as completed with results
func (jm *JobManager) CompleteJob(jobID string, response *TranscriptionResponse) {
	if job, exists := jm.jobs[jobID]; exists {
		job.Response = response
		job.Status.Status = "completed"
		job.Status.Progress = 1.0
		job.Status.Result = response.Result
		now := time.Now()
		job.Status.EndTime = &now
	}
}

// GetJob retrieves a transcription job by ID
func (jm *JobManager) GetJob(jobID string) (*TranscriptionJob, bool) {
	job, exists := jm.jobs[jobID]
	return job, exists
}

// GetJobStatus retrieves just the status of a transcription job
func (jm *JobManager) GetJobStatus(jobID string) (*TranscriptionStatus, bool) {
	if job, exists := jm.jobs[jobID]; exists {
		return job.Status, true
	}
	return nil, false
}

// ListJobs returns all jobs (optionally filtered by status)
func (jm *JobManager) ListJobs(statusFilter string) []*TranscriptionJob {
	var jobs []*TranscriptionJob
	for _, job := range jm.jobs {
		if statusFilter == "" || job.Status.Status == statusFilter {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// CleanupCompletedJobs removes completed/failed jobs older than the specified duration
func (jm *JobManager) CleanupCompletedJobs(olderThan time.Duration) int {
	cutoff := time.Now().Add(-olderThan)
	cleaned := 0

	for jobID, job := range jm.jobs {
		if (job.Status.Status == "completed" || job.Status.Status == "failed") &&
		   job.Status.EndTime != nil && job.Status.EndTime.Before(cutoff) {
			delete(jm.jobs, jobID)
			cleaned++
		}
	}

	return cleaned
}