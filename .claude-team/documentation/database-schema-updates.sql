-- =============================================================================
-- Database Schema Updates for CallRail Integration
-- Multi-Tenant Ingestion Pipeline - Standard Solution
-- =============================================================================

-- -----------------------------------------------------------------------------
-- 1. OFFICES TABLE UPDATES - CallRail Mapping & Configuration
-- -----------------------------------------------------------------------------

-- Add CallRail company mapping and API credentials
ALTER TABLE offices ADD COLUMN callrail_company_id STRING(50);
ALTER TABLE offices ADD COLUMN callrail_api_key STRING(100);

-- Update workflow_config to include CallRail-specific settings
COMMENT ON COLUMN offices.workflow_config IS 'JSON configuration for tenant workflow processing including CallRail settings';

-- Add indexes for CallRail lookups
CREATE INDEX idx_offices_callrail ON offices(callrail_company_id, tenant_id);
CREATE INDEX idx_offices_callrail_active ON offices(callrail_company_id, status) WHERE status = 'active';

-- -----------------------------------------------------------------------------
-- 2. REQUESTS TABLE UPDATES - Enhanced Call Processing
-- -----------------------------------------------------------------------------

-- Add CallRail-specific fields
ALTER TABLE requests ADD COLUMN call_id STRING(50);
ALTER TABLE requests ADD COLUMN recording_url STRING(500);
ALTER TABLE requests ADD COLUMN transcription_data JSON;
ALTER TABLE requests ADD COLUMN ai_analysis JSON;
ALTER TABLE requests ADD COLUMN lead_score INT64;
ALTER TABLE requests ADD COLUMN communication_mode STRING(20);
ALTER TABLE requests ADD COLUMN audio_duration_seconds INT64;
ALTER TABLE requests ADD COLUMN spam_likelihood FLOAT64;

-- Add processing metadata
ALTER TABLE requests ADD COLUMN processing_time_ms INT64;
ALTER TABLE requests ADD COLUMN gemini_model_used STRING(50);
ALTER TABLE requests ADD COLUMN speech_model_used STRING(50);

-- Add indexes for enhanced querying
CREATE INDEX idx_requests_call_id ON requests(call_id) WHERE call_id IS NOT NULL;
CREATE INDEX idx_requests_lead_score ON requests(tenant_id, lead_score DESC) WHERE lead_score IS NOT NULL;
CREATE INDEX idx_requests_communication_mode ON requests(tenant_id, communication_mode, created_at DESC);
CREATE INDEX idx_requests_spam ON requests(tenant_id, spam_likelihood DESC) WHERE spam_likelihood > 50;

-- Add comments for documentation
COMMENT ON COLUMN requests.call_id IS 'CallRail call identifier for phone call requests';
COMMENT ON COLUMN requests.recording_url IS 'Cloud Storage URL for call recording file';
COMMENT ON COLUMN requests.transcription_data IS 'JSON containing full transcription with speaker diarization and word timing';
COMMENT ON COLUMN requests.ai_analysis IS 'JSON containing Gemini analysis results including intent, sentiment, and lead scoring';
COMMENT ON COLUMN requests.lead_score IS 'AI-generated lead quality score from 1-100';
COMMENT ON COLUMN requests.communication_mode IS 'Type of communication: form, phone_call, calendar, chat';
COMMENT ON COLUMN requests.spam_likelihood IS 'Percentage confidence that request is spam (0-100)';

-- -----------------------------------------------------------------------------
-- 3. NEW TABLE: CALL_RECORDINGS - Audio File Management
-- -----------------------------------------------------------------------------

CREATE TABLE call_recordings (
  recording_id STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
  tenant_id STRING(36) NOT NULL,
  request_id STRING(36) NOT NULL,
  call_id STRING(50) NOT NULL,
  original_callrail_url STRING(500),
  storage_url STRING(500) NOT NULL,
  file_size_bytes INT64,
  duration_seconds INT64,
  format STRING(10),
  transcription_status STRING(20) DEFAULT 'pending',
  transcription_completed_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  archived_at TIMESTAMP,
  FOREIGN KEY(tenant_id) REFERENCES tenants(tenant_id),
  FOREIGN KEY(tenant_id, request_id) REFERENCES requests(tenant_id, request_id),
  PRIMARY KEY(tenant_id, recording_id)
) INTERLEAVE IN PARENT tenants ON DELETE CASCADE;

-- Indexes for call recording management
CREATE INDEX idx_recordings_call_id ON call_recordings(call_id);
CREATE INDEX idx_recordings_status ON call_recordings(tenant_id, transcription_status);
CREATE INDEX idx_recordings_created ON call_recordings(tenant_id, created_at DESC);

-- Comments
COMMENT ON TABLE call_recordings IS 'Audio file management for CallRail phone call recordings';
COMMENT ON COLUMN call_recordings.original_callrail_url IS 'Original download URL from CallRail API';
COMMENT ON COLUMN call_recordings.storage_url IS 'Google Cloud Storage URL for archived recording';
COMMENT ON COLUMN call_recordings.transcription_status IS 'Status: pending, processing, completed, failed';

-- -----------------------------------------------------------------------------
-- 4. NEW TABLE: WEBHOOK_EVENTS - Webhook Processing Log
-- -----------------------------------------------------------------------------

CREATE TABLE webhook_events (
  event_id STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
  tenant_id STRING(36),
  webhook_source STRING(50) NOT NULL,
  event_type STRING(50) NOT NULL,
  call_id STRING(50),
  raw_payload JSON NOT NULL,
  signature_verified BOOL NOT NULL,
  processing_status STRING(20) NOT NULL DEFAULT 'received',
  error_message STRING(MAX),
  retry_count INT64 DEFAULT 0,
  processed_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  PRIMARY KEY(event_id)
);

-- Indexes for webhook event tracking
CREATE INDEX idx_webhook_events_source ON webhook_events(webhook_source, event_type, created_at DESC);
CREATE INDEX idx_webhook_events_call ON webhook_events(call_id) WHERE call_id IS NOT NULL;
CREATE INDEX idx_webhook_events_status ON webhook_events(processing_status, created_at DESC);
CREATE INDEX idx_webhook_events_tenant ON webhook_events(tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL;

-- Comments
COMMENT ON TABLE webhook_events IS 'Log of all incoming webhook events for audit and debugging';
COMMENT ON COLUMN webhook_events.webhook_source IS 'Source of webhook: callrail, hubspot, calendly, etc.';
COMMENT ON COLUMN webhook_events.signature_verified IS 'Whether HMAC signature verification passed';
COMMENT ON COLUMN webhook_events.processing_status IS 'Status: received, processing, completed, failed, retrying';

-- -----------------------------------------------------------------------------
-- 5. UPDATE: CRM_INTEGRATIONS TABLE - Enhanced Tracking
-- -----------------------------------------------------------------------------

-- Add CallRail-specific fields to existing CRM integration tracking
ALTER TABLE crm_integrations ADD COLUMN call_id STRING(50);
ALTER TABLE crm_integrations ADD COLUMN lead_score INT64;
ALTER TABLE crm_integrations ADD COLUMN source_communication_mode STRING(20);

-- Update indexes
CREATE INDEX idx_crm_integrations_call ON crm_integrations(call_id) WHERE call_id IS NOT NULL;
CREATE INDEX idx_crm_integrations_score ON crm_integrations(tenant_id, lead_score DESC) WHERE lead_score IS NOT NULL;

-- Comments
COMMENT ON COLUMN crm_integrations.call_id IS 'CallRail call ID for phone-based leads';
COMMENT ON COLUMN crm_integrations.lead_score IS 'AI-generated lead quality score pushed to CRM';
COMMENT ON COLUMN crm_integrations.source_communication_mode IS 'Original communication type that generated this CRM record';

-- -----------------------------------------------------------------------------
-- 6. NEW TABLE: AI_PROCESSING_LOG - AI Analysis Tracking
-- -----------------------------------------------------------------------------

CREATE TABLE ai_processing_log (
  processing_id STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
  tenant_id STRING(36) NOT NULL,
  request_id STRING(36) NOT NULL,
  processing_type STRING(50) NOT NULL,
  model_used STRING(50) NOT NULL,
  input_tokens INT64,
  output_tokens INT64,
  processing_time_ms INT64,
  confidence_score FLOAT64,
  result_data JSON,
  cost_estimate_usd NUMERIC(10,6),
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  FOREIGN KEY(tenant_id) REFERENCES tenants(tenant_id),
  FOREIGN KEY(tenant_id, request_id) REFERENCES requests(tenant_id, request_id),
  PRIMARY KEY(tenant_id, processing_id)
) INTERLEAVE IN PARENT tenants ON DELETE CASCADE;

-- Indexes for AI processing analytics
CREATE INDEX idx_ai_processing_type ON ai_processing_log(tenant_id, processing_type, created_at DESC);
CREATE INDEX idx_ai_processing_model ON ai_processing_log(model_used, created_at DESC);
CREATE INDEX idx_ai_processing_cost ON ai_processing_log(tenant_id, cost_estimate_usd DESC);

-- Comments
COMMENT ON TABLE ai_processing_log IS 'Detailed log of AI processing operations for cost tracking and performance monitoring';
COMMENT ON COLUMN ai_processing_log.processing_type IS 'Type: transcription, content_analysis, spam_detection, sentiment_analysis';
COMMENT ON COLUMN ai_processing_log.confidence_score IS 'AI model confidence in result accuracy (0.0-1.0)';

-- -----------------------------------------------------------------------------
-- 7. UPDATE EXISTING TABLES - Enhanced Multi-tenancy
-- -----------------------------------------------------------------------------

-- Add CallRail configuration to tenant_genkit_config
ALTER TABLE tenant_genkit_config ADD COLUMN callrail_webhook_secret STRING(100);
ALTER TABLE tenant_genkit_config ADD COLUMN audio_processing_enabled BOOL DEFAULT true;
ALTER TABLE tenant_genkit_config ADD COLUMN transcription_language STRING(10) DEFAULT 'en-US';
ALTER TABLE tenant_genkit_config ADD COLUMN min_lead_score_threshold INT64 DEFAULT 30;

-- Comments
COMMENT ON COLUMN tenant_genkit_config.callrail_webhook_secret IS 'Secret token for verifying CallRail webhook signatures';
COMMENT ON COLUMN tenant_genkit_config.min_lead_score_threshold IS 'Minimum AI lead score to trigger CRM integration and notifications';

-- -----------------------------------------------------------------------------
-- 8. EXAMPLE WORKFLOW_CONFIG JSON STRUCTURE
-- -----------------------------------------------------------------------------

/*
Example workflow_config JSON for offices table:
{
  "communication_detection": {
    "enabled": true,
    "phone_processing": {
      "transcribe_audio": true,
      "extract_details": true,
      "sentiment_analysis": true,
      "speaker_diarization": true,
      "language": "en-US"
    }
  },
  "validation": {
    "spam_detection": {
      "enabled": true,
      "confidence_threshold": 75,
      "ml_model": "gemini-2.5-flash"
    }
  },
  "service_area": {
    "enabled": true,
    "validation_method": "zip_code",
    "allowed_areas": ["90210", "90211"],
    "buffer_miles": 25
  },
  "crm_integration": {
    "enabled": true,
    "provider": "hubspot",
    "credentials_secret_name": "hubspot-api-key-tenant-123",
    "field_mapping": {
      "name": "firstname",
      "phone": "phone",
      "lead_score": "hs_lead_score"
    },
    "push_immediately": true,
    "min_lead_score": 30
  },
  "email_notifications": {
    "enabled": true,
    "recipients": ["sales@company.com"],
    "conditions": {
      "send_for_spam": false,
      "min_lead_score": 30
    }
  },
  "callrail_integration": {
    "company_id": "12345",
    "api_key_secret_name": "callrail-api-key-tenant-123",
    "webhook_secret": "webhook_secret_token_123",
    "auto_download_recordings": true,
    "retention_days": 2555
  }
}
*/

-- -----------------------------------------------------------------------------
-- 9. PERFORMANCE OPTIMIZATION UPDATES
-- -----------------------------------------------------------------------------

-- Update existing property graph to include new CallRail relationships
CREATE OR REPLACE PROPERTY GRAPH agent_platform_graph
  NODE TABLES(
    -- Existing tables...
    call_recordings
      KEY(tenant_id, recording_id)
      LABEL Recording PROPERTIES(
        recording_id, call_id, storage_url, duration_seconds,
        transcription_status, created_at),

    webhook_events
      KEY(event_id)
      LABEL WebhookEvent PROPERTIES(
        event_id, webhook_source, event_type, call_id,
        processing_status, created_at)
  )
  EDGE TABLES(
    -- Add new relationships...
  );

-- -----------------------------------------------------------------------------
-- 10. DATA MIGRATION SCRIPTS (if needed)
-- -----------------------------------------------------------------------------

-- Update existing requests to have default communication_mode
UPDATE requests
SET communication_mode = 'form'
WHERE communication_mode IS NULL AND source != 'callrail_webhook';

-- Set default spam_likelihood for existing records
UPDATE requests
SET spam_likelihood = 0.0
WHERE spam_likelihood IS NULL;

-- -----------------------------------------------------------------------------
-- 11. SECURITY ENHANCEMENTS
-- -----------------------------------------------------------------------------

-- Row-level security for call recordings (ensure tenant isolation)
CREATE ROW ACCESS POLICY tenant_isolation_recordings ON call_recordings
  GRANT TO ('application_role')
  FILTER USING (tenant_id = @tenant_id_param);

-- Row-level security for webhook events
CREATE ROW ACCESS POLICY tenant_isolation_webhooks ON webhook_events
  GRANT TO ('application_role')
  FILTER USING (tenant_id IS NULL OR tenant_id = @tenant_id_param);

-- -----------------------------------------------------------------------------
-- 12. MONITORING AND ALERTING VIEWS
-- -----------------------------------------------------------------------------

-- View for webhook processing health
CREATE VIEW webhook_processing_health AS
SELECT
  webhook_source,
  processing_status,
  COUNT(*) as event_count,
  AVG(retry_count) as avg_retries,
  MAX(created_at) as last_event
FROM webhook_events
WHERE created_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
GROUP BY webhook_source, processing_status;

-- View for call processing metrics
CREATE VIEW call_processing_metrics AS
SELECT
  tenant_id,
  COUNT(*) as total_calls,
  AVG(lead_score) as avg_lead_score,
  AVG(audio_duration_seconds) as avg_duration,
  AVG(processing_time_ms) as avg_processing_time,
  SUM(CASE WHEN spam_likelihood > 75 THEN 1 ELSE 0 END) as spam_count
FROM requests
WHERE communication_mode = 'phone_call'
  AND created_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
GROUP BY tenant_id;

-- -----------------------------------------------------------------------------
-- DEPLOYMENT VERIFICATION QUERIES
-- -----------------------------------------------------------------------------

-- Verify schema updates
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name IN ('offices', 'requests', 'call_recordings', 'webhook_events')
ORDER BY table_name, ordinal_position;

-- Verify indexes
SELECT index_name, table_name, is_unique, index_type
FROM information_schema.indexes
WHERE table_name IN ('offices', 'requests', 'call_recordings', 'webhook_events')
ORDER BY table_name, index_name;