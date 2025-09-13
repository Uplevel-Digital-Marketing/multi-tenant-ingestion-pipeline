# Multi-Tenant Ingestion Pipeline Architecture Flowchart

```mermaid
graph TD
    %% Input Sources
    A1[🌐 Website Form] --> B[🔄 API Gateway]
    A2[📞 CallRail Webhook<br/>Post-Call Event] --> B
    A3[📅 Calendar Booking] --> B
    A4[💬 Chat Widget] --> B

    %% API Gateway & Load Balancer
    B --> C[☁️ Cloud Load Balancer]
    C --> D[🚀 Cloud Run Service<br/>Gemini/Go Agent]

    %% Initial Processing & Authentication
    D --> E{Extract tenant_id<br/>from JSON payload}
    E -->|Success| E1[🔐 HMAC Signature<br/>Verification (CallRail)]
    E1 --> F[🗄️ Cloud Spanner Query<br/>Get Tenant Config + CallRail ID]
    E -->|Missing/Invalid| G[❌ Reject Request<br/>Log Error]

    %% Tenant Configuration
    F --> H[📋 Load workflow_config JSON<br/>From offices table]
    H --> I[🎯 Initialize Workflow Engine]

    %% Step 1: Communication Detection
    I --> J{Communication Mode<br/>Detection}
    J -->|Form| K1[📝 Form Processor<br/>Document AI v1.5]
    J -->|Phone Call| K2[🎤 Audio Processor<br/>Speech-to-Text Chirp 3]
    J -->|Calendar| K3[📅 Calendar Processor<br/>Calendar API]
    J -->|Chat| K4[💬 Chat Processor<br/>Dialogflow CX]

    %% CallRail Audio Processing Branch
    K2 --> L0[📡 CallRail API<br/>Get Call Details]
    L0 --> L1[⬇️ Download Recording<br/>from CallRail API]
    L1 --> L2[🎵 Audio File Storage<br/>Cloud Storage]
    L2 --> L3[🗣️ Speech-to-Text Chirp 3<br/>Speaker Diarization]
    L3 --> L4[🧠 Gemini Content Analysis<br/>Intent & Lead Scoring]
    L4 --> L5[📊 AI Analysis Results<br/>Sentiment & Project Details]
    L5 --> M[📝 Enhanced JSON Payload<br/>with Call Intelligence]

    %% Form Processing Branch
    K1 --> N1[✅ Field Validation]
    N1 --> N2[🔍 Auto-complete Fields<br/>Document AI]
    N2 --> M

    %% Calendar Processing Branch
    K3 --> O1[📅 Sync Validation]
    O1 --> O2[⚠️ Conflict Detection]
    O2 --> M

    %% Chat Processing Branch
    K4 --> P1[💬 Message Analysis]
    P1 --> M

    %% Step 2: Spam Validation
    M --> Q[🤖 Spam Detection<br/>Gemini 2.5 Flash ML]
    Q --> R{Spam Likelihood<br/>Check}
    R -->|High (>85%)| S1[🚫 Mark as Spam<br/>Quarantine]
    R -->|Medium (50-85%)| S2[⚠️ Flag for Review<br/>Human Queue]
    R -->|Low (<50%)| T[✅ Continue Processing]

    %% Step 3: Service Area Validation
    T --> U[🗺️ Geographic Validation<br/>Maps API]
    U --> V{Service Area<br/>Check}
    V -->|Outside Area| W1[📍 Route to Partner<br/>or Reject]
    V -->|In Area| X[✅ Area Validated]

    %% Step 4: CRM Integration
    X --> Y[🔐 Load CRM Credentials<br/>Secret Manager]
    Y --> Z[🔌 Initialize CRM MCP<br/>Dynamic Provider]
    Z --> AA[📋 Get CRM Schema<br/>Field Mapping]
    AA --> BB[🔄 Transform Data<br/>JSON Mapping]
    BB --> CC[📤 Push to CRM<br/>API Integration]
    CC --> DD{CRM Push<br/>Success?}
    DD -->|Success| EE[✅ Log Success<br/>crm_integrations table]
    DD -->|Failure| FF[❌ Log Error<br/>Retry Queue]

    %% Step 5: Email Notifications
    EE --> GG[📧 SendGrid MCP<br/>Load Templates]
    GG --> HH{Email Conditions<br/>Met?}
    HH -->|Yes| II[📬 Send Notifications<br/>Multiple Recipients]
    HH -->|No| JJ[⏭️ Skip Email]

    %% Step 6: Database Storage
    II --> KK[🗄️ Insert to Spanner<br/>requests table]
    JJ --> KK

    %% Final Processing
    KK --> LL[📊 Update Analytics<br/>BigQuery Streaming]
    LL --> MM[✅ Process Complete<br/>Return Success]

    %% Error Handling
    FF --> NN[🔄 Retry Logic<br/>Exponential Backoff]
    NN --> OO{Max Retries<br/>Reached?}
    OO -->|No| CC
    OO -->|Yes| PP[❌ Dead Letter Queue<br/>Manual Review]

    %% Monitoring & Logging
    MM --> QQ[📈 Cloud Monitoring<br/>Metrics & Alerts]
    PP --> QQ
    S1 --> QQ
    G --> QQ

    %% Storage Systems
    KK --> RR[(🗄️ Cloud Spanner<br/>Primary Database)]
    LL --> SS[(📊 BigQuery<br/>Analytics Warehouse)]
    L1 --> TT[(💾 Cloud Storage<br/>Audio Files)]

    %% External Integrations
    CC --> UU[🏢 CRM Systems<br/>HubSpot/Salesforce/Custom]
    II --> VV[📧 SendGrid<br/>Email Service]

    %% Styling
    classDef inputSource fill:#e1f5fe
    classDef gcpService fill:#4caf50,color:#fff
    classDef processing fill:#fff3e0
    classDef decision fill:#f3e5f5
    classDef storage fill:#e8f5e8
    classDef external fill:#ffebee
    classDef error fill:#ffcdd2

    class A1,A2,A3,A4 inputSource
    class B,C,D,F,K1,K2,K3,K4,L1,L2,L3,L4,Q,U,Y,Z,GG,LL,QQ gcpService
    class I,M,N1,N2,O1,O2,P1,T,X,AA,BB,EE,KK,MM processing
    class E,J,R,V,DD,HH,OO decision
    class RR,SS,TT storage
    class UU,VV external
    class G,S1,FF,NN,PP error
```

## 🏗️ **Architecture Components Breakdown**

### **🌐 Input Layer**
- **API Gateway**: Route requests, rate limiting, authentication
- **Cloud Load Balancer**: Distribute traffic, SSL termination
- **Multiple Sources**: Forms, calls, calendar, chat

### **🚀 Processing Layer**
- **Cloud Run**: Serverless Gemini/Go agent (auto-scaling 0-1000 instances)
- **Gemini 2.5 Flash**: AI content analysis and decision making
- **Document AI v1.5**: Advanced form processing (30 pages/min)
- **Speech-to-Text Chirp 3**: Real-time audio transcription

### **🗄️ Data Layer**
- **Cloud Spanner**: Multi-tenant database with tenant isolation
- **BigQuery**: Analytics and reporting warehouse
- **Cloud Storage**: Audio file storage with lifecycle policies
- **Secret Manager**: Secure credential storage

### **🔌 Integration Layer**
- **MCP Framework**: Dynamic CRM integrations
- **SendGrid**: Professional email delivery
- **Maps API**: Geographic validation
- **Calendar APIs**: Booking synchronization

### **📊 Monitoring Layer**
- **Cloud Monitoring**: Performance metrics and alerts
- **Error Reporting**: Centralized error tracking
- **Cloud Logging**: Comprehensive audit trails
- **Trace**: Request flow analysis

## 📋 **Data Flow Summary**

1. **Request arrives** → API Gateway validation
2. **Extract tenant_id** → Load configuration from Spanner
3. **Communication detection** → Route to appropriate processor
4. **AI analysis** → Content extraction and enrichment
5. **Spam validation** → ML-powered fraud detection
6. **Geographic validation** → Service area verification
7. **CRM integration** → Dynamic field mapping and push
8. **Email notifications** → Template-based alerts
9. **Database storage** → Structured data persistence
10. **Analytics update** → Real-time reporting data

## 🔄 **Key Features**

- **Zero-downtime scaling**: Cloud Run auto-scales based on demand
- **Multi-tenant isolation**: Row-level security in Cloud Spanner
- **Configuration-driven**: No code changes for new tenants
- **Real-time processing**: <200ms average response time
- **Fault tolerance**: Retry logic with dead letter queues
- **Comprehensive monitoring**: Full observability stack