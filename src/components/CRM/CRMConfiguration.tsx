// CRM Configuration Component
// Location: ./src/components/CRM/CRMConfiguration.tsx

import React, { useState, useEffect } from 'react';
import { TenantConfig, CRMSettings, FieldMapping, CRMField, CRMFieldsResponse } from '@/types/api';
import { apiService } from '@/services/api';

interface CRMConfigurationProps {
  tenant: TenantConfig;
  onSave: (data: Partial<TenantConfig>) => void;
  isLoading: boolean;
}

type CRMProvider = 'salesforce' | 'hubspot' | 'pipedrive' | 'custom';

export const CRMConfiguration: React.FC<CRMConfigurationProps> = ({
  tenant,
  onSave,
  isLoading,
}) => {
  const [crmSettings, setCrmSettings] = useState<CRMSettings>(tenant.crm_settings);
  const [availableFields, setAvailableFields] = useState<CRMField[]>([]);
  const [isLoadingFields, setIsLoadingFields] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  // Load CRM fields when provider changes
  useEffect(() => {
    if (crmSettings.provider && crmSettings.provider !== 'custom') {
      loadCRMFields(crmSettings.provider);
    }
  }, [crmSettings.provider]);

  const loadCRMFields = async (provider: CRMProvider) => {
    setIsLoadingFields(true);
    try {
      const fieldsResponse: CRMFieldsResponse = await apiService.getCRMFieldMappings(tenant.id, provider);
      setAvailableFields([...fieldsResponse.fields, ...fieldsResponse.custom_fields]);
    } catch (error) {
      console.error('Failed to load CRM fields:', error);
      setAvailableFields([]);
    } finally {
      setIsLoadingFields(false);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave({ crm_settings: crmSettings });
  };

  const testCRMConnection = async () => {
    setIsTestingConnection(true);
    setTestResult(null);

    try {
      const result = await apiService.testCRMConnection(tenant.id, crmSettings);
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

  const addFieldMapping = () => {
    const newMapping: FieldMapping = {
      source_field: '',
      target_field: '',
      required: false,
    };

    setCrmSettings(prev => ({
      ...prev,
      field_mappings: [...prev.field_mappings, newMapping],
    }));
  };

  const updateFieldMapping = (index: number, field: keyof FieldMapping, value: any) => {
    setCrmSettings(prev => ({
      ...prev,
      field_mappings: prev.field_mappings.map((mapping, i) =>
        i === index ? { ...mapping, [field]: value } : mapping
      ),
    }));
  };

  const removeFieldMapping = (index: number) => {
    setCrmSettings(prev => ({
      ...prev,
      field_mappings: prev.field_mappings.filter((_, i) => i !== index),
    }));
  };

  // Source fields from our system
  const sourceFields = [
    { value: 'customer_name', label: 'Customer Name' },
    { value: 'customer_phone', label: 'Phone Number' },
    { value: 'customer_email', label: 'Email Address' },
    { value: 'customer_address', label: 'Address' },
    { value: 'lead_score', label: 'Lead Score' },
    { value: 'project_type', label: 'Project Type' },
    { value: 'estimated_budget', label: 'Estimated Budget' },
    { value: 'urgency_level', label: 'Urgency Level' },
    { value: 'call_duration', label: 'Call Duration' },
    { value: 'call_transcript', label: 'Call Transcript' },
    { value: 'sentiment_score', label: 'Sentiment Score' },
    { value: 'call_recording_url', label: 'Call Recording URL' },
  ];

  return (
    <form onSubmit={handleSubmit} className="p-6 space-y-8">
      {/* CRM Provider Selection */}
      <div>
        <h3 className="text-lg font-medium text-gray-900 mb-4">CRM Integration Settings</h3>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              CRM Provider
            </label>
            <select
              value={crmSettings.provider}
              onChange={(e) => setCrmSettings(prev => ({
                ...prev,
                provider: e.target.value as CRMProvider,
                field_mappings: [], // Reset mappings when provider changes
              }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">Select CRM Provider</option>
              <option value="salesforce">Salesforce</option>
              <option value="hubspot">HubSpot</option>
              <option value="pipedrive">Pipedrive</option>
              <option value="custom">Custom API</option>
            </select>
          </div>

          {crmSettings.provider === 'custom' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Custom API Endpoint
              </label>
              <input
                type="url"
                value={crmSettings.api_endpoint || ''}
                onChange={(e) => setCrmSettings(prev => ({ ...prev, api_endpoint: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
                placeholder="https://api.your-crm.com/leads"
              />
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Webhook URL (Optional)
            </label>
            <input
              type="url"
              value={crmSettings.webhook_url || ''}
              onChange={(e) => setCrmSettings(prev => ({ ...prev, webhook_url: e.target.value }))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="https://your-crm.com/webhook/leads"
            />
            <p className="text-xs text-gray-500 mt-1">
              Optional webhook URL for receiving lead data from our system
            </p>
          </div>

          <div className="flex items-center">
            <input
              type="checkbox"
              id="sync_enabled"
              checked={crmSettings.sync_enabled}
              onChange={(e) => setCrmSettings(prev => ({ ...prev, sync_enabled: e.target.checked }))}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <label htmlFor="sync_enabled" className="ml-2 text-sm font-medium text-gray-700">
              Enable automatic CRM sync
            </label>
          </div>

          {crmSettings.provider && (
            <div className="flex items-center space-x-4">
              <button
                type="button"
                onClick={testCRMConnection}
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
          )}
        </div>
      </div>

      {/* Field Mapping */}
      {crmSettings.provider && (
        <div>
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-medium text-gray-900">Field Mapping</h3>
            <button
              type="button"
              onClick={addFieldMapping}
              className="px-3 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500"
            >
              Add Mapping
            </button>
          </div>

          {isLoadingFields && (
            <div className="text-center py-4">
              <div className="inline-flex items-center space-x-2">
                <LoadingSpinner />
                <span className="text-sm text-gray-600">Loading CRM fields...</span>
              </div>
            </div>
          )}

          <div className="space-y-4">
            {crmSettings.field_mappings.map((mapping, index) => (
              <FieldMappingRow
                key={index}
                mapping={mapping}
                index={index}
                sourceFields={sourceFields}
                targetFields={availableFields}
                onUpdate={updateFieldMapping}
                onRemove={removeFieldMapping}
              />
            ))}

            {crmSettings.field_mappings.length === 0 && !isLoadingFields && (
              <div className="text-center py-8 text-gray-500">
                No field mappings configured. Click "Add Mapping" to get started.
              </div>
            )}
          </div>

          {crmSettings.field_mappings.length > 0 && (
            <div className="mt-6 bg-blue-50 border border-blue-200 rounded-md p-4">
              <h4 className="text-sm font-medium text-blue-900 mb-2">Field Mapping Summary</h4>
              <div className="text-sm text-blue-800">
                <p><strong>Required mappings:</strong> {crmSettings.field_mappings.filter(m => m.required).length}</p>
                <p><strong>Optional mappings:</strong> {crmSettings.field_mappings.filter(m => !m.required).length}</p>
                <p><strong>Total mappings:</strong> {crmSettings.field_mappings.length}</p>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Provider-specific Settings */}
      {crmSettings.provider === 'salesforce' && (
        <SalesforceSettings
          settings={crmSettings}
          onUpdate={setCrmSettings}
        />
      )}

      {crmSettings.provider === 'hubspot' && (
        <HubSpotSettings
          settings={crmSettings}
          onUpdate={setCrmSettings}
        />
      )}

      {crmSettings.provider === 'pipedrive' && (
        <PipedriveSettings
          settings={crmSettings}
          onUpdate={setCrmSettings}
        />
      )}

      <div className="pt-4 border-t">
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {isLoading ? 'Saving...' : 'Save CRM Configuration'}
        </button>
      </div>
    </form>
  );
};

// Field Mapping Row Component
const FieldMappingRow: React.FC<{
  mapping: FieldMapping;
  index: number;
  sourceFields: Array<{ value: string; label: string }>;
  targetFields: CRMField[];
  onUpdate: (index: number, field: keyof FieldMapping, value: any) => void;
  onRemove: (index: number) => void;
}> = ({ mapping, index, sourceFields, targetFields, onUpdate, onRemove }) => (
  <div className="border border-gray-200 rounded-lg p-4">
    <div className="flex items-center justify-between mb-4">
      <h4 className="text-sm font-medium text-gray-900">Mapping {index + 1}</h4>
      <button
        type="button"
        onClick={() => onRemove(index)}
        className="text-red-600 hover:text-red-800"
      >
        <TrashIcon className="w-4 h-4" />
      </button>
    </div>

    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Source Field</label>
        <select
          value={mapping.source_field}
          onChange={(e) => onUpdate(index, 'source_field', e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        >
          <option value="">Select Source Field</option>
          {sourceFields.map((field) => (
            <option key={field.value} value={field.value}>
              {field.label}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Target Field</label>
        <select
          value={mapping.target_field}
          onChange={(e) => onUpdate(index, 'target_field', e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        >
          <option value="">Select Target Field</option>
          {targetFields.map((field) => (
            <option key={field.id} value={field.id}>
              {field.label} ({field.type})
            </option>
          ))}
        </select>
      </div>

      <div className="flex items-center space-x-4">
        <label className="flex items-center">
          <input
            type="checkbox"
            checked={mapping.required}
            onChange={(e) => onUpdate(index, 'required', e.target.checked)}
            className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
          />
          <span className="ml-2 text-sm font-medium text-gray-700">Required</span>
        </label>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Transform</label>
          <input
            type="text"
            value={mapping.transformation || ''}
            onChange={(e) => onUpdate(index, 'transformation', e.target.value)}
            placeholder="e.g., uppercase"
            className="w-full px-2 py-1 text-sm border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
          />
        </div>
      </div>
    </div>
  </div>
);

// Provider-specific Settings Components
const SalesforceSettings: React.FC<{
  settings: CRMSettings;
  onUpdate: (settings: CRMSettings) => void;
}> = ({ settings, onUpdate }) => (
  <div className="bg-gray-50 p-4 rounded-lg">
    <h4 className="text-sm font-medium text-gray-900 mb-3">Salesforce Settings</h4>
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Salesforce Instance URL
        </label>
        <input
          type="url"
          placeholder="https://your-org.salesforce.com"
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          API Version
        </label>
        <select className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500">
          <option value="v58.0">v58.0 (Latest)</option>
          <option value="v57.0">v57.0</option>
          <option value="v56.0">v56.0</option>
        </select>
      </div>
    </div>
  </div>
);

const HubSpotSettings: React.FC<{
  settings: CRMSettings;
  onUpdate: (settings: CRMSettings) => void;
}> = ({ settings, onUpdate }) => (
  <div className="bg-gray-50 p-4 rounded-lg">
    <h4 className="text-sm font-medium text-gray-900 mb-3">HubSpot Settings</h4>
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          HubSpot API Key
        </label>
        <input
          type="password"
          placeholder="Enter your HubSpot API key"
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Portal ID
        </label>
        <input
          type="text"
          placeholder="Your HubSpot Portal ID"
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>
    </div>
  </div>
);

const PipedriveSettings: React.FC<{
  settings: CRMSettings;
  onUpdate: (settings: CRMSettings) => void;
}> = ({ settings, onUpdate }) => (
  <div className="bg-gray-50 p-4 rounded-lg">
    <h4 className="text-sm font-medium text-gray-900 mb-3">Pipedrive Settings</h4>
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Pipedrive API Token
        </label>
        <input
          type="password"
          placeholder="Enter your Pipedrive API token"
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Company Domain
        </label>
        <input
          type="text"
          placeholder="your-company"
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
        <p className="text-xs text-gray-500 mt-1">
          Your Pipedrive company domain (will be used as: your-company.pipedrive.com)
        </p>
      </div>
    </div>
  </div>
);

// Helper Components
const LoadingSpinner: React.FC = () => (
  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
);

const TrashIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
  </svg>
);

export default CRMConfiguration;