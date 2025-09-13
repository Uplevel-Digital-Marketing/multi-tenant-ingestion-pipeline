# 📋 Multi-Tenant Ingestion Pipeline - 2025 Requirements Analysis

## 🎯 **Project Overview - Requirements Analyst Report**
**Generated**: September 2025 | **Agent**: requirements-analyst | **Duration**: 15min

### **Mission Statement**
Build a cutting-edge multi-tenant ingestion pipeline for home remodeling companies that processes forms, phone calls (via CallRail), calendar bookings, and chat interactions through an intelligent Go/Gemini agent with configurable workflows.

### **Critical 2025 Technology Stack**
- **Vertex AI Gemini 2.5 Flash** (2025 updates - requires latest SDK)
- **Speech-to-Text Chirp 3** (2025 features - enhanced accuracy)
- **Cloud Spanner** (2025 multi-tenant improvements)
- **Cloud Run** (2025 scaling capabilities)
- **Go 1.23+** (2025 version with enhanced performance)

---

## 🏗️ **System Architecture Requirements**

### **Input Sources (Multi-Channel)**
1. 🌐 **Website Forms** → Direct API Gateway processing
2. 📞 **CallRail Webhooks** → Post-call webhook with audio download
3. 📅 **Calendar Bookings** → Calendar API integration
4. 💬 **Chat Widgets** → Real-time chat processing

### **Core Processing Flow**
```
Input → Cloud Load Balancer → Cloud Run (Go/Gemini Agent) →
tenant_id Authentication → Communication Detection →
AI Processing → Configurable Workflow → CRM/Email/Storage
```

### **Authentication Requirements**
- ❌ **NO API Keys** required in headers
- ✅ **tenant_id** in JSON payload for authentication
- ✅ **CallRail company ID** mapping in office settings
- 🔐 **HMAC signature verification** for CallRail webhooks

---

## 🗄️ **Database Requirements - Cloud Spanner**

### **Existing Infrastructure**
- **Project**: `account-strategy-464106`
- **Location**: `us-central1`
- **Instance**: `upai-customers` (Enterprise, us-central1, Autoscaling)
- **Database**: `agent_platform` (Google Standard SQL)

### **Required Schema Updates**
Based on database-schema-updates.sql analysis:

#### **Enhanced Tables**
1. **offices** - Add CallRail integration fields
2. **requests** - Add call processing and AI analysis fields
3. **call_recordings** - NEW table for audio file management
4. **webhook_events** - NEW table for webhook processing log
5. **ai_processing_log** - NEW table for AI analysis tracking

#### **Multi-Tenant Security**
- Row-level security policies for tenant isolation
- Interleaved tables for performance optimization
- Comprehensive indexing for query performance

---

## 🧠 **AI Intelligence Requirements**

### **Gemini Agent Architecture**
The Go/Gemini agent serves as the core intelligent processing service:

```go
type GeminiAgent struct {
    VertexAIClient   *aiplatform.PredictionClient
    WorkflowEngine   *workflow.Engine
    CRMConnectors    map[string]crm.Connector
    ConfigManager    *config.Manager
}
```

### **AI Processing Capabilities**
1. **Speech-to-Text Chirp 3** for call transcription
2. **Vertex AI Gemini 2.5 Flash** for content analysis and lead scoring
3. **Intelligent routing** with spam detection
4. **Service area validation**
5. **Automated CRM integration** with field mapping

---

## ⚡ **Performance Requirements**

### **Latency Targets**
- **<200ms latency** for webhook processing
- **<5s latency** for audio transcription
- **<3s latency** for AI analysis
- **<1s latency** for CRM integration

### **Scalability Requirements**
- **99.9% availability** SLA
- **Auto-scaling** 0-1000 Cloud Run instances
- **Multi-tenant isolation** with row-level security
- **Concurrent processing** of multiple webhook sources

---

## 💰 **Cost Requirements - Budget Management**

### **Monthly Operational Budget**: $4,300-8,700
- **Vertex AI Gemini**: $2,000-4,000/month
- **Cloud Run**: $500-1,000/month
- **Cloud Spanner**: $1,000-2,000/month
- **Speech-to-Text**: $400-800/month
- **Other services**: $400-900/month

### **Cost Optimization Requirements**
- Efficient AI token usage
- Optimized database queries
- Smart caching strategies
- Automated scaling policies

---

## 🔧 **Integration Requirements**

### **CallRail Integration**
- **Webhook Processing**: Real-time post-call webhooks
- **Audio Download**: Automatic recording retrieval
- **HMAC Verification**: Security signature validation
- **Company Mapping**: CallRail company ID to tenant mapping

### **CRM Integration (MCP Framework)**
Dynamic CRM connectors supporting:
- **HubSpot** (primary)
- **Salesforce**
- **Pipedrive**
- **Custom CRM APIs**

### **Email Integration**
- **SendGrid** for notifications
- **Template management**
- **Conditional sending** based on lead scores

---

## 🛡️ **Security Requirements**

### **Multi-Tenant Security**
- **Row-level security** in Cloud Spanner
- **Tenant isolation** for all data access
- **Secure credential management** with Secret Manager

### **API Security**
- **HMAC signature verification** for all webhooks
- **Input validation** and sanitization
- **Rate limiting** and DDoS protection

### **Data Security**
- **Encryption at rest** and in transit
- **Audit logging** for all operations
- **PII handling** compliance

---

## 📊 **Monitoring Requirements**

### **Operational Monitoring**
- **Real-time metrics** dashboard
- **Performance tracking** for AI processing
- **Cost monitoring** and alerting
- **Error tracking** and notification

### **Business Metrics**
- **Lead conversion rates**
- **Processing success rates**
- **AI accuracy metrics**
- **Multi-tenant usage statistics**

---

## 🚀 **2025 Implementation Priorities**

### **Phase 1 (Critical - 2025 Tech Stack)**
1. Latest Go 1.23+ setup with enhanced performance
2. Vertex AI Gemini 2.5 Flash integration (2025 features)
3. Speech-to-Text Chirp 3 implementation (2025 accuracy)
4. Cloud Spanner multi-tenant setup (2025 improvements)

### **Phase 2 (Core Features)**
1. CallRail webhook processing with HMAC verification
2. Audio transcription pipeline
3. AI analysis and lead scoring
4. Multi-tenant database operations

### **Phase 3 (Integration)**
1. MCP CRM integration framework
2. Email notifications with SendGrid
3. Real-time dashboard
4. Performance optimization

### **Phase 4 (Production)**
1. Comprehensive testing (unit, integration, load)
2. Security testing and compliance
3. Production deployment
4. Monitoring and alerting

---

## ✅ **Success Criteria**

### **Technical**
- All performance targets met
- 99.9% availability achieved
- Multi-tenant security validated
- Cost targets maintained

### **Business**
- Successful CallRail integration
- Accurate AI lead scoring
- Seamless CRM integration
- Real-time processing capability

---

**REQUIREMENTS ANALYSIS COMPLETE** ✅
**Next Phase**: 2025 Technology Research & Architecture Planning