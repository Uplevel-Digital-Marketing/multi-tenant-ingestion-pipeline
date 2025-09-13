// Call Processing Monitor Component
// Location: ./src/components/Calls/CallProcessingMonitor.tsx

import React, { useState, useMemo, useCallback } from 'react';
import { useCallProcessing, useRequestDetails } from '@/hooks/useCallProcessing';
import { ProcessingRequest } from '@/types/api';

interface CallProcessingMonitorProps {
  tenantId: string;
  timeRange: 'hour' | 'day' | 'week' | 'month';
}

type ViewMode = 'list' | 'grid' | 'timeline';

export const RequestsMonitor: React.FC<CallProcessingMonitorProps> = ({
  tenantId,
  timeRange,
}) => {
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [filters, setFilters] = useState({
    source: [] as string[],
    status: [] as string[],
    search: '',
  });
  const [selectedRequest, setSelectedRequest] = useState<string | null>(null);

  const {
    requests,
    isLoading,
    error,
    isConnected,
    refresh,
    loadMore,
    hasNextPage,
  } = useCallProcessing({
    tenantId,
    filters,
    realtime: true,
  });

  // Filter requests based on current filters
  const filteredRequests = useMemo(() => {
    return requests.filter(request => {
      // Source filter
      if (filters.source.length > 0 && !filters.source.includes(request.source)) {
        return false;
      }

      // Status filter
      if (filters.status.length > 0 && !filters.status.includes(request.status)) {
        return false;
      }

      // Search filter
      if (filters.search) {
        const searchTerm = filters.search.toLowerCase();
        return (
          request.ai_analysis?.customer_info?.name?.toLowerCase().includes(searchTerm) ||
          request.ai_analysis?.customer_info?.phone?.includes(searchTerm) ||
          request.transcription_data?.full_text?.toLowerCase().includes(searchTerm)
        );
      }

      return true;
    });
  }, [requests, filters]);

  const updateFilter = useCallback((key: keyof typeof filters, value: any) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  }, []);

  const toggleArrayFilter = useCallback((key: 'source' | 'status', value: string) => {
    setFilters(prev => {
      const currentArray = prev[key];
      const newArray = currentArray.includes(value)
        ? currentArray.filter(item => item !== value)
        : [...currentArray, value];
      return { ...prev, [key]: newArray };
    });
  }, []);

  if (error && requests.length === 0) {
    return <ErrorState error={error} onRetry={refresh} />;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <h2 className="text-2xl font-bold text-gray-900">Live Request Monitor</h2>
          <ConnectionIndicator isConnected={isConnected} />
        </div>

        <div className="flex items-center space-x-3">
          <ViewModeSelector viewMode={viewMode} onViewModeChange={setViewMode} />
          <button
            onClick={refresh}
            className="p-2 text-gray-600 hover:text-gray-900 transition-colors"
            title="Refresh requests"
          >
            <RefreshIcon className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Filters */}
      <FilterBar
        filters={filters}
        onUpdateFilter={updateFilter}
        onToggleArrayFilter={toggleArrayFilter}
      />

      {/* Statistics */}
      <RequestStatistics requests={filteredRequests} />

      {/* Request List/Grid */}
      {isLoading && requests.length === 0 ? (
        <LoadingSkeleton />
      ) : (
        <div className="space-y-6">
          {viewMode === 'list' && (
            <RequestsList
              requests={filteredRequests}
              onSelectRequest={setSelectedRequest}
            />
          )}

          {viewMode === 'grid' && (
            <RequestsGrid
              requests={filteredRequests}
              onSelectRequest={setSelectedRequest}
            />
          )}

          {viewMode === 'timeline' && (
            <RequestsTimeline
              requests={filteredRequests}
              onSelectRequest={setSelectedRequest}
            />
          )}

          {/* Load More */}
          {hasNextPage && (
            <div className="text-center">
              <button
                onClick={loadMore}
                disabled={isLoading}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
              >
                {isLoading ? 'Loading...' : 'Load More'}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Request Detail Modal */}
      {selectedRequest && (
        <RequestDetailModal
          requestId={selectedRequest}
          onClose={() => setSelectedRequest(null)}
        />
      )}
    </div>
  );
};

// Connection Indicator Component
const ConnectionIndicator: React.FC<{ isConnected: boolean }> = ({ isConnected }) => (
  <div className="flex items-center space-x-2">
    <div className={`w-3 h-3 rounded-full ${isConnected ? 'bg-green-400 animate-pulse' : 'bg-red-400'}`} />
    <span className="text-sm text-gray-600">
      {isConnected ? 'Live Updates' : 'Disconnected'}
    </span>
  </div>
);

// View Mode Selector Component
const ViewModeSelector: React.FC<{
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
}> = ({ viewMode, onViewModeChange }) => (
  <div className="flex rounded-lg border border-gray-300 overflow-hidden">
    {[
      { mode: 'list' as ViewMode, icon: ListIcon, label: 'List' },
      { mode: 'grid' as ViewMode, icon: GridIcon, label: 'Grid' },
      { mode: 'timeline' as ViewMode, icon: TimelineIcon, label: 'Timeline' },
    ].map(({ mode, icon: Icon, label }) => (
      <button
        key={mode}
        onClick={() => onViewModeChange(mode)}
        className={`px-3 py-2 text-sm font-medium transition-colors ${
          viewMode === mode
            ? 'bg-blue-600 text-white'
            : 'bg-white text-gray-700 hover:bg-gray-50'
        }`}
        title={label}
      >
        <Icon className="w-4 h-4" />
      </button>
    ))}
  </div>
);

// Filter Bar Component
const FilterBar: React.FC<{
  filters: any;
  onUpdateFilter: (key: string, value: any) => void;
  onToggleArrayFilter: (key: 'source' | 'status', value: string) => void;
}> = ({ filters, onUpdateFilter, onToggleArrayFilter }) => (
  <div className="bg-white p-4 rounded-lg shadow border">
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      {/* Search */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Search</label>
        <input
          type="text"
          value={filters.search}
          onChange={(e) => onUpdateFilter('search', e.target.value)}
          placeholder="Name, phone, or content..."
          className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
        />
      </div>

      {/* Source Filter */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Source</label>
        <div className="space-y-1">
          {['form', 'callrail', 'calendar', 'chat'].map((source) => (
            <label key={source} className="flex items-center">
              <input
                type="checkbox"
                checked={filters.source.includes(source)}
                onChange={() => onToggleArrayFilter('source', source)}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-700 capitalize">{source}</span>
            </label>
          ))}
        </div>
      </div>

      {/* Status Filter */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
        <div className="space-y-1">
          {['received', 'processing', 'completed', 'failed'].map((status) => (
            <label key={status} className="flex items-center">
              <input
                type="checkbox"
                checked={filters.status.includes(status)}
                onChange={() => onToggleArrayFilter('status', status)}
                className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-700 capitalize">{status}</span>
            </label>
          ))}
        </div>
      </div>

      {/* Clear Filters */}
      <div className="flex items-end">
        <button
          onClick={() => onUpdateFilter('', { source: [], status: [], search: '' })}
          className="px-3 py-2 text-sm text-gray-600 hover:text-gray-900 transition-colors"
        >
          Clear Filters
        </button>
      </div>
    </div>
  </div>
);

// Request Statistics Component
const RequestStatistics: React.FC<{ requests: ProcessingRequest[] }> = ({ requests }) => {
  const stats = useMemo(() => {
    const total = requests.length;
    const completed = requests.filter(r => r.status === 'completed').length;
    const processing = requests.filter(r => r.status === 'processing').length;
    const failed = requests.filter(r => r.status === 'failed').length;
    const avgLeadScore = requests.reduce((sum, r) => sum + (r.lead_score || 0), 0) / total || 0;

    return { total, completed, processing, failed, avgLeadScore };
  }, [requests]);

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
      <StatCard title="Total Requests" value={stats.total} color="blue" />
      <StatCard title="Completed" value={stats.completed} color="green" />
      <StatCard title="Processing" value={stats.processing} color="yellow" />
      <StatCard title="Failed" value={stats.failed} color="red" />
      <StatCard title="Avg Lead Score" value={Math.round(stats.avgLeadScore)} color="purple" />
    </div>
  );
};

const StatCard: React.FC<{
  title: string;
  value: number;
  color: string;
}> = ({ title, value, color }) => (
  <div className="bg-white p-4 rounded-lg shadow border">
    <div className="text-sm font-medium text-gray-600">{title}</div>
    <div className={`text-2xl font-bold text-${color}-600`}>{value}</div>
  </div>
);

// Requests List Component
const RequestsList: React.FC<{
  requests: ProcessingRequest[];
  onSelectRequest: (id: string) => void;
}> = ({ requests, onSelectRequest }) => (
  <div className="bg-white shadow rounded-lg overflow-hidden">
    <table className="min-w-full divide-y divide-gray-200">
      <thead className="bg-gray-50">
        <tr>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Request
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Customer
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Lead Score
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Status
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Time
          </th>
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {requests.map((request) => (
          <RequestRow
            key={request.id}
            request={request}
            onClick={() => onSelectRequest(request.id)}
          />
        ))}
      </tbody>
    </table>

    {requests.length === 0 && (
      <div className="text-center py-12">
        <div className="text-gray-500">No requests found matching your filters</div>
      </div>
    )}
  </div>
);

const RequestRow: React.FC<{
  request: ProcessingRequest;
  onClick: () => void;
}> = ({ request, onClick }) => (
  <tr
    className="hover:bg-gray-50 cursor-pointer"
    onClick={onClick}
  >
    <td className="px-6 py-4 whitespace-nowrap">
      <div className="flex items-center space-x-3">
        <SourceIcon source={request.source} />
        <div>
          <div className="text-sm font-medium text-gray-900">
            {request.communication_mode === 'phone' ? 'Phone Call' : request.source}
          </div>
          <div className="text-sm text-gray-500">{request.id.slice(-8)}</div>
        </div>
      </div>
    </td>
    <td className="px-6 py-4 whitespace-nowrap">
      <div className="text-sm font-medium text-gray-900">
        {request.ai_analysis?.customer_info?.name || 'Unknown'}
      </div>
      <div className="text-sm text-gray-500">
        {request.ai_analysis?.customer_info?.phone}
      </div>
    </td>
    <td className="px-6 py-4 whitespace-nowrap">
      <LeadScoreBadge score={request.lead_score || 0} />
    </td>
    <td className="px-6 py-4 whitespace-nowrap">
      <StatusBadge status={request.status} />
    </td>
    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
      {new Date(request.created_at).toLocaleTimeString()}
    </td>
  </tr>
);

// Request Detail Modal Component
const RequestDetailModal: React.FC<{
  requestId: string;
  onClose: () => void;
}> = ({ requestId, onClose }) => {
  const { request, isLoading, error } = useRequestDetails(requestId);

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl max-h-screen overflow-y-auto m-4">
        <div className="sticky top-0 bg-white border-b p-6 flex items-center justify-between">
          <h3 className="text-lg font-medium text-gray-900">Request Details</h3>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <CloseIcon className="w-6 h-6" />
          </button>
        </div>

        <div className="p-6">
          {isLoading && <div className="text-center py-8">Loading request details...</div>}
          {error && <div className="text-red-600 text-center py-8">{error}</div>}
          {request && <RequestDetailContent request={request} />}
        </div>
      </div>
    </div>
  );
};

const RequestDetailContent: React.FC<{ request: ProcessingRequest }> = ({ request }) => (
  <div className="space-y-6">
    {/* Request Overview */}
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div className="bg-gray-50 p-4 rounded-lg">
        <h4 className="font-medium text-gray-900 mb-2">Source</h4>
        <div className="flex items-center space-x-2">
          <SourceIcon source={request.source} />
          <span className="capitalize">{request.source}</span>
        </div>
      </div>
      <div className="bg-gray-50 p-4 rounded-lg">
        <h4 className="font-medium text-gray-900 mb-2">Status</h4>
        <StatusBadge status={request.status} />
      </div>
      <div className="bg-gray-50 p-4 rounded-lg">
        <h4 className="font-medium text-gray-900 mb-2">Lead Score</h4>
        <LeadScoreBadge score={request.lead_score || 0} />
      </div>
    </div>

    {/* Customer Information */}
    {request.ai_analysis?.customer_info && (
      <div className="bg-white border rounded-lg p-6">
        <h4 className="font-medium text-gray-900 mb-4">Customer Information</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium text-gray-700">Name</label>
            <div className="text-gray-900">{request.ai_analysis.customer_info.name || 'N/A'}</div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700">Phone</label>
            <div className="text-gray-900">{request.ai_analysis.customer_info.phone || 'N/A'}</div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700">Email</label>
            <div className="text-gray-900">{request.ai_analysis.customer_info.email || 'N/A'}</div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700">Address</label>
            <div className="text-gray-900">{request.ai_analysis.customer_info.address || 'N/A'}</div>
          </div>
        </div>
      </div>
    )}

    {/* Call Transcript */}
    {request.transcription_data && (
      <div className="bg-white border rounded-lg p-6">
        <h4 className="font-medium text-gray-900 mb-4">Call Transcript</h4>
        <div className="bg-gray-50 p-4 rounded-lg max-h-64 overflow-y-auto">
          <p className="text-gray-800 whitespace-pre-wrap">
            {request.transcription_data.full_text}
          </p>
        </div>
        <div className="mt-2 text-sm text-gray-500">
          Confidence: {Math.round((request.transcription_data.confidence || 0) * 100)}%
          • Duration: {request.transcription_data.duration_seconds}s
        </div>
      </div>
    )}

    {/* AI Analysis */}
    {request.ai_analysis && (
      <div className="bg-white border rounded-lg p-6">
        <h4 className="font-medium text-gray-900 mb-4">AI Analysis</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="text-sm font-medium text-gray-700">Intent</label>
            <div className="text-gray-900">{request.ai_analysis.intent}</div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700">Sentiment</label>
            <div className="text-gray-900 capitalize">{request.ai_analysis.sentiment}</div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700">Urgency</label>
            <div className="text-gray-900 capitalize">{request.ai_analysis.urgency_level}</div>
          </div>
          <div>
            <label className="text-sm font-medium text-gray-700">Topics</label>
            <div className="text-gray-900">{request.ai_analysis.topics.join(', ')}</div>
          </div>
        </div>
      </div>
    )}
  </div>
);

// Helper Components
const SourceIcon: React.FC<{ source: string }> = ({ source }) => {
  const icons = {
    form: '📝',
    callrail: '📞',
    calendar: '📅',
    chat: '💬',
  };
  return <span className="text-lg">{icons[source as keyof typeof icons] || '📊'}</span>;
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const colors = {
    received: 'bg-blue-100 text-blue-800',
    processing: 'bg-yellow-100 text-yellow-800',
    completed: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status as keyof typeof colors]}`}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
};

const LeadScoreBadge: React.FC<{ score: number }> = ({ score }) => {
  const getColor = (score: number) => {
    if (score >= 80) return 'text-green-600';
    if (score >= 60) return 'text-yellow-600';
    if (score >= 40) return 'text-orange-600';
    return 'text-red-600';
  };

  return (
    <div className={`text-sm font-medium ${getColor(score)}`}>
      {score}/100
    </div>
  );
};

// Placeholder components for Grid and Timeline views
const RequestsGrid: React.FC<{
  requests: ProcessingRequest[];
  onSelectRequest: (id: string) => void;
}> = ({ requests, onSelectRequest }) => (
  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {requests.map((request) => (
      <div
        key={request.id}
        onClick={() => onSelectRequest(request.id)}
        className="bg-white p-4 rounded-lg shadow border hover:shadow-md transition-shadow cursor-pointer"
      >
        <div className="flex items-center justify-between mb-3">
          <SourceIcon source={request.source} />
          <StatusBadge status={request.status} />
        </div>
        <div className="text-sm font-medium text-gray-900 mb-1">
          {request.ai_analysis?.customer_info?.name || 'Unknown Customer'}
        </div>
        <div className="text-sm text-gray-500 mb-2">
          {request.ai_analysis?.customer_info?.phone}
        </div>
        <div className="flex items-center justify-between">
          <LeadScoreBadge score={request.lead_score || 0} />
          <span className="text-xs text-gray-500">
            {new Date(request.created_at).toLocaleTimeString()}
          </span>
        </div>
      </div>
    ))}
  </div>
);

const RequestsTimeline: React.FC<{
  requests: ProcessingRequest[];
  onSelectRequest: (id: string) => void;
}> = ({ requests, onSelectRequest }) => (
  <div className="space-y-4">
    {requests.map((request, index) => (
      <div key={request.id} className="flex items-start space-x-4">
        <div className="flex-shrink-0">
          <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center">
            <span className="text-white text-xs font-medium">{index + 1}</span>
          </div>
        </div>
        <div
          onClick={() => onSelectRequest(request.id)}
          className="flex-1 bg-white p-4 rounded-lg shadow border hover:shadow-md transition-shadow cursor-pointer"
        >
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center space-x-3">
              <SourceIcon source={request.source} />
              <span className="font-medium text-gray-900">
                {request.ai_analysis?.customer_info?.name || 'Unknown'}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <StatusBadge status={request.status} />
              <LeadScoreBadge score={request.lead_score || 0} />
            </div>
          </div>
          <div className="text-sm text-gray-600">
            {new Date(request.created_at).toLocaleString()}
          </div>
        </div>
      </div>
    ))}
  </div>
);

// Error and Loading States
const ErrorState: React.FC<{
  error: string;
  onRetry: () => void;
}> = ({ error, onRetry }) => (
  <div className="bg-white rounded-lg shadow p-8 text-center">
    <div className="text-red-500 mb-4">
      <ErrorIcon className="w-12 h-12 mx-auto" />
    </div>
    <h3 className="text-lg font-medium text-gray-900 mb-2">Failed to load requests</h3>
    <p className="text-gray-600 mb-4">{error}</p>
    <button
      onClick={onRetry}
      className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
    >
      Try Again
    </button>
  </div>
);

const LoadingSkeleton: React.FC = () => (
  <div className="bg-white shadow rounded-lg overflow-hidden">
    <div className="animate-pulse">
      <div className="bg-gray-50 px-6 py-3">
        <div className="flex space-x-8">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="h-4 bg-gray-200 rounded w-20"></div>
          ))}
        </div>
      </div>
      <div className="divide-y divide-gray-200">
        {[...Array(10)].map((_, i) => (
          <div key={i} className="px-6 py-4">
            <div className="flex items-center space-x-4">
              <div className="h-10 w-10 bg-gray-200 rounded-full"></div>
              <div className="flex-1 space-y-2">
                <div className="h-4 bg-gray-200 rounded w-32"></div>
                <div className="h-3 bg-gray-200 rounded w-24"></div>
              </div>
              <div className="h-6 bg-gray-200 rounded w-16"></div>
            </div>
          </div>
        ))}
      </div>
    </div>
  </div>
);

// Icon components
const RefreshIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
  </svg>
);

const ListIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
  </svg>
);

const GridIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
  </svg>
);

const TimelineIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const CloseIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const ErrorIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="currentColor" viewBox="0 0 20 20">
    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
  </svg>
);

export default RequestsMonitor;