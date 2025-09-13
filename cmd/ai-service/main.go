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
	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

type AIAnalysisService struct {
	config       *config.Config
	authService  *auth.AuthService
	spannerRepo  *spanner.Repository
	aiService    *ai.Service
	pubsubClient *pubsub.Client
}

type AnalysisRequest struct {
	RequestID     string              `json:"request_id"`
	TenantID      string              `json:"tenant_id"`
	CallID        string              `json:"call_id"`
	Transcription string              `json:"transcription"`
	CallDetails   models.CallDetails  `json:"call_details"`
	AnalysisType  string              `json:"analysis_type"` // content_analysis, spam_detection, sentiment_analysis
	Priority      string              `json:"priority,omitempty"` // high, normal, low
}

type AnalysisResponse struct {
	Status           string                 `json:"status"`
	RequestID        string                 `json:"request_id"`
	AnalysisID       string                 `json:"analysis_id,omitempty"`
	CallAnalysis     *models.CallAnalysis   `json:"call_analysis,omitempty"`
	SpamLikelihood   *float64               `json:"spam_likelihood,omitempty"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
	Error            string                 `json:"error,omitempty"`
}

type BatchAnalysisRequest struct {
	Requests []AnalysisRequest `json:"requests"`
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

		log.Println("Shutting down AI analysis service...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("AI analysis service starting on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func initializeServices(ctx context.Context, cfg *config.Config) (*AIAnalysisService, error) {
	// Initialize Spanner repository
	spannerRepo, err := spanner.NewRepository(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize spanner repository: %w", err)
	}

	// Initialize authentication service
	authService := auth.NewAuthService(cfg, spannerRepo)

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

	return &AIAnalysisService{
		config:       cfg,
		authService:  authService,
		spannerRepo:  spannerRepo,
		aiService:    aiService,
		pubsubClient: pubsubClient,
	}, nil
}

func (s *AIAnalysisService) cleanup() {
	if s.spannerRepo != nil {
		s.spannerRepo.Close()
	}
	if s.aiService != nil {
		s.aiService.Close()
	}
	if s.pubsubClient != nil {
		s.pubsubClient.Close()
	}
}

func (s *AIAnalysisService) setupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", s.healthCheck)

	// API routes
	api := router.Group("/api/v1")
	{
		// AI analysis endpoints
		api.POST("/analysis/content", s.handleContentAnalysis)
		api.POST("/analysis/spam-detection", s.handleSpamDetection)
		api.POST("/analysis/sentiment", s.handleSentimentAnalysis)
		api.POST("/analysis/batch", s.handleBatchAnalysis)
		api.GET("/analysis/status/:analysis_id", s.handleGetAnalysisStatus)
	}
}

func (s *AIAnalysisService) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "ai-analysis-service",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"ai_model":  s.config.VertexAIModel,
	})
}

func (s *AIAnalysisService) handleContentAnalysis(c *gin.Context) {
	ctx := c.Request.Context()
	startTime := time.Now()

	var req AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.AnalysisType = "content_analysis"

	// Process content analysis
	result, err := s.processAnalysis(ctx, &req)
	if err != nil {
		log.Printf("Failed to process content analysis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Analysis failed"})
		return
	}

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, result)
}

func (s *AIAnalysisService) handleSpamDetection(c *gin.Context) {
	ctx := c.Request.Context()
	startTime := time.Now()

	var req AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.AnalysisType = "spam_detection"

	// Process spam detection
	result, err := s.processAnalysis(ctx, &req)
	if err != nil {
		log.Printf("Failed to process spam detection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Spam detection failed"})
		return
	}

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, result)
}

func (s *AIAnalysisService) handleSentimentAnalysis(c *gin.Context) {
	ctx := c.Request.Context()
	startTime := time.Now()

	var req AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.AnalysisType = "sentiment_analysis"

	// Process sentiment analysis
	result, err := s.processAnalysis(ctx, &req)
	if err != nil {
		log.Printf("Failed to process sentiment analysis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sentiment analysis failed"})
		return
	}

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, result)
}

func (s *AIAnalysisService) handleBatchAnalysis(c *gin.Context) {
	ctx := c.Request.Context()

	var batchReq BatchAnalysisRequest
	if err := c.ShouldBindJSON(&batchReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(batchReq.Requests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No requests provided"})
		return
	}

	if len(batchReq.Requests) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many requests (max 20)"})
		return
	}

	// Process batch requests
	results := make([]AnalysisResponse, len(batchReq.Requests))
	for i, req := range batchReq.Requests {
		result, err := s.processAnalysis(ctx, &req)
		if err != nil {
			result = &AnalysisResponse{
				Status:    "failed",
				RequestID: req.RequestID,
				Error:     err.Error(),
			}
		}
		results[i] = *result
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}

func (s *AIAnalysisService) handleGetAnalysisStatus(c *gin.Context) {
	ctx := c.Request.Context()
	analysisID := c.Param("analysis_id")
	tenantID := c.Query("tenant_id")

	if analysisID == "" || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing analysis_id or tenant_id"})
		return
	}

	// Get analysis status from database
	processingLog, err := s.spannerRepo.GetAIProcessingLog(ctx, tenantID, analysisID)
	if err != nil {
		log.Printf("Failed to get analysis status: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Analysis not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"analysis_id":     processingLog.LogID,
		"tenant_id":       processingLog.TenantID,
		"request_id":      processingLog.RequestID,
		"analysis_type":   processingLog.AnalysisType,
		"status":          processingLog.Status,
		"processing_data": processingLog.ProcessingData,
		"created_at":      processingLog.CreatedAt,
		"updated_at":      processingLog.UpdatedAt,
	})
}

func (s *AIAnalysisService) processAnalysis(ctx context.Context, req *AnalysisRequest) (*AnalysisResponse, error) {
	log.Printf("Processing %s for request %s, call %s", req.AnalysisType, req.RequestID, req.CallID)

	startTime := time.Now()
	var result *AnalysisResponse

	switch req.AnalysisType {
	case "content_analysis":
		result = s.processContentAnalysis(ctx, req)
	case "spam_detection":
		result = s.processSpamDetection(ctx, req)
	case "sentiment_analysis":
		result = s.processSentimentAnalysis(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported analysis type: %s", req.AnalysisType)
	}

	if result.Error != "" {
		return result, fmt.Errorf("analysis failed: %s", result.Error)
	}

	// Create AI processing log entry
	processingLog := &models.AIProcessingLog{
		LogID:          models.NewProcessingID(),
		TenantID:       req.TenantID,
		RequestID:      req.RequestID,
		AnalysisType:   req.AnalysisType,
		Status:         result.Status,
		ProcessingData: s.serializeProcessingData(result, time.Since(startTime)),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.spannerRepo.CreateAIProcessingLog(ctx, processingLog); err != nil {
		log.Printf("Failed to create AI processing log: %v", err)
		// Continue processing even if logging fails
	}

	result.AnalysisID = processingLog.LogID

	// Publish analysis completed event
	if err := s.publishAnalysisCompletedEvent(ctx, req, result); err != nil {
		log.Printf("Failed to publish analysis completed event: %v", err)
		// Continue processing even if event publishing fails
	}

	return result, nil
}

func (s *AIAnalysisService) processContentAnalysis(ctx context.Context, req *AnalysisRequest) *AnalysisResponse {
	analysis, err := s.aiService.AnalyzeCallContent(ctx, req.Transcription, req.CallDetails)
	if err != nil {
		return &AnalysisResponse{
			Status:    "failed",
			RequestID: req.RequestID,
			Error:     err.Error(),
		}
	}

	return &AnalysisResponse{
		Status:       "completed",
		RequestID:    req.RequestID,
		CallAnalysis: analysis,
	}
}

func (s *AIAnalysisService) processSpamDetection(ctx context.Context, req *AnalysisRequest) *AnalysisResponse {
	spamLikelihood, err := s.aiService.DetectSpam(ctx, req.Transcription, req.CallDetails)
	if err != nil {
		return &AnalysisResponse{
			Status:    "failed",
			RequestID: req.RequestID,
			Error:     err.Error(),
		}
	}

	return &AnalysisResponse{
		Status:         "completed",
		RequestID:      req.RequestID,
		SpamLikelihood: &spamLikelihood,
	}
}

func (s *AIAnalysisService) processSentimentAnalysis(ctx context.Context, req *AnalysisRequest) *AnalysisResponse {
	// Create a focused sentiment analysis prompt
	prompt := fmt.Sprintf(`
Analyze the sentiment of this call transcription:

TRANSCRIPT: %s

Return ONLY a JSON object with:
{
  "sentiment": "positive|neutral|negative",
  "confidence": 0.0-1.0,
  "emotional_tone": "string describing emotional tone",
  "customer_satisfaction": "high|medium|low"
}
`, req.Transcription)

	// Use the AI service with a custom prompt for sentiment analysis
	analysis, err := s.aiService.AnalyzeCallContent(ctx, prompt, req.CallDetails)
	if err != nil {
		return &AnalysisResponse{
			Status:    "failed",
			RequestID: req.RequestID,
			Error:     err.Error(),
		}
	}

	return &AnalysisResponse{
		Status:       "completed",
		RequestID:    req.RequestID,
		CallAnalysis: analysis,
	}
}

func (s *AIAnalysisService) calculateOutputTokens(result *AnalysisResponse) int64 {
	if result.CallAnalysis != nil {
		data, _ := json.Marshal(result.CallAnalysis)
		return int64(len(data))
	}
	if result.SpamLikelihood != nil {
		return 10 // Estimate for spam likelihood response
	}
	return 0
}

func (s *AIAnalysisService) extractConfidenceScore(result *AnalysisResponse) float64 {
	if result.CallAnalysis != nil {
		return 0.85 // Default confidence for successful analysis
	}
	if result.SpamLikelihood != nil {
		return 0.90 // Spam detection typically has high confidence
	}
	return 0.0
}

func (s *AIAnalysisService) extractResultData(result *AnalysisResponse) interface{} {
	if result.CallAnalysis != nil {
		return result.CallAnalysis
	}
	if result.SpamLikelihood != nil {
		return map[string]interface{}{
			"spam_likelihood": *result.SpamLikelihood,
		}
	}
	return nil
}

func (s *AIAnalysisService) publishAnalysisCompletedEvent(ctx context.Context, req *AnalysisRequest, result *AnalysisResponse) error {
	topic := s.pubsubClient.Topic("analysis-completed")

	event := map[string]interface{}{
		"event_type":    "analysis.completed",
		"tenant_id":     req.TenantID,
		"call_id":       req.CallID,
		"request_id":    req.RequestID,
		"analysis_id":   result.AnalysisID,
		"analysis_type": req.AnalysisType,
		"result":        s.extractResultData(result),
		"timestamp":     time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	pubsubResult := topic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"event_type":    "analysis.completed",
			"tenant_id":     req.TenantID,
			"call_id":       req.CallID,
			"analysis_type": req.AnalysisType,
		},
	})

	_, err = pubsubResult.Get(ctx)
	return err
}

func (s *AIAnalysisService) startPubSubListener(ctx context.Context) {
	sub := s.pubsubClient.Subscription("ai-analysis-requests")

	log.Println("Starting Pub/Sub listener for AI analysis requests...")

	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var req AnalysisRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("Failed to unmarshal analysis request: %v", err)
			msg.Nack()
			return
		}

		// Process the analysis
		result, err := s.processAnalysis(ctx, &req)
		if err != nil {
			log.Printf("Failed to process analysis from Pub/Sub: %v", err)
			msg.Nack()
			return
		}

		log.Printf("Successfully processed analysis from Pub/Sub: %s", result.AnalysisID)
		msg.Ack()
	})

	if err != nil {
		log.Printf("Pub/Sub receive error: %v", err)
	}
}

// serializeProcessingData converts processing result to JSON string
func (s *AIAnalysisService) serializeProcessingData(result *AnalysisResponse, duration time.Duration) string {
	data := map[string]interface{}{
		"processing_time_ms": duration.Milliseconds(),
		"model_used":         s.config.VertexAIModel,
		"result":             result,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to serialize processing data: %v", err)
		return "{}"
	}

	return string(jsonData)
}