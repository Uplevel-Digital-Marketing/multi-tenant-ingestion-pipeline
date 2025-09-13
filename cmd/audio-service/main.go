package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"cloud.google.com/go/pubsub"

	"github.com/home-renovators/ingestion-pipeline/internal/ai"
	"github.com/home-renovators/ingestion-pipeline/internal/auth"
	"github.com/home-renovators/ingestion-pipeline/internal/spanner"
	"github.com/home-renovators/ingestion-pipeline/internal/storage"
	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

type AudioService struct {
	config         *config.Config
	authService    *auth.AuthService
	spannerRepo    *spanner.Repository
	storageService *storage.Service
	aiService      *ai.Service
	pubsubClient   *pubsub.Client
}

type AudioProcessingRequest struct {
	RecordingID   string `json:"recording_id"`
	TenantID      string `json:"tenant_id"`
	CallID        string `json:"call_id"`
	StorageURL    string `json:"storage_url"`
	RequestID     string `json:"request_id"`
	Priority      string `json:"priority,omitempty"` // high, normal, low
}

type AudioProcessingResponse struct {
	Status           string                      `json:"status"`
	RecordingID      string                      `json:"recording_id"`
	TranscriptionID  string                      `json:"transcription_id,omitempty"`
	Transcription    *models.TranscriptionResult `json:"transcription,omitempty"`
	ProcessingTimeMs int64                       `json:"processing_time_ms"`
	Error            string                      `json:"error,omitempty"`
}

func main() {
	ctx := context.Background()

	// Load configuration
	cfg := config.DefaultConfig()
	if err := cfg.LoadSecrets(ctx); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize services
	service, err := initializeServices(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}
	defer service.cleanup()

	// Set up HTTP server
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	service.setupRoutes(router)

	// Start background workers
	go service.startPubSubListener(ctx)

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down audio service...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Audio service starting on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func initializeServices(ctx context.Context, cfg *config.Config) (*AudioService, error) {
	// Initialize Spanner repository
	spannerRepo, err := spanner.NewRepository(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize spanner repository: %w", err)
	}

	// Initialize authentication service
	authService := auth.NewAuthService(cfg, spannerRepo)

	// Initialize storage service
	storageService, err := storage.NewService(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage service: %w", err)
	}

	// Initialize AI service
	aiService, err := ai.NewService(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI service: %w", err)
	}

	// Initialize Pub/Sub client
	pubsubClient, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Pub/Sub client: %w", err)
	}

	return &AudioService{
		config:         cfg,
		authService:    authService,
		spannerRepo:    spannerRepo,
		storageService: storageService,
		aiService:      aiService,
		pubsubClient:   pubsubClient,
	}, nil
}

func (s *AudioService) cleanup() {
	if s.spannerRepo != nil {
		s.spannerRepo.Close()
	}
	if s.storageService != nil {
		s.storageService.Close()
	}
	if s.aiService != nil {
		s.aiService.Close()
	}
	if s.pubsubClient != nil {
		s.pubsubClient.Close()
	}
}

func (s *AudioService) setupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", s.healthCheck)

	// API routes
	api := router.Group("/api/v1")
	{
		// Audio processing endpoints
		api.POST("/audio/transcribe", s.handleTranscribeAudio)
		api.GET("/audio/status/:recording_id", s.handleGetTranscriptionStatus)
		api.POST("/audio/process-batch", s.handleBatchProcess)
	}
}

func (s *AudioService) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "audio-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *AudioService) handleTranscribeAudio(c *gin.Context) {
	ctx := c.Request.Context()
	startTime := time.Now()

	var req AudioProcessingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.TenantID == "" || req.StorageURL == "" || req.CallID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	// Process audio transcription
	result, err := s.processAudioTranscription(ctx, &req)
	if err != nil {
		log.Printf("Failed to process audio transcription: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transcription failed"})
		return
	}

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, result)
}

func (s *AudioService) handleGetTranscriptionStatus(c *gin.Context) {
	ctx := c.Request.Context()
	recordingID := c.Param("recording_id")
	tenantID := c.Query("tenant_id")

	if recordingID == "" || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing recording_id or tenant_id"})
		return
	}

	// Get recording status from database
	recording, err := s.spannerRepo.GetCallRecording(ctx, tenantID, recordingID)
	if err != nil {
		log.Printf("Failed to get recording status: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Recording not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recording_id":         recording.RecordingID,
		"tenant_id":           recording.TenantID,
		"call_id":             recording.CallID,
		"transcription_status": recording.TranscriptionStatus,
		"created_at":          recording.CreatedAt,
	})
}

func (s *AudioService) handleBatchProcess(c *gin.Context) {
	ctx := c.Request.Context()

	var requests []AudioProcessingRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No requests provided"})
		return
	}

	if len(requests) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many requests (max 10)"})
		return
	}

	// Process batch requests
	results := make([]AudioProcessingResponse, len(requests))
	for i, req := range requests {
		result, err := s.processAudioTranscription(ctx, &req)
		if err != nil {
			result = &AudioProcessingResponse{
				Status:      "failed",
				RecordingID: req.RecordingID,
				Error:       err.Error(),
			}
		}
		results[i] = *result
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}

func (s *AudioService) processAudioTranscription(ctx context.Context, req *AudioProcessingRequest) (*AudioProcessingResponse, error) {
	log.Printf("Processing audio transcription for recording %s, call %s", req.RecordingID, req.CallID)

	// Update transcription status to processing
	if req.RecordingID != "" {
		if err := s.spannerRepo.UpdateCallRecordingStatus(ctx, req.RecordingID, "processing"); err != nil {
			log.Printf("Failed to update recording status to processing: %v", err)
		}
	}

	// Transcribe audio using AI service
	transcription, err := s.aiService.TranscribeAudio(ctx, req.StorageURL)
	if err != nil {
		// Update status to failed
		if req.RecordingID != "" {
			s.spannerRepo.UpdateCallRecordingStatus(ctx, req.RecordingID, "failed")
		}
		return nil, fmt.Errorf("transcription failed: %w", err)
	}

	// Serialize transcription result
	transcriptionJSON, _ := json.Marshal(transcription)

	// Update transcription status to completed
	if req.RecordingID != "" {
		if err := s.spannerRepo.UpdateCallRecordingStatus(ctx, req.RecordingID, "completed"); err != nil {
			log.Printf("Failed to update recording status to completed: %v", err)
		}

		// Store transcription completion time
		if err := s.spannerRepo.UpdateCallRecordingTranscription(ctx, req.RecordingID, string(transcriptionJSON)); err != nil {
			log.Printf("Failed to update transcription completion time: %v", err)
		}
	}

	// Create AI processing log entry
	processingLog := &models.AIProcessingLog{
		LogID:          models.NewProcessingID(),
		TenantID:       req.TenantID,
		RequestID:      req.RequestID,
		AnalysisType:   "transcription",
		Status:         "completed",
		ProcessingData: string(transcriptionJSON),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.spannerRepo.CreateAIProcessingLog(ctx, processingLog); err != nil {
		log.Printf("Failed to create AI processing log: %v", err)
		// Continue processing even if logging fails
	}

	// Publish transcription completed event
	if err := s.publishTranscriptionCompletedEvent(ctx, req, transcription); err != nil {
		log.Printf("Failed to publish transcription completed event: %v", err)
		// Continue processing even if event publishing fails
	}

	return &AudioProcessingResponse{
		Status:          "completed",
		RecordingID:     req.RecordingID,
		TranscriptionID: processingLog.LogID,
		Transcription:   transcription,
	}, nil
}

func (s *AudioService) publishTranscriptionCompletedEvent(ctx context.Context, req *AudioProcessingRequest, transcription *models.TranscriptionResult) error {
	topic := s.pubsubClient.Topic("transcription-completed")

	event := map[string]interface{}{
		"event_type":    "transcription.completed",
		"tenant_id":     req.TenantID,
		"call_id":       req.CallID,
		"recording_id":  req.RecordingID,
		"request_id":    req.RequestID,
		"transcription": transcription,
		"timestamp":     time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	result := topic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"event_type": "transcription.completed",
			"tenant_id":  req.TenantID,
			"call_id":    req.CallID,
		},
	})

	_, err = result.Get(ctx)
	return err
}

func (s *AudioService) startPubSubListener(ctx context.Context) {
	sub := s.pubsubClient.Subscription("audio-processing-requests")

	log.Println("Starting Pub/Sub listener for audio processing requests...")

	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var req AudioProcessingRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("Failed to unmarshal audio processing request: %v", err)
			msg.Nack()
			return
		}

		// Process the audio transcription
		result, err := s.processAudioTranscription(ctx, &req)
		if err != nil {
			log.Printf("Failed to process audio from Pub/Sub: %v", err)
			msg.Nack()
			return
		}

		log.Printf("Successfully processed audio from Pub/Sub: %s", result.RecordingID)
		msg.Ack()
	})

	if err != nil {
		log.Printf("Pub/Sub receive error: %v", err)
	}
}