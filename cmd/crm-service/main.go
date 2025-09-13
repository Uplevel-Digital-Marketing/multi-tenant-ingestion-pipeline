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

	"github.com/home-renovators/ingestion-pipeline/internal/auth"
	"github.com/home-renovators/ingestion-pipeline/internal/spanner"
	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

type CRMService struct {
	config       *config.Config
	authService  *auth.AuthService
	spannerRepo  *spanner.Repository
	pubsubClient *pubsub.Client
	crmClients   map[string]CRMClient // provider -> client
}

// CRMClient interface for different CRM providers
type CRMClient interface {
	CreateLead(ctx context.Context, lead *LeadData, config *CRMConfig) (*CRMResponse, error)
	UpdateLead(ctx context.Context, leadID string, lead *LeadData, config *CRMConfig) (*CRMResponse, error)
	GetLead(ctx context.Context, leadID string, config *CRMConfig) (*LeadData, error)
	TestConnection(ctx context.Context, config *CRMConfig) error
}

type CRMConfig struct {
	Provider    string            `json:"provider"`
	APIKey      string            `json:"api_key"`
	APISecret   string            `json:"api_secret,omitempty"`
	BaseURL     string            `json:"base_url,omitempty"`
	FieldMapping map[string]string `json:"field_mapping"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

type LeadData struct {
	ID                  string                 `json:"id,omitempty"`
	TenantID           string                 `json:"tenant_id"`
	RequestID          string                 `json:"request_id"`
	CallID             string                 `json:"call_id,omitempty"`
	CustomerName       string                 `json:"customer_name"`
	CustomerPhone      string                 `json:"customer_phone"`
	CustomerEmail      string                 `json:"customer_email,omitempty"`
	CustomerAddress    string                 `json:"customer_address,omitempty"`
	CustomerCity       string                 `json:"customer_city,omitempty"`
	CustomerState      string                 `json:"customer_state,omitempty"`
	CustomerZip        string                 `json:"customer_zip,omitempty"`
	ProjectType        string                 `json:"project_type"`
	ProjectDescription string                 `json:"project_description,omitempty"`
	LeadScore          int                    `json:"lead_score"`
	LeadSource         string                 `json:"lead_source"`
	LeadStatus         string                 `json:"lead_status"`
	Sentiment          string                 `json:"sentiment,omitempty"`
	Urgency            string                 `json:"urgency,omitempty"`
	Timeline           string                 `json:"timeline,omitempty"`
	BudgetIndicator    string                 `json:"budget_indicator,omitempty"`
	Notes              string                 `json:"notes,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	CustomFields       map[string]interface{} `json:"custom_fields,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

type CRMResponse struct {
	Success     bool                   `json:"success"`
	LeadID      string                 `json:"lead_id,omitempty"`
	ExternalID  string                 `json:"external_id,omitempty"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type CRMIntegrationRequest struct {
	TenantID         string                 `json:"tenant_id"`
	RequestID        string                 `json:"request_id"`
	CallID           string                 `json:"call_id,omitempty"`
	LeadData         *LeadData              `json:"lead_data"`
	CRMProvider      string                 `json:"crm_provider"`
	Action           string                 `json:"action"` // create, update, get
	ExistingLeadID   string                 `json:"existing_lead_id,omitempty"`
	Priority         string                 `json:"priority,omitempty"`
}

type CRMIntegrationResponse struct {
	Status           string       `json:"status"`
	RequestID        string       `json:"request_id"`
	IntegrationID    string       `json:"integration_id,omitempty"`
	CRMResponse      *CRMResponse `json:"crm_response,omitempty"`
	ProcessingTimeMs int64        `json:"processing_time_ms"`
	Error            string       `json:"error,omitempty"`
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

		log.Println("Shutting down CRM service...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("CRM service starting on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func initializeServices(ctx context.Context, cfg *config.Config) (*CRMService, error) {
	// Initialize Spanner repository
	spannerRepo, err := spanner.NewRepository(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize spanner repository: %w", err)
	}

	// Initialize authentication service
	authService := auth.NewAuthService(cfg, spannerRepo)

	// Initialize Pub/Sub client
	pubsubClient, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Pub/Sub client: %w", err)
	}

	// Initialize CRM clients
	crmClients := map[string]CRMClient{
		"hubspot":    NewHubSpotClient(),
		"salesforce": NewSalesforceClient(),
		"pipedrive":  NewPipedriveClient(),
		"custom":     NewCustomCRMClient(),
	}

	return &CRMService{
		config:       cfg,
		authService:  authService,
		spannerRepo:  spannerRepo,
		pubsubClient: pubsubClient,
		crmClients:   crmClients,
	}, nil
}

func (s *CRMService) cleanup() {
	if s.spannerRepo != nil {
		s.spannerRepo.Close()
	}
	if s.pubsubClient != nil {
		s.pubsubClient.Close()
	}
}

func (s *CRMService) setupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", s.healthCheck)

	// API routes
	api := router.Group("/api/v1")
	{
		// CRM integration endpoints
		api.POST("/crm/create-lead", s.handleCreateLead)
		api.PUT("/crm/update-lead/:lead_id", s.handleUpdateLead)
		api.GET("/crm/get-lead/:lead_id", s.handleGetLead)
		api.POST("/crm/test-connection", s.handleTestConnection)
		api.POST("/crm/batch-process", s.handleBatchProcess)
		api.GET("/crm/status/:integration_id", s.handleGetIntegrationStatus)
	}
}

func (s *CRMService) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":           "healthy",
		"service":          "crm-service",
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"supported_crms":   []string{"hubspot", "salesforce", "pipedrive", "custom"},
	})
}

func (s *CRMService) handleCreateLead(c *gin.Context) {
	ctx := c.Request.Context()
	startTime := time.Now()

	var req CRMIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.Action = "create"

	// Process CRM integration
	result, err := s.processCRMIntegration(ctx, &req)
	if err != nil {
		log.Printf("Failed to process CRM integration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CRM integration failed"})
		return
	}

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, result)
}

func (s *CRMService) handleUpdateLead(c *gin.Context) {
	ctx := c.Request.Context()
	startTime := time.Now()
	leadID := c.Param("lead_id")

	var req CRMIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.Action = "update"
	req.ExistingLeadID = leadID

	// Process CRM integration
	result, err := s.processCRMIntegration(ctx, &req)
	if err != nil {
		log.Printf("Failed to update CRM lead: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CRM update failed"})
		return
	}

	result.ProcessingTimeMs = time.Since(startTime).Milliseconds()
	c.JSON(http.StatusOK, result)
}

func (s *CRMService) handleGetLead(c *gin.Context) {
	ctx := c.Request.Context()
	leadID := c.Param("lead_id")
	tenantID := c.Query("tenant_id")
	provider := c.Query("provider")

	if leadID == "" || tenantID == "" || provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
		return
	}

	// Get CRM configuration for tenant
	office, err := s.spannerRepo.GetOfficeByTenantID(ctx, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	crmConfig, err := s.parseCRMConfig(office.WorkflowConfig, provider)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CRM configuration"})
		return
	}

	// Get CRM client
	client, exists := s.crmClients[provider]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported CRM provider"})
		return
	}

	// Get lead from CRM
	lead, err := client.GetLead(ctx, leadID, crmConfig)
	if err != nil {
		log.Printf("Failed to get lead from CRM: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve lead"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"lead_id": leadID,
		"provider": provider,
		"lead_data": lead,
	})
}

func (s *CRMService) handleTestConnection(c *gin.Context) {
	ctx := c.Request.Context()

	var crmConfig CRMConfig
	if err := c.ShouldBindJSON(&crmConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get CRM client
	client, exists := s.crmClients[crmConfig.Provider]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported CRM provider"})
		return
	}

	// Test connection
	err := client.TestConnection(ctx, &crmConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection test successful",
	})
}

func (s *CRMService) handleBatchProcess(c *gin.Context) {
	ctx := c.Request.Context()

	var requests []CRMIntegrationRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(requests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No requests provided"})
		return
	}

	if len(requests) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many requests (max 50)"})
		return
	}

	// Process batch requests
	results := make([]CRMIntegrationResponse, len(requests))
	for i, req := range requests {
		result, err := s.processCRMIntegration(ctx, &req)
		if err != nil {
			result = &CRMIntegrationResponse{
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

func (s *CRMService) handleGetIntegrationStatus(c *gin.Context) {
	ctx := c.Request.Context()
	integrationID := c.Param("integration_id")
	tenantID := c.Query("tenant_id")

	if integrationID == "" || tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing integration_id or tenant_id"})
		return
	}

	// Get integration status from database
	integration, err := s.spannerRepo.GetCRMIntegration(ctx, tenantID, integrationID)
	if err != nil {
		log.Printf("Failed to get integration status: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"integration_id": integration.IntegrationID,
		"tenant_id":      integration.TenantID,
		"crm_type":       integration.CRMType,
		"config":         integration.Config,
		"status":         integration.Status,
		"created_at":     integration.CreatedAt,
		"updated_at":     integration.UpdatedAt,
	})
}

func (s *CRMService) processCRMIntegration(ctx context.Context, req *CRMIntegrationRequest) (*CRMIntegrationResponse, error) {
	log.Printf("Processing CRM integration for request %s, action %s, provider %s", req.RequestID, req.Action, req.CRMProvider)

	// Get CRM configuration for tenant
	office, err := s.spannerRepo.GetOfficeByTenantID(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant configuration: %w", err)
	}

	crmConfig, err := s.parseCRMConfig(office.WorkflowConfig, req.CRMProvider)
	if err != nil {
		return nil, fmt.Errorf("invalid CRM configuration: %w", err)
	}

	// Get CRM client
	client, exists := s.crmClients[req.CRMProvider]
	if !exists {
		return nil, fmt.Errorf("unsupported CRM provider: %s", req.CRMProvider)
	}

	// Serialize integration config
	configData := map[string]interface{}{
		"request_id":    req.RequestID,
		"call_id":       req.CallID,
		"provider":      req.CRMProvider,
		"lead_score":    req.LeadData.LeadScore,
		"lead_source":   req.LeadData.LeadSource,
		"lead_data":     req.LeadData,
	}
	configJSON, _ := json.Marshal(configData)

	// Create integration record
	integrationID := models.NewIntegrationID()
	integration := &models.CRMIntegration{
		IntegrationID: integrationID,
		TenantID:      req.TenantID,
		CRMType:       req.CRMProvider,
		Config:        string(configJSON),
		Status:        "processing",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := s.spannerRepo.CreateCRMIntegration(ctx, integration); err != nil {
		log.Printf("Failed to create CRM integration record: %v", err)
		// Continue processing even if record creation fails
	}

	// Process the CRM action
	var crmResponse *CRMResponse
	switch req.Action {
	case "create":
		crmResponse, err = client.CreateLead(ctx, req.LeadData, crmConfig)
	case "update":
		crmResponse, err = client.UpdateLead(ctx, req.ExistingLeadID, req.LeadData, crmConfig)
	default:
		err = fmt.Errorf("unsupported action: %s", req.Action)
	}

	// Update integration record with results
	if err != nil {
		integration.Status = "failed"
		// Update config with error information
		var configData map[string]interface{}
		json.Unmarshal([]byte(integration.Config), &configData)
		configData["error_message"] = err.Error()
		configJSON, _ := json.Marshal(configData)
		integration.Config = string(configJSON)
	} else {
		integration.Status = "completed"
		if crmResponse.ExternalID != "" {
			// Update config with external ID
			var configData map[string]interface{}
			json.Unmarshal([]byte(integration.Config), &configData)
			configData["external_id"] = crmResponse.ExternalID
			configJSON, _ := json.Marshal(configData)
			integration.Config = string(configJSON)
		}
	}
	integration.UpdatedAt = time.Now().UTC()

	if updateErr := s.spannerRepo.UpdateCRMIntegration(ctx, integration); updateErr != nil {
		log.Printf("Failed to update CRM integration record: %v", updateErr)
	}

	if err != nil {
		return &CRMIntegrationResponse{
			Status:        "failed",
			RequestID:     req.RequestID,
			IntegrationID: integrationID,
			Error:         err.Error(),
		}, nil
	}

	// Publish integration completed event
	if err := s.publishIntegrationCompletedEvent(ctx, req, crmResponse); err != nil {
		log.Printf("Failed to publish integration completed event: %v", err)
		// Continue processing even if event publishing fails
	}

	return &CRMIntegrationResponse{
		Status:        "completed",
		RequestID:     req.RequestID,
		IntegrationID: integrationID,
		CRMResponse:   crmResponse,
	}, nil
}

func (s *CRMService) parseCRMConfig(workflowConfigJSON, provider string) (*CRMConfig, error) {
	var workflowConfig models.WorkflowConfig
	if err := json.Unmarshal([]byte(workflowConfigJSON), &workflowConfig); err != nil {
		return nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}

	if !workflowConfig.CRMIntegration.Enabled {
		return nil, fmt.Errorf("CRM integration is disabled")
	}

	if workflowConfig.CRMIntegration.Provider != provider {
		return nil, fmt.Errorf("provider mismatch: expected %s, got %s", workflowConfig.CRMIntegration.Provider, provider)
	}

	return &CRMConfig{
		Provider:     workflowConfig.CRMIntegration.Provider,
		FieldMapping: workflowConfig.CRMIntegration.FieldMapping,
		// API keys would be loaded from secret manager
	}, nil
}

func (s *CRMService) publishIntegrationCompletedEvent(ctx context.Context, req *CRMIntegrationRequest, response *CRMResponse) error {
	topic := s.pubsubClient.Topic("crm-integration-completed")

	event := map[string]interface{}{
		"event_type":     "crm.integration.completed",
		"tenant_id":      req.TenantID,
		"request_id":     req.RequestID,
		"call_id":        req.CallID,
		"crm_provider":   req.CRMProvider,
		"action":         req.Action,
		"lead_id":        response.LeadID,
		"external_id":    response.ExternalID,
		"success":        response.Success,
		"timestamp":      time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	result := topic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"event_type":   "crm.integration.completed",
			"tenant_id":    req.TenantID,
			"crm_provider": req.CRMProvider,
		},
	})

	_, err = result.Get(ctx)
	return err
}

func (s *CRMService) startPubSubListener(ctx context.Context) {
	sub := s.pubsubClient.Subscription("crm-integration-requests")

	log.Println("Starting Pub/Sub listener for CRM integration requests...")

	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var req CRMIntegrationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("Failed to unmarshal CRM integration request: %v", err)
			msg.Nack()
			return
		}

		// Process the CRM integration
		result, err := s.processCRMIntegration(ctx, &req)
		if err != nil {
			log.Printf("Failed to process CRM integration from Pub/Sub: %v", err)
			msg.Nack()
			return
		}

		log.Printf("Successfully processed CRM integration from Pub/Sub: %s", result.IntegrationID)
		msg.Ack()
	})

	if err != nil {
		log.Printf("Pub/Sub receive error: %v", err)
	}
}

// Placeholder CRM client implementations (simplified for demonstration)
type HubSpotClient struct{}
type SalesforceClient struct{}
type PipedriveClient struct{}
type CustomCRMClient struct{}

func NewHubSpotClient() *HubSpotClient { return &HubSpotClient{} }
func NewSalesforceClient() *SalesforceClient { return &SalesforceClient{} }
func NewPipedriveClient() *PipedriveClient { return &PipedriveClient{} }
func NewCustomCRMClient() *CustomCRMClient { return &CustomCRMClient{} }

// Implementation stubs - would contain actual API calls
func (h *HubSpotClient) CreateLead(ctx context.Context, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	// TODO: Implement HubSpot API integration
	return &CRMResponse{Success: true, LeadID: "hubspot_" + lead.ID, ExternalID: "hs_12345"}, nil
}
func (h *HubSpotClient) UpdateLead(ctx context.Context, leadID string, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	return &CRMResponse{Success: true, LeadID: leadID, ExternalID: "hs_12345"}, nil
}
func (h *HubSpotClient) GetLead(ctx context.Context, leadID string, config *CRMConfig) (*LeadData, error) {
	return &LeadData{ID: leadID}, nil
}
func (h *HubSpotClient) TestConnection(ctx context.Context, config *CRMConfig) error {
	return nil
}

func (s *SalesforceClient) CreateLead(ctx context.Context, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	return &CRMResponse{Success: true, LeadID: "sf_" + lead.ID, ExternalID: "sf_12345"}, nil
}
func (s *SalesforceClient) UpdateLead(ctx context.Context, leadID string, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	return &CRMResponse{Success: true, LeadID: leadID, ExternalID: "sf_12345"}, nil
}
func (s *SalesforceClient) GetLead(ctx context.Context, leadID string, config *CRMConfig) (*LeadData, error) {
	return &LeadData{ID: leadID}, nil
}
func (s *SalesforceClient) TestConnection(ctx context.Context, config *CRMConfig) error {
	return nil
}

func (p *PipedriveClient) CreateLead(ctx context.Context, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	return &CRMResponse{Success: true, LeadID: "pd_" + lead.ID, ExternalID: "pd_12345"}, nil
}
func (p *PipedriveClient) UpdateLead(ctx context.Context, leadID string, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	return &CRMResponse{Success: true, LeadID: leadID, ExternalID: "pd_12345"}, nil
}
func (p *PipedriveClient) GetLead(ctx context.Context, leadID string, config *CRMConfig) (*LeadData, error) {
	return &LeadData{ID: leadID}, nil
}
func (p *PipedriveClient) TestConnection(ctx context.Context, config *CRMConfig) error {
	return nil
}

func (c *CustomCRMClient) CreateLead(ctx context.Context, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	return &CRMResponse{Success: true, LeadID: "custom_" + lead.ID, ExternalID: "custom_12345"}, nil
}
func (c *CustomCRMClient) UpdateLead(ctx context.Context, leadID string, lead *LeadData, config *CRMConfig) (*CRMResponse, error) {
	return &CRMResponse{Success: true, LeadID: leadID, ExternalID: "custom_12345"}, nil
}
func (c *CustomCRMClient) GetLead(ctx context.Context, leadID string, config *CRMConfig) (*LeadData, error) {
	return &LeadData{ID: leadID}, nil
}
func (c *CustomCRMClient) TestConnection(ctx context.Context, config *CRMConfig) error {
	return nil
}