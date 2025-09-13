# 🔄 Multi-Tenant Ingestion Pipeline - System Flowchart

## 📊 Complete System Architecture Flow

```mermaid
graph TD
    %% External Input Sources
    A1[🌐 Website Form<br/>Contact Form Submission] --> B1[🔄 API Gateway<br/>Rate Limiting & SSL]
    A2[📞 CallRail Webhook<br/>Post-Call Event] --> B1
    A3[📅 Calendar Booking<br/>Calendly/Acuity] --> B1
    A4[💬 Chat Widget<br/>Intercom/Zendesk] --> B1

    %% API Gateway & Load Balancing
    B1 --> C1[☁️ Cloud Load Balancer<br/>Global Traffic Distribution]
    C1 --> D1[🚀 Cloud Run Service<br/>webhook-processor]
    C1 --> D2[🚀 Cloud Run Service<br/>api-gateway]

    %% Authentication & Tenant Resolution
    D1 --> E1{🔐 Extract tenant_id<br/>from JSON Payload}
    D2 --> E1
    E1 -->|Valid| F1[🗄️ Cloud Spanner Query<br/>SELECT * FROM offices WHERE tenant_id=?]
    E1 -->|Invalid/Missing| G1[❌ HTTP 401<br/>Log Security Event]

    %% Tenant Configuration & Webhook Verification
    F1 --> H1[📋 Load Tenant Config<br/>workflow_config JSON]
    H1 --> I1{🔐 Webhook Source<br/>Verification}
    I1 -->|CallRail| J1[🔑 HMAC-SHA256<br/>Signature Verification]
    I1 -->|Other| J2[✅ Basic Validation]
    J1 -->|Valid| K1[✅ Webhook Authenticated]
    J1 -->|Invalid| G1
    J2 --> K1

    %% Communication Mode Detection & Processing
    K1 --> L1{📡 Communication Mode<br/>Detection}

    %% Form Processing Branch
    L1 -->|Website Form| M1[📝 Document AI v1.5<br/>Form Field Extraction]
    M1 --> M2[✅ Field Validation<br/>Required Fields Check]
    M2 --> M3[🔍 Auto-complete Fields<br/>Address/Phone Normalization]
    M3 --> N1[📊 Enhanced Form Payload]

    %% CallRail Phone Call Processing Branch
    L1 -->|Phone Call| O1[📞 CallRail API Client<br/>GET /v3/a/{account}/calls/{id}]
    O1 --> O2[⬇️ Download Recording<br/>GET recording.json + audio file]
    O2 --> O3[💾 Cloud Storage<br/>gs://tenant-audio-files/{tenant}/{call}.mp3]
    O3 --> O4[🎤 Speech-to-Text Chirp 3<br/>Transcription + Speaker Diarization]
    O4 --> O5[🧠 Vertex AI Gemini 2.5 Flash<br/>Content Analysis & Lead Scoring]
    O5 --> O6[📊 AI Analysis Results<br/>Intent, Sentiment, Project Type, Lead Score]
    O6 --> N1

    %% Calendar Processing Branch
    L1 -->|Calendar Booking| P1[📅 Calendar API Integration<br/>Event Details Extraction]
    P1 --> P2[⚠️ Conflict Detection<br/>Double-booking Check]
    P2 --> N1

    %% Chat Processing Branch
    L1 -->|Chat Message| Q1[💬 Chat API Integration<br/>Message Thread Analysis]
    Q1 --> Q2[🧠 Gemini Content Analysis<br/>Intent Detection]
    Q2 --> N1

    %% Enhanced Payload Creation
    N1 --> R1[📝 Create Enhanced JSON<br/>Unified Request Format]
    R1 --> S1[🗄️ Insert webhook_events<br/>Audit Log Entry]

    %% Spam Detection & Validation
    S1 --> T1[🤖 AI Spam Detection<br/>Gemini 2.5 Flash ML Analysis]
    T1 --> U1{🚨 Spam Likelihood<br/>Assessment}
    U1 -->|High >85%| V1[🚫 Mark as Spam<br/>Quarantine & Log]
    U1 -->|Medium 50-85%| V2[⚠️ Flag for Review<br/>Human Validation Queue]
    U1 -->|Low <50%| W1[✅ Continue Processing]

    %% Service Area Validation
    W1 --> X1[🗺️ Google Maps API<br/>Geographic Validation]
    X1 --> Y1{📍 Service Area<br/>Coverage Check}
    Y1 -->|Outside Area| Z1[📍 Route to Partner<br/>or Polite Decline]
    Y1 -->|In Service Area| AA1[✅ Area Validated]

    %% CRM Integration Pipeline
    AA1 --> BB1[🔐 Load CRM Credentials<br/>Secret Manager Lookup]
    BB1 --> CC1[🔌 Initialize CRM MCP<br/>Dynamic Provider Selection]
    CC1 --> DD1{🏢 CRM Provider<br/>Detection}
    DD1 -->|HubSpot| EE1[🟠 HubSpot API<br/>Contact Creation]
    DD1 -->|Salesforce| EE2[🔵 Salesforce API<br/>Lead Creation]
    DD1 -->|Pipedrive| EE3[🟢 Pipedrive API<br/>Deal Creation]
    DD1 -->|Custom| EE4[🔗 Custom Webhook<br/>Field Mapping]

    %% CRM Processing Results
    EE1 --> FF1[📋 Transform Data<br/>Field Mapping & Validation]
    EE2 --> FF1
    EE3 --> FF1
    EE4 --> FF1
    FF1 --> GG1[📤 Push to CRM<br/>API Integration]
    GG1 --> HH1{✅ CRM Push<br/>Success Status}
    HH1 -->|Success| II1[✅ Log Success<br/>crm_integrations table]
    HH1 -->|Failure| JJ1[❌ Retry Queue<br/>Exponential Backoff]

    %% Email Notification System
    II1 --> KK1[📧 Email Template Engine<br/>SendGrid MCP Integration]
    KK1 --> LL1{📮 Notification Rules<br/>Condition Check}
    LL1 -->|Conditions Met| MM1[📬 Send Notifications<br/>Multi-recipient Delivery]
    LL1 -->|Skip Email| NN1[⏭️ Continue to Storage]

    %% Database Storage & Analytics
    MM1 --> OO1[🗄️ Cloud Spanner Insert<br/>requests table with tenant isolation]
    NN1 --> OO1
    OO1 --> PP1[📊 BigQuery Streaming<br/>Analytics Data Pipeline]
    PP1 --> QQ1[📈 Update Real-time Metrics<br/>Dashboard Data Refresh]

    %% Error Handling & Retry Logic
    JJ1 --> RR1[🔄 Exponential Backoff<br/>1s, 2s, 4s, 8s delays]
    RR1 --> SS1{🔁 Max Retries<br/>Reached (3 attempts)}
    SS1 -->|No| GG1
    SS1 -->|Yes| TT1[💀 Dead Letter Queue<br/>Manual Review Required]

    %% Monitoring & Alerting
    QQ1 --> UU1[📈 Cloud Monitoring<br/>Metrics Collection]
    TT1 --> UU1
    V1 --> UU1
    G1 --> UU1
    UU1 --> VV1[🚨 Alert Manager<br/>Slack/PagerDuty Integration]

    %% Final Response
    QQ1 --> WW1[✅ HTTP 200 Response<br/>Processing Complete]

    %% Data Storage Systems
    OO1 --> XX1[(🗄️ Cloud Spanner<br/>Primary Multi-tenant Database)]
    PP1 --> YY1[(📊 BigQuery<br/>Analytics Warehouse)]
    O3 --> ZZ1[(💾 Cloud Storage<br/>Audio File Archive)]

    %% External Integrations
    GG1 --> AAA1[🏢 External CRM Systems<br/>HubSpot/Salesforce/Custom]
    MM1 --> BBB1[📧 SendGrid<br/>Email Delivery Service]
    X1 --> CCC1[🗺️ Google Maps API<br/>Geocoding Service]

    %% Real-time Dashboard Updates
    QQ1 --> DDD1[📱 Real-time Dashboard<br/>Server-Sent Events]
    DDD1 --> EEE1[👥 Tenant Admin Interface<br/>React/TypeScript UI]

    %% Styling for better readability
    classDef inputSource fill:#e3f2fd,stroke:#1976d2,stroke-width:2px
    classDef processing fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px
    classDef decision fill:#fff3e0,stroke:#f57f17,stroke-width:2px
    classDef storage fill:#e8f5e8,stroke:#388e3c,stroke-width:2px
    classDef external fill:#ffebee,stroke:#d32f2f,stroke-width:2px
    classDef error fill:#ffcdd2,stroke:#c62828,stroke-width:2px
    classDef success fill:#c8e6c9,stroke:#2e7d32,stroke-width:2px

    class A1,A2,A3,A4 inputSource
    class B1,C1,D1,D2,F1,H1,M1,M2,M3,O1,O2,O4,O5,P1,P2,Q1,Q2,R1,T1,X1,BB1,CC1,FF1,KK1,OO1,PP1,UU1 processing
    class E1,I1,L1,U1,Y1,DD1,HH1,LL1,SS1 decision
    class XX1,YY1,ZZ1 storage
    class AAA1,BBB1,CCC1 external
    class G1,JJ1,RR1,TT1,V1 error
    class II1,WW1,QQ1 success
```

## 🎯 **Key Processing Stages Explained**

### **1. Input Reception (Lines 1-4)**
- Multiple communication channels funnel through API Gateway
- Rate limiting and SSL termination protect the system
- All requests normalized to unified format

### **2. Authentication & Tenant Resolution (Lines 5-8)**
- Extract `tenant_id` from JSON payload (no API keys required)
- Query Cloud Spanner for tenant configuration
- Load workflow rules and CRM settings

### **3. Webhook Verification (Lines 9-12)**
- CallRail webhooks verified with HMAC-SHA256 signatures
- Other sources get basic validation
- Security events logged for monitoring

### **4. Communication Processing (Lines 13-24)**
- **Forms**: Document AI field extraction and validation
- **Phone Calls**: Full CallRail integration with recording download, transcription, and AI analysis
- **Calendar**: Booking validation and conflict detection
- **Chat**: Message analysis and intent detection

### **5. AI Enhancement (Lines 25-27)**
- Spam detection using Gemini 2.5 Flash
- Content analysis for lead scoring
- Sentiment and intent classification

### **6. Business Logic (Lines 28-32)**
- Service area validation with Google Maps
- Geographic coverage checking
- Partner routing for out-of-area requests

### **7. CRM Integration (Lines 33-40)**
- Dynamic CRM provider selection
- Field mapping and data transformation
- Multi-provider support (HubSpot, Salesforce, Pipedrive, Custom)
- Retry logic with exponential backoff

### **8. Notifications & Storage (Lines 41-46)**
- Conditional email notifications via SendGrid
- Multi-tenant database storage in Cloud Spanner
- Real-time analytics streaming to BigQuery
- Dashboard updates via Server-Sent Events

### **9. Error Handling (Lines 47-50)**
- Comprehensive retry mechanisms
- Dead letter queues for failed operations
- Monitoring and alerting integration

## 💾 **Data Flow Summary**

1. **Request** → API Gateway → Cloud Run
2. **Authentication** → Cloud Spanner tenant lookup
3. **Processing** → AI analysis (Speech-to-Text + Gemini)
4. **Validation** → Spam detection + service area check
5. **Integration** → CRM push + email notifications
6. **Storage** → Cloud Spanner + BigQuery analytics
7. **Monitoring** → Real-time dashboard updates

## 🔧 **Technology Stack**

- **Compute**: Cloud Run (auto-scaling 0-1000 instances)
- **Database**: Cloud Spanner (multi-tenant with row-level security)
- **AI Services**: Vertex AI Gemini 2.5 Flash + Speech-to-Text Chirp 3
- **Storage**: Cloud Storage (audio files) + BigQuery (analytics)
- **Integration**: Dynamic CRM connectors + SendGrid
- **Monitoring**: Cloud Monitoring + real-time dashboards

This flowchart represents the complete end-to-end processing pipeline for the multi-tenant ingestion system we just built!