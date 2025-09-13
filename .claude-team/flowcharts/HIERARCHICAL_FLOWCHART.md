# 📊 Multi-Tenant Pipeline - Hierarchical Box Flowchart

## 🔄 Top-Down System Flow (Human-Readable)

```mermaid
flowchart TD
    %% Level 1: Input Sources
    A[🌐 Website Form<br/>Contact submission]
    B[📞 CallRail Webhook<br/>Post-call event]
    C[📅 Calendar Booking<br/>Appointment scheduled]
    D[💬 Chat Message<br/>Live chat widget]

    %% Level 2: Gateway
    E[🚪 API Gateway<br/>Rate limiting, SSL termination<br/>Route to Cloud Run services]

    %% Level 3: Authentication
    F[🔐 Authentication Layer<br/>Extract tenant_id from JSON<br/>Query Cloud Spanner for config<br/>HMAC verification for CallRail]

    %% Level 4: Communication Detection
    G{📡 Detect Communication Type<br/>Route to appropriate processor}

    %% Level 5A: Form Processing
    H1[📝 Form Processor<br/>Document AI v1.5<br/>Extract fields & validate]

    %% Level 5B: CallRail Processing
    H2[📞 CallRail Processor<br/>Download call details<br/>Download audio recording<br/>Store in Cloud Storage]

    %% Level 5C: Calendar Processing
    H3[📅 Calendar Processor<br/>Extract booking details<br/>Check for conflicts]

    %% Level 5D: Chat Processing
    H4[💬 Chat Processor<br/>Extract conversation<br/>Analyze message thread]

    %% Level 6A: Audio Processing (CallRail only)
    I[🎤 Audio Processing<br/>Speech-to-Text Chirp 3<br/>Transcription + Speaker diarization<br/>Extract conversation details]

    %% Level 7: AI Analysis
    J[🧠 AI Content Analysis<br/>Vertex AI Gemini 2.5 Flash<br/>• Intent detection (quote, info, complaint)<br/>• Project type (kitchen, bathroom, etc.)<br/>• Lead quality scoring (1-100)<br/>• Sentiment analysis<br/>• Timeline urgency assessment]

    %% Level 8: Enhanced Payload
    K[📊 Create Enhanced Payload<br/>Combine original data + AI insights<br/>Unified JSON format for workflow]

    %% Level 9: Spam Detection
    L[🚨 Spam Detection<br/>AI-powered filtering<br/>Confidence scoring]

    %% Level 10: Spam Decision
    M{🤖 Spam Assessment<br/>Check likelihood score}

    %% Level 11A: Spam Handling
    N1[🚫 High Spam >85%<br/>Quarantine & log<br/>No further processing]

    %% Level 11B: Review Queue
    N2[⚠️ Medium Spam 50-85%<br/>Flag for human review<br/>Queue for manual validation]

    %% Level 11C: Continue Processing
    N3[✅ Low Spam <50%<br/>Continue automated processing]

    %% Level 12: Service Area Check
    O[🗺️ Service Area Validation<br/>Google Maps API geocoding<br/>Check against tenant coverage zones<br/>Buffer zone calculation (25 mile radius)]

    %% Level 13: Area Decision
    P{📍 Coverage Assessment<br/>Is customer in service area?}

    %% Level 14A: Outside Area
    Q1[❌ Outside Service Area<br/>Route to partner network<br/>or polite decline]

    %% Level 14B: In Area
    Q2[✅ In Service Area<br/>Continue to CRM integration]

    %% Level 15: CRM Preparation
    R[🔐 CRM Integration Setup<br/>Load credentials from Secret Manager<br/>Determine CRM provider<br/>Load field mapping configuration]

    %% Level 16: CRM Selection
    S{🏢 CRM Provider<br/>Which system to integrate?}

    %% Level 17: CRM Integrations
    T1[🟠 HubSpot Integration<br/>Create/update contact<br/>Add to appropriate pipeline<br/>Set lead score property]

    T2[🔵 Salesforce Integration<br/>Create/update lead<br/>Assign to sales rep<br/>Set priority based on score]

    T3[🟢 Pipedrive Integration<br/>Create deal in pipeline<br/>Set value based on project type<br/>Schedule follow-up activity]

    T4[🔗 Custom CRM Webhook<br/>Transform data to custom format<br/>POST to client's API endpoint<br/>Handle custom field mapping]

    %% Level 18: CRM Result
    U{✅ CRM Integration<br/>Was the push successful?}

    %% Level 19A: CRM Success
    V1[✅ CRM Success<br/>Log success in database<br/>Update integration metrics]

    %% Level 19B: CRM Failure
    V2[❌ CRM Failure<br/>Add to retry queue<br/>Exponential backoff<br/>Alert on max retries]

    %% Level 20: Email Notifications
    W[📧 Email Notification Engine<br/>Check notification rules<br/>Load email templates<br/>SendGrid integration]

    %% Level 21: Email Decision
    X{📮 Send Notifications?<br/>Check conditions:<br/>• Lead score threshold<br/>• Communication type<br/>• Time of day rules}

    %% Level 22A: Send Email
    Y1[📬 Send Email Alerts<br/>Multi-recipient delivery<br/>• Sales team notification<br/>• Manager alert for high-value leads<br/>• Customer confirmation]

    %% Level 22B: Skip Email
    Y2[⏭️ Skip Email<br/>Conditions not met<br/>Log decision]

    %% Level 23: Database Storage
    Z[🗄️ Database Storage<br/>Insert into Cloud Spanner<br/>• requests table (main record)<br/>• call_recordings (if audio)<br/>• webhook_events (audit log)<br/>• crm_integrations (success tracking)]

    %% Level 24: Analytics Pipeline
    AA[📊 Analytics Update<br/>Stream data to BigQuery<br/>Update real-time metrics<br/>Refresh dashboard data]

    %% Level 25: Monitoring & Alerts
    BB[📈 Monitoring Dashboard<br/>Real-time metrics via SSE<br/>Performance tracking<br/>Alert on thresholds<br/>Cost analysis]

    %% Level 26: Final Response
    CC[✅ Processing Complete<br/>Return HTTP 200<br/>Include processing summary<br/>Webhook acknowledged]

    %% Flow Connections
    A --> E
    B --> E
    C --> E
    D --> E

    E --> F
    F --> G

    G -->|Form| H1
    G -->|CallRail| H2
    G -->|Calendar| H3
    G -->|Chat| H4

    H1 --> J
    H2 --> I
    H3 --> J
    H4 --> J

    I --> J
    J --> K
    K --> L
    L --> M

    M -->|High >85%| N1
    M -->|Medium 50-85%| N2
    M -->|Low <50%| N3

    N3 --> O
    O --> P

    P -->|Outside| Q1
    P -->|Inside| Q2

    Q2 --> R
    R --> S

    S -->|HubSpot| T1
    S -->|Salesforce| T2
    S -->|Pipedrive| T3
    S -->|Custom| T4

    T1 --> U
    T2 --> U
    T3 --> U
    T4 --> U

    U -->|Success| V1
    U -->|Failure| V2

    V1 --> W
    V2 --> W

    W --> X

    X -->|Yes| Y1
    X -->|No| Y2

    Y1 --> Z
    Y2 --> Z

    Z --> AA
    AA --> BB
    BB --> CC

    %% Error paths
    N1 --> BB
    Q1 --> BB

    %% Retry loop
    V2 -.->|Retry| R

    %% Styling for readability
    classDef inputLevel fill:#e3f2fd,stroke:#1976d2,stroke-width:2px,color:#000
    classDef gatewayLevel fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px,color:#000
    classDef authLevel fill:#fff3e0,stroke:#f57f17,stroke-width:2px,color:#000
    classDef processLevel fill:#e8f5e8,stroke:#388e3c,stroke-width:2px,color:#000
    classDef decisionLevel fill:#ffebee,stroke:#d32f2f,stroke-width:2px,color:#000
    classDef integrationLevel fill:#e1f5fe,stroke:#0277bd,stroke-width:2px,color:#000
    classDef storageLevel fill:#f1f8e9,stroke:#689f38,stroke-width:2px,color:#000
    classDef errorLevel fill:#ffcdd2,stroke:#c62828,stroke-width:2px,color:#000

    class A,B,C,D inputLevel
    class E gatewayLevel
    class F authLevel
    class G,H1,H2,H3,H4,I,J,K,L processLevel
    class M,P,S,U,X decisionLevel
    class R,T1,T2,T3,T4,W,Y1 integrationLevel
    class Z,AA,BB,CC storageLevel
    class N1,Q1,V2 errorLevel
```

## 📋 Process Flow Summary (Level by Level)

### **🎯 INPUT LEVEL (1)**
Four communication channels feed into the system

### **🚪 GATEWAY LEVEL (2)**
Single entry point with rate limiting and security

### **🔐 AUTHENTICATION LEVEL (3)**
Tenant identification and webhook verification

### **📡 ROUTING LEVEL (4-5)**
Intelligent routing based on communication type:
- **Forms** → Document AI processing
- **CallRail** → Audio download and storage
- **Calendar** → Booking validation
- **Chat** → Message analysis

### **🎤 AUDIO PROCESSING LEVEL (6)**
CallRail-specific: Speech-to-Text transcription

### **🧠 AI ANALYSIS LEVEL (7-8)**
Gemini 2.5 Flash processes all content types:
- Intent detection
- Project type classification
- Lead scoring (1-100)
- Sentiment analysis
- Urgency assessment

### **🚨 SPAM FILTERING LEVELS (9-11)**
Three-tier spam detection:
- **High (>85%)** → Quarantine
- **Medium (50-85%)** → Human review
- **Low (<50%)** → Continue processing

### **🗺️ SERVICE AREA LEVELS (12-14)**
Geographic validation:
- Geocode customer location
- Check against service zones
- Route out-of-area to partners

### **🔌 CRM INTEGRATION LEVELS (15-19)**
Dynamic CRM selection and processing:
- **HubSpot** → Contact management
- **Salesforce** → Lead creation
- **Pipedrive** → Deal pipeline
- **Custom** → Webhook integration
- Retry logic for failures

### **📧 EMAIL NOTIFICATION LEVELS (20-22)**
Conditional email alerts:
- Check notification rules
- Multi-recipient delivery
- SendGrid integration

### **💾 STORAGE LEVELS (23-24)**
Data persistence and analytics:
- Cloud Spanner primary storage
- BigQuery analytics streaming

### **📈 MONITORING LEVEL (25)**
Real-time dashboard and alerting

### **✅ COMPLETION LEVEL (26)**
HTTP response and acknowledgment

## 🎯 **Key Decision Points**

1. **Communication Type** → Routes to appropriate processor
2. **Spam Assessment** → Determines if processing continues
3. **Service Area** → Validates customer location
4. **CRM Provider** → Selects integration method
5. **Email Conditions** → Decides on notifications

This hierarchical flowchart shows the complete 26-level processing pipeline from initial input to final completion!