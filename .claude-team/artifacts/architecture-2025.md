# 🏗️ Multi-Tenant Ingestion Pipeline - 2025 Architecture Blueprint

## 🎯 **System Architect Report - 2025 Technology Stack**
**Generated**: September 2025 | **Agent**: system-architect | **Duration**: 15min
**Architecture Focus**: Multi-tenant CallRail integration with 2025 GCP services

---

## 🏛️ **Overall Architecture Vision**

### **High-Level System Design**
```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          MULTI-TENANT INGESTION PIPELINE                    │
│                              (2025 Technology Stack)                        │
├─────────────────────────────────────────────────────────────────────────────┤
│  Input Sources                Cloud Run Services              Output Targets │
│  ┌─────────────┐             ┌─────────────────┐             ┌─────────────┐ │
│  │ Website     │────────────▶│  API Gateway    │────────────▶│ HubSpot     │ │
│  │ Forms       │             │  (tenant auth)  │             │ CRM         │ │
│  └─────────────┘             └─────────────────┘             └─────────────┘ │
│  ┌─────────────┐             ┌─────────────────┐             ┌─────────────┐ │
│  │ CallRail    │────────────▶│ Webhook         │────────────▶│ Salesforce  │ │
│  │ Webhooks    │   HMAC      │ Processor       │             │ CRM         │ │
│  └─────────────┘             └─────────────────┘             └─────────────┘ │
│  ┌─────────────┐             ┌─────────────────┐             ┌─────────────┐ │
│  │ Calendar    │────────────▶│ Audio           │────────────▶│ Email       │ │
│  │ Bookings    │             │ Processor       │             │ (SendGrid)  │ │
│  └─────────────┘             └─────────────────┘             └─────────────┘ │
│  ┌─────────────┐             ┌─────────────────┐             ┌─────────────┐ │
│  │ Chat        │────────────▶│ AI Analyzer     │────────────▶│ Custom      │ │
│  │ Widgets     │             │ (Gemini 2.5)    │             │ CRMs        │ │
│  └─────────────┘             └─────────────────┘             └─────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
                            ┌─────────────────┐
                            │ Cloud Spanner   │
                            │ (Multi-tenant   │
                            │  Database)      │
                            └─────────────────┘
```

---

## 🚀 **2025 Cloud Run Microservices Architecture**

### **Service Decomposition Strategy**
Based on existing code structure in `/internal/` and 2025 capabilities:

#### **1. API Gateway Service** (`cmd/api-gateway/`)
- **Responsibility**: Authentication, routing, rate limiting
- **2025 Features**: Enhanced autoscaling, worker pool optimization
- **Scaling**: 0-100 instances, CPU-based scaling
- **Dependencies**: Cloud Spanner (tenant validation)

```yaml
# 2025 Cloud Run Configuration
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: api-gateway
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "100"
        autoscaling.knative.dev/target: "70"
        run.googleapis.com/cpu-throttling: "false"
    spec:
      containerConcurrency: 100
      containers:
      - image: gcr.io/account-strategy-464106/api-gateway:2025
        resources:
          limits:
            cpu: "2"
            memory: "2Gi"
        env:
        - name: GOOGLE_CLOUD_PROJECT
          value: "account-strategy-464106"
```

#### **2. Webhook Processor Service** (`cmd/webhook-processor/`)
- **Responsibility**: CallRail webhook handling, HMAC verification
- **2025 Features**: Worker pools for queue-based processing
- **Scaling**: 0-200 instances, event-driven scaling
- **Security**: HMAC signature validation, tenant isolation

```go
// 2025 Enhanced Webhook Processor
type WebhookProcessor2025 struct {
    tenantRepo     TenantRepository
    audioQueue     chan AudioJob
    workerPool     *WorkerPool2025
    hmacValidator  *HMACValidator
    logger         *zap.Logger
}

// Enhanced HMAC validation with 2025 security standards
func (p *WebhookProcessor2025) ValidateSignature(payload []byte, signature, tenantID string) error {
    secret, err := p.getWebhookSecret(tenantID)
    if err != nil {
        return fmt.Errorf("retrieving webhook secret: %w", err)
    }

    return p.hmacValidator.Verify(payload, signature, secret)
}
```

#### **3. Audio Processor Service** (`cmd/audio-processor/`)
- **Responsibility**: Speech-to-Text Chirp 3 integration
- **2025 Features**: Enhanced diarization, auto-language detection
- **Scaling**: 5-50 instances, memory-intensive processing
- **Performance**: Streaming audio processing, 64KB chunks

```go
// 2025 Chirp 3 Integration
type AudioProcessor2025 struct {
    speechClient *speechv2.Client
    storage      *storage.Client
    config       *Chirp3Config
}

type Chirp3Config struct {
    Model                    string `json:"model"` // "chirp_3_transcription"
    DiarizationEnabled       bool   `json:"diarization_enabled"`
    AutoLanguageDetection    bool   `json:"auto_language_detection"`
    SpeakerLabels           bool   `json:"speaker_labels"`
    EnhancedAccuracy        bool   `json:"enhanced_accuracy"`
}
```

#### **4. AI Analyzer Service** (`cmd/ai-analyzer/`)
- **Responsibility**: Vertex AI Gemini 2.5 Flash analysis
- **2025 Features**: Thinking capabilities, enhanced reasoning
- **Scaling**: 2-100 instances, token-based optimization
- **Intelligence**: Lead scoring, spam detection, sentiment analysis

```go
// 2025 Gemini 2.5 Flash Integration
type AIAnalyzer2025 struct {
    geminiClient *aiplatform.PredictionClient
    config       *Gemini2025Config
    tokenTracker *TokenUsageTracker
}

type Gemini2025Config struct {
    ModelID            string `json:"model_id"` // "gemini-2.5-flash"
    ThinkingEnabled    bool   `json:"thinking_enabled"`
    ThinkingBudget     int    `json:"thinking_budget"`
    SecurityLevel      string `json:"security_level"`
    MaxTokensPerMinute int    `json:"max_tokens_per_minute"`
}
```

#### **5. Workflow Engine Service** (`cmd/workflow-engine/`)
- **Responsibility**: Configurable tenant workflows, CRM integration
- **2025 Features**: MCP framework integration, dynamic routing
- **Scaling**: 1-50 instances, configuration-driven
- **Integration**: HubSpot, Salesforce, Pipedrive, Custom APIs

---

## 🗄️ **2025 Cloud Spanner Multi-Tenant Database Architecture**

### **Enhanced Schema Design for 2025**
Building on existing schema with 2025 improvements:

#### **Core Multi-Tenant Structure**
```sql
-- Enhanced tenants table with 2025 features
CREATE TABLE tenants (
  tenant_id STRING(36) NOT NULL,
  name STRING(100) NOT NULL,
  status STRING(20) DEFAULT 'active',
  -- 2025 vector search capabilities
  embedding_enabled BOOL DEFAULT true,
  graph_features_enabled BOOL DEFAULT true,
  -- Enhanced isolation
  isolation_level STRING(20) DEFAULT 'STRICT',
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  PRIMARY KEY(tenant_id)
);

-- Enhanced offices table with CallRail integration
CREATE TABLE offices (
  tenant_id STRING(36) NOT NULL,
  office_id STRING(36) NOT NULL,
  name STRING(100) NOT NULL,
  -- CallRail integration fields
  callrail_company_id STRING(50),
  callrail_api_key STRING(100),
  -- 2025 enhanced workflow configuration
  workflow_config JSON,
  -- 2025 vector search for office data
  office_embedding ARRAY<FLOAT64>,
  status STRING(20) DEFAULT 'active',
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  FOREIGN KEY(tenant_id) REFERENCES tenants(tenant_id),
  PRIMARY KEY(tenant_id, office_id)
) INTERLEAVE IN PARENT tenants ON DELETE CASCADE;
```

#### **2025 Enhanced Request Processing**
```sql
-- Requests table with 2025 AI capabilities
CREATE TABLE requests (
  tenant_id STRING(36) NOT NULL,
  request_id STRING(36) NOT NULL,
  office_id STRING(36) NOT NULL,

  -- Core request data
  communication_mode STRING(20) NOT NULL, -- form, phone_call, calendar, chat
  caller_name STRING(100),
  caller_phone STRING(20),
  caller_email STRING(100),

  -- CallRail integration
  call_id STRING(50),
  recording_url STRING(500),
  audio_duration_seconds INT64,

  -- 2025 AI analysis with Gemini 2.5 Flash
  transcription_data JSON,
  ai_analysis JSON,
  lead_score INT64,
  spam_likelihood FLOAT64,
  sentiment_score FLOAT64,

  -- 2025 vector search capabilities
  content_embedding ARRAY<FLOAT64>,
  semantic_tags ARRAY<STRING(50)>,

  -- Processing metadata
  processing_time_ms INT64,
  gemini_model_used STRING(50) DEFAULT 'gemini-2.5-flash',
  speech_model_used STRING(50) DEFAULT 'chirp_3_transcription',

  -- Enhanced timestamps
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  processed_at TIMESTAMP,

  FOREIGN KEY(tenant_id) REFERENCES tenants(tenant_id),
  FOREIGN KEY(tenant_id, office_id) REFERENCES offices(tenant_id, office_id),
  PRIMARY KEY(tenant_id, request_id)
) INTERLEAVE IN PARENT tenants ON DELETE CASCADE;
```

#### **2025 Vector Search Integration**
```sql
-- 2025 vector search indexes for AI-powered queries
CREATE VECTOR INDEX requests_content_embedding_index
ON requests(content_embedding)
OPTIONS (
  distance_type = 'COSINE',
  approximate = true
);

-- Graph relationships for enhanced analytics
CREATE OR REPLACE PROPERTY GRAPH multi_tenant_pipeline_graph
NODE TABLES(
  tenants KEY(tenant_id) LABEL Tenant,
  offices KEY(tenant_id, office_id) LABEL Office,
  requests KEY(tenant_id, request_id) LABEL Request
)
EDGE TABLES(
  offices SOURCE KEY(tenant_id) REFERENCES tenants(tenant_id)
          DESTINATION KEY(tenant_id) REFERENCES tenants(tenant_id)
          LABEL OWNS,
  requests SOURCE KEY(tenant_id, office_id) REFERENCES offices(tenant_id, office_id)
           DESTINATION KEY(tenant_id, request_id) REFERENCES requests(tenant_id, request_id)
           LABEL PROCESSES
);
```

---

## 🔐 **2025 Security Architecture**

### **Multi-Tenant Isolation Strategy**
```sql
-- 2025 enhanced row-level security
CREATE ROW ACCESS POLICY tenant_isolation_requests ON requests
  GRANT TO ('application_role')
  FILTER USING (tenant_id = @tenant_id_param);

CREATE ROW ACCESS POLICY tenant_isolation_offices ON offices
  GRANT TO ('application_role')
  FILTER USING (tenant_id = @tenant_id_param);
```

### **HMAC Signature Validation (2025 Enhanced)**
```go
// 2025 security standards implementation
type HMACValidator2025 struct {
    secretManager *secretmanager.Client
    logger        *zap.Logger
    rateLimiter   *RateLimiter
}

func (v *HMACValidator2025) Verify(payload []byte, signature, tenantID string) error {
    // Rate limiting for security
    if !v.rateLimiter.Allow(tenantID) {
        return errors.New("rate limit exceeded for tenant")
    }

    // Retrieve secret from Secret Manager
    secret, err := v.getWebhookSecret(tenantID)
    if err != nil {
        return fmt.Errorf("retrieving webhook secret: %w", err)
    }

    // Enhanced HMAC verification with timing attack protection
    return v.verifyHMACConstantTime(payload, signature, secret)
}
```

---

## 🤖 **2025 AI Integration Architecture**

### **Vertex AI Gemini 2.5 Flash Integration**
```go
// 2025 enhanced AI analysis pipeline
type AIAnalysisPipeline2025 struct {
    geminiClient     *aiplatform.PredictionClient
    speechClient     *speechv2.Client
    vectorSearch     *VectorSearchClient
    thinkingTracker  *ThinkingUsageTracker
}

func (p *AIAnalysisPipeline2025) AnalyzeCallContent(ctx context.Context, request *models.Request) (*models.AIAnalysis, error) {
    // 1. Speech-to-Text with Chirp 3
    transcription, err := p.transcribeAudio(ctx, request.RecordingURL)
    if err != nil {
        return nil, fmt.Errorf("transcribing audio: %w", err)
    }

    // 2. Gemini 2.5 Flash analysis with thinking capabilities
    analysis, err := p.analyzeWithGemini(ctx, transcription, true) // thinking enabled
    if err != nil {
        return nil, fmt.Errorf("AI analysis: %w", err)
    }

    // 3. Vector embedding for semantic search
    embedding, err := p.generateEmbedding(ctx, transcription.Text)
    if err != nil {
        return nil, fmt.Errorf("generating embedding: %w", err)
    }

    return &models.AIAnalysis{
        LeadScore:       analysis.LeadScore,
        SpamLikelihood:  analysis.SpamLikelihood,
        SentimentScore:  analysis.SentimentScore,
        ContentEmbedding: embedding,
        ThinkingProcess: analysis.ThinkingSteps, // 2025 feature
        ConfidenceScore: analysis.Confidence,
    }, nil
}
```

### **Enhanced Chirp 3 Configuration**
```go
// 2025 Speech-to-Text configuration
type SpeechConfig2025 struct {
    Model:                   "chirp_3_transcription",
    LanguageCode:           "en-US",
    DiarizationEnabled:     true,
    AutoLanguageDetection:  true,
    SpeakerLabels:          true,
    EnhancedAccuracy:       true,
    WordLevelTimestamps:    true,
    PunctuationEnabled:     true,
    ProfanityFilter:        false, // Keep original for analysis
}
```

---

## 🔄 **2025 Workflow Engine Architecture**

### **MCP Framework Integration**
```go
// 2025 MCP-based CRM integration
type MCPWorkflowEngine2025 struct {
    crmConnectors map[string]MCPConnector
    ruleEngine    *WorkflowRuleEngine
    eventBus      *EventBus
}

type MCPConnector interface {
    CreateLead(ctx context.Context, leadData *models.LeadData) error
    UpdateContact(ctx context.Context, contactID string, updates map[string]interface{}) error
    ValidateConnection(ctx context.Context) error
}

// Dynamic CRM integration based on tenant configuration
func (e *MCPWorkflowEngine2025) ProcessWorkflow(ctx context.Context, request *models.Request, analysis *models.AIAnalysis) error {
    workflow, err := e.getWorkflowConfig(ctx, request.TenantID)
    if err != nil {
        return fmt.Errorf("getting workflow config: %w", err)
    }

    // Execute workflow steps based on configuration
    for _, step := range workflow.Steps {
        if err := e.executeWorkflowStep(ctx, step, request, analysis); err != nil {
            return fmt.Errorf("executing workflow step %s: %w", step.Type, err)
        }
    }

    return nil
}
```

---

## 📊 **2025 Monitoring & Observability Architecture**

### **Enhanced Metrics Collection**
```go
// 2025 monitoring configuration
type MonitoringConfig2025 struct {
    // Vertex AI metrics
    GeminiTokenUsage      bool `json:"gemini_token_usage"`
    GeminiThinkingMetrics bool `json:"gemini_thinking_metrics"`

    // Speech-to-Text metrics
    Chirp3AccuracyMetrics bool `json:"chirp3_accuracy_metrics"`
    AudioProcessingTimes  bool `json:"audio_processing_times"`

    // Spanner metrics
    VectorSearchMetrics   bool `json:"vector_search_metrics"`
    GraphQueryMetrics     bool `json:"graph_query_metrics"`

    // Cloud Run metrics
    WorkerPoolMetrics     bool `json:"worker_pool_metrics"`
    ScalingMetrics        bool `json:"scaling_metrics"`
}
```

### **Real-time Dashboard Configuration**
```json
{
  "dashboard_2025": {
    "ai_processing_metrics": {
      "gemini_thinking_usage": true,
      "chirp3_accuracy_scores": true,
      "vector_search_performance": true
    },
    "multi_tenant_metrics": {
      "tenant_isolation_compliance": true,
      "per_tenant_costs": true,
      "tenant_scaling_patterns": true
    },
    "cost_optimization": {
      "token_usage_optimization": true,
      "scaling_efficiency": true,
      "storage_optimization": true
    }
  }
}
```

---

## 🎯 **2025 Performance Targets**

### **Enhanced SLA Requirements**
- **Webhook Processing**: <100ms (improved from 200ms)
- **Audio Transcription**: <3s (improved from 5s)
- **AI Analysis**: <2s (improved from 3s)
- **CRM Integration**: <500ms (improved from 1s)
- **Vector Search**: <50ms for similarity queries
- **Graph Queries**: <100ms for relationship analysis

### **Scalability Targets**
- **Concurrent Requests**: 10,000+ simultaneous webhook processing
- **Audio Processing**: 500+ concurrent transcriptions
- **Database Operations**: 50,000+ QPS with multi-tenant isolation
- **Vector Search**: 1,000+ similarity searches per second

---

## 🚀 **Deployment Strategy 2025**

### **Infrastructure as Code (Enhanced)**
```yaml
# 2025 Terraform configuration
resource "google_cloud_run_v2_service" "ingestion_pipeline" {
  count    = length(var.services_2025)
  name     = var.services_2025[count.index].name
  location = var.region

  template {
    scaling {
      min_instance_count = var.services_2025[count.index].min_instances
      max_instance_count = var.services_2025[count.index].max_instances
    }

    # 2025 enhanced configuration
    service_account = google_service_account.pipeline_sa.email

    containers {
      image = "gcr.io/${var.project_id}/${var.services_2025[count.index].name}:2025"

      resources {
        limits = {
          cpu    = var.services_2025[count.index].cpu_limit
          memory = var.services_2025[count.index].memory_limit
        }
      }

      # 2025 environment variables
      env {
        name  = "GEMINI_MODEL"
        value = "gemini-2.5-flash"
      }

      env {
        name  = "SPEECH_MODEL"
        value = "chirp_3_transcription"
      }
    }
  }
}
```

---

**2025 ARCHITECTURE BLUEPRINT COMPLETE** ✅
**Next Phase**: Backend Implementation & Testing Framework Setup