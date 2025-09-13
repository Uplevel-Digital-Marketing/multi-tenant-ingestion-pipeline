// Multi-Tenant Ingestion Pipeline - TypeScript Interfaces
// Location: ./src/types/tenant.ts

export interface TenantConfig {
  id: string;
  name: string;
  status: 'active' | 'inactive' | 'suspended';
  callrail_company_id?: string;
  callrail_api_key?: string;
  workflow_config: WorkflowConfig;
  service_areas: ServiceArea[];
  crm_settings: CRMSettings;
  created_at: string;
  updated_at: string;
}

export interface WorkflowConfig {
  // Lead routing configuration
  lead_routing: {
    priority_threshold: number;
    hot_lead_threshold: number;
    routing_rules: RoutingRule[];
  };

  // AI processing settings
  ai_settings: {
    enable_sentiment_analysis: boolean;
    enable_lead_scoring: boolean;
    enable_spam_detection: boolean;
    confidence_threshold: number;
    gemini_model: string;
    speech_model: string;
  };

  // Notification preferences
  notifications: {
    email_alerts: boolean;
    slack_webhook?: string;
    sms_alerts: boolean;
    alert_conditions: AlertCondition[];
  };

  // Form processing rules
  form_processing: {
    required_fields: string[];
    validation_rules: ValidationRule[];
    auto_response_template?: string;
  };

  // Call processing settings
  call_processing: {
    auto_transcription: boolean;
    speaker_diarization: boolean;
    keyword_detection: string[];
    call_scoring_enabled: boolean;
  };
}

export interface ServiceArea {
  id: string;
  name: string;
  zip_codes: string[];
  service_types: string[];
  priority_level: number;
}

export interface CRMSettings {
  provider: 'salesforce' | 'hubspot' | 'pipedrive' | 'custom';
  api_endpoint?: string;
  field_mappings: FieldMapping[];
  sync_enabled: boolean;
  webhook_url?: string;
}

export interface FieldMapping {
  source_field: string;
  target_field: string;
  transformation?: string;
  required: boolean;
}

export interface RoutingRule {
  id: string;
  condition: string;
  action: 'assign_to' | 'notify' | 'priority' | 'tag';
  value: string;
  priority: number;
}

export interface AlertCondition {
  type: 'lead_score' | 'response_time' | 'error_rate' | 'volume';
  threshold: number;
  operator: 'greater_than' | 'less_than' | 'equals';
  frequency: 'immediate' | 'hourly' | 'daily';
}

export interface ValidationRule {
  field: string;
  type: 'required' | 'email' | 'phone' | 'regex';
  pattern?: string;
  message: string;
}

// Request Processing Types
export interface ProcessingRequest {
  id: string;
  tenant_id: string;
  source: 'form' | 'callrail' | 'calendar' | 'chat';
  status: 'received' | 'processing' | 'completed' | 'failed';
  communication_mode: 'phone' | 'email' | 'form' | 'chat';

  // Call-specific fields
  call_id?: string;
  recording_url?: string;
  audio_duration_seconds?: number;

  // Processing data
  transcription_data?: TranscriptionData;
  ai_analysis?: AIAnalysis;
  lead_score?: number;
  spam_likelihood?: number;

  // Performance metrics
  processing_time_ms: number;
  gemini_model_used?: string;
  speech_model_used?: string;

  // Timestamps
  created_at: string;
  processed_at?: string;
  updated_at: string;
}

export interface TranscriptionData {
  full_text: string;
  confidence: number;
  language: string;
  speakers: SpeakerSegment[];
  keywords_detected: string[];
  duration_seconds: number;
}

export interface SpeakerSegment {
  speaker_id: string;
  start_time: number;
  end_time: number;
  text: string;
  confidence: number;
}

export interface AIAnalysis {
  intent: string;
  sentiment: 'positive' | 'neutral' | 'negative';
  sentiment_score: number;
  urgency_level: 'low' | 'medium' | 'high' | 'urgent';
  topics: string[];
  action_items: string[];
  next_steps: string[];
  customer_info: CustomerInfo;
  project_details?: ProjectDetails;
}

export interface CustomerInfo {
  name?: string;
  phone?: string;
  email?: string;
  address?: string;
  estimated_budget?: number;
  timeline?: string;
  previous_customer: boolean;
}

export interface ProjectDetails {
  project_type: string;
  rooms_involved: string[];
  estimated_cost: number;
  timeline_weeks: number;
  special_requirements: string[];
}

// Dashboard Metrics Types
export interface DashboardMetrics {
  tenant_id: string;
  time_range: 'hour' | 'day' | 'week' | 'month';

  // Volume metrics
  total_requests: number;
  requests_by_source: Record<string, number>;
  requests_by_hour: Array<{ hour: string; count: number }>;

  // Performance metrics
  avg_processing_time: number;
  success_rate: number;
  error_rate: number;

  // Lead metrics
  total_leads: number;
  qualified_leads: number;
  avg_lead_score: number;
  conversion_rate: number;

  // Call metrics
  total_calls: number;
  avg_call_duration: number;
  transcription_accuracy: number;
  spam_calls_blocked: number;

  // Cost metrics
  processing_costs: CostBreakdown;

  // Real-time status
  current_processing_queue: number;
  active_integrations: string[];
  last_updated: string;
}

export interface CostBreakdown {
  total_cost: number;
  gemini_api_cost: number;
  speech_api_cost: number;
  storage_cost: number;
  compute_cost: number;
  estimated_monthly: number;
}

// Real-time Event Types for Server-Sent Events
export interface RealtimeEvent {
  type: 'request_received' | 'processing_complete' | 'error' | 'metric_update';
  tenant_id: string;
  data: any;
  timestamp: string;
}

export interface ProcessingComplete extends RealtimeEvent {
  type: 'processing_complete';
  data: {
    request_id: string;
    processing_time_ms: number;
    lead_score: number;
    success: boolean;
  };
}

export interface MetricUpdate extends RealtimeEvent {
  type: 'metric_update';
  data: {
    metric_name: string;
    current_value: number;
    previous_value: number;
    change_percentage: number;
  };
}

// UI State Types
export interface DashboardState {
  selectedTenant: string | null;
  timeRange: 'hour' | 'day' | 'week' | 'month';
  filters: DashboardFilters;
  realtime_enabled: boolean;
  auto_refresh_interval: number;
}

export interface DashboardFilters {
  source: string[];
  status: string[];
  lead_score_min?: number;
  lead_score_max?: number;
  date_range?: {
    start: string;
    end: string;
  };
}