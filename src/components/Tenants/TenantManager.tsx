// Tenant Management Interface Component
// Location: ./src/components/Tenants/TenantManager.tsx

import React, { useState, useEffect } from 'react';
import { TenantConfig, WorkflowConfig, CRMSettings } from '@/types/api';
import { apiService } from '@/services/api';
import { CRMConfiguration } from '../CRM/CRMConfiguration';

interface TenantManagerProps {
  selectedTenant: string;
  tenants: TenantConfig[];
  onTenantUpdate: (tenant: TenantConfig) => void;
}

type TabType = 'general' | 'workflow' | 'callrail' | 'crm' | 'service-areas';

export const TenantManager: React.FC<TenantManagerProps> = ({
  selectedTenant,
  tenants,
  onTenantUpdate,
}) => {
  const [activeTab, setActiveTab] = useState<TabType>('general');
  const [currentTenant, setCurrentTenant] = useState<TenantConfig | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Load tenant details
  useEffect(() => {
    const loadTenant = async () => {
      if (!selectedTenant) return;

      setIsLoading(true);
      setError(null);

      try {
        const tenant = await apiService.getTenant(selectedTenant);
        setCurrentTenant(tenant);
      } catch (error) {
        setError(error instanceof Error ? error.message : 'Failed to load tenant');
      } finally {
        setIsLoading(false);
      }
    };

    loadTenant();
  }, [selectedTenant]);

  // Save tenant configuration
  const saveTenant = async (updatedConfig: Partial<TenantConfig>) => {
    if (!currentTenant) return;

    setIsSaving(true);
    setError(null);
    setSuccessMessage(null);

    try {
      const updatedTenant = await apiService.updateTenant(currentTenant.id, updatedConfig);
      setCurrentTenant(updatedTenant);
      onTenantUpdate(updatedTenant);
      setSuccessMessage('Tenant configuration updated successfully');

      // Clear success message after 3 seconds
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to update tenant');
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return <TenantLoadingSkeleton />;
  }

  if (!currentTenant) {
    return <div className="text-center py-8 text-gray-500">No tenant selected</div>;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Tenant Management</h2>
          <p className="text-gray-600">Configure {currentTenant.name} settings</p>
        </div>
        <div className="flex items-center space-x-3">
          <StatusBadge status={currentTenant.status} />
          {isSaving && <LoadingSpinner />}
        </div>
      </div>

      {/* Status Messages */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex">
            <ErrorIcon className="h-5 w-5 text-red-400 mr-3 mt-0.5" />
            <p className="text-sm text-red-800">{error}</p>
          </div>
        </div>
      )}

      {successMessage && (
        <div className="bg-green-50 border border-green-200 rounded-md p-4">
          <div className="flex">
            <CheckIcon className="h-5 w-5 text-green-400 mr-3 mt-0.5" />
            <p className="text-sm text-green-800">{successMessage}</p>
          </div>
        </div>
      )}

      {/* Tab Navigation */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`py-2 px-1 border-b-2 font-medium text-sm transition-colors ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <div className="flex items-center space-x-2">
                <tab.icon className="w-5 h-5" />
                <span>{tab.label}</span>
              </div>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="bg-white rounded-lg shadow">
        {activeTab === 'general' && (
          <GeneralSettings
            tenant={currentTenant}
            onSave={saveTenant}
            isLoading={isSaving}
          />
        )}

        {activeTab === 'workflow' && (
          <WorkflowSettings
            tenant={currentTenant}
            onSave={saveTenant}
            isLoading={isSaving}
          />
        )}

        {activeTab === 'callrail' && (
          <CallRailSettings
            tenant={currentTenant}
            onSave={saveTenant}
            isLoading={isSaving}
          />
        )}

        {activeTab === 'crm' && (
          <CRMConfiguration
            tenant={currentTenant}
            onSave={saveTenant}
            isLoading={isSaving}
          />
        )}

        {activeTab === 'service-areas' && (
          <ServiceAreaSettings
            tenant={currentTenant}
            onSave={saveTenant}
            isLoading={isSaving}
          />
        )}
      </div>
    </div>
  );
};

// Tab definitions
const tabs = [
  { id: 'general' as TabType, label: 'General', icon: GeneralIcon },
  { id: 'workflow' as TabType, label: 'Workflow', icon: WorkflowIcon },
  { id: 'callrail' as TabType, label: 'CallRail', icon: CallRailIcon },
  { id: 'crm' as TabType, label: 'CRM Integration', icon: CRMIcon },
  { id: 'service-areas' as TabType, label: 'Service Areas', icon: ServiceAreaIcon },
];

// General Settings Component
const GeneralSettings: React.FC<{
  tenant: TenantConfig;
  onSave: (data: Partial<TenantConfig>) => void;
  isLoading: boolean;
}> = ({ tenant, onSave, isLoading }) => {
  const [formData, setFormData] = useState({
    name: tenant.name,
    status: tenant.status,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(formData);
  };

  return (
    <form onSubmit={handleSubmit} className="p-6 space-y-6">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Tenant Name
        </label>
        <input
          type="text"
          value={formData.name}
          onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          required
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Status
        </label>
        <select
          value={formData.status}
          onChange={(e) => setFormData(prev => ({ ...prev, status: e.target.value as any }))}
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        >
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="suspended">Suspended</option>
        </select>
      </div>

      <div className="pt-4 border-t">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {isLoading ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </form>
  );
};

// Workflow Settings Component
const WorkflowSettings: React.FC<{
  tenant: TenantConfig;
  onSave: (data: Partial<TenantConfig>) => void;
  isLoading: boolean;
}> = ({ tenant, onSave, isLoading }) => {
  const [workflowConfig, setWorkflowConfig] = useState<WorkflowConfig>(
    tenant.workflow_config
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({ workflow_config: workflowConfig });
  };

  const updateAISetting = (key: keyof WorkflowConfig['ai_settings'], value: any) => {
    setWorkflowConfig(prev => ({
      ...prev,
      ai_settings: {
        ...prev.ai_settings,
        [key]: value,
      },
    }));
  };

  return (
    <form onSubmit={handleSubmit} className="p-6 space-y-8">
      {/* AI Processing Settings */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">AI Processing</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-4">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={workflowConfig.ai_settings.enable_sentiment_analysis}
                onChange={(e) => updateAISetting('enable_sentiment_analysis', e.target.checked)}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Enable Sentiment Analysis
              </span>
            </label>

            <label className="flex items-center">
              <input
                type="checkbox"
                checked={workflowConfig.ai_settings.enable_lead_scoring}
                onChange={(e) => updateAISetting('enable_lead_scoring', e.target.checked)}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Enable Lead Scoring
              </span>
            </label>

            <label className="flex items-center">
              <input
                type="checkbox"
                checked={workflowConfig.ai_settings.enable_spam_detection}
                onChange={(e) => updateAISetting('enable_spam_detection', e.target.checked)}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm font-medium text-gray-700">
                Enable Spam Detection
              </span>
            </label>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Confidence Threshold
              </label>
              <input
                type="range"
                min="0"
                max="100"
                value={workflowConfig.ai_settings.confidence_threshold}
                onChange={(e) => updateAISetting('confidence_threshold', parseInt(e.target.value))}
                className="w-full"
              />
              <div className="text-sm text-gray-600 text-center">
                {workflowConfig.ai_settings.confidence_threshold}%
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Gemini Model
              </label>
              <select
                value={workflowConfig.ai_settings.gemini_model}
                onChange={(e) => updateAISetting('gemini_model', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="gemini-2.5-flash">Gemini 2.5 Flash</option>
                <option value="gemini-1.5-pro">Gemini 1.5 Pro</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      <div className="pt-4 border-t">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {isLoading ? 'Saving...' : 'Save Workflow Configuration'}
        </button>
      </div>
    </form>
  );
};

// CallRail Settings Component
const CallRailSettings: React.FC<{
  tenant: TenantConfig;
  onSave: (data: Partial<TenantConfig>) => void;
  isLoading: boolean;
}> = ({ tenant, onSave, isLoading }) => {
  const [formData, setFormData] = useState({
    callrail_company_id: tenant.callrail_company_id || '',
    callrail_api_key: tenant.callrail_api_key || '',
  });
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(formData);
  };

  const testConnection = async () => {
    if (!formData.callrail_company_id || !formData.callrail_api_key) {
      setTestResult({ success: false, message: 'Please fill in Company ID and API Key' });
      return;
    }

    setIsTestingConnection(true);
    setTestResult(null);

    try {
      const result = await apiService.testCallRailIntegration(tenant.id, {
        company_id: formData.callrail_company_id,
        api_key: formData.callrail_api_key,
      });

      setTestResult({
        success: result.success,
        message: result.success ? 'Connection successful!' : result.error_message || 'Connection failed',
      });
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : 'Connection test failed',
      });
    } finally {
      setIsTestingConnection(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="p-6 space-y-6">
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">CallRail Integration</h3>
        <p className="text-sm text-gray-600 mb-6">
          Configure CallRail API settings to enable phone call processing and webhook integration.
        </p>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              CallRail Company ID
            </label>
            <input
              type="text"
              value={formData.callrail_company_id}
              onChange={(e) => setFormData(prev => ({ ...prev, callrail_company_id: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter your CallRail Company ID"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              CallRail API Key
            </label>
            <input
              type="password"
              value={formData.callrail_api_key}
              onChange={(e) => setFormData(prev => ({ ...prev, callrail_api_key: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter your CallRail API Key"
            />
          </div>

          <div className="flex items-center space-x-4">
            <button
              type="button"
              onClick={testConnection}
              disabled={isTestingConnection}
              className="px-4 py-2 border border-gray-300 rounded-md shadow-sm bg-white text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
            >
              {isTestingConnection ? 'Testing...' : 'Test Connection'}
            </button>

            {testResult && (
              <div className={`text-sm ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
                {testResult.message}
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="pt-4 border-t">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {isLoading ? 'Saving...' : 'Save CallRail Settings'}
        </button>
      </div>
    </form>
  );
};

// Service Area Settings Component
const ServiceAreaSettings: React.FC<{
  tenant: TenantConfig;
  onSave: (data: Partial<TenantConfig>) => void;
  isLoading: boolean;
}> = ({ tenant, onSave, isLoading }) => {
  const [serviceAreas, setServiceAreas] = useState(tenant.service_areas || []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({ service_areas: serviceAreas });
  };

  const addServiceArea = () => {
    setServiceAreas(prev => [...prev, {
      id: `area_${Date.now()}`,
      name: '',
      zip_codes: [],
      service_types: [],
      priority_level: 1,
    }]);
  };

  const removeServiceArea = (index: number) => {
    setServiceAreas(prev => prev.filter((_, i) => i !== index));
  };

  return (
    <form onSubmit={handleSubmit} className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900">Service Areas</h3>
        <button
          type="button"
          onClick={addServiceArea}
          className="px-3 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500"
        >
          Add Area
        </button>
      </div>

      <div className="space-y-4">
        {serviceAreas.map((area, index) => (
          <div key={area.id} className="border border-gray-200 rounded-lg p-4">
            <div className="flex items-center justify-between mb-4">
              <h4 className="text-sm font-medium text-gray-900">Service Area {index + 1}</h4>
              <button
                type="button"
                onClick={() => removeServiceArea(index)}
                className="text-red-600 hover:text-red-800"
              >
                Remove
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input
                  type="text"
                  value={area.name}
                  onChange={(e) => {
                    const newAreas = [...serviceAreas];
                    newAreas[index] = { ...area, name: e.target.value };
                    setServiceAreas(newAreas);
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="e.g., Downtown LA"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">ZIP Codes</label>
                <input
                  type="text"
                  value={area.zip_codes.join(', ')}
                  onChange={(e) => {
                    const newAreas = [...serviceAreas];
                    newAreas[index] = {
                      ...area,
                      zip_codes: e.target.value.split(',').map(zip => zip.trim()).filter(Boolean)
                    };
                    setServiceAreas(newAreas);
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                  placeholder="90210, 90211, 90212"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Priority</label>
                <select
                  value={area.priority_level}
                  onChange={(e) => {
                    const newAreas = [...serviceAreas];
                    newAreas[index] = { ...area, priority_level: parseInt(e.target.value) };
                    setServiceAreas(newAreas);
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                >
                  <option value={1}>High</option>
                  <option value={2}>Medium</option>
                  <option value={3}>Low</option>
                </select>
              </div>
            </div>
          </div>
        ))}

        {serviceAreas.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No service areas configured. Click "Add Area" to get started.
          </div>
        )}
      </div>

      <div className="pt-4 border-t">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {isLoading ? 'Saving...' : 'Save Service Areas'}
        </button>
      </div>
    </form>
  );
};

// Helper Components
const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
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
  <div className="space-y-6">
    <div className="animate-pulse">
      <div className="flex items-center justify-between mb-6">
        <div>
          <div className="h-8 bg-gray-200 rounded w-64 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-48"></div>
        </div>
        <div className="h-6 bg-gray-200 rounded w-16"></div>
      </div>

      <div className="border-b border-gray-200">
        <div className="flex space-x-8">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="py-2">
              <div className="h-5 bg-gray-200 rounded w-20"></div>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <div className="h-64 bg-gray-200 rounded"></div>
      </div>
    </div>
  </div>
);

// Icon Components
const LoadingSpinner: React.FC = () => (
  <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600"></div>
);

const ErrorIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="currentColor" viewBox="0 0 20 20">
    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
  </svg>
);

const CheckIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="currentColor" viewBox="0 0 20 20">
    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
  </svg>
);

// Tab Icons
const GeneralIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
  </svg>
);

const WorkflowIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
  </svg>
);

const CallRailIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 5a2 2 0 012-2h3.28a1 1 0 01.948.684l1.498 4.493a1 1 0 01-.502 1.21l-2.257 1.13a11.042 11.042 0 005.516 5.516l1.13-2.257a1 1 0 011.21-.502l4.493 1.498a1 1 0 01.684.949V19a2 2 0 01-2 2h-1C9.716 21 3 14.284 3 6V5z" />
  </svg>
);

const CRMIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
  </svg>
);

const ServiceAreaIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17.657 16.657L13.414 20.9a1.998 1.998 0 01-2.827 0l-4.244-4.243a8 8 0 1111.314 0z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 11a3 3 0 11-6 0 3 3 0 016 0z" />
  </svg>
);

export default TenantManager;