package config

import (
	"context"
	"fmt"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// Config holds all application configuration
type Config struct {
	// Server Configuration
	Port        string `json:"port"`
	Environment string `json:"environment"`

	// Google Cloud Project Configuration
	ProjectID string `json:"project_id"`
	Location  string `json:"location"`

	// Cloud Spanner Configuration
	SpannerInstance string `json:"spanner_instance"`
	SpannerDatabase string `json:"spanner_database"`

	// Vertex AI Configuration
	VertexAIProject  string `json:"vertex_ai_project"`
	VertexAILocation string `json:"vertex_ai_location"`
	VertexAIModel    string `json:"vertex_ai_model"`

	// Speech-to-Text Configuration
	SpeechToTextProject  string `json:"speech_to_text_project"`
	SpeechToTextLocation string `json:"speech_to_text_location"`
	SpeechToTextModel    string `json:"speech_to_text_model"`
	SpeechLanguage       string `json:"speech_language"`
	EnableDiarization    bool   `json:"enable_diarization"`

	// Cloud Storage Configuration
	StorageProject   string `json:"storage_project"`
	AudioBucket      string `json:"audio_bucket"`
	StorageLocation  string `json:"storage_location"`
	RetentionDays    int    `json:"retention_days"`

	// Webhook Configuration
	CallRailWebhookSecret string `json:"callrail_webhook_secret"`

	// Cloud Run Configuration
	CloudRunProject string `json:"cloud_run_project"`
	CloudRunRegion  string `json:"cloud_run_region"`

	// Cloud Tasks Configuration
	CloudTasksProject  string `json:"cloud_tasks_project"`
	CloudTasksLocation string `json:"cloud_tasks_location"`
}

// DefaultConfig returns configuration with default values from environment variables
func DefaultConfig() *Config {
	return &Config{
		Port:        getEnvOrDefault("PORT", "8080"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),

		// GCP Project Configuration
		ProjectID: getEnvOrDefault("GOOGLE_CLOUD_PROJECT", "account-strategy-464106"),
		Location:  getEnvOrDefault("GOOGLE_CLOUD_LOCATION", "us-central1"),

		// Cloud Spanner Configuration
		SpannerInstance: getEnvOrDefault("SPANNER_INSTANCE", "upai-customers"),
		SpannerDatabase: getEnvOrDefault("SPANNER_DATABASE", "agent_platform"),

		// Vertex AI Configuration
		VertexAIProject:  getEnvOrDefault("VERTEX_AI_PROJECT", "account-strategy-464106"),
		VertexAILocation: getEnvOrDefault("VERTEX_AI_LOCATION", "us-central1"),
		VertexAIModel:    getEnvOrDefault("VERTEX_AI_MODEL", "gemini-2.5-flash"),

		// Speech-to-Text Configuration
		SpeechToTextProject:  getEnvOrDefault("SPEECH_TO_TEXT_PROJECT", "account-strategy-464106"),
		SpeechToTextLocation: getEnvOrDefault("SPEECH_TO_TEXT_LOCATION", "us-central1"),
		SpeechToTextModel:    getEnvOrDefault("SPEECH_TO_TEXT_MODEL", "chirp-3"),
		SpeechLanguage:       getEnvOrDefault("SPEECH_LANGUAGE", "en-US"),
		EnableDiarization:    getEnvOrDefault("ENABLE_DIARIZATION", "true") == "true",

		// Cloud Storage Configuration
		StorageProject:  getEnvOrDefault("STORAGE_PROJECT", "account-strategy-464106"),
		AudioBucket:     getEnvOrDefault("AUDIO_STORAGE_BUCKET", "tenant-audio-files"),
		StorageLocation: getEnvOrDefault("STORAGE_LOCATION", "us-central1"),
		RetentionDays:   getEnvIntOrDefault("RETENTION_DAYS", 2555),

		// Webhook Configuration
		CallRailWebhookSecret: getEnvOrDefault("CALLRAIL_WEBHOOK_SECRET_NAME", "callrail-webhook-secret"),

		// Cloud Run Configuration
		CloudRunProject: getEnvOrDefault("CLOUD_RUN_PROJECT", "account-strategy-464106"),
		CloudRunRegion:  getEnvOrDefault("CLOUD_RUN_REGION", "us-central1"),

		// Cloud Tasks Configuration
		CloudTasksProject:  getEnvOrDefault("CLOUD_TASKS_PROJECT", "account-strategy-464106"),
		CloudTasksLocation: getEnvOrDefault("CLOUD_TASKS_LOCATION", "us-central1"),
	}
}

// LoadSecrets loads sensitive configuration from Google Secret Manager
func (c *Config) LoadSecrets(ctx context.Context) error {
	if c.Environment == "development" {
		return nil // Skip secret loading in development
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client: %w", err)
	}
	defer client.Close()

	// Load CallRail webhook secret
	webhookSecret, err := accessSecret(ctx, client, c.ProjectID, c.CallRailWebhookSecret)
	if err != nil {
		return fmt.Errorf("failed to load webhook secret: %w", err)
	}
	c.CallRailWebhookSecret = webhookSecret

	return nil
}

// accessSecret retrieves a secret from Google Secret Manager
func accessSecret(ctx context.Context, client *secretmanager.Client, projectID, secretID string) (string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to access secret %s: %w", secretID, err)
	}

	return string(result.Payload.Data), nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault returns environment variable as int or default
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}