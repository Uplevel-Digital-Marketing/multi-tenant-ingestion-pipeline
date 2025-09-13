// Call Processing Hook for Real-time Monitoring
// Location: ./src/hooks/useCallProcessing.ts

import { useState, useEffect, useCallback, useRef } from 'react';
import { ProcessingRequest, RealtimeEvent } from '@/types/api';
import { apiService } from '@/services/api';

interface UseCallProcessingOptions {
  tenantId: string;
  filters?: {
    source?: string[];
    status?: string[];
    limit?: number;
  };
  realtime?: boolean;
}

interface CallProcessingState {
  requests: ProcessingRequest[];
  isLoading: boolean;
  error: string | null;
  totalCount: number;
  hasNextPage: boolean;
  isConnected: boolean;
  filters: any;
}

export function useCallProcessing(options: UseCallProcessingOptions) {
  const { tenantId, filters = {}, realtime = true } = options;

  const [state, setState] = useState<CallProcessingState>({
    requests: [],
    isLoading: true,
    error: null,
    totalCount: 0,
    hasNextPage: false,
    isConnected: false,
    filters,
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const requestsRef = useRef<ProcessingRequest[]>([]);

  // Fetch initial data
  const fetchRequests = useCallback(async () => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));

    try {
      const requests = await apiService.getProcessingRequests(tenantId, filters);
      setState(prev => ({
        ...prev,
        requests,
        isLoading: false,
        totalCount: requests.length,
        hasNextPage: requests.length >= (filters.limit || 50),
      }));
      requestsRef.current = requests;
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to fetch requests',
        isLoading: false,
      }));
    }
  }, [tenantId, filters]);

  // Real-time connection management
  const connectRealtime = useCallback(() => {
    if (!realtime || eventSourceRef.current) return;

    const url = new URL('/api/dashboard/realtime-requests', window.location.origin);
    url.searchParams.set('tenant_id', tenantId);

    const eventSource = new EventSource(url.toString());
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      console.log('[Call Processing] Real-time connection opened');
      setState(prev => ({ ...prev, isConnected: true }));
    };

    eventSource.onmessage = (event) => {
      try {
        const data: RealtimeEvent = JSON.parse(event.data);

        switch (data.type) {
          case 'request_received':
            handleNewRequest(data.data);
            break;
          case 'processing_complete':
            handleRequestUpdate(data.data);
            break;
          default:
            console.log('[Call Processing] Unknown event type:', data.type);
        }
      } catch (error) {
        console.error('[Call Processing] Failed to parse event:', error);
      }
    };

    eventSource.onerror = (error) => {
      console.error('[Call Processing] Connection error:', error);
      setState(prev => ({ ...prev, isConnected: false }));

      // Attempt reconnection after 5 seconds
      setTimeout(connectRealtime, 5000);
    };
  }, [tenantId, realtime]);

  // Handle new request from real-time stream
  const handleNewRequest = useCallback((newRequest: ProcessingRequest) => {
    setState(prev => {
      const updatedRequests = [newRequest, ...prev.requests.slice(0, 99)]; // Keep last 100
      return {
        ...prev,
        requests: updatedRequests,
        totalCount: prev.totalCount + 1,
      };
    });
    requestsRef.current = [newRequest, ...requestsRef.current.slice(0, 99)];
  }, []);

  // Handle request status updates
  const handleRequestUpdate = useCallback((updateData: any) => {
    setState(prev => {
      const updatedRequests = prev.requests.map(req =>
        req.id === updateData.request_id
          ? { ...req, status: updateData.status, ...updateData }
          : req
      );
      return {
        ...prev,
        requests: updatedRequests,
      };
    });

    requestsRef.current = requestsRef.current.map(req =>
      req.id === updateData.request_id
        ? { ...req, status: updateData.status, ...updateData }
        : req
    );
  }, []);

  // Manual refresh
  const refresh = useCallback(() => {
    fetchRequests();
  }, [fetchRequests]);

  // Load more requests (pagination)
  const loadMore = useCallback(async () => {
    if (!state.hasNextPage || state.isLoading) return;

    setState(prev => ({ ...prev, isLoading: true }));

    try {
      const moreRequests = await apiService.getProcessingRequests(tenantId, {
        ...filters,
        page: Math.floor(state.requests.length / (filters.limit || 50)) + 1,
      });

      setState(prev => ({
        ...prev,
        requests: [...prev.requests, ...moreRequests],
        isLoading: false,
        hasNextPage: moreRequests.length >= (filters.limit || 50),
      }));
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to load more requests',
        isLoading: false,
      }));
    }
  }, [tenantId, filters, state.requests.length, state.hasNextPage, state.isLoading]);

  // Update filters
  const updateFilters = useCallback((newFilters: any) => {
    setState(prev => ({ ...prev, filters: { ...prev.filters, ...newFilters } }));
  }, []);

  // Initialize
  useEffect(() => {
    fetchRequests();
    if (realtime) {
      connectRealtime();
    }

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [fetchRequests, connectRealtime, realtime]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  return {
    ...state,
    refresh,
    loadMore,
    updateFilters,
    disconnect: () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
        setState(prev => ({ ...prev, isConnected: false }));
      }
    },
  };
}

// Hook for individual request details
export function useRequestDetails(requestId: string) {
  const [request, setRequest] = useState<ProcessingRequest | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRequest = useCallback(async () => {
    if (!requestId) return;

    setIsLoading(true);
    setError(null);

    try {
      const requestData = await apiService.getProcessingRequest(requestId);
      setRequest(requestData);
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to fetch request details');
    } finally {
      setIsLoading(false);
    }
  }, [requestId]);

  useEffect(() => {
    fetchRequest();
  }, [fetchRequest]);

  return {
    request,
    isLoading,
    error,
    refresh: fetchRequest,
  };
}