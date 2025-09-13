package database

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

// SpannerClient provides enhanced database operations for Cloud Spanner
type SpannerClient struct {
	client   *spanner.Client
	database string
}

// Config contains Spanner configuration
type Config struct {
	ProjectID  string `json:"project_id"`
	Instance   string `json:"instance"`
	Database   string `json:"database"`
	MaxSessions int   `json:"max_sessions,omitempty"`
	MinSessions int   `json:"min_sessions,omitempty"`
}

// NewSpannerClient creates a new enhanced Spanner client
func NewSpannerClient(ctx context.Context, config *Config) (*SpannerClient, error) {
	clientConfig := spanner.ClientConfig{}

	if config.MaxSessions > 0 {
		clientConfig.SessionPoolConfig.MaxOpened = uint64(config.MaxSessions)
	}
	if config.MinSessions > 0 {
		clientConfig.SessionPoolConfig.MinOpened = uint64(config.MinSessions)
	}

	database := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		config.ProjectID, config.Instance, config.Database)

	client, err := spanner.NewClientWithConfig(ctx, database, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Spanner client: %w", err)
	}

	return &SpannerClient{
		client:   client,
		database: database,
	}, nil
}

// Close closes the Spanner client
func (sc *SpannerClient) Close() {
	sc.client.Close()
}

// Transaction represents a database transaction
type Transaction struct {
	txn *spanner.ReadWriteTransaction
}

// RunTransaction runs a function in a database transaction
func (sc *SpannerClient) RunTransaction(ctx context.Context, fn func(*Transaction) error) error {
	_, err := sc.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		t := &Transaction{txn: txn}
		return fn(t)
	})
	return err
}

// QueryOptions contains options for queries
type QueryOptions struct {
	Limit  int                    `json:"limit,omitempty"`
	Offset int                    `json:"offset,omitempty"`
	OrderBy string               `json:"order_by,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// ===== REQUEST OPERATIONS =====

// CreateRequest creates a new request record
func (sc *SpannerClient) CreateRequest(ctx context.Context, request *models.Request) error {
	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdate("requests", []string{
			"request_id", "tenant_id", "source", "request_type", "status",
			"data", "ai_normalized", "ai_extracted", "call_id", "recording_url",
			"transcription_data", "ai_analysis", "lead_score", "communication_mode",
			"spam_likelihood", "created_at", "updated_at",
		}, []interface{}{
			request.RequestID, request.TenantID, request.Source, request.RequestType, request.Status,
			request.Data, request.AINormalized, request.AIExtracted, request.CallID, request.RecordingURL,
			request.TranscriptionData, request.AIAnalysis, request.LeadScore, request.CommunicationMode,
			request.SpamLikelihood, request.CreatedAt, request.UpdatedAt,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// GetRequest retrieves a request by ID
func (sc *SpannerClient) GetRequest(ctx context.Context, tenantID, requestID string) (*models.Request, error) {
	stmt := spanner.Statement{
		SQL: `SELECT request_id, tenant_id, source, request_type, status, data, ai_normalized,
		      ai_extracted, call_id, recording_url, transcription_data, ai_analysis,
		      lead_score, communication_mode, spam_likelihood, created_at, updated_at
		      FROM requests WHERE tenant_id = @tenant_id AND request_id = @request_id`,
		Params: map[string]interface{}{
			"tenant_id":  tenantID,
			"request_id": requestID,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("request not found")
	}
	if err != nil {
		return nil, err
	}

	request := &models.Request{}
	err = row.Columns(
		&request.RequestID, &request.TenantID, &request.Source, &request.RequestType,
		&request.Status, &request.Data, &request.AINormalized, &request.AIExtracted,
		&request.CallID, &request.RecordingURL, &request.TranscriptionData,
		&request.AIAnalysis, &request.LeadScore, &request.CommunicationMode,
		&request.SpamLikelihood, &request.CreatedAt, &request.UpdatedAt,
	)
	return request, err
}

// ListRequests lists requests for a tenant with filtering
func (sc *SpannerClient) ListRequests(ctx context.Context, tenantID string, options *QueryOptions) ([]*models.Request, error) {
	var requests []*models.Request

	sql := `SELECT request_id, tenant_id, source, request_type, status, data, ai_normalized,
	        ai_extracted, call_id, recording_url, transcription_data, ai_analysis,
	        lead_score, communication_mode, spam_likelihood, created_at, updated_at
	        FROM requests WHERE tenant_id = @tenant_id`

	params := map[string]interface{}{
		"tenant_id": tenantID,
	}

	if options != nil {
		if options.OrderBy != "" {
			sql += " ORDER BY " + options.OrderBy
		} else {
			sql += " ORDER BY created_at DESC"
		}

		if options.Limit > 0 {
			sql += " LIMIT @limit"
			params["limit"] = options.Limit
		}

		if options.Offset > 0 {
			sql += " OFFSET @offset"
			params["offset"] = options.Offset
		}

		// Add custom parameters
		for k, v := range options.Params {
			params[k] = v
		}
	}

	stmt := spanner.Statement{SQL: sql, Params: params}
	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		request := &models.Request{}
		err = row.Columns(
			&request.RequestID, &request.TenantID, &request.Source, &request.RequestType,
			&request.Status, &request.Data, &request.AINormalized, &request.AIExtracted,
			&request.CallID, &request.RecordingURL, &request.TranscriptionData,
			&request.AIAnalysis, &request.LeadScore, &request.CommunicationMode,
			&request.SpamLikelihood, &request.CreatedAt, &request.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		requests = append(requests, request)
	}

	return requests, nil
}

// ===== CALL RECORDING OPERATIONS =====

// CreateCallRecording creates a new call recording record
func (sc *SpannerClient) CreateCallRecording(ctx context.Context, recording *models.CallRecording) error {
	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdate("call_recordings", []string{
			"recording_id", "tenant_id", "call_id", "storage_url",
			"transcription_status", "created_at",
		}, []interface{}{
			recording.RecordingID, recording.TenantID, recording.CallID,
			recording.StorageURL, recording.TranscriptionStatus, recording.CreatedAt,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// GetCallRecording retrieves a call recording by ID
func (sc *SpannerClient) GetCallRecording(ctx context.Context, tenantID, recordingID string) (*models.CallRecording, error) {
	stmt := spanner.Statement{
		SQL: `SELECT recording_id, tenant_id, call_id, storage_url, transcription_status, created_at
		      FROM call_recordings WHERE tenant_id = @tenant_id AND recording_id = @recording_id`,
		Params: map[string]interface{}{
			"tenant_id":    tenantID,
			"recording_id": recordingID,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("recording not found")
	}
	if err != nil {
		return nil, err
	}

	recording := &models.CallRecording{}
	err = row.Columns(
		&recording.RecordingID, &recording.TenantID, &recording.CallID,
		&recording.StorageURL, &recording.TranscriptionStatus, &recording.CreatedAt,
	)
	return recording, err
}

// UpdateCallRecordingStatus updates the transcription status of a call recording
func (sc *SpannerClient) UpdateCallRecordingStatus(ctx context.Context, tenantID, recordingID, status string) error {
	mutations := []*spanner.Mutation{
		spanner.Update("call_recordings", []string{
			"tenant_id", "recording_id", "transcription_status",
		}, []interface{}{
			tenantID, recordingID, status,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// UpdateCallRecordingTranscription updates the transcription completion time
func (sc *SpannerClient) UpdateCallRecordingTranscription(ctx context.Context, tenantID, recordingID string, completedAt time.Time) error {
	mutations := []*spanner.Mutation{
		spanner.Update("call_recordings", []string{
			"tenant_id", "recording_id", "transcription_completed_at",
		}, []interface{}{
			tenantID, recordingID, completedAt,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// ===== WEBHOOK EVENT OPERATIONS =====

// CreateWebhookEvent creates a new webhook event record
func (sc *SpannerClient) CreateWebhookEvent(ctx context.Context, event *models.WebhookEvent) error {
	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdate("webhook_events", []string{
			"event_id", "webhook_source", "call_id", "processing_status", "created_at",
		}, []interface{}{
			event.EventID, event.WebhookSource, event.CallID,
			event.ProcessingStatus, event.CreatedAt,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// UpdateWebhookEventStatus updates the processing status of a webhook event
func (sc *SpannerClient) UpdateWebhookEventStatus(ctx context.Context, eventID, status string) error {
	mutations := []*spanner.Mutation{
		spanner.Update("webhook_events", []string{
			"event_id", "processing_status",
		}, []interface{}{
			eventID, status,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// ===== AI PROCESSING LOG OPERATIONS =====

// CreateAIProcessingLog creates a new AI processing log entry
func (sc *SpannerClient) CreateAIProcessingLog(ctx context.Context, log *models.AIProcessingLog) error {
	// Convert ResultData to JSON string for storage
	resultDataJSON := ""
	if log.ResultData != nil {
		if data, err := spanner.NullJSON{Valid: true, Value: log.ResultData}.MarshalJSON(); err == nil {
			resultDataJSON = string(data)
		}
	}

	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdate("ai_processing_log", []string{
			"processing_id", "tenant_id", "request_id", "processing_type", "model_used",
			"input_tokens", "output_tokens", "processing_time_ms", "confidence_score",
			"result_data", "cost_estimate_usd", "created_at",
		}, []interface{}{
			log.ProcessingID, log.TenantID, log.RequestID, log.ProcessingType, log.ModelUsed,
			log.InputTokens, log.OutputTokens, log.ProcessingTimeMs, log.ConfidenceScore,
			resultDataJSON, log.CostEstimateUSD, log.CreatedAt,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// GetAIProcessingLog retrieves an AI processing log entry
func (sc *SpannerClient) GetAIProcessingLog(ctx context.Context, tenantID, processingID string) (*models.AIProcessingLog, error) {
	stmt := spanner.Statement{
		SQL: `SELECT processing_id, tenant_id, request_id, processing_type, model_used,
		      input_tokens, output_tokens, processing_time_ms, confidence_score,
		      result_data, cost_estimate_usd, created_at
		      FROM ai_processing_log WHERE tenant_id = @tenant_id AND processing_id = @processing_id`,
		Params: map[string]interface{}{
			"tenant_id":     tenantID,
			"processing_id": processingID,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("processing log not found")
	}
	if err != nil {
		return nil, err
	}

	log := &models.AIProcessingLog{}
	var resultDataStr spanner.NullString
	err = row.Columns(
		&log.ProcessingID, &log.TenantID, &log.RequestID, &log.ProcessingType, &log.ModelUsed,
		&log.InputTokens, &log.OutputTokens, &log.ProcessingTimeMs, &log.ConfidenceScore,
		&resultDataStr, &log.CostEstimateUSD, &log.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse result data if available
	if resultDataStr.Valid && resultDataStr.StringVal != "" {
		var resultData interface{}
		nullJSON := spanner.NullJSON{}
		if err := nullJSON.UnmarshalJSON([]byte(resultDataStr.StringVal)); err == nil {
			resultData = nullJSON.Value
		}
		log.ResultData = resultData
	}

	return log, nil
}

// ===== CRM INTEGRATION OPERATIONS =====

// CreateCRMIntegration creates a new CRM integration record
func (sc *SpannerClient) CreateCRMIntegration(ctx context.Context, integration *models.CRMIntegration) error {
	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdate("crm_integrations", []string{
			"integration_id", "tenant_id", "request_id", "call_id", "provider",
			"external_id", "lead_score", "source_communication_mode", "status",
			"error_message", "created_at", "updated_at",
		}, []interface{}{
			integration.IntegrationID, integration.TenantID, integration.RequestID,
			integration.CallID, integration.Provider, integration.ExternalID,
			integration.LeadScore, integration.SourceCommunicationMode, integration.Status,
			integration.ErrorMessage, integration.CreatedAt, integration.UpdatedAt,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// UpdateCRMIntegration updates a CRM integration record
func (sc *SpannerClient) UpdateCRMIntegration(ctx context.Context, integration *models.CRMIntegration) error {
	mutations := []*spanner.Mutation{
		spanner.Update("crm_integrations", []string{
			"tenant_id", "integration_id", "external_id", "status", "error_message", "updated_at",
		}, []interface{}{
			integration.TenantID, integration.IntegrationID, integration.ExternalID,
			integration.Status, integration.ErrorMessage, integration.UpdatedAt,
		}),
	}

	_, err := sc.client.Apply(ctx, mutations)
	return err
}

// GetCRMIntegration retrieves a CRM integration record
func (sc *SpannerClient) GetCRMIntegration(ctx context.Context, tenantID, integrationID string) (*models.CRMIntegration, error) {
	stmt := spanner.Statement{
		SQL: `SELECT integration_id, tenant_id, request_id, call_id, provider,
		      external_id, lead_score, source_communication_mode, status,
		      error_message, created_at, updated_at
		      FROM crm_integrations WHERE tenant_id = @tenant_id AND integration_id = @integration_id`,
		Params: map[string]interface{}{
			"tenant_id":      tenantID,
			"integration_id": integrationID,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("CRM integration not found")
	}
	if err != nil {
		return nil, err
	}

	integration := &models.CRMIntegration{}
	err = row.Columns(
		&integration.IntegrationID, &integration.TenantID, &integration.RequestID,
		&integration.CallID, &integration.Provider, &integration.ExternalID,
		&integration.LeadScore, &integration.SourceCommunicationMode, &integration.Status,
		&integration.ErrorMessage, &integration.CreatedAt, &integration.UpdatedAt,
	)
	return integration, err
}

// ===== OFFICE OPERATIONS =====

// GetOfficeByTenantID retrieves office configuration by tenant ID
func (sc *SpannerClient) GetOfficeByTenantID(ctx context.Context, tenantID string) (*models.Office, error) {
	stmt := spanner.Statement{
		SQL: `SELECT tenant_id, office_id, callrail_company_id, callrail_api_key,
		      workflow_config, status, created_at, updated_at
		      FROM offices WHERE tenant_id = @tenant_id AND status = 'active' LIMIT 1`,
		Params: map[string]interface{}{
			"tenant_id": tenantID,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("office not found for tenant")
	}
	if err != nil {
		return nil, err
	}

	office := &models.Office{}
	err = row.Columns(
		&office.TenantID, &office.OfficeID, &office.CallRailCompanyID,
		&office.CallRailAPIKey, &office.WorkflowConfig, &office.Status,
		&office.CreatedAt, &office.UpdatedAt,
	)
	return office, err
}

// GetOfficeByCallRailCompanyID retrieves office by CallRail company ID
func (sc *SpannerClient) GetOfficeByCallRailCompanyID(ctx context.Context, callrailCompanyID, tenantID string) (*models.Office, error) {
	stmt := spanner.Statement{
		SQL: `SELECT tenant_id, office_id, callrail_company_id, callrail_api_key,
		      workflow_config, status, created_at, updated_at
		      FROM offices WHERE callrail_company_id = @callrail_company_id
		      AND tenant_id = @tenant_id AND status = 'active' LIMIT 1`,
		Params: map[string]interface{}{
			"callrail_company_id": callrailCompanyID,
			"tenant_id":          tenantID,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("office not found for CallRail company ID")
	}
	if err != nil {
		return nil, err
	}

	office := &models.Office{}
	err = row.Columns(
		&office.TenantID, &office.OfficeID, &office.CallRailCompanyID,
		&office.CallRailAPIKey, &office.WorkflowConfig, &office.Status,
		&office.CreatedAt, &office.UpdatedAt,
	)
	return office, err
}

// ===== ANALYTICS OPERATIONS =====

// GetRequestCountsByTenant gets request counts by communication mode for a tenant
func (sc *SpannerClient) GetRequestCountsByTenant(ctx context.Context, tenantID string, since time.Time) (map[string]int64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT communication_mode, COUNT(*) as count
		      FROM requests
		      WHERE tenant_id = @tenant_id AND created_at >= @since
		      GROUP BY communication_mode`,
		Params: map[string]interface{}{
			"tenant_id": tenantID,
			"since":     since,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	counts := make(map[string]int64)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var mode string
		var count int64
		if err := row.Columns(&mode, &count); err != nil {
			return nil, err
		}

		counts[mode] = count
	}

	return counts, nil
}

// GetAverageLeadScoreByTenant gets average lead score for a tenant
func (sc *SpannerClient) GetAverageLeadScoreByTenant(ctx context.Context, tenantID string, since time.Time) (float64, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AVG(CAST(lead_score AS FLOAT64)) as avg_score
		      FROM requests
		      WHERE tenant_id = @tenant_id AND lead_score IS NOT NULL AND created_at >= @since`,
		Params: map[string]interface{}{
			"tenant_id": tenantID,
			"since":     since,
		},
	}

	iter := sc.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	var avgScore spanner.NullFloat64
	if err := row.Column(0, &avgScore); err != nil {
		return 0, err
	}

	if avgScore.Valid {
		return avgScore.Float64, nil
	}
	return 0, nil
}

// ===== TRANSACTION OPERATIONS =====

// CreateRequestWithRecording creates a request and recording in a single transaction
func (sc *SpannerClient) CreateRequestWithRecording(ctx context.Context, request *models.Request, recording *models.CallRecording) error {
	return sc.RunTransaction(ctx, func(txn *Transaction) error {
		// Insert request
		err := txn.InsertOrUpdate("requests", []string{
			"request_id", "tenant_id", "source", "request_type", "status",
			"data", "ai_normalized", "ai_extracted", "call_id", "recording_url",
			"transcription_data", "ai_analysis", "lead_score", "communication_mode",
			"spam_likelihood", "created_at", "updated_at",
		}, []interface{}{
			request.RequestID, request.TenantID, request.Source, request.RequestType, request.Status,
			request.Data, request.AINormalized, request.AIExtracted, request.CallID, request.RecordingURL,
			request.TranscriptionData, request.AIAnalysis, request.LeadScore, request.CommunicationMode,
			request.SpamLikelihood, request.CreatedAt, request.UpdatedAt,
		})
		if err != nil {
			return err
		}

		// Insert recording if provided
		if recording != nil {
			err = txn.InsertOrUpdate("call_recordings", []string{
				"recording_id", "tenant_id", "call_id", "storage_url",
				"transcription_status", "created_at",
			}, []interface{}{
				recording.RecordingID, recording.TenantID, recording.CallID,
				recording.StorageURL, recording.TranscriptionStatus, recording.CreatedAt,
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// InsertOrUpdate is a transaction helper method
func (t *Transaction) InsertOrUpdate(table string, columns []string, values []interface{}) error {
	mutation := spanner.InsertOrUpdate(table, columns, values)
	return t.txn.BufferWrite([]*spanner.Mutation{mutation})
}

// Update is a transaction helper method
func (t *Transaction) Update(table string, columns []string, values []interface{}) error {
	mutation := spanner.Update(table, columns, values)
	return t.txn.BufferWrite([]*spanner.Mutation{mutation})
}