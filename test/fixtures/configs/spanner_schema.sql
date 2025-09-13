-- Multi-Tenant Ingestion Pipeline Database Schema
-- Google Cloud Spanner DDL for testing

-- Tenant configurations table
CREATE TABLE tenant_configurations (
  tenant_id STRING(36) NOT NULL,
  tenant_name STRING(100) NOT NULL,
  tenant_type STRING(20) NOT NULL, -- basic, standard, premium, enterprise
  is_active BOOL NOT NULL DEFAULT true,
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  updated_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- Configuration JSON fields
  crm_settings JSON,
  ai_prompts JSON,
  processing_rules JSON,
  notification_settings JSON,

  -- Security and limits
  encryption_key_id STRING(255),
  rate_limit_per_minute INT64,
  storage_limit_gb INT64,
  retention_days INT64,

  -- Billing
  subscription_tier STRING(20),
  billing_email STRING(255),

) PRIMARY KEY (tenant_id);

-- Main ingestion records table (partitioned by tenant)
CREATE TABLE ingestion_records (
  tenant_id STRING(36) NOT NULL,
  id STRING(36) NOT NULL,
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  updated_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- Audio file information
  audio_file_name STRING(255),
  audio_file_size_bytes INT64,
  audio_duration_ms INT64,
  audio_format STRING(10),
  audio_hash STRING(64) NOT NULL,
  audio_storage_path STRING(500),

  -- Processing information
  processing_status STRING(20) NOT NULL, -- uploaded, processing, completed, failed
  processing_stage STRING(50), -- audio_validation, transcription, ai_extraction, crm_integration
  processing_started_at TIMESTAMP,
  processing_completed_at TIMESTAMP,
  processing_error TEXT,

  -- Transcription results
  transcript TEXT,
  transcript_confidence FLOAT64,
  language_code STRING(10),

  -- AI extraction results
  extracted_data JSON,
  extraction_confidence FLOAT64,
  extraction_model_version STRING(50),

  -- CRM integration
  crm_lead_id STRING(100),
  crm_status STRING(20),
  crm_error TEXT,
  crm_created_at TIMESTAMP,

  -- Metadata and tags
  tags ARRAY<STRING(50)>,
  metadata JSON,
  source_ip STRING(45),
  user_agent STRING(500),

) PRIMARY KEY (tenant_id, id);

-- Audio processing queue for workflow management
CREATE TABLE processing_queue (
  id STRING(36) NOT NULL,
  tenant_id STRING(36) NOT NULL,
  ingestion_record_id STRING(36) NOT NULL,
  queue_name STRING(50) NOT NULL, -- audio_processing, ai_extraction, crm_integration
  priority INT64 NOT NULL DEFAULT 5, -- 1 (highest) to 10 (lowest)
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  scheduled_at TIMESTAMP,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,

  -- Processing details
  processor_id STRING(100), -- ID of the worker processing this item
  retry_count INT64 NOT NULL DEFAULT 0,
  max_retries INT64 NOT NULL DEFAULT 3,
  processing_timeout_ms INT64,

  -- Status tracking
  status STRING(20) NOT NULL, -- pending, processing, completed, failed, cancelled
  error_message TEXT,
  processing_metadata JSON,

) PRIMARY KEY (id);

-- Audit logging for security and compliance
CREATE TABLE audit_logs (
  id STRING(36) NOT NULL,
  tenant_id STRING(36) NOT NULL,
  timestamp TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- Event details
  event_type STRING(50) NOT NULL, -- upload, access, delete, update, login, error
  resource_type STRING(50), -- ingestion_record, tenant_config, user_account
  resource_id STRING(36),

  -- User and session info
  user_id STRING(36),
  session_id STRING(36),
  ip_address STRING(45),
  user_agent STRING(500),

  -- Event outcome
  success BOOL NOT NULL,
  error_type STRING(50),
  error_message TEXT,

  -- Additional context
  details JSON,
  security_level STRING(20), -- info, warning, critical

) PRIMARY KEY (tenant_id, timestamp, id);

-- Performance metrics for monitoring
CREATE TABLE performance_metrics (
  id STRING(36) NOT NULL,
  tenant_id STRING(36) NOT NULL,
  timestamp TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),

  -- Metric details
  metric_name STRING(100) NOT NULL,
  metric_value FLOAT64 NOT NULL,
  metric_unit STRING(20), -- ms, bytes, count, percent

  -- Context
  resource_id STRING(36),
  component STRING(50), -- audio_processor, ai_service, crm_integration
  operation STRING(50), -- upload, transcribe, extract, integrate

  -- Dimensions for grouping
  dimensions JSON,

) PRIMARY KEY (tenant_id, timestamp, id);

-- User sessions for authentication tracking
CREATE TABLE user_sessions (
  session_id STRING(36) NOT NULL,
  tenant_id STRING(36) NOT NULL,
  user_id STRING(36) NOT NULL,
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  last_accessed_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
  expires_at TIMESTAMP NOT NULL,

  -- Session details
  ip_address STRING(45),
  user_agent STRING(500),
  is_active BOOL NOT NULL DEFAULT true,

  -- Security
  jwt_token_hash STRING(64),
  permissions ARRAY<STRING(50)>,
  role STRING(20),

) PRIMARY KEY (session_id);

-- Tenant usage statistics for billing
CREATE TABLE tenant_usage (
  tenant_id STRING(36) NOT NULL,
  period_start TIMESTAMP NOT NULL,
  period_end TIMESTAMP NOT NULL,

  -- Usage counters
  total_uploads INT64 NOT NULL DEFAULT 0,
  total_audio_minutes INT64 NOT NULL DEFAULT 0,
  total_storage_bytes INT64 NOT NULL DEFAULT 0,
  total_api_calls INT64 NOT NULL DEFAULT 0,
  total_crm_integrations INT64 NOT NULL DEFAULT 0,

  -- Performance stats
  avg_processing_time_ms FLOAT64,
  success_rate FLOAT64,
  error_count INT64 NOT NULL DEFAULT 0,

  -- Billing
  billing_period STRING(20), -- daily, weekly, monthly
  computed_cost FLOAT64,

) PRIMARY KEY (tenant_id, period_start);

-- Create indexes for efficient querying

-- Ingestion records indexes
CREATE INDEX idx_ingestion_records_status ON ingestion_records (tenant_id, processing_status, created_at DESC);
CREATE INDEX idx_ingestion_records_created_at ON ingestion_records (tenant_id, created_at DESC);
CREATE INDEX idx_ingestion_records_audio_hash ON ingestion_records (tenant_id, audio_hash);
CREATE INDEX idx_ingestion_records_crm_status ON ingestion_records (tenant_id, crm_status, crm_created_at DESC);

-- Processing queue indexes
CREATE INDEX idx_processing_queue_status ON processing_queue (queue_name, status, priority, created_at);
CREATE INDEX idx_processing_queue_tenant ON processing_queue (tenant_id, status, created_at);
CREATE INDEX idx_processing_queue_scheduled ON processing_queue (scheduled_at, status) WHERE scheduled_at IS NOT NULL;

-- Audit logs indexes
CREATE INDEX idx_audit_logs_event_type ON audit_logs (tenant_id, event_type, timestamp DESC);
CREATE INDEX idx_audit_logs_user ON audit_logs (tenant_id, user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_security ON audit_logs (security_level, timestamp DESC) WHERE security_level IN ('warning', 'critical');

-- Performance metrics indexes
CREATE INDEX idx_performance_metrics_name ON performance_metrics (tenant_id, metric_name, timestamp DESC);
CREATE INDEX idx_performance_metrics_component ON performance_metrics (component, operation, timestamp DESC);

-- User sessions indexes
CREATE INDEX idx_user_sessions_tenant_user ON user_sessions (tenant_id, user_id, last_accessed_at DESC);
CREATE INDEX idx_user_sessions_expires ON user_sessions (expires_at) WHERE is_active = true;

-- Tenant usage indexes
CREATE INDEX idx_tenant_usage_period ON tenant_usage (tenant_id, billing_period, period_start DESC);

-- Views for common queries

-- Active ingestion records view
CREATE VIEW active_ingestion_records AS
SELECT
  ir.*,
  tc.tenant_name,
  tc.tenant_type
FROM ingestion_records ir
JOIN tenant_configurations tc ON ir.tenant_id = tc.tenant_id
WHERE tc.is_active = true
  AND ir.processing_status IN ('uploaded', 'processing', 'completed');

-- Tenant performance summary view
CREATE VIEW tenant_performance_summary AS
SELECT
  tc.tenant_id,
  tc.tenant_name,
  tc.tenant_type,
  COUNT(ir.id) as total_records,
  COUNT(CASE WHEN ir.processing_status = 'completed' THEN 1 END) as completed_records,
  COUNT(CASE WHEN ir.processing_status = 'failed' THEN 1 END) as failed_records,
  AVG(ir.extraction_confidence) as avg_confidence,
  AVG(TIMESTAMP_DIFF(ir.processing_completed_at, ir.processing_started_at, MILLISECOND)) as avg_processing_time_ms
FROM tenant_configurations tc
LEFT JOIN ingestion_records ir ON tc.tenant_id = ir.tenant_id
WHERE tc.is_active = true
GROUP BY tc.tenant_id, tc.tenant_name, tc.tenant_type;

-- Security events view
CREATE VIEW security_events AS
SELECT
  al.*,
  tc.tenant_name,
  tc.tenant_type
FROM audit_logs al
JOIN tenant_configurations tc ON al.tenant_id = tc.tenant_id
WHERE al.security_level IN ('warning', 'critical')
  OR al.success = false
ORDER BY al.timestamp DESC;