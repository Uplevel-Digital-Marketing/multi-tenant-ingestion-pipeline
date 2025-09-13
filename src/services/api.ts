// API Service Layer for Multi-Tenant Ingestion Pipeline
// Location: ./src/services/api.ts

import axios, { AxiosInstance, AxiosResponse } from 'axios';
import { TenantConfig, DashboardMetrics, ProcessingRequest } from '@/types/api';

interface ApiConfig {
  baseURL: string;
  timeout: number;
  tenantId?: string;
}

class ApiService {
  private client: AxiosInstance;
  private tenantId: string | null = null;

  constructor(config: ApiConfig) {
    this.client = axios.create({
      baseURL: config.baseURL,
      timeout: config.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor to add tenant context
    this.client.interceptors.request.use((config) => {
      if (this.tenantId) {
        config.headers['X-Tenant-ID'] = this.tenantId;
        config.params = { ...config.params, tenant_id: this.tenantId };
      }
      return config;
    });

    // Response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        console.error('[API Service] Request failed:', {
          url: error.config?.url,
          status: error.response?.status,
          message: error.message,
        });
        return Promise.reject(error);
      }
    );
  }

  setTenant(tenantId: string) {
    this.tenantId = tenantId;
  }

  // Tenant Management
  async getTenants(): Promise<TenantConfig[]> {
    const response = await this.client.get<TenantConfig[]>('/api/tenants');
    return response.data;
  }

  async getTenant(tenantId: string): Promise<TenantConfig> {
    const response = await this.client.get<TenantConfig>(`/api/tenants/${tenantId}`);
    return response.data;
  }

  async updateTenant(tenantId: string, data: Partial<TenantConfig>): Promise<TenantConfig> {
    const response = await this.client.put<TenantConfig>(`/api/tenants/${tenantId}`, data);
    return response.data;
  }

  async testCallRailIntegration(tenantId: string, config: { company_id: string; api_key: string }) {
    const response = await this.client.post(`/api/integrations/callrail/test`, {
      tenant_id: tenantId,
      ...config,
    });
    return response.data;
  }

  // Dashboard Metrics
  async getDashboardMetrics(tenantId: string, timeRange: string): Promise<DashboardMetrics> {
    const response = await this.client.get<DashboardMetrics>('/api/dashboard/metrics', {
      params: { tenant_id: tenantId, time_range: timeRange },
    });
    return response.data;
  }

  async getProcessingRequests(tenantId: string, filters?: any): Promise<ProcessingRequest[]> {
    const response = await this.client.get<ProcessingRequest[]>('/api/dashboard/requests', {
      params: { tenant_id: tenantId, ...filters },
    });
    return response.data;
  }

  async getProcessingRequest(requestId: string): Promise<ProcessingRequest> {
    const response = await this.client.get<ProcessingRequest>(`/api/dashboard/requests/${requestId}`);
    return response.data;
  }

  // Health and Status
  async getTenantHealth(tenantId: string) {
    const response = await this.client.get(`/api/dashboard/health`, {
      params: { tenant_id: tenantId },
    });
    return response.data;
  }

  // CRM Integration Management
  async testCRMConnection(tenantId: string, crmConfig: any) {
    const response = await this.client.post(`/api/integrations/crm/test`, {
      tenant_id: tenantId,
      ...crmConfig,
    });
    return response.data;
  }

  async getCRMFieldMappings(tenantId: string, crmProvider: string) {
    const response = await this.client.get(`/api/integrations/crm/fields/${crmProvider}`, {
      params: { tenant_id: tenantId },
    });
    return response.data;
  }
}

// Create default API instance
export const apiService = new ApiService({
  baseURL: process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080',
  timeout: 30000,
});

export default apiService;