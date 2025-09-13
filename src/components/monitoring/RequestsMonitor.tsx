// Real-time Request Processing Monitor
// Location: ./src/components/monitoring/RequestsMonitor.tsx

import React, { useState, useEffect, useMemo } from 'react';
import { useRealtimeRequests } from '@/hooks/useRealtimeMetrics';
import { ProcessingRequest, AIAnalysis } from '@/types/tenant';

interface RequestsMonitorProps {
  tenantId: string;
  autoRefresh?: boolean;
  showFilters?: boolean;
}

export const RequestsMonitor: React.FC<RequestsMonitorProps> = ({
  tenantId,
  autoRefresh = true,
  showFilters = true,
}) => {
  const { requests, isConnected } = useRealtimeRequests(tenantId);
  const [filters, setFilters] = useState({
    source: 'all',
    status: 'all',
    timeRange: '1h',
    search: '',
  });
  const [selectedRequest, setSelectedRequest] = useState<ProcessingRequest | null>(null);
  const [viewMode, setViewMode] = useState<'list' | 'grid' | 'timeline'>('list');

  // Filter and sort requests
  const filteredRequests = useMemo(() => {
    let filtered = [...requests];

    // Apply filters
    if (filters.source !== 'all') {
      filtered = filtered.filter(req => req.source === filters.source);
    }

    if (filters.status !== 'all') {
      filtered = filtered.filter(req => req.status === filters.status);
    }

    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      filtered = filtered.filter(req =>
        req.id.toLowerCase().includes(searchLower) ||
        req.ai_analysis?.customer_info?.name?.toLowerCase().includes(searchLower) ||
        req.ai_analysis?.intent?.toLowerCase().includes(searchLower)
      );
    }

    // Apply time range filter
    const now = new Date();
    const timeLimit = new Date();

    switch (filters.timeRange) {
      case '1h':
        timeLimit.setHours(now.getHours() - 1);
        break;
      case '6h':
        timeLimit.setHours(now.getHours() - 6);
        break;
      case '24h':
        timeLimit.setDate(now.getDate() - 1);
        break;
      case '7d':
        timeLimit.setDate(now.getDate() - 7);
        break;
    }

    filtered = filtered.filter(req => new Date(req.created_at) >= timeLimit);

    // Sort by creation time (newest first)
    return filtered.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
  }, [requests, filters]);

  // Calculate summary stats
  const stats = useMemo(() => {
    const total = filteredRequests.length;
    const completed = filteredRequests.filter(r => r.status === 'completed').length;
    const processing = filteredRequests.filter(r => r.status === 'processing').length;
    const failed = filteredRequests.filter(r => r.status === 'failed').length;
    const avgProcessingTime = filteredRequests
      .filter(r => r.processing_time_ms > 0)
      .reduce((sum, r) => sum + r.processing_time_ms, 0) /
      Math.max(1, filteredRequests.filter(r => r.processing_time_ms > 0).length);

    return { total, completed, processing, failed, avgProcessingTime };
  }, [filteredRequests]);

  return (
    <div className="space-y-6">
      {/* Header with Connection Status */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <h2 className="text-2xl font-bold text-gray-900">Request Monitor</h2>
          <div className="flex items-center space-x-2">
            <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-400' : 'bg-red-400'}`} />
            <span className="text-sm text-gray-600">
              {isConnected ? 'Live Updates' : 'Disconnected'}
            </span>
          </div>
        </div>

        <div className="flex items-center space-x-3">
          <ViewModeSelector value={viewMode} onChange={setViewMode} />
          <button
            onClick={() => setFilters(prev => ({ ...prev, search: '', source: 'all', status: 'all' }))}
            className="px-3 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
          >
            Clear Filters
          </button>
        </div>
      </div>

      {/* Stats Summary */}
      <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <StatCard label="Total Requests" value={stats.total} color="blue" />
        <StatCard label="Completed" value={stats.completed} color="green" />
        <StatCard label="Processing" value={stats.processing} color="yellow" />
        <StatCard label="Failed" value={stats.failed} color="red" />
        <StatCard
          label="Avg Time"
          value={stats.avgProcessingTime > 0 ? `${Math.round(stats.avgProcessingTime)}ms` : 'N/A'}
          color="purple"
        />
      </div>

      {/* Filters */}
      {showFilters && (
        <RequestFilters
          filters={filters}
          onFiltersChange={setFilters}
          requestCount={filteredRequests.length}
        />
      )}

      {/* Requests Display */}
      <div className="bg-white rounded-lg shadow-sm border">
        {filteredRequests.length === 0 ? (
          <EmptyState filters={filters} />
        ) : (
          <>
            {viewMode === 'list' && (
              <RequestsList
                requests={filteredRequests}
                onRequestSelect={setSelectedRequest}
                selectedRequest={selectedRequest}
              />
            )}
            {viewMode === 'grid' && (
              <RequestsGrid
                requests={filteredRequests}
                onRequestSelect={setSelectedRequest}
              />
            )}
            {viewMode === 'timeline' && (
              <RequestsTimeline
                requests={filteredRequests}
                onRequestSelect={setSelectedRequest}
              />
            )}
          </>
        )}
      </div>

      {/* Request Detail Modal */}
      {selectedRequest && (
        <RequestDetailModal
          request={selectedRequest}
          onClose={() => setSelectedRequest(null)}
        />
      )}
    </div>
  );
};

// Supporting Components

const StatCard: React.FC<{
  label: string;
  value: string | number;
  color: 'blue' | 'green' | 'yellow' | 'red' | 'purple';
}> = ({ label, value, color }) => {
  const colorClasses = {
    blue: 'bg-blue-50 text-blue-700 border-blue-200',
    green: 'bg-green-50 text-green-700 border-green-200',
    yellow: 'bg-yellow-50 text-yellow-700 border-yellow-200',
    red: 'bg-red-50 text-red-700 border-red-200',
    purple: 'bg-purple-50 text-purple-700 border-purple-200',
  };

  return (
    <div className={`p-4 rounded-lg border ${colorClasses[color]}`}>
      <div className="text-sm font-medium">{label}</div>
      <div className="text-2xl font-bold mt-1">{value}</div>
    </div>
  );
};

const ViewModeSelector: React.FC<{
  value: 'list' | 'grid' | 'timeline';
  onChange: (mode: 'list' | 'grid' | 'timeline') => void;
}> = ({ value, onChange }) => (
  <div className="flex border border-gray-300 rounded-md">
    {[
      { id: 'list', label: 'List', icon: '📋' },
      { id: 'grid', label: 'Grid', icon: '▦' },
      { id: 'timeline', label: 'Timeline', icon: '📊' },
    ].map((mode) => (
      <button
        key={mode.id}
        onClick={() => onChange(mode.id as any)}
        className={`px-3 py-1 text-sm font-medium ${
          value === mode.id
            ? 'bg-blue-600 text-white'
            : 'bg-white text-gray-700 hover:bg-gray-50'
        } ${mode.id === 'list' ? 'rounded-l-md' : mode.id === 'timeline' ? 'rounded-r-md' : ''}`}
      >
        <span className="mr-1">{mode.icon}</span>
        {mode.label}
      </button>
    ))}
  </div>
);

const RequestFilters: React.FC<{
  filters: any;
  onFiltersChange: (filters: any) => void;
  requestCount: number;
}> = ({ filters, onFiltersChange, requestCount }) => (
  <div className="bg-white p-4 rounded-lg shadow-sm border">
    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Source</label>
        <select
          value={filters.source}
          onChange={(e) => onFiltersChange({ ...filters, source: e.target.value })}
          className="w-full px-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">All Sources</option>
          <option value="form">Website Forms</option>
          <option value="callrail">Phone Calls</option>
          <option value="calendar">Calendar Bookings</option>
          <option value="chat">Chat Messages</option>
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
        <select
          value={filters.status}
          onChange={(e) => onFiltersChange({ ...filters, status: e.target.value })}
          className="w-full px-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="all">All Statuses</option>
          <option value="received">Received</option>
          <option value="processing">Processing</option>
          <option value="completed">Completed</option>
          <option value="failed">Failed</option>
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Time Range</label>
        <select
          value={filters.timeRange}
          onChange={(e) => onFiltersChange({ ...filters, timeRange: e.target.value })}
          className="w-full px-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          <option value="1h">Last Hour</option>
          <option value="6h">Last 6 Hours</option>
          <option value="24h">Last 24 Hours</option>
          <option value="7d">Last 7 Days</option>
        </select>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Search</label>
        <input
          type="text"
          value={filters.search}
          onChange={(e) => onFiltersChange({ ...filters, search: e.target.value })}
          placeholder="Search by ID, name, intent..."
          className="w-full px-3 py-2 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>
    </div>

    <div className="mt-3 text-sm text-gray-600">
      Showing {requestCount} request{requestCount !== 1 ? 's' : ''}
    </div>
  </div>
);

const RequestsList: React.FC<{
  requests: ProcessingRequest[];
  onRequestSelect: (request: ProcessingRequest) => void;
  selectedRequest: ProcessingRequest | null;
}> = ({ requests, onRequestSelect, selectedRequest }) => (
  <div className="overflow-hidden">
    <table className="min-w-full divide-y divide-gray-200">
      <thead className="bg-gray-50">
        <tr>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Request
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Source
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Status
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Lead Score
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Processing Time
          </th>
          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Created
          </th>
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {requests.map((request) => (
          <tr
            key={request.id}
            onClick={() => onRequestSelect(request)}
            className={`cursor-pointer hover:bg-gray-50 ${
              selectedRequest?.id === request.id ? 'bg-blue-50' : ''
            }`}
          >
            <td className="px-6 py-4 whitespace-nowrap">
              <div>
                <div className="text-sm font-medium text-gray-900">
                  {request.id.slice(0, 8)}...
                </div>
                <div className="text-sm text-gray-500">
                  {request.ai_analysis?.customer_info?.name || 'Unknown'}
                </div>
              </div>
            </td>
            <td className="px-6 py-4 whitespace-nowrap">
              <SourceBadge source={request.source} />
            </td>
            <td className="px-6 py-4 whitespace-nowrap">
              <StatusBadge status={request.status} />
            </td>
            <td className="px-6 py-4 whitespace-nowrap">
              <LeadScoreBadge score={request.lead_score} />
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
              {request.processing_time_ms ? `${request.processing_time_ms}ms` : 'N/A'}
            </td>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {new Date(request.created_at).toLocaleTimeString()}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
);

const RequestsGrid: React.FC<{
  requests: ProcessingRequest[];
  onRequestSelect: (request: ProcessingRequest) => void;
}> = ({ requests, onRequestSelect }) => (
  <div className="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {requests.map((request) => (
      <RequestCard
        key={request.id}
        request={request}
        onSelect={() => onRequestSelect(request)}
      />
    ))}
  </div>
);

const RequestCard: React.FC<{
  request: ProcessingRequest;
  onSelect: () => void;
}> = ({ request, onSelect }) => (
  <div
    onClick={onSelect}
    className="p-4 border border-gray-200 rounded-lg hover:shadow-md transition-shadow cursor-pointer"
  >
    <div className="flex items-center justify-between mb-3">
      <SourceBadge source={request.source} />
      <StatusBadge status={request.status} />
    </div>

    <div className="space-y-2">
      <div className="text-sm font-medium text-gray-900">
        {request.ai_analysis?.customer_info?.name || 'Unknown Customer'}
      </div>

      <div className="text-xs text-gray-500">
        ID: {request.id.slice(0, 12)}...
      </div>

      {request.ai_analysis?.intent && (
        <div className="text-sm text-gray-600">
          Intent: {request.ai_analysis.intent}
        </div>
      )}

      <div className="flex items-center justify-between mt-3">
        <LeadScoreBadge score={request.lead_score} />
        <span className="text-xs text-gray-500">
          {new Date(request.created_at).toLocaleTimeString()}
        </span>
      </div>
    </div>
  </div>
);

const RequestsTimeline: React.FC<{
  requests: ProcessingRequest[];
  onRequestSelect: (request: ProcessingRequest) => void;
}> = ({ requests, onRequestSelect }) => (
  <div className="p-6">
    <div className="relative">
      <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200"></div>
      <div className="space-y-6">
        {requests.map((request, index) => (
          <div key={request.id} className="relative flex items-start space-x-4">
            <div className="relative z-10 flex items-center justify-center w-8 h-8 bg-white border-2 border-gray-300 rounded-full">
              <SourceIcon source={request.source} />
            </div>
            <div
              onClick={() => onRequestSelect(request)}
              className="flex-1 min-w-0 p-4 bg-white border border-gray-200 rounded-lg hover:shadow-md transition-shadow cursor-pointer"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="text-sm font-medium text-gray-900">
                  {request.ai_analysis?.customer_info?.name || 'Unknown Customer'}
                </div>
                <div className="text-xs text-gray-500">
                  {new Date(request.created_at).toLocaleTimeString()}
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <StatusBadge status={request.status} />
                <LeadScoreBadge score={request.lead_score} />
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  </div>
);

// Badge Components
const SourceBadge: React.FC<{ source: string }> = ({ source }) => {
  const colors = {
    form: 'bg-blue-100 text-blue-800',
    callrail: 'bg-green-100 text-green-800',
    calendar: 'bg-purple-100 text-purple-800',
    chat: 'bg-yellow-100 text-yellow-800',
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[source as keyof typeof colors] || 'bg-gray-100 text-gray-800'}`}>
      {source}
    </span>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const colors = {
    received: 'bg-blue-100 text-blue-800',
    processing: 'bg-yellow-100 text-yellow-800',
    completed: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status as keyof typeof colors] || 'bg-gray-100 text-gray-800'}`}>
      {status}
    </span>
  );
};

const LeadScoreBadge: React.FC<{ score?: number }> = ({ score }) => {
  if (!score) return <span className="text-xs text-gray-400">No Score</span>;

  const getColor = (score: number) => {
    if (score >= 80) return 'bg-green-100 text-green-800';
    if (score >= 60) return 'bg-yellow-100 text-yellow-800';
    if (score >= 40) return 'bg-orange-100 text-orange-800';
    return 'bg-red-100 text-red-800';
  };

  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${getColor(score)}`}>
      Score: {score}
    </span>
  );
};

const SourceIcon: React.FC<{ source: string }> = ({ source }) => {
  const icons: Record<string, string> = {
    form: '📝',
    callrail: '📞',
    calendar: '📅',
    chat: '💬',
  };
  return <span className="text-xs">{icons[source] || '📊'}</span>;
};

const EmptyState: React.FC<{ filters: any }> = ({ filters }) => (
  <div className="p-12 text-center">
    <div className="text-gray-400 mb-4">📭</div>
    <h3 className="text-lg font-medium text-gray-900 mb-2">No requests found</h3>
    <p className="text-gray-600">
      {filters.search || filters.source !== 'all' || filters.status !== 'all'
        ? 'Try adjusting your filters to see more results.'
        : 'No requests have been received yet.'}
    </p>
  </div>
);

const RequestDetailModal: React.FC<{
  request: ProcessingRequest;
  onClose: () => void;
}> = ({ request, onClose }) => (
  <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
    <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-screen overflow-auto">
      <div className="p-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-bold text-gray-900">Request Details</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 text-xl"
          >
            ×
          </button>
        </div>

        {/* Request details content would go here */}
        <div className="space-y-4">
          <div>
            <h3 className="text-sm font-medium text-gray-700">Request ID</h3>
            <p className="text-sm text-gray-900">{request.id}</p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <h3 className="text-sm font-medium text-gray-700">Source</h3>
              <SourceBadge source={request.source} />
            </div>
            <div>
              <h3 className="text-sm font-medium text-gray-700">Status</h3>
              <StatusBadge status={request.status} />
            </div>
          </div>

          {request.ai_analysis && (
            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">AI Analysis</h3>
              <pre className="text-xs bg-gray-100 p-3 rounded overflow-auto">
                {JSON.stringify(request.ai_analysis, null, 2)}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  </div>
);