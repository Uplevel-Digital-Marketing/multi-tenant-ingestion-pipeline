package spanner

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

// Repository handles Cloud Spanner database operations
type Repository struct {
	client   *spanner.Client
	database string
}

// NewRepository creates a new Spanner repository
func NewRepository(ctx context.Context, cfg *config.Config) (*Repository, error) {
	databasePath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.ProjectID, cfg.SpannerInstance, cfg.SpannerDatabase)

	client, err := spanner.NewClient(ctx, databasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create spanner client: %w", err)
	}

	return &Repository{
		client:   client,
		database: databasePath,
	}, nil
}

// Close closes the Spanner client
func (r *Repository) Close() {
	r.client.Close()
}

// GetOfficeByCallRailCompanyID retrieves an office by CallRail company ID and tenant ID
func (r *Repository) GetOfficeByCallRailCompanyID(ctx context.Context, callRailCompanyID, tenantID string) (*models.Office, error) {
	stmt := spanner.Statement{
		SQL: `SELECT tenant_id, office_id, callrail_company_id, callrail_api_key,
		             workflow_config, status, created_at, updated_at
		      FROM offices
		      WHERE callrail_company_id = @callrail_company_id
		        AND tenant_id = @tenant_id
		        AND status = 'active'`,
		Params: map[string]interface{}{
			"callrail_company_id": callRailCompanyID,
			"tenant_id":          tenantID,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // Office not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query office: %w", err)
	}

	var office models.Office
	err = row.Columns(
		&office.TenantID,
		&office.OfficeID,
		&office.CallRailCompanyID,
		&office.CallRailAPIKey,
		&office.WorkflowConfig,
		&office.Status,
		&office.CreatedAt,
		&office.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan office row: %w", err)
	}

	return &office, nil
}

// TenantExists checks if a tenant exists and is active
func (r *Repository) TenantExists(ctx context.Context, tenantID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM tenants WHERE tenant_id = @tenant_id AND status = 'active'`,
		Params: map[string]interface{}{
			"tenant_id": tenantID,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return false, fmt.Errorf("failed to query tenant: %w", err)
	}

	var count int64
	err = row.Columns(&count)
	if err != nil {
		return false, fmt.Errorf("failed to scan count: %w", err)
	}

	return count > 0, nil
}

// CreateRequest creates a new request record
func (r *Repository) CreateRequest(ctx context.Context, req *models.Request) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("requests",
			[]string{
				"request_id", "tenant_id", "source", "request_type", "status",
				"data", "ai_normalized", "ai_extracted", "call_id", "recording_url",
				"transcription_data", "ai_analysis", "lead_score", "communication_mode",
				"spam_likelihood", "created_at", "updated_at",
			},
			[]interface{}{
				req.RequestID,
				req.TenantID,
				req.Source,
				req.RequestType,
				req.Status,
				req.Data,
				req.AINormalized,
				req.AIExtracted,
				req.CallID,
				req.RecordingURL,
				req.TranscriptionData,
				req.AIAnalysis,
				req.LeadScore,
				req.CommunicationMode,
				req.SpamLikelihood,
				req.CreatedAt,
				req.UpdatedAt,
			},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	return nil
}

// CreateCallRecording creates a new call recording record
func (r *Repository) CreateCallRecording(ctx context.Context, recording *models.CallRecording) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("call_recordings",
			[]string{
				"recording_id", "tenant_id", "call_id", "storage_url",
				"transcription_status", "created_at",
			},
			[]interface{}{
				recording.RecordingID,
				recording.TenantID,
				recording.CallID,
				recording.StorageURL,
				recording.TranscriptionStatus,
				recording.CreatedAt,
			},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to create call recording: %w", err)
	}

	return nil
}

// CreateWebhookEvent creates a new webhook event record
func (r *Repository) CreateWebhookEvent(ctx context.Context, event *models.WebhookEvent) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("webhook_events",
			[]string{
				"event_id", "webhook_source", "call_id", "processing_status", "created_at",
			},
			[]interface{}{
				event.EventID,
				event.WebhookSource,
				event.CallID,
				event.ProcessingStatus,
				event.CreatedAt,
			},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to create webhook event: %w", err)
	}

	return nil
}

// UpdateWebhookEventStatus updates the processing status of a webhook event
func (r *Repository) UpdateWebhookEventStatus(ctx context.Context, eventID, status string) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("webhook_events",
			[]string{"event_id", "processing_status"},
			[]interface{}{eventID, status},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to update webhook event status: %w", err)
	}

	return nil
}

// UpdateCallRecordingStatus updates the transcription status of a call recording
func (r *Repository) UpdateCallRecordingStatus(ctx context.Context, recordingID, status string) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("call_recordings",
			[]string{"recording_id", "transcription_status"},
			[]interface{}{recordingID, status},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to update call recording status: %w", err)
	}

	return nil
}

// GetRequestByCallID retrieves a request by call ID
func (r *Repository) GetRequestByCallID(ctx context.Context, callID string) (*models.Request, error) {
	stmt := spanner.Statement{
		SQL: `SELECT request_id, tenant_id, source, request_type, status, data,
		             ai_normalized, ai_extracted, call_id, recording_url,
		             transcription_data, ai_analysis, lead_score, communication_mode,
		             spam_likelihood, created_at, updated_at
		      FROM requests
		      WHERE call_id = @call_id`,
		Params: map[string]interface{}{
			"call_id": callID,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // Request not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query request: %w", err)
	}

	var req models.Request
	err = row.Columns(
		&req.RequestID,
		&req.TenantID,
		&req.Source,
		&req.RequestType,
		&req.Status,
		&req.Data,
		&req.AINormalized,
		&req.AIExtracted,
		&req.CallID,
		&req.RecordingURL,
		&req.TranscriptionData,
		&req.AIAnalysis,
		&req.LeadScore,
		&req.CommunicationMode,
		&req.SpamLikelihood,
		&req.CreatedAt,
		&req.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan request row: %w", err)
	}

	return &req, nil
}

// GetRequestsByTenant retrieves requests for a specific tenant
func (r *Repository) GetRequestsByTenant(ctx context.Context, tenantID string, limit int, offset int) ([]*models.Request, error) {
	stmt := spanner.Statement{
		SQL: `SELECT request_id, tenant_id, source, request_type, status, data,
		             ai_normalized, ai_extracted, call_id, recording_url,
		             transcription_data, ai_analysis, lead_score, communication_mode,
		             spam_likelihood, created_at, updated_at
		      FROM requests
		      WHERE tenant_id = @tenant_id
		      ORDER BY created_at DESC
		      LIMIT @limit OFFSET @offset`,
		Params: map[string]interface{}{
			"tenant_id": tenantID,
			"limit":     limit,
			"offset":    offset,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var requests []*models.Request
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate requests: %w", err)
		}

		var req models.Request
		err = row.Columns(
			&req.RequestID,
			&req.TenantID,
			&req.Source,
			&req.RequestType,
			&req.Status,
			&req.Data,
			&req.AINormalized,
			&req.AIExtracted,
			&req.CallID,
			&req.RecordingURL,
			&req.TranscriptionData,
			&req.AIAnalysis,
			&req.LeadScore,
			&req.CommunicationMode,
			&req.SpamLikelihood,
			&req.CreatedAt,
			&req.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan request row: %w", err)
		}

		requests = append(requests, &req)
	}

	return requests, nil
}

// Helper function to marshal JSON data
func marshalJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetOfficeByTenantID retrieves an office by tenant ID
func (r *Repository) GetOfficeByTenantID(ctx context.Context, tenantID string) (*models.Office, error) {
	stmt := spanner.Statement{
		SQL: `SELECT tenant_id, office_id, callrail_company_id, callrail_api_key,
		             workflow_config, status, created_at, updated_at
		      FROM offices
		      WHERE tenant_id = @tenant_id
		        AND status = 'active'`,
		Params: map[string]interface{}{
			"tenant_id": tenantID,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("office not found for tenant %s", tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query office: %w", err)
	}

	var office models.Office
	err = row.Columns(
		&office.TenantID,
		&office.OfficeID,
		&office.CallRailCompanyID,
		&office.CallRailAPIKey,
		&office.WorkflowConfig,
		&office.Status,
		&office.CreatedAt,
		&office.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan office row: %w", err)
	}

	return &office, nil
}

// GetCallRecording retrieves a call recording by tenant and recording ID
func (r *Repository) GetCallRecording(ctx context.Context, tenantID, recordingID string) (*models.CallRecording, error) {
	stmt := spanner.Statement{
		SQL: `SELECT recording_id, tenant_id, call_id, storage_url,
		             transcription_status, created_at
		      FROM call_recordings
		      WHERE tenant_id = @tenant_id
		        AND recording_id = @recording_id`,
		Params: map[string]interface{}{
			"tenant_id":    tenantID,
			"recording_id": recordingID,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("call recording not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query call recording: %w", err)
	}

	var recording models.CallRecording
	err = row.Columns(
		&recording.RecordingID,
		&recording.TenantID,
		&recording.CallID,
		&recording.StorageURL,
		&recording.TranscriptionStatus,
		&recording.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan call recording row: %w", err)
	}

	return &recording, nil
}

// UpdateCallRecordingTranscription updates the transcription data of a call recording
func (r *Repository) UpdateCallRecordingTranscription(ctx context.Context, recordingID string, transcriptionData string) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("call_recordings",
			[]string{"recording_id", "transcription_data"},
			[]interface{}{recordingID, transcriptionData},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to update call recording transcription: %w", err)
	}

	return nil
}

// CreateAIProcessingLog creates a new AI processing log record
func (r *Repository) CreateAIProcessingLog(ctx context.Context, log *models.AIProcessingLog) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("ai_processing_logs",
			[]string{
				"log_id", "tenant_id", "request_id", "analysis_type",
				"status", "processing_data", "created_at", "updated_at",
			},
			[]interface{}{
				log.LogID,
				log.TenantID,
				log.RequestID,
				log.AnalysisType,
				log.Status,
				log.ProcessingData,
				log.CreatedAt,
				log.UpdatedAt,
			},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to create AI processing log: %w", err)
	}

	return nil
}

// GetAIProcessingLog retrieves an AI processing log
func (r *Repository) GetAIProcessingLog(ctx context.Context, tenantID, logID string) (*models.AIProcessingLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT log_id, tenant_id, request_id, analysis_type,
		             status, processing_data, created_at, updated_at
		      FROM ai_processing_logs
		      WHERE tenant_id = @tenant_id
		        AND log_id = @log_id`,
		Params: map[string]interface{}{
			"tenant_id": tenantID,
			"log_id":    logID,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("AI processing log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query AI processing log: %w", err)
	}

	var log models.AIProcessingLog
	err = row.Columns(
		&log.LogID,
		&log.TenantID,
		&log.RequestID,
		&log.AnalysisType,
		&log.Status,
		&log.ProcessingData,
		&log.CreatedAt,
		&log.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan AI processing log row: %w", err)
	}

	return &log, nil
}

// CreateCRMIntegration creates a new CRM integration record
func (r *Repository) CreateCRMIntegration(ctx context.Context, integration *models.CRMIntegration) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("crm_integrations",
			[]string{
				"integration_id", "tenant_id", "crm_type", "config",
				"status", "created_at", "updated_at",
			},
			[]interface{}{
				integration.IntegrationID,
				integration.TenantID,
				integration.CRMType,
				integration.Config,
				integration.Status,
				integration.CreatedAt,
				integration.UpdatedAt,
			},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to create CRM integration: %w", err)
	}

	return nil
}

// GetCRMIntegration retrieves a CRM integration by tenant and integration ID
func (r *Repository) GetCRMIntegration(ctx context.Context, tenantID, integrationID string) (*models.CRMIntegration, error) {
	stmt := spanner.Statement{
		SQL: `SELECT integration_id, tenant_id, crm_type, config,
		             status, created_at, updated_at
		      FROM crm_integrations
		      WHERE tenant_id = @tenant_id
		        AND integration_id = @integration_id`,
		Params: map[string]interface{}{
			"tenant_id":      tenantID,
			"integration_id": integrationID,
		},
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("CRM integration not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query CRM integration: %w", err)
	}

	var integration models.CRMIntegration
	err = row.Columns(
		&integration.IntegrationID,
		&integration.TenantID,
		&integration.CRMType,
		&integration.Config,
		&integration.Status,
		&integration.CreatedAt,
		&integration.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan CRM integration row: %w", err)
	}

	return &integration, nil
}

// UpdateCRMIntegration updates a CRM integration
func (r *Repository) UpdateCRMIntegration(ctx context.Context, integration *models.CRMIntegration) error {
	_, err := r.client.Apply(ctx, []*spanner.Mutation{
		spanner.Update("crm_integrations",
			[]string{
				"integration_id", "tenant_id", "crm_type", "config",
				"status", "updated_at",
			},
			[]interface{}{
				integration.IntegrationID,
				integration.TenantID,
				integration.CRMType,
				integration.Config,
				integration.Status,
				integration.UpdatedAt,
			},
		),
	})

	if err != nil {
		return fmt.Errorf("failed to update CRM integration: %w", err)
	}

	return nil
}

// Helper function to unmarshal JSON data
func unmarshalJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}