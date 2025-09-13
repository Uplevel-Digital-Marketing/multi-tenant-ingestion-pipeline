// Real-time Metrics Hook using Server-Sent Events
// Location: ./src/hooks/useRealtimeMetrics.ts

import { useState, useEffect, useRef, useCallback } from 'react';
import { DashboardMetrics, RealtimeEvent, TenantConfig } from '@/types/tenant';

interface UseRealtimeMetricsOptions {
  tenantId: string;
  timeRange: 'hour' | 'day' | 'week' | 'month';
  autoRefresh?: boolean;
  refreshInterval?: number;
}

interface RealtimeMetricsState {
  metrics: DashboardMetrics | null;
  isConnected: boolean;
  isLoading: boolean;
  error: string | null;
  lastUpdated: Date | null;
  connectionRetries: number;
}

export function useRealtimeMetrics(options: UseRealtimeMetricsOptions) {
  const [state, setState] = useState<RealtimeMetricsState>({
    metrics: null,
    isConnected: false,
    isLoading: true,
    error: null,
    lastUpdated: null,
    connectionRetries: 0,
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const retryTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const { tenantId, timeRange, autoRefresh = true, refreshInterval = 5000 } = options;

  // Connection management
  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const url = new URL('/api/dashboard/realtime-metrics', window.location.origin);
    url.searchParams.set('tenant_id', tenantId);
    url.searchParams.set('time_range', timeRange);

    console.log(`[Realtime Metrics] Connecting to: ${url.toString()}`);

    const eventSource = new EventSource(url.toString());
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      console.log('[Realtime Metrics] Connection opened');
      setState(prev => ({
        ...prev,
        isConnected: true,
        isLoading: false,
        error: null,
        connectionRetries: 0,
      }));
    };

    eventSource.onmessage = (event) => {
      try {
        const data: RealtimeEvent = JSON.parse(event.data);

        if (data.type === 'metric_update') {
          setState(prev => ({
            ...prev,
            metrics: data.data as DashboardMetrics,
            lastUpdated: new Date(),
            error: null,
          }));
        }
      } catch (error) {
        console.error('[Realtime Metrics] Failed to parse event data:', error);
      }
    };

    eventSource.onerror = (error) => {
      console.error('[Realtime Metrics] Connection error:', error);

      setState(prev => ({
        ...prev,
        isConnected: false,
        error: 'Connection lost. Attempting to reconnect...',
        connectionRetries: prev.connectionRetries + 1,
      }));

      // Exponential backoff retry strategy
      const retryDelay = Math.min(1000 * Math.pow(2, state.connectionRetries), 30000);

      retryTimeoutRef.current = setTimeout(() => {
        if (state.connectionRetries < 5) {
          connect();
        } else {
          setState(prev => ({
            ...prev,
            error: 'Max retries exceeded. Please refresh the page.',
            isLoading: false,
          }));
        }
      }, retryDelay);
    };
  }, [tenantId, timeRange, state.connectionRetries]);

  // Manual refresh function
  const refresh = useCallback(async () => {
    setState(prev => ({ ...prev, isLoading: true }));

    try {
      const response = await fetch(`/api/dashboard/metrics?tenant_id=${tenantId}&time_range=${timeRange}`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const metrics: DashboardMetrics = await response.json();
      setState(prev => ({
        ...prev,
        metrics,
        lastUpdated: new Date(),
        isLoading: false,
        error: null,
      }));
    } catch (error) {
      console.error('[Realtime Metrics] Manual refresh failed:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to refresh metrics',
        isLoading: false,
      }));
    }
  }, [tenantId, timeRange]);

  // Initialize connection
  useEffect(() => {
    if (autoRefresh) {
      connect();
    } else {
      refresh();
    }

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
      }
    };
  }, [tenantId, timeRange, autoRefresh, connect, refresh]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
      }
    };
  }, []);

  // Manual disconnect
  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (retryTimeoutRef.current) {
      clearTimeout(retryTimeoutRef.current);
    }
    setState(prev => ({
      ...prev,
      isConnected: false,
    }));
  }, []);

  return {
    ...state,
    refresh,
    connect,
    disconnect,
  };
}

// Hook for processing request real-time updates
export function useRealtimeRequests(tenantId: string) {
  const [requests, setRequests] = useState<any[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const url = `/api/dashboard/realtime-requests?tenant_id=${tenantId}`;
    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      setIsConnected(true);
    };

    eventSource.onmessage = (event) => {
      try {
        const data: RealtimeEvent = JSON.parse(event.data);

        if (data.type === 'request_received') {
          setRequests(prev => [data.data, ...prev.slice(0, 99)]); // Keep last 100
        } else if (data.type === 'processing_complete') {
          setRequests(prev => prev.map(req =>
            req.id === data.data.request_id
              ? { ...req, status: 'completed', ...data.data }
              : req
          ));
        }
      } catch (error) {
        console.error('[Realtime Requests] Parse error:', error);
      }
    };

    eventSource.onerror = () => {
      setIsConnected(false);
    };

    return () => {
      eventSource.close();
    };
  }, [tenantId]);

  return {
    requests,
    isConnected,
  };
}

// Hook for tenant health monitoring
export function useTenantHealth(tenantId: string) {
  const [health, setHealth] = useState({
    status: 'unknown' as 'healthy' | 'warning' | 'error' | 'unknown',
    uptime: 0,
    error_rate: 0,
    avg_response_time: 0,
    active_integrations: [] as string[],
    last_error: null as string | null,
  });

  useEffect(() => {
    const checkHealth = async () => {
      try {
        const response = await fetch(`/api/dashboard/health?tenant_id=${tenantId}`);
        const healthData = await response.json();
        setHealth(healthData);
      } catch (error) {
        setHealth(prev => ({
          ...prev,
          status: 'error',
          last_error: 'Health check failed',
        }));
      }
    };

    // Initial check
    checkHealth();

    // Set up periodic health checks
    const interval = setInterval(checkHealth, 30000); // Every 30 seconds

    return () => clearInterval(interval);
  }, [tenantId]);

  return health;
}