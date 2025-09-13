# 🚀 Multi-Tenant Ingestion Pipeline - Complete Implementation Guide

## 📋 **Project Overview**

**Mission**: Build a cutting-edge multi-tenant ingestion pipeline for home remodeling companies that processes forms, phone calls (via CallRail), calendar bookings, and chat interactions through an intelligent Go/Gemini agent with configurable workflows.

**Selected Solution**: **Standard Solution** - Balanced performance with advanced features
**Budget**: $4,300-8,700/month | **Timeline**: 26 weeks | **Team**: 5-7 engineers

---

## 🏗️ **Architecture Overview**

### **Input Sources**
- 🌐 **Website Forms** → API Gateway processing
- 📞 **CallRail Webhooks** → Post-call webhook with recording download
- 📅 **Calendar Bookings** → Calendar API integration
- 💬 **Chat Widgets** → Real-time chat processing

### **Core Processing Flow**
```
Input → Cloud Load Balancer → Cloud Run (Go/Gemini Agent) →
tenant_id Authentication → Communication Detection →
AI Processing → Configurable Workflow → CRM/Email/Storage
```

### **Authentication Method**
- ❌ **NO API Keys** required in headers
- ✅ **tenant_id** in JSON payload for authentication
- ✅ **CallRail company ID** mapping in office settings
- 🔐 **HMAC signature verification** for CallRail webhooks

---

## 🗄️ **Database Architecture**

### **Existing Cloud Spanner Instance**
- **Project**: `account-strategy-464106`
- **Location**: `us-central1`
- **Instance**: `upai-customers` (Enterprise, us-central1, Autoscaling)
- **Database**: `agent_platform` (Google Standard SQL)

### **Required Schema Updates**

#### **Enhanced Offices Table**
```sql
-- Add CallRail integration fields
ALTER TABLE offices ADD COLUMN callrail_company_id STRING(50);
ALTER TABLE offices ADD COLUMN callrail_api_key STRING(100);
ALTER TABLE offices ADD COLUMN workflow_config JSON;

-- Indexes
CREATE INDEX idx_offices_callrail ON offices(callrail_company_id, tenant_id);
```

#### **Enhanced Requests Table**
```sql
-- Add CallRail and processing fields
ALTER TABLE requests ADD COLUMN call_id STRING(50);
ALTER TABLE requests ADD COLUMN recording_url STRING(500);
ALTER TABLE requests ADD COLUMN transcription_data JSON;
ALTER TABLE requests ADD COLUMN ai_analysis JSON;
ALTER TABLE requests ADD COLUMN lead_score INT64;
ALTER TABLE requests ADD COLUMN communication_mode STRING(20);
ALTER TABLE requests ADD COLUMN spam_likelihood FLOAT64;

-- Indexes
CREATE INDEX idx_requests_call_id ON requests(call_id);
CREATE INDEX idx_requests_lead_score ON requests(tenant_id, lead_score DESC);
```

#### **New Tables**
```sql
-- Call recording management
CREATE TABLE call_recordings (
  recording_id STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
  tenant_id STRING(36) NOT NULL,
  call_id STRING(50) NOT NULL,
  storage_url STRING(500) NOT NULL,
  transcription_status STRING(20) DEFAULT 'pending',
  created_at TIMESTAMP NOT NULL,
  PRIMARY KEY(tenant_id, recording_id)
) INTERLEAVE IN PARENT tenants ON DELETE CASCADE;

-- Webhook event tracking
CREATE TABLE webhook_events (
  event_id STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
  webhook_source STRING(50) NOT NULL,
  call_id STRING(50),
  processing_status STRING(20) NOT NULL DEFAULT 'received',
  created_at TIMESTAMP NOT NULL,
  PRIMARY KEY(event_id)
);
```

### **Workflow Configuration JSON**
```json
{
  "communication_detection": {
    "enabled": true,
    "phone_processing": {
      "transcribe_audio": true,
      "extract_details": true,
      "sentiment_analysis": true,
      "speaker_diarization": true
    }
  },
  "validation": {
    "spam_detection": {
      "enabled": true,
      "confidence_threshold": 75,
      "ml_model": "gemini-2.5-flash"
    }
  },
  "service_area": {
    "enabled": true,
    "validation_method": "zip_code",
    "allowed_areas": ["90210", "90211"],
    "buffer_miles": 25
  },
  "crm_integration": {
    "enabled": true,
    "provider": "hubspot",
    "field_mapping": {
      "name": "firstname",
      "phone": "phone",
      "lead_score": "hs_lead_score"
    },
    "push_immediately": true
  },
  "email_notifications": {
    "enabled": true,
    "recipients": ["sales@company.com"],
    "conditions": {
      "min_lead_score": 30
    }
  }
}
```

---

## 🔄 **CallRail Integration Flow**

### **Step 1: Webhook Reception**
**Endpoint**: `POST /api/v1/callrail/webhook`

**Expected Payload**:
```json
{
  "call_id": "CAL123456789",
  "caller_id": "+15551234567",
  "duration": "180",
  "recording_url": "https://api.callrail.com/...",
  "tenant_id": "tenant_12345",
  "callrail_company_id": "12345"
}
```

### **Step 2: Authentication & Verification**
```go
// Verify HMAC signature
signature := r.Header.Get("x-callrail-signature")
if !verifyHMACSignature(payload, signature, webhookSecret) {
    return http.StatusUnauthorized
}

// Query tenant mapping
office := queryOfficeByCallRailCompanyID(callrailCompanyID, tenantID)
if office == nil {
    return errors.New("invalid tenant mapping")
}
```

### **Step 3: Call Data Retrieval**
```go
// Get call details from CallRail API
callDetails := getCallDetails(accountID, callID, office.CallRailAPIKey)

// Download recording
recordingData := downloadRecording(callDetails.RecordingURL, office.CallRailAPIKey)

// Store in Cloud Storage
storageURL := storeAudioFile(tenantID, callID, recordingData)
```

### **Step 4: Audio Processing**
```go
// Transcribe with Speech-to-Text Chirp 3
transcription := transcribeAudio(storageURL, speechConfig{
    EnableSpeakerDiarization: true,
    EnableAutomaticPunctuation: true,
    LanguageCode: "en-US",
})

// AI analysis with Gemini 2.5 Flash
analysis := analyzeCallContent(transcription, callDetails)
```

### **Step 5: Enhanced Payload Creation**
```json
{
  "tenant_id": "tenant_12345",
  "source": "callrail_webhook",
  "communication_mode": "phone_call",
  "call_details": {
    "customer_name": "John Smith",
    "customer_phone": "+15551234567",
    "duration": 180
  },
  "audio_processing": {
    "recording_url": "gs://tenant-audio-files/...",
    "transcription": "Hi, I'm interested in kitchen remodel...",
    "confidence": 0.95
  },
  "ai_analysis": {
    "intent": "quote_request",
    "project_type": "kitchen",
    "lead_score": 85,
    "sentiment": "positive",
    "urgency": "medium"
  },
  "spam_likelihood": 5
}
```

---

## 🚀 **Google Cloud Services Stack**

### **Core Services**
| Service | Purpose | Configuration |
|---------|---------|---------------|
| **Cloud Run** | Go microservices hosting | 2nd gen, 4 vCPU, 16GB RAM, 0-1000 instances, us-central1 |
| **Cloud Spanner** | Multi-tenant database | Enterprise, autoscaling, row-level security, us-central1 |
| **Vertex AI Gemini 2.5 Flash** | AI content analysis | Enterprise with data residency, us-central1 |
| **Speech-to-Text Chirp 3** | Audio transcription | Real-time, speaker diarization, us-central1 |
| **Document AI v1.5** | Form processing | 30 pages/min processing, us-central1 |
| **Cloud Storage** | Audio file storage | Regional buckets, lifecycle policies, us-central1 |
| **Cloud Tasks** | Workflow orchestration | Queue-based processing, us-central1 |
| **Secret Manager** | API credential storage | Regional replication, us-central1 |

### **Integration Services**
| Service | Purpose | Usage |
|---------|---------|--------|
| **API Gateway** | Request routing | Rate limiting, SSL termination, us-central1 |
| **Cloud Load Balancer** | Traffic distribution | Global load balancing, us-central1 backend |
| **Cloud Monitoring** | Performance metrics | SLOs, alerting, dashboards, us-central1 |
| **Cloud Logging** | Audit trails | Structured logging, us-central1 |
| **Maps API** | Geographic validation | Service area verification, us-central1 |

### **Cost Breakdown (Monthly)**
- Vertex AI Gemini 2.5 Flash: $2,000-4,000
- Cloud Run: $500-1,000
- Cloud Spanner: $1,000-2,000
- Speech-to-Text Chirp 3: $400-800
- Document AI v1.5: $200-400
- Cloud Storage: $50-100
- Other services: $500-800
- **CallRail Integration**: +$62 per tenant
- **Total**: $4,300-8,700/month

---

## 📅 **Implementation Timeline (26 Weeks)**

### **Phase 1: Infrastructure Setup (Weeks 1-4)**
**Team**: Cloud Architect (1 FTE), DevOps Engineer (1 FTE)

**Tasks**:
- [ ] Configure GCP project with standard billing
- [ ] Set up Cloud Run services with autoscaling
- [ ] Optimize Cloud Spanner for multi-tenancy
- [ ] Configure Vertex AI Gemini 2.5 Flash quotas
- [ ] Set up Speech-to-Text and Document AI APIs
- [ ] Configure monitoring and logging
- [ ] Establish CI/CD pipeline

**Deliverables**:
- ✅ Standard GCP infrastructure operational
- ✅ AI services configured and tested
- ✅ Development environment ready

### **Phase 2: Core Development (Weeks 5-12)**
**Team**: Backend Engineers (2 FTE), ML Engineer (1 FTE), Frontend Engineer (1 FTE)

**Go Microservices**:
- [ ] Tenant configuration service with Spanner integration
- [ ] CallRail webhook processing with HMAC verification
- [ ] Audio download and transcription service
- [ ] AI-powered call analysis service
- [ ] Document processing service (forms)
- [ ] Spam detection service
- [ ] Service area validation service

**Database Updates**:
- [ ] Apply schema updates for CallRail integration
- [ ] Implement row-level security policies
- [ ] Create tenant isolation mechanisms
- [ ] Set up data archival policies

**Workflow Engine**:
- [ ] Cloud Tasks-based orchestration
- [ ] MCP framework for CRM integration
- [ ] SendGrid integration for notifications
- [ ] Configuration-driven processing

### **Phase 3: Integration & Features (Weeks 13-18)**
**Team**: Backend Engineers (2 FTE), Integration Specialist (1 FTE)

**Advanced Processing**:
- [ ] Intelligent content routing
- [ ] Confidence scoring for spam detection
- [ ] Geographic boundary validation
- [ ] Tenant-specific configuration management
- [ ] Real-time analytics and reporting

**External Integrations**:
- [ ] Multiple CRM connectors (HubSpot, Salesforce)
- [ ] Email template management
- [ ] Audit logging and compliance
- [ ] Data export capabilities

**Performance Optimization**:
- [ ] Caching strategies for configurations
- [ ] Database query optimization
- [ ] Auto-scaling policies
- [ ] Request rate limiting

### **Phase 4: Testing & QA (Weeks 19-22)**
**Team**: QA Lead (1 FTE), Performance Engineer (0.5 FTE)

**Testing Suite**:
- [ ] Unit testing for all microservices
- [ ] Integration testing with CallRail API
- [ ] Load testing with realistic scenarios
- [ ] Multi-tenant isolation testing
- [ ] Security testing and vulnerability assessment

**Performance Validation**:
- [ ] Latency testing (<200ms target)
- [ ] Auto-scaling behavior validation
- [ ] Audio processing optimization
- [ ] End-to-end workflow testing

### **Phase 5: Production Deployment (Weeks 23-26)**
**Team**: DevOps Lead (1 FTE), SRE Engineer (0.5 FTE)

**Production Setup**:
- [ ] Production environment configuration
- [ ] Monitoring and alerting setup
- [ ] Backup and disaster recovery
- [ ] Security hardening

**Go-Live**:
- [ ] Staged rollout to pilot tenants
- [ ] Performance baseline establishment
- [ ] Support team training
- [ ] Documentation finalization

---

## 🛠️ **Technical Implementation Details**

### **Go Application Structure**
```
cmd/
├── webhook-processor/     # CallRail webhook handler
├── audio-processor/       # Speech-to-Text service
├── ai-analyzer/          # Gemini content analysis
├── workflow-engine/       # Task orchestration
└── api-gateway/          # Main API server

internal/
├── auth/                 # Tenant authentication
├── callrail/            # CallRail API client
├── spanner/             # Database operations
├── storage/             # Cloud Storage operations
├── ai/                  # Vertex AI integration
└── workflow/            # Workflow processing

pkg/
├── models/              # Data structures
├── config/              # Configuration management
└── utils/               # Shared utilities
```

### **Key Configuration Files**
```yaml
# config/app.yaml
project_id: "account-strategy-464106"
location: "us-central1"
spanner_instance: "upai-customers"
spanner_database: "agent_platform"

# AI Services
vertex_ai:
  project: "account-strategy-464106"
  location: "us-central1"
  model: "gemini-2.5-flash"

speech_to_text:
  project: "account-strategy-464106"
  location: "us-central1"
  model: "chirp-3"
  language: "en-US"
  enable_diarization: true

document_ai:
  project: "account-strategy-464106"
  location: "us-central1"
  processor_version: "v1.5"

# Storage
cloud_storage:
  project: "account-strategy-464106"
  audio_bucket: "tenant-audio-files"
  location: "us-central1"
  retention_days: 2555

# Security
webhook_secrets:
  callrail: "projects/account-strategy-464106/secrets/callrail-webhook-secret"

# Networking
cloud_run:
  project: "account-strategy-464106"
  region: "us-central1"

cloud_tasks:
  project: "account-strategy-464106"
  location: "us-central1"
```

### **Environment Variables**
```bash
# Required for all services
export GOOGLE_CLOUD_PROJECT="account-strategy-464106"
export GOOGLE_CLOUD_LOCATION="us-central1"
export SPANNER_INSTANCE="upai-customers"
export SPANNER_DATABASE="agent_platform"

# AI Services
export VERTEX_AI_PROJECT="account-strategy-464106"
export VERTEX_AI_LOCATION="us-central1"
export SPEECH_TO_TEXT_PROJECT="account-strategy-464106"
export DOCUMENT_AI_PROJECT="account-strategy-464106"

# Service-specific
export CALLRAIL_WEBHOOK_SECRET_NAME="callrail-webhook-secret"
export AUDIO_STORAGE_BUCKET="tenant-audio-files"
export CLOUD_RUN_REGION="us-central1"
export CLOUD_TASKS_LOCATION="us-central1"
```

---

## 🔐 **Security Implementation**

### **Authentication & Authorization**
- **Tenant Isolation**: Row-level security in Cloud Spanner
- **API Security**: HMAC signature verification for webhooks
- **Credential Management**: Google Secret Manager for API keys
- **Network Security**: VPC Service Controls and private networking

### **Data Protection**
- **Encryption**: Customer-managed encryption keys (CMEK)
- **Access Control**: IAM with least privilege principles
- **Audit Logging**: Comprehensive activity tracking
- **Data Loss Prevention**: Automated sensitive data detection

### **Compliance Features**
- **GDPR**: Data retention and deletion policies
- **SOC 2**: Security controls and monitoring
- **HIPAA**: Healthcare data protection (if applicable)
- **Industry Standards**: Secure coding practices

---

## 📊 **Monitoring & Performance**

### **Key Metrics**
- **Latency**: <200ms for forms, <1s for AI analysis
- **Throughput**: 1,000+ requests/minute per tenant
- **Availability**: 99.9% SLA target
- **Audio Processing**: <5s transcription latency
- **Cost**: Stay within $4,300-8,700/month budget

### **Alerting Conditions**
- Webhook processing failures (>5%)
- High latency (>500ms average)
- Audio download failures
- Spam detection anomalies
- CRM integration errors
- Cost threshold breaches

### **Dashboard Components**
- Real-time request processing
- Tenant-specific performance metrics
- Cost analysis and projections
- AI model performance tracking
- Error rates and resolution times

---

## 🧪 **Testing Strategy**

### **Unit Testing**
- Go service unit tests (>90% coverage)
- Database operation tests
- AI service integration tests
- Error handling validation

### **Integration Testing**
- CallRail webhook end-to-end flow
- Multi-tenant isolation verification
- CRM integration testing
- Email notification validation

### **Load Testing**
- Concurrent webhook processing
- Audio transcription at scale
- Database performance under load
- Auto-scaling behavior validation

### **Security Testing**
- Penetration testing
- Vulnerability scanning
- Access control verification
- Data encryption validation

---

## 📚 **Documentation Requirements**

### **Technical Documentation**
- [ ] API specification (OpenAPI/Swagger)
- [ ] Database schema documentation
- [ ] Deployment procedures
- [ ] Configuration management guide
- [ ] Troubleshooting runbook

### **User Documentation**
- [ ] Tenant onboarding guide
- [ ] Webhook configuration instructions
- [ ] Dashboard user manual
- [ ] CRM integration setup
- [ ] Billing and cost management

### **Operational Documentation**
- [ ] Monitoring and alerting guide
- [ ] Incident response procedures
- [ ] Backup and recovery procedures
- [ ] Performance tuning guide
- [ ] Security best practices

---

## 🚀 **Deployment Strategy**

### **Environment Setup**
```bash
# Development Environment Setup
gcloud config set project account-strategy-464106
gcloud config set compute/region us-central1

# Enable required APIs
gcloud services enable run.googleapis.com
gcloud services enable spanner.googleapis.com
gcloud services enable aiplatform.googleapis.com
gcloud services enable speech.googleapis.com
gcloud services enable documentai.googleapis.com
gcloud services enable storage.googleapis.com
gcloud services enable cloudtasks.googleapis.com
gcloud services enable secretmanager.googleapis.com

# Apply database schema updates
gcloud spanner databases ddl update agent_platform \
  --instance=upai-customers \
  --ddl-file=database-schema-updates.sql \
  --project=account-strategy-464106

# Create Cloud Storage bucket for audio files
gsutil mb -p account-strategy-464106 \
  -c STANDARD \
  -l us-central1 \
  gs://tenant-audio-files

# Deploy main webhook processor service
gcloud run deploy webhook-processor \
  --source=cmd/webhook-processor \
  --region=us-central1 \
  --project=account-strategy-464106 \
  --allow-unauthenticated \
  --memory=1Gi \
  --cpu=2 \
  --max-instances=100

# Deploy audio processing service
gcloud run deploy audio-processor \
  --source=cmd/audio-processor \
  --region=us-central1 \
  --project=account-strategy-464106 \
  --memory=2Gi \
  --cpu=4 \
  --timeout=900s
```

### **Production Checklist**
- [ ] Database schema applied and tested
- [ ] All services deployed and healthy
- [ ] Monitoring and alerting configured
- [ ] Security policies applied
- [ ] Backup procedures verified
- [ ] Performance baselines established
- [ ] Documentation complete
- [ ] Team training completed

---

## 💡 **Success Criteria**

### **Technical Success**
- ✅ 99.9% availability SLA achieved
- ✅ <200ms latency for standard requests
- ✅ 100-500 concurrent tenants supported
- ✅ CallRail integration fully operational
- ✅ Audio processing with <5s latency
- ✅ Multi-tenant isolation verified

### **Business Success**
- ✅ Cost targets maintained ($4,300-8,700/month)
- ✅ Lead processing accuracy >95%
- ✅ CRM integration success rate >99%
- ✅ Customer satisfaction >90%
- ✅ Time to value <30 days for new tenants

### **Operational Success**
- ✅ Zero-downtime deployments achieved
- ✅ Incident response time <15 minutes
- ✅ Data consistency 99.99%
- ✅ Team productivity maintained
- ✅ Knowledge transfer completed

---

## 📋 **Project Files Reference**

### **Planning Documents**
- `/home/brandon/pipe/.claude-team/planning/options/standard-solution.md` - Selected solution details
- `/home/brandon/pipe/.claude-team/planning/phases/standard-phases.md` - Implementation timeline
- `/home/brandon/pipe/.claude-team/planning/quality-review.md` - Quality assurance plan

### **Technical Documentation**
- `/home/brandon/pipe/callrail-integration-flow.md` - Complete CallRail integration process
- `/home/brandon/pipe/multi-tenant-ingestion-flowchart.md` - Architecture flowchart
- `/home/brandon/pipe/google-cloud-services-mapping.md` - GCP services configuration
- `/home/brandon/pipe/database-schema-updates.sql` - Database schema changes

### **Research Files**
- `/home/brandon/pipe/gcp-services-research-2025.md` - Latest GCP services research
- `/home/brandon/pipe/github-research-analysis.md` - Implementation patterns analysis
- `/home/brandon/pipe/learning-materials-research.md` - Technical learning resources
- `/home/brandon/pipe/library-research-findings.md` - Go library recommendations

---

## 🎯 **Next Steps for Implementation Team**

1. **Week 1**: Review all documentation and establish development environment
2. **Week 2**: Begin Phase 1 infrastructure setup following the timeline
3. **Week 3**: Start database schema updates and basic service development
4. **Week 4**: Implement CallRail webhook processing and authentication
5. **Week 5**: Begin audio processing and AI integration development

**Key Focus Areas**:
- Prioritize CallRail integration for immediate value
- Ensure proper tenant isolation and security from day one
- Implement comprehensive monitoring and alerting early
- Maintain cost discipline throughout development
- Plan for scalability and future enhancements

This comprehensive guide provides everything needed for your development team to successfully implement the multi-tenant ingestion pipeline with CallRail integration. The Standard Solution offers the optimal balance of advanced features, reasonable costs, and manageable complexity for your home remodeling company use case.