# 🎨 Multi-Tenant Pipeline - Visual Flowchart for Human Reading

## 🔄 Simplified System Overview

```mermaid
flowchart TD
    %% Input Layer
    subgraph INPUTS ["📥 INPUT SOURCES"]
        A1["🌐<br/>Website<br/>Forms"]
        A2["📞<br/>CallRail<br/>Calls"]
        A3["📅<br/>Calendar<br/>Bookings"]
        A4["💬<br/>Chat<br/>Messages"]
    end

    %% Gateway Layer
    subgraph GATEWAY ["🚪 ENTRY POINT"]
        B1["🔄 API Gateway<br/>Rate Limiting & SSL"]
        B2["☁️ Load Balancer<br/>Traffic Distribution"]
    end

    %% Authentication Layer
    subgraph AUTH ["🔐 AUTHENTICATION"]
        C1["🆔 Extract tenant_id<br/>from JSON payload"]
        C2["🗄️ Query Cloud Spanner<br/>Load tenant config"]
        C3["🔑 HMAC Verification<br/>(CallRail webhooks)"]
    end

    %% Processing Layer
    subgraph PROCESS ["⚙️ INTELLIGENT PROCESSING"]
        D1["📝 Form Processing<br/>Document AI v1.5"]
        D2["🎤 Audio Processing<br/>Speech-to-Text Chirp 3"]
        D3["📅 Calendar Processing<br/>Conflict Detection"]
        D4["💬 Chat Processing<br/>Intent Analysis"]

        E1["🧠 AI Analysis<br/>Gemini 2.5 Flash"]
        E2["🎯 Lead Scoring<br/>Intent & Sentiment"]
    end

    %% Validation Layer
    subgraph VALIDATE ["✅ VALIDATION & FILTERING"]
        F1["🚨 Spam Detection<br/>AI-powered filtering"]
        F2["🗺️ Service Area Check<br/>Geographic validation"]
    end

    %% Integration Layer
    subgraph INTEGRATE ["🔌 CRM INTEGRATION"]
        G1["🟠 HubSpot"]
        G2["🔵 Salesforce"]
        G3["🟢 Pipedrive"]
        G4["🔗 Custom CRM"]

        H1["📧 Email Notifications<br/>SendGrid"]
    end

    %% Storage Layer
    subgraph STORE ["💾 DATA STORAGE"]
        I1["🗄️ Cloud Spanner<br/>Primary Database"]
        I2["📊 BigQuery<br/>Analytics"]
        I3["💾 Cloud Storage<br/>Audio Files"]
    end

    %% Monitoring Layer
    subgraph MONITOR ["📈 MONITORING"]
        J1["📱 Real-time Dashboard<br/>Live metrics"]
        J2["🚨 Alerts & Notifications<br/>System health"]
    end

    %% Flow Connections
    INPUTS --> GATEWAY
    GATEWAY --> AUTH
    AUTH --> PROCESS
    PROCESS --> VALIDATE
    VALIDATE --> INTEGRATE
    INTEGRATE --> STORE
    STORE --> MONITOR

    %% Internal Connections
    A1 --> D1
    A2 --> D2
    A3 --> D3
    A4 --> D4

    D1 --> E1
    D2 --> E1
    D3 --> E1
    D4 --> E1

    E1 --> E2
    E2 --> F1
    F1 --> F2

    F2 --> G1
    F2 --> G2
    F2 --> G3
    F2 --> G4

    G1 --> H1
    G2 --> H1
    G3 --> H1
    G4 --> H1

    H1 --> I1
    I1 --> I2
    I1 --> I3

    I2 --> J1
    I3 --> J1
    I1 --> J2

    %% Styling
    classDef inputBox fill:#e3f2fd,stroke:#1976d2,stroke-width:3px,color:#000
    classDef gatewayBox fill:#f3e5f5,stroke:#7b1fa2,stroke-width:3px,color:#000
    classDef authBox fill:#fff3e0,stroke:#f57f17,stroke-width:3px,color:#000
    classDef processBox fill:#e8f5e8,stroke:#388e3c,stroke-width:3px,color:#000
    classDef validateBox fill:#ffebee,stroke:#d32f2f,stroke-width:3px,color:#000
    classDef integrateBox fill:#e1f5fe,stroke:#0277bd,stroke-width:3px,color:#000
    classDef storeBox fill:#f1f8e9,stroke:#689f38,stroke-width:3px,color:#000
    classDef monitorBox fill:#fce4ec,stroke:#c2185b,stroke-width:3px,color:#000

    class A1,A2,A3,A4 inputBox
    class B1,B2 gatewayBox
    class C1,C2,C3 authBox
    class D1,D2,D3,D4,E1,E2 processBox
    class F1,F2 validateBox
    class G1,G2,G3,G4,H1 integrateBox
    class I1,I2,I3 storeBox
    class J1,J2 monitorBox
```

## 📋 Detailed CallRail Processing Flow

```mermaid
flowchart LR
    subgraph CALLRAIL ["📞 CALLRAIL INTEGRATION FLOW"]
        direction TB

        CR1["📞 Call Ends<br/>CallRail webhook fires"]
        CR2["🔐 HMAC Signature<br/>Verification"]
        CR3["📡 Download Call Details<br/>CallRail API"]
        CR4["⬇️ Download Recording<br/>Audio file (.mp3)"]
        CR5["💾 Store in Cloud Storage<br/>gs://tenant-audio/..."]
        CR6["🎤 Speech-to-Text<br/>Chirp 3 Transcription"]
        CR7["🧠 AI Analysis<br/>Gemini 2.5 Flash"]
        CR8["📊 Enhanced Payload<br/>Combined data"]

        CR1 --> CR2
        CR2 --> CR3
        CR3 --> CR4
        CR4 --> CR5
        CR5 --> CR6
        CR6 --> CR7
        CR7 --> CR8
    end

    subgraph ANALYSIS ["🎯 AI ANALYSIS RESULTS"]
        direction TB

        AN1["🎯 Intent Detection<br/>Quote request, Info seeking"]
        AN2["🏠 Project Type<br/>Kitchen, Bathroom, etc."]
        AN3["📈 Lead Score<br/>1-100 quality rating"]
        AN4["😊 Sentiment<br/>Positive, Neutral, Negative"]
        AN5["⏰ Urgency Level<br/>Immediate, 1-3 months"]

        AN1 --> AN2
        AN2 --> AN3
        AN3 --> AN4
        AN4 --> AN5
    end

    CALLRAIL --> ANALYSIS
```

## 🎨 Human-Readable Process Summary

### **🔄 Step-by-Step Flow**

```
1. 📥 INPUT RECEIVED
   ├── Website form submission
   ├── CallRail phone call webhook
   ├── Calendar booking notification
   └── Chat message from widget

2. 🚪 GATEWAY PROCESSING
   ├── API Gateway handles request
   ├── Load balancer distributes traffic
   └── SSL termination & rate limiting

3. 🔐 AUTHENTICATION
   ├── Extract tenant_id from JSON
   ├── Query tenant configuration
   └── Verify webhook signatures (CallRail)

4. ⚙️ INTELLIGENT PROCESSING
   ├── Route by communication type
   ├── AI analysis (Gemini 2.5 Flash)
   ├── Audio transcription (Speech-to-Text)
   └── Lead scoring & sentiment analysis

5. ✅ VALIDATION & FILTERING
   ├── AI-powered spam detection
   └── Geographic service area check

6. 🔌 CRM INTEGRATION
   ├── Dynamic CRM selection
   ├── Field mapping & data transform
   ├── Push to HubSpot/Salesforce/etc.
   └── Email notifications (SendGrid)

7. 💾 DATA STORAGE
   ├── Cloud Spanner (primary database)
   ├── BigQuery (analytics warehouse)
   └── Cloud Storage (audio files)

8. 📈 MONITORING & ALERTS
   ├── Real-time dashboard updates
   └── System health monitoring
```

## 🎯 **Key Features Highlighted**

### **🔒 Security First**
- HMAC signature verification for CallRail webhooks
- Multi-tenant data isolation in Cloud Spanner
- Row-level security for all database operations

### **🧠 AI-Powered Intelligence**
- **Speech-to-Text Chirp 3**: Audio transcription with speaker diarization
- **Gemini 2.5 Flash**: Content analysis, lead scoring, sentiment analysis
- **Smart Routing**: AI-powered spam detection and service area validation

### **⚡ Performance & Scale**
- **Auto-scaling**: Cloud Run scales 0-1000 instances automatically
- **<200ms latency**: Fast webhook processing
- **99.9% availability**: Enterprise-grade reliability

### **🔌 Integration Flexibility**
- **Multi-CRM support**: HubSpot, Salesforce, Pipedrive, Custom APIs
- **Real-time notifications**: Email alerts via SendGrid
- **Live dashboard**: Server-Sent Events for real-time updates

## 💰 **Cost Structure**
- **Budget**: $4,300-8,700/month
- **Scales**: 100-500 concurrent tenants
- **Value**: Complete automation of lead processing pipeline

This visual flowchart shows how your multi-tenant ingestion pipeline processes customer communications through intelligent AI analysis and seamlessly integrates with existing business systems!