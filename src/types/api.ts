// API Types for Multi-Tenant Ingestion Pipeline
// Location: ./src/types/api.ts

// Re-export all types from tenant.ts for convenience
export * from './tenant';

// Additional API-specific types
export interface ApiResponse<T = any> {
  success: boolean;
  data: T;
  message?: string;
  errors?: string[];
  timestamp: string;
}

export interface ApiError {
  code: string;
  message: string;
  details?: any;
  timestamp: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

// Request filters and parameters
export interface RequestFilters {
  source?: string[];
  status?: string[];
  lead_score_min?: number;
  lead_score_max?: number;
  date_range?: {
    start: string;
    end: string;
  };
  search?: string;
  page?: number;
  limit?: number;
}

// Connection test results
export interface ConnectionTestResult {
  success: boolean;
  response_time_ms: number;
  error_message?: string;
  details?: {
    endpoint: string;
    status_code?: number;
    headers?: Record<string, string>;
  };
}

// Health check response
export interface HealthCheckResponse {
  status: 'healthy' | 'warning' | 'error' | 'unknown';
  uptime_seconds: number;
  error_rate: number;
  avg_response_time_ms: number;
  active_integrations: string[];
  last_error?: string;
  checks: {
    database: 'healthy' | 'error';
    callrail_api: 'healthy' | 'error';
    crm_integrations: 'healthy' | 'error';
    ai_services: 'healthy' | 'error';
  };
}

// CRM field options for mapping
export interface CRMField {
  id: string;
  label: string;
  type: 'text' | 'email' | 'phone' | 'number' | 'date' | 'boolean' | 'select';
  required: boolean;
  options?: string[];
}

export interface CRMFieldsResponse {
  provider: string;
  fields: CRMField[];
  custom_fields: CRMField[];
}