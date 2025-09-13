// Analytics Dashboard Component
// Location: ./src/components/Analytics/AnalyticsDashboard.tsx

import React, { useState, useMemo } from 'react';
import { useRealtimeMetrics } from '@/hooks/useRealtimeMetrics';
import { DashboardMetrics } from '@/types/api';

interface AnalyticsDashboardProps {
  tenantId: string;
  timeRange: 'hour' | 'day' | 'week' | 'month';
  onTimeRangeChange: (range: 'hour' | 'day' | 'week' | 'month') => void;
}

export const AnalyticsDashboard: React.FC<AnalyticsDashboardProps> = ({
  tenantId,
  timeRange,
  onTimeRangeChange,
}) => {
  const [selectedChart, setSelectedChart] = useState<string>('overview');

  const { metrics, isLoading, error, refresh } = useRealtimeMetrics({
    tenantId,
    timeRange,
    autoRefresh: false, // Disable auto-refresh for analytics view
  });

  // Calculate analytics data
  const analyticsData = useMemo(() => {
    if (!metrics) return null;

    const conversionRate = metrics.total_leads > 0
      ? (metrics.qualified_leads / metrics.total_leads) * 100
      : 0;

    const costPerLead = metrics.qualified_leads > 0
      ? metrics.processing_costs.total_cost / metrics.qualified_leads
      : 0;

    const callSuccessRate = metrics.total_calls > 0
      ? ((metrics.total_calls - metrics.spam_calls_blocked) / metrics.total_calls) * 100
      : 0;

    return {
      conversionRate,
      costPerLead,
      callSuccessRate,
      metrics,
    };
  }, [metrics]);

  if (isLoading) {
    return <AnalyticsLoadingSkeleton />;
  }

  if (error || !analyticsData) {
    return <AnalyticsErrorState error={error || 'No data available'} onRetry={refresh} />;
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">Analytics Dashboard</h2>
          <p className="text-gray-600">Detailed insights and performance metrics</p>
        </div>

        <div className="flex items-center space-x-4">
          <TimeRangeSelector
            value={timeRange}
            onChange={onTimeRangeChange}
          />
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
          >
            Refresh Data
          </button>
        </div>
      </div>

      {/* Key Performance Indicators */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <KPICard
          title="Conversion Rate"
          value={`${analyticsData.conversionRate.toFixed(1)}%`}
          change={15.3}
          icon="📈"
          color="green"
        />
        <KPICard
          title="Cost per Lead"
          value={`$${analyticsData.costPerLead.toFixed(2)}`}
          change={-8.7}
          icon="💰"
          color="blue"
        />
        <KPICard
          title="Call Success Rate"
          value={`${analyticsData.callSuccessRate.toFixed(1)}%`}
          change={4.2}
          icon="📞"
          color="purple"
        />
        <KPICard
          title="Avg Lead Score"
          value={analyticsData.metrics.avg_lead_score.toFixed(0)}
          change={2.8}
          icon="🎯"
          color="orange"
        />
      </div>

      {/* Chart Selection */}
      <div className="bg-white rounded-lg shadow">
        <div className="border-b border-gray-200">
          <nav className="-mb-px flex space-x-8 px-6">
            {chartTabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setSelectedChart(tab.id)}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  selectedChart === tab.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>
        </div>

        <div className="p-6">
          {selectedChart === 'overview' && (
            <OverviewCharts metrics={analyticsData.metrics} />
          )}
          {selectedChart === 'requests' && (
            <RequestsAnalytics metrics={analyticsData.metrics} />
          )}
          {selectedChart === 'leads' && (
            <LeadsAnalytics metrics={analyticsData.metrics} />
          )}
          {selectedChart === 'costs' && (
            <CostAnalytics metrics={analyticsData.metrics} />
          )}
          {selectedChart === 'performance' && (
            <PerformanceAnalytics metrics={analyticsData.metrics} />
          )}
        </div>
      </div>

      {/* Insights Panel */}
      <InsightsPanel metrics={analyticsData.metrics} />
    </div>
  );
};

// Chart tabs configuration
const chartTabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'requests', label: 'Request Volume' },
  { id: 'leads', label: 'Lead Analysis' },
  { id: 'costs', label: 'Cost Breakdown' },
  { id: 'performance', label: 'Performance' },
];

// KPI Card Component
const KPICard: React.FC<{
  title: string;
  value: string;
  change: number;
  icon: string;
  color: string;
}> = ({ title, value, change, icon, color }) => (
  <div className="bg-white p-6 rounded-lg shadow border">
    <div className="flex items-center justify-between mb-4">
      <div className={`p-3 rounded-lg bg-${color}-100`}>
        <span className="text-2xl">{icon}</span>
      </div>
      <div className={`text-sm font-medium ${change >= 0 ? 'text-green-600' : 'text-red-600'}`}>
        {change >= 0 ? '+' : ''}{change.toFixed(1)}%
      </div>
    </div>
    <div>
      <h3 className="text-sm font-medium text-gray-600 mb-1">{title}</h3>
      <p className="text-2xl font-bold text-gray-900">{value}</p>
    </div>
  </div>
);

// Overview Charts Component
const OverviewCharts: React.FC<{ metrics: DashboardMetrics }> = ({ metrics }) => (
  <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
    {/* Request Volume Chart */}
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Request Volume by Hour</h3>
      <div className="h-64 bg-gray-50 rounded-lg flex items-center justify-center">
        <SimpleBarChart data={metrics.requests_by_hour} />
      </div>
    </div>

    {/* Source Distribution */}
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Requests by Source</h3>
      <div className="h-64 bg-gray-50 rounded-lg p-4">
        <SourceDistribution data={metrics.requests_by_source} />
      </div>
    </div>
  </div>
);

// Requests Analytics Component
const RequestsAnalytics: React.FC<{ metrics: DashboardMetrics }> = ({ metrics }) => (
  <div className="space-y-6">
    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Request Volume Trends</h3>
      <div className="h-80 bg-gray-50 rounded-lg flex items-center justify-center">
        <RequestVolumeChart data={metrics.requests_by_hour} />
      </div>
    </div>

    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div className="bg-gray-50 p-4 rounded-lg">
        <div className="text-sm font-medium text-gray-600">Peak Hour</div>
        <div className="text-xl font-bold text-gray-900">2:00 PM</div>
      </div>
      <div className="bg-gray-50 p-4 rounded-lg">
        <div className="text-sm font-medium text-gray-600">Avg Response Time</div>
        <div className="text-xl font-bold text-gray-900">{metrics.avg_processing_time}ms</div>
      </div>
      <div className="bg-gray-50 p-4 rounded-lg">
        <div className="text-sm font-medium text-gray-600">Success Rate</div>
        <div className="text-xl font-bold text-gray-900">{((1 - metrics.error_rate) * 100).toFixed(1)}%</div>
      </div>
    </div>
  </div>
);

// Leads Analytics Component
const LeadsAnalytics: React.FC<{ metrics: DashboardMetrics }> = ({ metrics }) => (
  <div className="space-y-6">
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Lead Quality Distribution</h3>
        <div className="h-64 bg-gray-50 rounded-lg flex items-center justify-center">
          <LeadScoreDistribution avgScore={metrics.avg_lead_score} />
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Conversion Funnel</h3>
        <div className="h-64 bg-gray-50 rounded-lg p-4">
          <ConversionFunnel
            totalRequests={metrics.total_requests}
            totalLeads={metrics.total_leads}
            qualifiedLeads={metrics.qualified_leads}
          />
        </div>
      </div>
    </div>

    <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
      <h4 className="text-sm font-medium text-blue-900 mb-2">Lead Insights</h4>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm text-blue-800">
        <div>
          <strong>Conversion Rate:</strong> {((metrics.qualified_leads / metrics.total_leads) * 100).toFixed(1)}%
        </div>
        <div>
          <strong>Avg Lead Score:</strong> {metrics.avg_lead_score.toFixed(1)}/100
        </div>
        <div>
          <strong>Lead Velocity:</strong> {(metrics.qualified_leads / 24).toFixed(1)}/hour
        </div>
      </div>
    </div>
  </div>
);

// Cost Analytics Component
const CostAnalytics: React.FC<{ metrics: DashboardMetrics }> = ({ metrics }) => (
  <div className="space-y-6">
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Cost Breakdown</h3>
        <div className="space-y-3">
          {Object.entries({
            'AI Processing': metrics.processing_costs.gemini_api_cost,
            'Speech-to-Text': metrics.processing_costs.speech_api_cost,
            'Storage': metrics.processing_costs.storage_cost,
            'Compute': metrics.processing_costs.compute_cost,
          }).map(([service, cost]) => (
            <div key={service} className="flex items-center justify-between">
              <span className="text-sm text-gray-600">{service}</span>
              <div className="flex items-center space-x-3">
                <div className="flex-1 bg-gray-200 rounded-full h-2 w-32">
                  <div
                    className="bg-blue-500 h-2 rounded-full"
                    style={{ width: `${(cost / metrics.processing_costs.total_cost) * 100}%` }}
                  />
                </div>
                <span className="text-sm font-medium text-gray-900">${cost.toFixed(2)}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Cost Projections</h3>
        <div className="space-y-4">
          <div className="bg-gray-50 p-4 rounded-lg">
            <div className="text-sm font-medium text-gray-600">Current Monthly Estimate</div>
            <div className="text-2xl font-bold text-gray-900">
              ${metrics.processing_costs.estimated_monthly.toFixed(2)}
            </div>
          </div>
          <div className="bg-gray-50 p-4 rounded-lg">
            <div className="text-sm font-medium text-gray-600">Cost per Request</div>
            <div className="text-2xl font-bold text-gray-900">
              ${(metrics.processing_costs.total_cost / metrics.total_requests).toFixed(3)}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
);

// Performance Analytics Component
const PerformanceAnalytics: React.FC<{ metrics: DashboardMetrics }> = ({ metrics }) => (
  <div className="space-y-6">
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <div className="bg-gray-50 p-4 rounded-lg">
        <div className="text-sm font-medium text-gray-600">Avg Processing Time</div>
        <div className="text-xl font-bold text-gray-900">{metrics.avg_processing_time}ms</div>
        <div className="text-xs text-green-600 mt-1">↓ 12% from last period</div>
      </div>
      <div className="bg-gray-50 p-4 rounded-lg">
        <div className="text-sm font-medium text-gray-600">Success Rate</div>
        <div className="text-xl font-bold text-gray-900">{((1 - metrics.error_rate) * 100).toFixed(1)}%</div>
        <div className="text-xs text-green-600 mt-1">↑ 3% from last period</div>
      </div>
      <div className="bg-gray-50 p-4 rounded-lg">
        <div className="text-sm font-medium text-gray-600">Transcription Accuracy</div>
        <div className="text-xl font-bold text-gray-900">{(metrics.transcription_accuracy * 100).toFixed(1)}%</div>
        <div className="text-xs text-green-600 mt-1">↑ 1.2% from last period</div>
      </div>
      <div className="bg-gray-50 p-4 rounded-lg">
        <div className="text-sm font-medium text-gray-600">Queue Depth</div>
        <div className="text-xl font-bold text-gray-900">{metrics.current_processing_queue}</div>
        <div className="text-xs text-gray-500 mt-1">Current backlog</div>
      </div>
    </div>

    <div>
      <h3 className="text-lg font-semibold text-gray-900 mb-4">System Performance Over Time</h3>
      <div className="h-64 bg-gray-50 rounded-lg flex items-center justify-center">
        <div className="text-gray-500">Performance trend chart would go here</div>
      </div>
    </div>
  </div>
);

// Insights Panel Component
const InsightsPanel: React.FC<{ metrics: DashboardMetrics }> = ({ metrics }) => {
  const insights = useMemo(() => {
    const insights = [];

    if (metrics.conversion_rate > 0.8) {
      insights.push({
        type: 'positive',
        title: 'High Conversion Rate',
        description: 'Your lead conversion rate is excellent at ' + (metrics.conversion_rate * 100).toFixed(1) + '%',
      });
    }

    if (metrics.avg_processing_time > 2000) {
      insights.push({
        type: 'warning',
        title: 'Processing Time Alert',
        description: 'Average processing time is higher than recommended. Consider optimizing workflows.',
      });
    }

    if (metrics.spam_calls_blocked / metrics.total_calls > 0.3) {
      insights.push({
        type: 'info',
        title: 'High Spam Detection',
        description: 'Spam detection is working well, blocking ' + Math.round((metrics.spam_calls_blocked / metrics.total_calls) * 100) + '% of calls.',
      });
    }

    return insights;
  }, [metrics]);

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Insights & Recommendations</h3>
      <div className="space-y-3">
        {insights.map((insight, index) => (
          <div
            key={index}
            className={`p-4 rounded-lg border ${
              insight.type === 'positive' ? 'bg-green-50 border-green-200' :
              insight.type === 'warning' ? 'bg-yellow-50 border-yellow-200' :
              'bg-blue-50 border-blue-200'
            }`}
          >
            <h4 className={`font-medium ${
              insight.type === 'positive' ? 'text-green-900' :
              insight.type === 'warning' ? 'text-yellow-900' :
              'text-blue-900'
            }`}>
              {insight.title}
            </h4>
            <p className={`text-sm mt-1 ${
              insight.type === 'positive' ? 'text-green-800' :
              insight.type === 'warning' ? 'text-yellow-800' :
              'text-blue-800'
            }`}>
              {insight.description}
            </p>
          </div>
        ))}

        {insights.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No significant insights to display. Keep monitoring your metrics!
          </div>
        )}
      </div>
    </div>
  );
};

// Simple Chart Components (replace with actual charting library)
const SimpleBarChart: React.FC<{ data: Array<{ hour: string; count: number }> }> = ({ data }) => {
  const maxCount = Math.max(...data.map(d => d.count));

  return (
    <div className="w-full h-full p-4">
      <div className="flex items-end h-full space-x-1">
        {data.slice(-12).map((item, index) => (
          <div key={index} className="flex-1 flex flex-col items-center">
            <div
              className="w-full bg-blue-500 rounded-t min-h-1"
              style={{ height: `${(item.count / maxCount) * 100}%` }}
            />
            <div className="text-xs text-gray-500 mt-2 transform -rotate-45">
              {item.hour}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

const RequestVolumeChart: React.FC<{ data: Array<{ hour: string; count: number }> }> = ({ data }) => (
  <div className="w-full h-full p-4 flex items-center justify-center">
    <div className="text-gray-500">Advanced line chart would be implemented here with a charting library</div>
  </div>
);

const SourceDistribution: React.FC<{ data: Record<string, number> }> = ({ data }) => {
  const total = Object.values(data).reduce((sum, count) => sum + count, 0);

  return (
    <div className="space-y-3">
      {Object.entries(data).map(([source, count]) => (
        <div key={source} className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-4 h-4 bg-blue-500 rounded"></div>
            <span className="text-sm font-medium text-gray-700 capitalize">{source}</span>
          </div>
          <div className="text-right">
            <div className="text-sm font-medium text-gray-900">{count}</div>
            <div className="text-xs text-gray-500">{((count / total) * 100).toFixed(1)}%</div>
          </div>
        </div>
      ))}
    </div>
  );
};

const LeadScoreDistribution: React.FC<{ avgScore: number }> = ({ avgScore }) => (
  <div className="w-full h-full p-4 flex items-center justify-center">
    <div className="text-center">
      <div className="text-4xl font-bold text-blue-600">{avgScore.toFixed(1)}</div>
      <div className="text-sm text-gray-500">Average Lead Score</div>
    </div>
  </div>
);

const ConversionFunnel: React.FC<{
  totalRequests: number;
  totalLeads: number;
  qualifiedLeads: number;
}> = ({ totalRequests, totalLeads, qualifiedLeads }) => (
  <div className="space-y-4">
    <div className="text-center">
      <div className="text-lg font-medium text-gray-900">Conversion Funnel</div>
    </div>

    <div className="space-y-2">
      <div className="bg-blue-100 p-3 rounded">
        <div className="flex justify-between">
          <span className="text-sm font-medium">Total Requests</span>
          <span className="text-sm">{totalRequests}</span>
        </div>
      </div>
      <div className="bg-blue-200 p-3 rounded ml-4">
        <div className="flex justify-between">
          <span className="text-sm font-medium">Leads Generated</span>
          <span className="text-sm">{totalLeads} ({((totalLeads / totalRequests) * 100).toFixed(1)}%)</span>
        </div>
      </div>
      <div className="bg-blue-300 p-3 rounded ml-8">
        <div className="flex justify-between">
          <span className="text-sm font-medium">Qualified Leads</span>
          <span className="text-sm">{qualifiedLeads} ({((qualifiedLeads / totalLeads) * 100).toFixed(1)}%)</span>
        </div>
      </div>
    </div>
  </div>
);

// Time Range Selector Component
const TimeRangeSelector: React.FC<{
  value: string;
  onChange: (value: 'hour' | 'day' | 'week' | 'month') => void;
}> = ({ value, onChange }) => (
  <select
    value={value}
    onChange={(e) => onChange(e.target.value as 'hour' | 'day' | 'week' | 'month')}
    className="px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
  >
    <option value="hour">Last Hour</option>
    <option value="day">Last 24 Hours</option>
    <option value="week">Last Week</option>
    <option value="month">Last Month</option>
  </select>
);

// Loading and Error States
const AnalyticsLoadingSkeleton: React.FC = () => (
  <div className="space-y-8">
    <div className="animate-pulse">
      <div className="flex justify-between items-center mb-8">
        <div>
          <div className="h-8 bg-gray-200 rounded w-64 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-48"></div>
        </div>
        <div className="h-10 bg-gray-200 rounded w-32"></div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="bg-white p-6 rounded-lg shadow">
            <div className="h-12 bg-gray-200 rounded w-12 mb-4"></div>
            <div className="h-4 bg-gray-200 rounded w-24 mb-2"></div>
            <div className="h-8 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <div className="h-64 bg-gray-200 rounded"></div>
      </div>
    </div>
  </div>
);

const AnalyticsErrorState: React.FC<{
  error: string;
  onRetry: () => void;
}> = ({ error, onRetry }) => (
  <div className="bg-white rounded-lg shadow p-8 text-center">
    <div className="text-red-500 mb-4">
      <ErrorIcon className="w-12 h-12 mx-auto" />
    </div>
    <h3 className="text-lg font-medium text-gray-900 mb-2">Failed to load analytics</h3>
    <p className="text-gray-600 mb-4">{error}</p>
    <button
      onClick={onRetry}
      className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
    >
      Try Again
    </button>
  </div>
);

const ErrorIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="currentColor" viewBox="0 0 20 20">
    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
  </svg>
);

export default AnalyticsDashboard;