package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/home-renovators/ingestion-pipeline/internal/spanner"
	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

var (
	ErrInvalidSignature    = errors.New("invalid webhook signature")
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrInvalidTenantMapping = errors.New("invalid tenant mapping")
)

// AuthService handles authentication and authorization
type AuthService struct {
	config       *config.Config
	spannerRepo  *spanner.Repository
	webhookSecret string
}

// NewAuthService creates a new authentication service
func NewAuthService(cfg *config.Config, spannerRepo *spanner.Repository) *AuthService {
	return &AuthService{
		config:      cfg,
		spannerRepo: spannerRepo,
		webhookSecret: cfg.CallRailWebhookSecret,
	}
}

// VerifyCallRailWebhook verifies the HMAC signature of a CallRail webhook
func (a *AuthService) VerifyCallRailWebhook(payload []byte, signature string) error {
	if signature == "" {
		return ErrInvalidSignature
	}

	// Remove "sha256=" prefix if present
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signature = signature[7:]
	}

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(a.webhookSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant-time comparison
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return ErrInvalidSignature
	}

	return nil
}

// AuthenticateTenant validates tenant authentication using CallRail company mapping
func (a *AuthService) AuthenticateTenant(ctx context.Context, tenantID, callRailCompanyID string) (*models.Office, error) {
	if tenantID == "" || callRailCompanyID == "" {
		return nil, ErrInvalidTenantMapping
	}

	// Query office by CallRail company ID and tenant ID
	office, err := a.spannerRepo.GetOfficeByCallRailCompanyID(ctx, callRailCompanyID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get office: %w", err)
	}

	if office == nil {
		return nil, ErrTenantNotFound
	}

	// Verify office is active
	if office.Status != "active" {
		return nil, fmt.Errorf("office is not active: status=%s", office.Status)
	}

	return office, nil
}

// ValidateAPIAccess validates that a tenant has access to specific API operations
func (a *AuthService) ValidateAPIAccess(ctx context.Context, tenantID string, operation string) error {
	// For now, we'll implement basic validation
	// In the future, this could be extended with more granular permissions

	if tenantID == "" {
		return errors.New("tenant_id is required")
	}

	// Check if tenant exists and is active
	exists, err := a.spannerRepo.TenantExists(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to check tenant existence: %w", err)
	}

	if !exists {
		return ErrTenantNotFound
	}

	return nil
}

// GetTenantWorkflowConfig retrieves the workflow configuration for a tenant
func (a *AuthService) GetTenantWorkflowConfig(ctx context.Context, office *models.Office) (*models.WorkflowConfig, error) {
	if office.WorkflowConfig == "" {
		// Return default configuration
		return getDefaultWorkflowConfig(), nil
	}

	// Parse JSON workflow configuration
	config, err := parseWorkflowConfig(office.WorkflowConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow config: %w", err)
	}

	return config, nil
}

// getDefaultWorkflowConfig returns a default workflow configuration
func getDefaultWorkflowConfig() *models.WorkflowConfig {
	return &models.WorkflowConfig{
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
				ConfidenceThreshold: 75,
				MLModel:             "gemini-2.5-flash",
			},
		},
		ServiceArea: models.ServiceAreaConfig{
			Enabled:          true,
			ValidationMethod: "zip_code",
			AllowedAreas:     []string{},
			BufferMiles:      25,
		},
		CRMIntegration: models.CRMIntegrationConfig{
			Enabled:         true,
			Provider:        "hubspot",
			FieldMapping:    map[string]string{
				"name":       "firstname",
				"phone":      "phone",
				"lead_score": "hs_lead_score",
			},
			PushImmediately: true,
		},
		EmailNotifications: models.EmailNotificationsConfig{
			Enabled:    true,
			Recipients: []string{},
			Conditions: models.EmailConditionsConfig{
				MinLeadScore: 30,
			},
		},
	}
}

// parseWorkflowConfig parses JSON workflow configuration
func parseWorkflowConfig(jsonConfig string) (*models.WorkflowConfig, error) {
	// For now, return default config
	// TODO: Implement JSON parsing
	return getDefaultWorkflowConfig(), nil
}

// SetWebhookSecret sets the webhook secret (for testing)
func (a *AuthService) SetWebhookSecret(secret string) {
	a.webhookSecret = secret
}