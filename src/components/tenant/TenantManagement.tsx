// Tenant Management Interface with Workflow Configuration
// Location: ./src/components/tenant/TenantManagement.tsx

import React, { useState, useEffect, useCallback } from 'react';
import { TenantConfig, WorkflowConfig, ServiceArea, CRMSettings } from '@/types/tenant';

interface TenantManagementProps {
  tenantId: string;
  onTenantUpdate?: (tenant: TenantConfig) => void;
}

export const TenantManagement: React.FC<TenantManagementProps> = ({
  tenantId,
  onTenantUpdate,
}) => {
  const [tenant, setTenant] = useState<TenantConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [activeTab, setActiveTab] = useState<'general' | 'workflow' | 'callrail' | 'crm' | 'areas'>('general');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [hasChanges, setHasChanges] = useState(false);

  // Load tenant configuration
  useEffect(() => {
    const loadTenant = async () => {
      setIsLoading(true);
      try {
        const response = await fetch(`/api/tenants/${tenantId}`);
        if (!response.ok) throw new Error('Failed to load tenant');

        const tenantData = await response.json();
        setTenant(tenantData);
      } catch (error) {
        console.error('Failed to load tenant:', error);
        setErrors({ general: 'Failed to load tenant configuration' });
      } finally {
        setIsLoading(false);
      }
    };

    loadTenant();
  }, [tenantId]);

  // Save tenant configuration
  const saveTenant = useCallback(async () => {
    if (!tenant || !hasChanges) return;

    setIsSaving(true);
    setErrors({});

    try {
      const response = await fetch(`/api/tenants/${tenantId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(tenant),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to save tenant');
      }

      const updatedTenant = await response.json();
      setTenant(updatedTenant);
      setHasChanges(false);
      onTenantUpdate?.(updatedTenant);
    } catch (error) {
      console.error('Failed to save tenant:', error);
      setErrors({ general: error instanceof Error ? error.message : 'Failed to save changes' });
    } finally {
      setIsSaving(false);
    }
  }, [tenant, tenantId, hasChanges, onTenantUpdate]);

  // Update tenant field
  const updateTenant = useCallback((updates: Partial<TenantConfig>) => {
    setTenant(prev => prev ? { ...prev, ...updates } : null);
    setHasChanges(true);
    setErrors({});
  }, []);

  if (isLoading) {
    return <TenantLoadingSkeleton />;
  }

  if (!tenant) {
    return <TenantErrorState />;
  }

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{tenant.name}</h1>
          <p className="text-gray-600">Tenant ID: {tenant.id}</p>
        </div>

        <div className="flex items-center space-x-3">
          <TenantStatusBadge status={tenant.status} />
          <button
            onClick={saveTenant}
            disabled={!hasChanges || isSaving}
            className={`px-4 py-2 rounded-md font-medium transition-colors ${
              hasChanges && !isSaving
                ? 'bg-blue-600 hover:bg-blue-700 text-white'
                : 'bg-gray-100 text-gray-400 cursor-not-allowed'
            }`}
          >
            {isSaving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>

      {/* Error Banner */}
      {errors.general && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex">
            <div className="text-red-400">⚠️</div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">Error</h3>
              <p className="text-sm text-red-700 mt-1">{errors.general}</p>
            </div>
          </div>
        </div>
      )}

      {/* Navigation Tabs */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          {[
            { id: 'general', label: 'General Settings', icon: '⚙️' },
            { id: 'workflow', label: 'Workflow Config', icon: '🔄' },
            { id: 'callrail', label: 'CallRail Integration', icon: '📞' },
            { id: 'crm', label: 'CRM Settings', icon: '🏢' },
            { id: 'areas', label: 'Service Areas', icon: '📍' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`py-2 px-1 border-b-2 font-medium text-sm flex items-center space-x-2 ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <span>{tab.icon}</span>
              <span>{tab.label}</span>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="bg-white rounded-lg shadow-sm border p-6">
        {activeTab === 'general' && (
          <GeneralSettings tenant={tenant} onUpdate={updateTenant} errors={errors} />
        )}
        {activeTab === 'workflow' && (
          <WorkflowConfiguration
            config={tenant.workflow_config}
            onUpdate={(config) => updateTenant({ workflow_config: config })}
            errors={errors}
          />
        )}
        {activeTab === 'callrail' && (
          <CallRailSettings tenant={tenant} onUpdate={updateTenant} errors={errors} />
        )}
        {activeTab === 'crm' && (
          <CRMConfiguration
            settings={tenant.crm_settings}
            onUpdate={(settings) => updateTenant({ crm_settings: settings })}
            errors={errors}
          />
        )}
        {activeTab === 'areas' && (
          <ServiceAreasManagement
            areas={tenant.service_areas}
            onUpdate={(areas) => updateTenant({ service_areas: areas })}
            errors={errors}
          />
        )}
      </div>
    </div>
  );
};

// General Settings Tab
const GeneralSettings: React.FC<{
  tenant: TenantConfig;
  onUpdate: (updates: Partial<TenantConfig>) => void;
  errors: Record<string, string>;
}> = ({ tenant, onUpdate, errors }) => (
  <div className="space-y-6">
    <div>
      <h3 className="text-lg font-medium text-gray-900 mb-4">Basic Information</h3>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Tenant Name
          </label>
          <input
            type="text"
            value={tenant.name}
            onChange={(e) => onUpdate({ name: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          {errors.name && <p className="mt-1 text-sm text-red-600">{errors.name}</p>}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Status
          </label>
          <select
            value={tenant.status}
            onChange={(e) => onUpdate({ status: e.target.value as any })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
            <option value="suspended">Suspended</option>
          </select>
        </div>
      </div>
    </div>

    <div>
      <h4 className="text-md font-medium text-gray-900 mb-3">Account Information</h4>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
        <div>
          <span className="text-gray-600">Created:</span>
          <span className="ml-2 text-gray-900">
            {new Date(tenant.created_at).toLocaleDateString()}
          </span>
        </div>
        <div>
          <span className="text-gray-600">Last Updated:</span>
          <span className="ml-2 text-gray-900">
            {new Date(tenant.updated_at).toLocaleDateString()}
          </span>
        </div>
        <div>
          <span className="text-gray-600">Service Areas:</span>
          <span className="ml-2 text-gray-900">{tenant.service_areas.length}</span>
        </div>
        <div>
          <span className="text-gray-600">CRM Integration:</span>
          <span className="ml-2 text-gray-900">
            {tenant.crm_settings.provider || 'Not configured'}
          </span>
        </div>
      </div>
    </div>
  </div>
);

// Workflow Configuration Tab
const WorkflowConfiguration: React.FC<{
  config: WorkflowConfig;
  onUpdate: (config: WorkflowConfig) => void;
  errors: Record<string, string>;
}> = ({ config, onUpdate, errors }) => {
  const [activeSection, setActiveSection] = useState<'routing' | 'ai' | 'notifications' | 'processing'>('routing');

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">Workflow Configuration</h3>
        <p className="text-gray-600 mb-6">
          Configure how requests are processed, routed, and handled by your AI system.
        </p>
      </div>

      {/* Section Navigation */}
      <div className="flex space-x-4 border-b border-gray-200">
        {[
          { id: 'routing', label: 'Lead Routing', icon: '🎯' },
          { id: 'ai', label: 'AI Processing', icon: '🤖' },
          { id: 'notifications', label: 'Notifications', icon: '🔔' },
          { id: 'processing', label: 'Form & Call Processing', icon: '📞' },
        ].map((section) => (
          <button
            key={section.id}
            onClick={() => setActiveSection(section.id as any)}
            className={`pb-2 px-1 border-b-2 font-medium text-sm flex items-center space-x-2 ${
              activeSection === section.id
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            <span>{section.icon}</span>
            <span>{section.label}</span>
          </button>
        ))}
      </div>

      {/* Section Content */}
      <div className="mt-6">
        {activeSection === 'routing' && (
          <LeadRoutingSection
            routing={config.lead_routing}
            onUpdate={(routing) => onUpdate({ ...config, lead_routing: routing })}
          />
        )}
        {activeSection === 'ai' && (
          <AISettingsSection
            settings={config.ai_settings}
            onUpdate={(ai_settings) => onUpdate({ ...config, ai_settings })}
          />
        )}
        {activeSection === 'notifications' && (
          <NotificationsSection
            notifications={config.notifications}
            onUpdate={(notifications) => onUpdate({ ...config, notifications })}
          />
        )}
        {activeSection === 'processing' && (
          <ProcessingSection
            formProcessing={config.form_processing}
            callProcessing={config.call_processing}
            onUpdateForm={(form_processing) => onUpdate({ ...config, form_processing })}
            onUpdateCall={(call_processing) => onUpdate({ ...config, call_processing })}
          />
        )}
      </div>
    </div>
  );
};

// CallRail Integration Settings
const CallRailSettings: React.FC<{
  tenant: TenantConfig;
  onUpdate: (updates: Partial<TenantConfig>) => void;
  errors: Record<string, string>;
}> = ({ tenant, onUpdate, errors }) => {
  const [testingConnection, setTestingConnection] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<'unknown' | 'success' | 'error'>('unknown');

  const testConnection = async () => {
    if (!tenant.callrail_company_id || !tenant.callrail_api_key) return;

    setTestingConnection(true);
    try {
      const response = await fetch('/api/integrations/callrail/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          company_id: tenant.callrail_company_id,
          api_key: tenant.callrail_api_key,
        }),
      });

      setConnectionStatus(response.ok ? 'success' : 'error');
    } catch (error) {
      setConnectionStatus('error');
    } finally {
      setTestingConnection(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">CallRail Integration</h3>
        <p className="text-gray-600 mb-6">
          Configure CallRail integration to process phone calls and webhooks.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            CallRail Company ID
          </label>
          <input
            type="text"
            value={tenant.callrail_company_id || ''}
            onChange={(e) => onUpdate({ callrail_company_id: e.target.value })}
            placeholder="Enter your CallRail Company ID"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          {errors.callrail_company_id && (
            <p className="mt-1 text-sm text-red-600">{errors.callrail_company_id}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            CallRail API Key
          </label>
          <input
            type="password"
            value={tenant.callrail_api_key || ''}
            onChange={(e) => onUpdate({ callrail_api_key: e.target.value })}
            placeholder="Enter your CallRail API Key"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          {errors.callrail_api_key && (
            <p className="mt-1 text-sm text-red-600">{errors.callrail_api_key}</p>
          )}
        </div>
      </div>

      {/* Connection Test */}
      <div className="border rounded-md p-4 bg-gray-50">
        <div className="flex items-center justify-between">
          <div>
            <h4 className="text-sm font-medium text-gray-900">Connection Status</h4>
            <p className="text-sm text-gray-600">Test your CallRail integration</p>
          </div>
          <div className="flex items-center space-x-3">
            {connectionStatus !== 'unknown' && (
              <div className={`flex items-center space-x-2 ${
                connectionStatus === 'success' ? 'text-green-600' : 'text-red-600'
              }`}>
                <span className="text-sm">
                  {connectionStatus === 'success' ? '✅ Connected' : '❌ Failed'}
                </span>
              </div>
            )}
            <button
              onClick={testConnection}
              disabled={testingConnection || !tenant.callrail_company_id || !tenant.callrail_api_key}
              className="px-3 py-1 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed"
            >
              {testingConnection ? 'Testing...' : 'Test Connection'}
            </button>
          </div>
        </div>
      </div>

      {/* Webhook Configuration */}
      <div className="border rounded-md p-4">
        <h4 className="text-sm font-medium text-gray-900 mb-3">Webhook Configuration</h4>
        <div className="space-y-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 mb-1">
              Webhook URL (Configure this in CallRail)
            </label>
            <div className="flex">
              <input
                type="text"
                value={`${window.location.origin}/api/webhooks/callrail/${tenant.id}`}
                readOnly
                className="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-l-md bg-gray-50"
              />
              <button
                onClick={() => navigator.clipboard.writeText(`${window.location.origin}/api/webhooks/callrail/${tenant.id}`)}
                className="px-3 py-2 text-sm bg-gray-100 border border-l-0 border-gray-300 rounded-r-md hover:bg-gray-200"
              >
                📋 Copy
              </button>
            </div>
          </div>
          <p className="text-xs text-gray-500">
            Configure this URL in your CallRail account webhook settings to receive call notifications.
          </p>
        </div>
      </div>
    </div>
  );
};

// Supporting Components
const TenantStatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const colors = {
    active: 'bg-green-100 text-green-800',
    inactive: 'bg-gray-100 text-gray-800',
    suspended: 'bg-red-100 text-red-800',
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status as keyof typeof colors]}`}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
};

const TenantLoadingSkeleton: React.FC = () => (
  <div className="max-w-6xl mx-auto space-y-6">
    <div className="flex justify-between items-center">
      <div>
        <div className="h-8 bg-gray-200 rounded w-48 animate-pulse mb-2" />
        <div className="h-4 bg-gray-200 rounded w-32 animate-pulse" />
      </div>
      <div className="h-10 bg-gray-200 rounded w-32 animate-pulse" />
    </div>
    <div className="bg-white rounded-lg shadow-sm border p-6">
      <div className="space-y-4">
        {[...Array(6)].map((_, i) => (
          <div key={i} className="h-4 bg-gray-200 rounded animate-pulse" />
        ))}
      </div>
    </div>
  </div>
);

const TenantErrorState: React.FC = () => (
  <div className="max-w-6xl mx-auto">
    <div className="bg-white rounded-lg shadow-sm border p-8 text-center">
      <div className="text-red-500 mb-4">❌</div>
      <h3 className="text-lg font-medium text-gray-900 mb-2">Failed to load tenant</h3>
      <p className="text-gray-600">Please check the tenant ID and try again.</p>
    </div>
  </div>
);

// Placeholder components for complex sections
const LeadRoutingSection: React.FC<any> = () => <div>Lead Routing Configuration...</div>;
const AISettingsSection: React.FC<any> = () => <div>AI Processing Settings...</div>;
const NotificationsSection: React.FC<any> = () => <div>Notification Preferences...</div>;
const ProcessingSection: React.FC<any> = () => <div>Form & Call Processing...</div>;
const CRMConfiguration: React.FC<any> = () => <div>CRM Integration Settings...</div>;
const ServiceAreasManagement: React.FC<any> = () => <div>Service Areas Management...</div>;