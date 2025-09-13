# 🎨 Multi-Tenant Ingestion Pipeline - Frontend Dashboard

A comprehensive React/TypeScript dashboard for monitoring and managing the multi-tenant ingestion pipeline with real-time capabilities.

## 🚀 Features Implemented

### ✅ Core Components
- **MainDashboard** - Complete dashboard orchestration with navigation
- **MetricsOverview** - Real-time metrics with Server-Sent Events
- **CallProcessingMonitor** - Live request monitoring with filtering
- **TenantManager** - Comprehensive tenant configuration interface
- **CRMConfiguration** - Multi-provider CRM integration setup
- **AnalyticsDashboard** - Advanced analytics and insights

### ✅ Real-time Capabilities
- **Server-Sent Events** - Live connection with automatic reconnection
- **Real-time Metrics** - Dashboard updates without page refresh
- **Live Request Stream** - Monitor incoming requests as they happen
- **Connection Health** - Visual indicators for connection status

### ✅ State Management
- **React Hooks** - Modern state management patterns
- **Custom Hooks** - Reusable logic for API calls and real-time data
- **Error Handling** - Comprehensive error states and recovery

### ✅ UI/UX Features
- **Responsive Design** - Mobile-first design approach
- **Loading States** - Skeleton screens and spinners
- **Error States** - User-friendly error messages with retry options
- **Empty States** - Helpful guidance when no data is available

## 📂 Project Structure

```
src/
├── components/
│   ├── Dashboard/
│   │   └── MainDashboard.tsx         # Main orchestration component
│   ├── Tenants/
│   │   └── TenantManager.tsx         # Tenant configuration interface
│   ├── CRM/
│   │   └── CRMConfiguration.tsx      # CRM integration setup
│   ├── Calls/
│   │   └── CallProcessingMonitor.tsx # Real-time request monitoring
│   ├── Analytics/
│   │   └── AnalyticsDashboard.tsx    # Analytics and reporting
│   └── common/
│       └── Layout.tsx                # Reusable UI components
├── hooks/
│   ├── useRealtimeMetrics.ts         # Real-time metrics hook
│   └── useCallProcessing.ts          # Call processing data hook
├── services/
│   └── api.ts                        # API service layer
├── types/
│   ├── api.ts                        # API type definitions
│   └── tenant.ts                     # Tenant and metrics types
└── pages/
    ├── index.tsx                     # Entry point
    └── Dashboard.tsx                 # Main dashboard page
```

## 🛠️ Technology Stack

- **Framework**: Next.js 14 with TypeScript
- **Styling**: Tailwind CSS
- **State Management**: React Hooks + Custom Hooks
- **Real-time**: Server-Sent Events (EventSource)
- **HTTP Client**: Axios with interceptors
- **Icons**: Heroicons and Lucide React
- **Build Tools**: Next.js built-in tooling

## 🚦 Getting Started

### Prerequisites
- Node.js 18+ and npm 8+
- Backend API running on port 8080 (configurable)

### Installation & Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Access the dashboard
open http://localhost:3000
```

### Environment Variables

Create `.env.local`:

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
NODE_ENV=development
```

## 📡 Real-time Architecture

### Server-Sent Events Implementation

The dashboard uses Server-Sent Events for real-time updates:

```typescript
// Real-time metrics connection
const { metrics, isConnected } = useRealtimeMetrics({
  tenantId: 'tenant_123',
  timeRange: 'day',
  autoRefresh: true
});

// Real-time request monitoring
const { requests, isConnected } = useCallProcessing({
  tenantId: 'tenant_123',
  realtime: true
});
```

### Expected API Endpoints

#### SSE Endpoints
- `GET /api/dashboard/realtime-metrics?tenant_id={id}` - Real-time metrics
- `GET /api/dashboard/realtime-requests?tenant_id={id}` - Live request stream

#### REST Endpoints
- `GET /api/tenants` - List tenants
- `GET /api/tenants/{id}` - Get tenant details
- `PUT /api/tenants/{id}` - Update tenant
- `GET /api/dashboard/metrics?tenant_id={id}&time_range={range}` - Get metrics
- `POST /api/integrations/callrail/test` - Test CallRail connection
- `POST /api/integrations/crm/test` - Test CRM connection

## 🎛️ Dashboard Features

### 1. Overview Dashboard
- **Real-time metrics** with trend indicators
- **System health** monitoring
- **Cost analysis** with projections
- **Request volume** charts

### 2. Live Request Monitor
- **Real-time request stream** with filtering
- **Multiple view modes**: List, Grid, Timeline
- **Request details** with AI analysis
- **Status tracking** with real-time updates

### 3. Tenant Management
- **Multi-tab configuration** interface
- **CallRail integration** with connection testing
- **Workflow configuration** for AI processing
- **Service area management**

### 4. CRM Configuration
- **Multi-provider support**: Salesforce, HubSpot, Pipedrive
- **Field mapping interface** with drag-and-drop
- **Connection testing** with real-time validation
- **Provider-specific settings**

### 5. Analytics Dashboard
- **Performance metrics** with historical trends
- **Conversion funnel** analysis
- **Cost breakdown** and optimization insights
- **Lead quality** distribution

## 🔧 Configuration Options

### API Service Configuration

```typescript
// API service with tenant context
apiService.setTenant('tenant_123');

// Test integrations
await apiService.testCallRailIntegration(tenantId, {
  company_id: 'CR123',
  api_key: 'key_123'
});
```

### Real-time Connection Options

```typescript
// Configure SSE connection
const { metrics } = useRealtimeMetrics({
  tenantId: 'tenant_123',
  timeRange: 'day',
  autoRefresh: true,        // Enable auto-refresh
  refreshInterval: 5000,    // Refresh every 5 seconds
});
```

## 📱 Mobile Responsiveness

The dashboard is fully responsive with:

- **Mobile-first design** approach
- **Collapsible navigation** for small screens
- **Touch-friendly interactions**
- **Adaptive layouts** for different screen sizes
- **Optimized performance** for mobile networks

## 🧪 Testing

### Component Testing
```bash
npm run test           # Run unit tests
npm run test:watch     # Watch mode
```

### E2E Testing
```bash
npm run test:e2e       # Run Playwright tests
```

### Type Checking
```bash
npm run type-check     # TypeScript validation
```

## 📈 Performance Optimizations

### Real-time Optimizations
- **Connection pooling** for SSE
- **Automatic reconnection** with exponential backoff
- **Message batching** to prevent UI choking
- **Memory management** with request history limits

### UI Optimizations
- **React.memo** for expensive components
- **Virtual scrolling** for large lists
- **Lazy loading** with code splitting
- **Image optimization** with Next.js

## 🔒 Security Features

- **Tenant isolation** at the UI level
- **API authentication** with tenant context
- **Input validation** and sanitization
- **XSS protection** with proper escaping
- **CSRF protection** via API design

## 🚀 Production Deployment

### Build for Production
```bash
npm run build          # Build optimized production bundle
npm run start          # Start production server
```

### Static Export (Optional)
```bash
BUILD_MODE=static npm run build
npm run export
```

### Environment Configuration
```env
NEXT_PUBLIC_API_BASE_URL=https://api.your-domain.com
NODE_ENV=production
```

## 🔮 Future Enhancements

### Phase 2 Features
- [ ] **Advanced filtering** with saved filters
- [ ] **Custom dashboards** with drag-and-drop widgets
- [ ] **Export functionality** for reports
- [ ] **Notification center** with real-time alerts
- [ ] **Dark mode** theme switching

### Phase 3 Features
- [ ] **Mobile app** (React Native)
- [ ] **Advanced charting** with time-series data
- [ ] **Real-time collaboration** features
- [ ] **Workflow builder** visual interface
- [ ] **Multi-language support** (i18n)

## 🤝 Contributing

### Code Standards
- **TypeScript strict mode** enabled
- **ESLint + Prettier** for consistent formatting
- **Component documentation** with JSDoc
- **Comprehensive error handling**

### Development Workflow
1. Create feature branch
2. Implement with tests
3. Run type checking and linting
4. Submit pull request
5. Code review and merge

## 📞 Support

For technical support or questions about the frontend implementation:

1. Check the component documentation in the code
2. Review the API integration patterns
3. Test real-time connections with the backend
4. Verify environment configuration

---

**Built with ❤️ for Multi-Tenant Pipeline Management**

This dashboard provides a comprehensive, real-time interface for managing the multi-tenant ingestion pipeline, with modern React patterns, TypeScript safety, and production-ready performance optimizations.