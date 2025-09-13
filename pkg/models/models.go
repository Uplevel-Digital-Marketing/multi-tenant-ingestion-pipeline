package models

import (
	"time"

	"github.com/google/uuid"
)

// CallRailWebhook represents the incoming webhook payload from CallRail
type CallRailWebhook struct {
	CallID            string    `json:"call_id"`
	AccountID         string    `json:"account_id"`
	CompanyID         string    `json:"company_id"`
	CallerID          string    `json:"caller_id"`
	CalledNumber      string    `json:"called_number"`
	Duration          string    `json:"duration"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Direction         string    `json:"direction"`
	RecordingURL      string    `json:"recording_url"`
	Answered          bool      `json:"answered"`
	FirstCall         bool      `json:"first_call"`
	Value             string    `json:"value"`
	GoodCall          *bool     `json:"good_call"`
	Tags              []string  `json:"tags"`
	Note              string    `json:"note"`
	BusinessPhoneNumber string  `json:"business_phone_number"`
	CustomerName      string    `json:"customer_name"`
	CustomerPhoneNumber string `json:"customer_phone_number"`
	CustomerCity      string    `json:"customer_city"`
	CustomerState     string    `json:"customer_state"`
	CustomerCountry   string    `json:"customer_country"`
	LeadStatus        string    `json:"lead_status"`

	// Custom fields for our application
	TenantID           string `json:"tenant_id"`
	CallRailCompanyID  string `json:"callrail_company_id"`
}

// CallDetails represents detailed call information from CallRail API
type CallDetails struct {
	ID                    string    `json:"id"`
	Answered              bool      `json:"answered"`
	BusinessPhoneNumber   string    `json:"business_phone_number"`
	CallerID              string    `json:"caller_id"`
	CompanyID             string    `json:"company_id"`
	CreatedAt             time.Time `json:"created_at"`
	CustomerCity          string    `json:"customer_city"`
	CustomerCountry       string    `json:"customer_country"`
	CustomerName          string    `json:"customer_name"`
	CustomerPhoneNumber   string    `json:"customer_phone_number"`
	CustomerState         string    `json:"customer_state"`
	Direction             string    `json:"direction"`
	Duration              int       `json:"duration"`
	FirstCall             bool      `json:"first_call"`
	FormattedBusinessPhoneNumber string `json:"formatted_business_phone_number"`
	FormattedCustomerLocation    string `json:"formatted_customer_location"`
	FormattedCustomerPhoneNumber string `json:"formatted_customer_phone_number"`
	FormattedDuration            string `json:"formatted_duration"`
	GoodCall              *bool     `json:"good_call"`
	LeadStatus            string    `json:"lead_status"`
	Note                  string    `json:"note"`
	Source                string    `json:"source"`
	StartTime             time.Time `json:"start_time"`
	Tags                  []string  `json:"tags"`
	TrackingPhoneNumber   string    `json:"tracking_phone_number"`
	Value                 string    `json:"value"`
	Recording             string    `json:"recording"`
}

// RecordingDetails represents call recording information
type RecordingDetails struct {
	CallID       string    `json:"call_id"`
	RecordingURL string    `json:"recording_url"`
	Duration     int       `json:"duration"`
	FileSize     int64     `json:"file_size"`
	Format       string    `json:"format"`
	CreatedAt    time.Time `json:"created_at"`
}

// TranscriptionResult represents the result of speech-to-text processing
type TranscriptionResult struct {
	Transcript          string                    `json:"transcript"`
	Confidence          float32                   `json:"confidence"`
	SpeakerDiarization  []SpeakerSegment         `json:"speaker_diarization"`
	WordDetails         []WordDetail             `json:"word_details"`
	Duration            float64                  `json:"duration"`
	SpeakerCount        int                      `json:"speaker_count"`
}

// SpeakerSegment represents a segment of speech from one speaker
type SpeakerSegment struct {
	Speaker   int     `json:"speaker"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Text      string  `json:"text"`
}

// WordDetail represents detailed information about a transcribed word
type WordDetail struct {
	Word       string  `json:"word"`
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
	Confidence float32 `json:"confidence"`
}

// CallAnalysis represents AI-powered analysis of the call content
type CallAnalysis struct {
	Intent              string   `json:"intent"`
	ProjectType         string   `json:"project_type"`
	Timeline            string   `json:"timeline"`
	BudgetIndicator     string   `json:"budget_indicator"`
	Sentiment           string   `json:"sentiment"`
	LeadScore           int      `json:"lead_score"`
	Urgency             string   `json:"urgency"`
	AppointmentRequested bool    `json:"appointment_requested"`
	FollowUpRequired    bool     `json:"follow_up_required"`
	KeyDetails          []string `json:"key_details"`
}

// EnhancedPayload represents the final structured data for workflow processing
type EnhancedPayload struct {
	RequestID         string                 `json:"request_id"`
	TenantID          string                 `json:"tenant_id"`
	Source            string                 `json:"source"`
	RequestType       string                 `json:"request_type"`
	CommunicationMode string                 `json:"communication_mode"`
	CreatedAt         time.Time              `json:"created_at"`

	OriginalWebhook   CallRailWebhook        `json:"original_webhook"`
	CallDetails       CallDetails            `json:"call_details"`
	AudioProcessing   AudioProcessingData    `json:"audio_processing"`
	AIAnalysis        CallAnalysis           `json:"ai_analysis"`
	SpamLikelihood    float64                `json:"spam_likelihood"`
	ProcessingMetadata ProcessingMetadata    `json:"processing_metadata"`
}

// AudioProcessingData represents processed audio information
type AudioProcessingData struct {
	RecordingURL    string              `json:"recording_url"`
	Transcription   string              `json:"transcription"`
	Confidence      float32             `json:"confidence"`
	Duration        float64             `json:"duration"`
	SpeakerCount    int                 `json:"speaker_count"`
	Transcription_Details TranscriptionResult `json:"transcription_details"`
}

// ProcessingMetadata tracks processing information
type ProcessingMetadata struct {
	ProcessedAt       time.Time `json:"processed_at"`
	ProcessingTimeMs  int64     `json:"processing_time_ms"`
	GeminiModel       string    `json:"gemini_model"`
	SpeechModel       string    `json:"speech_model"`
}

// Office represents a tenant office configuration
type Office struct {
	TenantID           string `json:"tenant_id" spanner:"tenant_id"`
	OfficeID           string `json:"office_id" spanner:"office_id"`
	CallRailCompanyID  string `json:"callrail_company_id" spanner:"callrail_company_id"`
	CallRailAPIKey     string `json:"callrail_api_key" spanner:"callrail_api_key"`
	WorkflowConfig     string `json:"workflow_config" spanner:"workflow_config"` // JSON string
	Status             string `json:"status" spanner:"status"`
	CreatedAt          time.Time `json:"created_at" spanner:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" spanner:"updated_at"`
}

// WorkflowConfig represents the tenant's workflow configuration
type WorkflowConfig struct {
	CommunicationDetection CommunicationDetectionConfig `json:"communication_detection"`
	Validation            ValidationConfig             `json:"validation"`
	ServiceArea           ServiceAreaConfig            `json:"service_area"`
	CRMIntegration        CRMIntegrationConfig         `json:"crm_integration"`
	EmailNotifications    EmailNotificationsConfig     `json:"email_notifications"`
}

// CommunicationDetectionConfig configures how communications are processed
type CommunicationDetectionConfig struct {
	Enabled         bool               `json:"enabled"`
	PhoneProcessing PhoneProcessingConfig `json:"phone_processing"`
}

// PhoneProcessingConfig configures phone call processing
type PhoneProcessingConfig struct {
	TranscribeAudio     bool `json:"transcribe_audio"`
	ExtractDetails      bool `json:"extract_details"`
	SentimentAnalysis   bool `json:"sentiment_analysis"`
	SpeakerDiarization  bool `json:"speaker_diarization"`
}

// ValidationConfig configures request validation
type ValidationConfig struct {
	SpamDetection SpamDetectionConfig `json:"spam_detection"`
}

// SpamDetectionConfig configures spam detection
type SpamDetectionConfig struct {
	Enabled             bool   `json:"enabled"`
	ConfidenceThreshold int    `json:"confidence_threshold"`
	MLModel             string `json:"ml_model"`
}

// ServiceAreaConfig configures service area validation
type ServiceAreaConfig struct {
	Enabled          bool     `json:"enabled"`
	ValidationMethod string   `json:"validation_method"`
	AllowedAreas     []string `json:"allowed_areas"`
	BufferMiles      int      `json:"buffer_miles"`
}

// CRMIntegrationConfig configures CRM integration
type CRMIntegrationConfig struct {
	Enabled         bool              `json:"enabled"`
	Provider        string            `json:"provider"`
	FieldMapping    map[string]string `json:"field_mapping"`
	PushImmediately bool              `json:"push_immediately"`
}

// EmailNotificationsConfig configures email notifications
type EmailNotificationsConfig struct {
	Enabled    bool                  `json:"enabled"`
	Recipients []string              `json:"recipients"`
	Conditions EmailConditionsConfig `json:"conditions"`
}

// EmailConditionsConfig defines conditions for sending email notifications
type EmailConditionsConfig struct {
	MinLeadScore int `json:"min_lead_score"`
}

// Request represents a stored request in the database
type Request struct {
	RequestID          string    `json:"request_id" spanner:"request_id"`
	TenantID           string    `json:"tenant_id" spanner:"tenant_id"`
	Source             string    `json:"source" spanner:"source"`
	RequestType        string    `json:"request_type" spanner:"request_type"`
	Status             string    `json:"status" spanner:"status"`
	Data               string    `json:"data" spanner:"data"` // JSON string
	AINormalized       string    `json:"ai_normalized" spanner:"ai_normalized"` // JSON string
	AIExtracted        string    `json:"ai_extracted" spanner:"ai_extracted"` // JSON string
	CallID             *string   `json:"call_id" spanner:"call_id"`
	RecordingURL       *string   `json:"recording_url" spanner:"recording_url"`
	TranscriptionData  *string   `json:"transcription_data" spanner:"transcription_data"` // JSON string
	AIAnalysis         *string   `json:"ai_analysis" spanner:"ai_analysis"` // JSON string
	LeadScore          *int      `json:"lead_score" spanner:"lead_score"`
	CommunicationMode  string    `json:"communication_mode" spanner:"communication_mode"`
	SpamLikelihood     *float64  `json:"spam_likelihood" spanner:"spam_likelihood"`
	CreatedAt          time.Time `json:"created_at" spanner:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" spanner:"updated_at"`
}

// CallRecording represents a call recording record
type CallRecording struct {
	RecordingID         string    `json:"recording_id" spanner:"recording_id"`
	TenantID            string    `json:"tenant_id" spanner:"tenant_id"`
	CallID              string    `json:"call_id" spanner:"call_id"`
	StorageURL          string    `json:"storage_url" spanner:"storage_url"`
	TranscriptionStatus string    `json:"transcription_status" spanner:"transcription_status"`
	CreatedAt           time.Time `json:"created_at" spanner:"created_at"`
}

// WebhookEvent represents a webhook event record
type WebhookEvent struct {
	EventID          string    `json:"event_id" spanner:"event_id"`
	WebhookSource    string    `json:"webhook_source" spanner:"webhook_source"`
	CallID           *string   `json:"call_id" spanner:"call_id"`
	ProcessingStatus string    `json:"processing_status" spanner:"processing_status"`
	CreatedAt        time.Time `json:"created_at" spanner:"created_at"`
}

// NewRequestID generates a new request ID
func NewRequestID() string {
	return "req_" + uuid.New().String()
}

// NewRecordingID generates a new recording ID
func NewRecordingID() string {
	return "rec_" + uuid.New().String()
}

// NewEventID generates a new event ID
func NewEventID() string {
	return "evt_" + uuid.New().String()
}

// NewProcessingID generates a new processing ID
func NewProcessingID() string {
	return "proc_" + uuid.New().String()
}

// NewIntegrationID generates a new integration ID
func NewIntegrationID() string {
	return "integ_" + uuid.New().String()
}

// AIProcessingLog represents AI processing operations log
type AIProcessingLog struct {
	LogID           string    `json:"log_id" spanner:"log_id"`
	TenantID        string    `json:"tenant_id" spanner:"tenant_id"`
	RequestID       string    `json:"request_id" spanner:"request_id"`
	AnalysisType    string    `json:"analysis_type" spanner:"analysis_type"`
	Status          string    `json:"status" spanner:"status"`
	ProcessingData  string    `json:"processing_data" spanner:"processing_data"` // JSON string
	CreatedAt       time.Time `json:"created_at" spanner:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" spanner:"updated_at"`
}

// CRMIntegration represents a CRM integration record
type CRMIntegration struct {
	IntegrationID string    `json:"integration_id" spanner:"integration_id"`
	TenantID      string    `json:"tenant_id" spanner:"tenant_id"`
	CRMType       string    `json:"crm_type" spanner:"crm_type"`
	Config        string    `json:"config" spanner:"config"` // JSON string
	Status        string    `json:"status" spanner:"status"`
	CreatedAt     time.Time `json:"created_at" spanner:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" spanner:"updated_at"`
}