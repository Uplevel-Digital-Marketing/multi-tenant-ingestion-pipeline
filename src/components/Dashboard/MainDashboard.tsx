// Main Dashboard Component
// Location: ./src/components/Dashboard/MainDashboard.tsx

import React, { useState, useEffect, useMemo } from 'react';
import { MetricsOverview } from './MetricsOverview';
import { RequestsMonitor } from '../Calls/CallProcessingMonitor';
import { TenantManager } from '../Tenants/TenantManager';
import { AnalyticsDashboard } from '../Analytics/AnalyticsDashboard';
import { useRealtimeMetrics } from '@/hooks/useRealtimeMetrics';
import { apiService } from '@/services/api';
import { TenantConfig } from '@/types/api';

interface MainDashboardProps {
  initialTenantId?: string;
}

type DashboardView = 'overview' | 'requests' | 'tenants' | 'analytics' | 'settings';

export const MainDashboard: React.FC<MainDashboardProps> = ({ initialTenantId }) => {
  const [currentView, setCurrentView] = useState<DashboardView>('overview');
  const [selectedTenant, setSelectedTenant] = useState<string>(initialTenantId || '');
  const [timeRange, setTimeRange] = useState<'hour' | 'day' | 'week' | 'month'>('day');
  const [tenants, setTenants] = useState<TenantConfig[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Set API tenant context when selected tenant changes
  useEffect(() => {
    if (selectedTenant) {
      apiService.setTenant(selectedTenant);
    }
  }, [selectedTenant]);

  // Fetch available tenants
  useEffect(() => {
    const fetchTenants = async () => {
      try {
        const tenantsData = await apiService.getTenants();
        setTenants(tenantsData);

        // Set first tenant as default if none selected
        if (!selectedTenant && tenantsData.length > 0) {
          setSelectedTenant(tenantsData[0].id);
        }
      } catch (error) {
        console.error('Failed to fetch tenants:', error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchTenants();
  }, [selectedTenant]);

  // Get current tenant info
  const currentTenant = useMemo(() => {
    return tenants.find(t => t.id === selectedTenant);
  }, [tenants, selectedTenant]);

  if (isLoading) {
    return <DashboardLoadingSkeleton />;
  }

  if (tenants.length === 0) {
    return <NoTenantsState />;
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-4">
              <h1 className="text-xl font-semibold text-gray-900">
                Multi-Tenant Pipeline Dashboard
              </h1>
              <div className="text-sm text-gray-500">
                {currentTenant?.name}
              </div>
            </div>

            {/* Tenant Selector */}
            <div className="flex items-center space-x-4">
              <select
                value={selectedTenant}
                onChange={(e) => setSelectedTenant(e.target.value)}
                className="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                {tenants.map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>
                    {tenant.name} ({tenant.status})
                  </option>
                ))}
              </select>

              <button
                onClick={() => window.location.reload()}
                className="p-2 text-gray-600 hover:text-gray-900 transition-colors"
                title="Refresh"
              >
                <RefreshIcon className="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex space-x-8">
            {navigationItems.map((item) => (
              <button
                key={item.id}
                onClick={() => setCurrentView(item.id)}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  currentView === item.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                <div className="flex items-center space-x-2">
                  <item.icon className="w-5 h-5" />
                  <span>{item.label}</span>
                </div>
              </button>
            ))}
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {currentView === 'overview' && (
          <MetricsOverview
            tenantId={selectedTenant}
            timeRange={timeRange}
            onTimeRangeChange={setTimeRange}
          />
        )}

        {currentView === 'requests' && (
          <RequestsMonitor
            tenantId={selectedTenant}
            timeRange={timeRange}
          />
        )}

        {currentView === 'tenants' && (
          <TenantManager
            selectedTenant={selectedTenant}
            tenants={tenants}
            onTenantUpdate={(updatedTenant) => {
              setTenants(prev => prev.map(t => t.id === updatedTenant.id ? updatedTenant : t));
            }}
          />
        )}

        {currentView === 'analytics' && (
          <AnalyticsDashboard
            tenantId={selectedTenant}
            timeRange={timeRange}
            onTimeRangeChange={setTimeRange}
          />
        )}

        {currentView === 'settings' && (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-medium text-gray-900 mb-4">Dashboard Settings</h2>
            <p className="text-gray-600">Settings panel coming soon...</p>
          </div>
        )}
      </main>
    </div>
  );
};

// Navigation configuration
const navigationItems = [
  { id: 'overview' as DashboardView, label: 'Overview', icon: DashboardIcon },
  { id: 'requests' as DashboardView, label: 'Live Requests', icon: RequestsIcon },
  { id: 'tenants' as DashboardView, label: 'Tenant Management', icon: TenantsIcon },
  { id: 'analytics' as DashboardView, label: 'Analytics', icon: AnalyticsIcon },
  { id: 'settings' as DashboardView, label: 'Settings', icon: SettingsIcon },
];

// Loading and Error States
const DashboardLoadingSkeleton: React.FC = () => (
  <div className="min-h-screen bg-gray-50">
    <div className="animate-pulse">
      {/* Header skeleton */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="h-6 bg-gray-200 rounded w-64"></div>
            <div className="h-8 bg-gray-200 rounded w-32"></div>
          </div>
        </div>
      </div>

      {/* Navigation skeleton */}
      <div className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex space-x-8">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="py-4">
                <div className="h-5 bg-gray-200 rounded w-20"></div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Content skeleton */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="bg-white p-6 rounded-lg shadow">
              <div className="h-12 bg-gray-200 rounded w-12 mb-4"></div>
              <div className="h-4 bg-gray-200 rounded w-24 mb-2"></div>
              <div className="h-8 bg-gray-200 rounded w-16"></div>
            </div>
          ))}
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <div className="h-64 bg-gray-200 rounded"></div>
        </div>
      </div>
    </div>
  </div>
);

const NoTenantsState: React.FC = () => (
  <div className="min-h-screen bg-gray-50 flex items-center justify-center">
    <div className="text-center">
      <div className="text-6xl mb-4">🏢</div>
      <h2 className="text-xl font-medium text-gray-900 mb-2">No Tenants Found</h2>
      <p className="text-gray-600 mb-6">
        No tenant configurations are available. Please contact your administrator.
      </p>
      <button
        onClick={() => window.location.reload()}
        className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
      >
        Refresh
      </button>
    </div>
  </div>
);

// Icon components (replace with your preferred icon library)
const RefreshIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
  </svg>
);

const DashboardIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
  </svg>
);

const RequestsIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
  </svg>
);

const TenantsIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
  </svg>
);

const AnalyticsIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
  </svg>
);

const SettingsIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
  </svg>
);

export default MainDashboard;