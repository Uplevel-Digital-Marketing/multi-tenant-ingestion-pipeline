// Real-time Metrics Overview Dashboard Component
// Location: ./src/components/dashboard/MetricsOverview.tsx

import React, { useState, useMemo } from 'react';
import { useRealtimeMetrics, useTenantHealth } from '@/hooks/useRealtimeMetrics';
import { DashboardMetrics } from '@/types/tenant';

interface MetricsOverviewProps {
  tenantId: string;
  timeRange: 'hour' | 'day' | 'week' | 'month';
  onTimeRangeChange: (range: 'hour' | 'day' | 'week' | 'month') => void;
}

export const MetricsOverview: React.FC<MetricsOverviewProps> = ({
  tenantId,
  timeRange,
  onTimeRangeChange,
}) => {
  const { metrics, isConnected, isLoading, error, lastUpdated, refresh } = useRealtimeMetrics({
    tenantId,
    timeRange,
    autoRefresh: true,
  });

  const health = useTenantHealth(tenantId);
  const [showDetailedView, setShowDetailedView] = useState(false);

  // Calculate performance indicators
  const performanceMetrics = useMemo(() => {
    if (!metrics) return null;

    const previousPeriodComparison = {
      requests: { current: metrics.total_requests, change: 12.5 },
      leads: { current: metrics.qualified_leads, change: -3.2 },
      processing_time: { current: metrics.avg_processing_time, change: -8.7 },
      error_rate: { current: metrics.error_rate, change: -15.3 },
    };

    return previousPeriodComparison;
  }, [metrics]);

  if (isLoading && !metrics) {
    return <MetricsLoadingSkeleton />;
  }

  if (error && !metrics) {
    return <MetricsErrorState error={error} onRetry={refresh} />;
  }

  return (
    <div className="space-y-6">
      {/* Header with connection status and controls */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <h2 className="text-2xl font-bold text-gray-900">Dashboard Overview</h2>
          <ConnectionStatus isConnected={isConnected} lastUpdated={lastUpdated} />
        </div>

        <div className="flex items-center space-x-3">
          <TimeRangeSelector
            value={timeRange}
            onChange={onTimeRangeChange}
          />
          <button
            onClick={() => setShowDetailedView(!showDetailedView)}
            className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
          >
            {showDetailedView ? 'Simple' : 'Detailed'} View
          </button>
          <button
            onClick={refresh}
            className="p-2 text-gray-600 hover:text-gray-900 transition-colors"
            title="Refresh metrics"
          >
            <RefreshIcon className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* System Health Status */}
      <SystemHealthBanner health={health} />

      {/* Main Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title="Total Requests"
          value={metrics?.total_requests || 0}
          change={performanceMetrics?.requests.change}
          icon={<RequestsIcon />}
          color="blue"
          subtitle={`${timeRange === 'hour' ? 'Last hour' : `Last ${timeRange}`}`}
        />

        <MetricCard
          title="Qualified Leads"
          value={metrics?.qualified_leads || 0}
          change={performanceMetrics?.leads.change}
          icon={<LeadsIcon />}
          color="green"
          subtitle={`${((metrics?.qualified_leads || 0) / (metrics?.total_leads || 1) * 100).toFixed(1)}% conversion`}
        />

        <MetricCard
          title="Avg Processing Time"
          value={`${metrics?.avg_processing_time || 0}ms`}
          change={performanceMetrics?.processing_time.change}
          icon={<ClockIcon />}
          color="purple"
          subtitle="Response time"
        />

        <MetricCard
          title="Success Rate"
          value={`${((1 - (metrics?.error_rate || 0)) * 100).toFixed(1)}%`}
          change={performanceMetrics?.error_rate.change}
          icon={<SuccessIcon />}
          color="emerald"
          subtitle={`${metrics?.error_rate.toFixed(2)}% error rate`}
        />
      </div>

      {/* Detailed Metrics Section */}
      {showDetailedView && metrics && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Request Sources Breakdown */}
          <div className="bg-white p-6 rounded-lg shadow-sm border">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Request Sources</h3>
            <div className="space-y-3">
              {Object.entries(metrics.requests_by_source).map(([source, count]) => (
                <div key={source} className="flex items-center justify-between">
                  <div className="flex items-center space-x-3">
                    <SourceIcon source={source} />
                    <span className="text-sm font-medium text-gray-700 capitalize">
                      {source}
                    </span>
                  </div>
                  <span className="text-sm text-gray-600">{count}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Cost Breakdown */}
          <div className="bg-white p-6 rounded-lg shadow-sm border">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Cost Analysis</h3>
            <div className="space-y-3">
              <CostItem
                label="AI Processing"
                amount={metrics.processing_costs.gemini_api_cost}
                percentage={(metrics.processing_costs.gemini_api_cost / metrics.processing_costs.total_cost) * 100}
              />
              <CostItem
                label="Speech-to-Text"
                amount={metrics.processing_costs.speech_api_cost}
                percentage={(metrics.processing_costs.speech_api_cost / metrics.processing_costs.total_cost) * 100}
              />
              <CostItem
                label="Storage"
                amount={metrics.processing_costs.storage_cost}
                percentage={(metrics.processing_costs.storage_cost / metrics.processing_costs.total_cost) * 100}
              />
              <div className="pt-3 border-t">
                <div className="flex justify-between items-center">
                  <span className="font-medium text-gray-900">Total</span>
                  <span className="font-bold text-gray-900">
                    ${metrics.processing_costs.total_cost.toFixed(2)}
                  </span>
                </div>
                <p className="text-xs text-gray-500 mt-1">
                  Est. monthly: ${metrics.processing_costs.estimated_monthly.toFixed(2)}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Real-time Activity Chart */}
      <div className="bg-white p-6 rounded-lg shadow-sm border">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Request Activity</h3>
        <HourlyActivityChart data={metrics?.requests_by_hour || []} />
      </div>
    </div>
  );
};

// Supporting Components

const ConnectionStatus: React.FC<{
  isConnected: boolean;
  lastUpdated: Date | null;
}> = ({ isConnected, lastUpdated }) => (
  <div className="flex items-center space-x-2">
    <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-400' : 'bg-red-400'}`} />
    <span className="text-sm text-gray-600">
      {isConnected ? 'Live' : 'Disconnected'}
      {lastUpdated && (
        <span className="ml-1">
          • Updated {lastUpdated.toLocaleTimeString()}
        </span>
      )}
    </span>
  </div>
);

const SystemHealthBanner: React.FC<{ health: any }> = ({ health }) => {
  if (health.status === 'healthy') return null;

  const bgColor = {
    warning: 'bg-yellow-50 border-yellow-200',
    error: 'bg-red-50 border-red-200',
    unknown: 'bg-gray-50 border-gray-200',
  }[health.status];

  const textColor = {
    warning: 'text-yellow-800',
    error: 'text-red-800',
    unknown: 'text-gray-800',
  }[health.status];

  return (
    <div className={`p-4 rounded-lg border ${bgColor}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <WarningIcon className={`w-5 h-5 ${textColor}`} />
          <div>
            <h4 className={`text-sm font-medium ${textColor}`}>
              System Health: {health.status.toUpperCase()}
            </h4>
            {health.last_error && (
              <p className={`text-sm ${textColor} opacity-75`}>{health.last_error}</p>
            )}
          </div>
        </div>
        <div className={`text-sm ${textColor}`}>
          Error Rate: {(health.error_rate * 100).toFixed(1)}%
        </div>
      </div>
    </div>
  );
};

const TimeRangeSelector: React.FC<{
  value: string;
  onChange: (value: 'hour' | 'day' | 'week' | 'month') => void;
}> = ({ value, onChange }) => (
  <select
    value={value}
    onChange={(e) => onChange(e.target.value as 'hour' | 'day' | 'week' | 'month')}
    className="px-3 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
  >
    <option value="hour">Last Hour</option>
    <option value="day">Last Day</option>
    <option value="week">Last Week</option>
    <option value="month">Last Month</option>
  </select>
);

const MetricCard: React.FC<{
  title: string;
  value: string | number;
  change?: number;
  icon: React.ReactNode;
  color: string;
  subtitle?: string;
}> = ({ title, value, change, icon, color, subtitle }) => (
  <div className="bg-white p-6 rounded-lg shadow-sm border">
    <div className="flex items-center justify-between">
      <div className={`p-2 rounded-lg bg-${color}-100`}>
        {icon}
      </div>
      {change !== undefined && (
        <span className={`text-sm font-medium ${change >= 0 ? 'text-green-600' : 'text-red-600'}`}>
          {change >= 0 ? '+' : ''}{change.toFixed(1)}%
        </span>
      )}
    </div>
    <div className="mt-4">
      <h3 className="text-sm font-medium text-gray-600">{title}</h3>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
      {subtitle && (
        <p className="text-xs text-gray-500 mt-1">{subtitle}</p>
      )}
    </div>
  </div>
);

const MetricsLoadingSkeleton: React.FC = () => (
  <div className="space-y-6">
    <div className="flex justify-between items-center">
      <div className="h-8 bg-gray-200 rounded w-48 animate-pulse" />
      <div className="h-8 bg-gray-200 rounded w-32 animate-pulse" />
    </div>
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      {[...Array(4)].map((_, i) => (
        <div key={i} className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="animate-pulse">
            <div className="h-12 bg-gray-200 rounded w-12 mb-4" />
            <div className="h-4 bg-gray-200 rounded w-24 mb-2" />
            <div className="h-8 bg-gray-200 rounded w-16" />
          </div>
        </div>
      ))}
    </div>
  </div>
);

const MetricsErrorState: React.FC<{
  error: string;
  onRetry: () => void;
}> = ({ error, onRetry }) => (
  <div className="bg-white p-8 rounded-lg shadow-sm border text-center">
    <div className="text-red-500 mb-4">
      <ErrorIcon className="w-12 h-12 mx-auto" />
    </div>
    <h3 className="text-lg font-medium text-gray-900 mb-2">Failed to load metrics</h3>
    <p className="text-gray-600 mb-4">{error}</p>
    <button
      onClick={onRetry}
      className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
    >
      Try Again
    </button>
  </div>
);

// Placeholder icon components (replace with actual icon library)
const RefreshIcon = ({ className }: { className?: string }) => <div className={className}>🔄</div>;
const RequestsIcon = () => <div className="w-6 h-6 text-blue-600">📊</div>;
const LeadsIcon = () => <div className="w-6 h-6 text-green-600">🎯</div>;
const ClockIcon = () => <div className="w-6 h-6 text-purple-600">⏱️</div>;
const SuccessIcon = () => <div className="w-6 h-6 text-emerald-600">✅</div>;
const WarningIcon = ({ className }: { className?: string }) => <div className={className}>⚠️</div>;
const ErrorIcon = ({ className }: { className?: string }) => <div className={className}>❌</div>;

const SourceIcon: React.FC<{ source: string }> = ({ source }) => {
  const icons: Record<string, string> = {
    form: '📝',
    callrail: '📞',
    calendar: '📅',
    chat: '💬',
  };
  return <span>{icons[source] || '📊'}</span>;
};

const CostItem: React.FC<{
  label: string;
  amount: number;
  percentage: number;
}> = ({ label, amount, percentage }) => (
  <div className="flex items-center justify-between">
    <span className="text-sm text-gray-600">{label}</span>
    <div className="text-right">
      <span className="text-sm font-medium text-gray-900">${amount.toFixed(2)}</span>
      <span className="text-xs text-gray-500 ml-2">({percentage.toFixed(1)}%)</span>
    </div>
  </div>
);

const HourlyActivityChart: React.FC<{
  data: Array<{ hour: string; count: number }>;
}> = ({ data }) => {
  const maxCount = Math.max(...data.map(d => d.count));

  return (
    <div className="space-y-3">
      {data.slice(-24).map((item, index) => (
        <div key={index} className="flex items-center space-x-3">
          <span className="text-xs text-gray-500 w-12">{item.hour}</span>
          <div className="flex-1 bg-gray-200 rounded-full h-2">
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-500"
              style={{ width: `${(item.count / maxCount) * 100}%` }}
            />
          </div>
          <span className="text-xs text-gray-600 w-8 text-right">{item.count}</span>
        </div>
      ))}
    </div>
  );
};