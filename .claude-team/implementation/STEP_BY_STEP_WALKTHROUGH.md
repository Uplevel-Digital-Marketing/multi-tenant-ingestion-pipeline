# 🚶‍♂️ Step-by-Step Walkthrough: Multi-Tenant Ingestion Pipeline

## 📞 **Example Scenario: CallRail Phone Call Processing**

Let's follow a real phone call from a potential customer interested in a kitchen remodel through the entire system.

---

## **LEVEL 1: INPUT RECEIVED** 📞

**What happens:**
- Customer calls (555) 987-6543 (a tracking number for "ABC Home Remodeling")
- Call lasts 3 minutes, customer says: "Hi, I'm interested in a kitchen remodel. Can someone call me back to discuss pricing?"
- CallRail records the call and prepares to send a webhook

**Technical details:**
- CallRail creates call ID: `CAL123456789`
- Recording saved to CallRail's servers
- Webhook payload prepared with call metadata

---

## **LEVEL 2: GATEWAY PROCESSING** 🚪

**What happens:**
- CallRail sends POST webhook to: `https://your-domain.com/api/v1/callrail/webhook`
- API Gateway receives the request
- Rate limiting checked (max 1000 requests/minute per source)
- SSL certificate verified
- Request routed to Cloud Run service

**Technical details:**
```http
POST /api/v1/callrail/webhook
Headers:
  x-callrail-signature: sha256=abc123...
  content-type: application/json

Body:
{
  "call_id": "CAL123456789",
  "caller_id": "+15551234567",
  "duration": "180",
  "recording_url": "https://api.callrail.com/...",
  "tenant_id": "tenant_abc_remodeling",
  "callrail_company_id": "12345"
}
```

---

## **LEVEL 3: AUTHENTICATION** 🔐

**What happens:**
1. **Extract tenant_id**: System pulls `tenant_abc_remodeling` from JSON payload
2. **Database lookup**: Query Cloud Spanner for tenant configuration
3. **HMAC verification**: Verify CallRail signature using secret key

**Technical details:**
```sql
SELECT tenant_id, workflow_config, callrail_api_key, callrail_company_id
FROM offices
WHERE tenant_id = 'tenant_abc_remodeling'
  AND callrail_company_id = '12345'
  AND status = 'active'
```

**Result**: Tenant found, configuration loaded, webhook signature verified ✅

---

## **LEVEL 4: COMMUNICATION DETECTION** 📡

**What happens:**
- System examines the payload
- Detects `source: "callrail_webhook"`
- Routes to CallRail processor
- Loads tenant-specific CallRail settings

**Technical details:**
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
  }
}
```

---

## **LEVEL 5: CALLRAIL PROCESSING** 📞

**What happens:**
1. **API call to CallRail**: Get detailed call information
2. **Download recording**: Retrieve the actual audio file
3. **Store audio**: Save to Cloud Storage for processing

**Technical details:**
```go
// API call to get call details
callDetails := getCallDetails("AC987654321", "CAL123456789", apiKey)

// Download the recording
audioData := downloadRecording(callDetails.RecordingURL, apiKey)

// Store in Cloud Storage
storageURL := "gs://tenant-audio-files/tenant_abc_remodeling/calls/CAL123456789.mp3"
```

---

## **LEVEL 6: AUDIO PROCESSING** 🎤

**What happens:**
1. **Speech-to-Text**: Audio sent to Google's Chirp 3 model
2. **Transcription**: Convert speech to text with speaker identification
3. **Timing data**: Extract word-level timing and confidence scores

**Technical details:**
```
Processing: 180 seconds of audio
Model: Speech-to-Text Chirp 3
Features: Speaker diarization, automatic punctuation, word timing
Language: en-US
```

**Transcription result:**
```json
{
  "transcript": "Hi, I'm interested in a kitchen remodel. Can someone call me back to discuss pricing?",
  "confidence": 0.95,
  "speaker_diarization": [
    {
      "speaker": 1,
      "start_time": "0.0s",
      "end_time": "4.2s",
      "text": "Hi, I'm interested in a kitchen remodel."
    },
    {
      "speaker": 1,
      "start_time": "4.5s",
      "end_time": "8.1s",
      "text": "Can someone call me back to discuss pricing?"
    }
  ]
}
```

---

## **LEVEL 7: AI ANALYSIS** 🧠

**What happens:**
- Transcription + call metadata sent to Vertex AI Gemini 2.5 Flash
- AI analyzes content for business intelligence
- Extracts key insights about the lead

**AI prompt:**
```
Analyze this phone call for a home remodeling company:

TRANSCRIPT: "Hi, I'm interested in a kitchen remodel. Can someone call me back to discuss pricing?"
CALL DATA: Duration 180s, first-time caller, Los Angeles CA

Extract: intent, project type, urgency, budget indicators, sentiment, lead score
```

**AI analysis result:**
```json
{
  "intent": "quote_request",
  "project_type": "kitchen",
  "timeline": "1-3_months",
  "budget_indicator": "medium",
  "sentiment": "positive",
  "lead_score": 85,
  "urgency": "medium",
  "follow_up_required": true,
  "key_details": [
    "Kitchen remodel interest",
    "Pricing discussion needed",
    "Callback requested"
  ]
}
```

---

## **LEVEL 8: ENHANCED PAYLOAD** 📊

**What happens:**
- Combine original webhook data + transcription + AI analysis
- Create unified JSON payload for workflow processing
- Add processing metadata

**Enhanced payload:**
```json
{
  "request_id": "req_987654321",
  "tenant_id": "tenant_abc_remodeling",
  "source": "callrail_webhook",
  "communication_mode": "phone_call",
  "call_details": {
    "customer_phone": "+15551234567",
    "duration": 180,
    "location": "Los Angeles, CA"
  },
  "audio_processing": {
    "recording_url": "gs://tenant-audio-files/...",
    "transcription": "Hi, I'm interested in...",
    "confidence": 0.95
  },
  "ai_analysis": {
    "intent": "quote_request",
    "project_type": "kitchen",
    "lead_score": 85,
    "sentiment": "positive"
  }
}
```

---

## **LEVEL 9: SPAM DETECTION** 🚨

**What happens:**
- Enhanced payload sent to AI spam detector
- Gemini analyzes for spam indicators
- Generates confidence score

**Spam analysis:**
```
Input: Lead score 85, positive sentiment, kitchen remodel, callback request
Indicators: Legitimate business inquiry, specific project type, normal duration
Result: 5% spam likelihood (very low)
```

---

## **LEVEL 10: SPAM DECISION** 🤖

**What happens:**
- System checks spam likelihood: 5%
- Since 5% < 50%, classified as "Low Spam"
- Processing continues automatically

**Decision logic:**
- High (>85%) → Quarantine
- Medium (50-85%) → Human review
- **Low (<50%) → Continue** ✅

---

## **LEVEL 11: SERVICE AREA CHECK** 🗺️

**What happens:**
1. **Geocoding**: "Los Angeles, CA" sent to Google Maps API
2. **Coordinates**: Returns lat/lng coordinates
3. **Coverage check**: Compare against tenant's service areas
4. **Buffer calculation**: Check 25-mile radius

**Technical details:**
```
Customer location: Los Angeles, CA (34.0522, -118.2437)
Tenant service areas: ["90210", "90211", "90401", ...]
Buffer zone: 25 miles
Result: INSIDE service area ✅
```

---

## **LEVEL 12: CRM INTEGRATION SETUP** 🔐

**What happens:**
1. **Load credentials**: Get CRM API keys from Secret Manager
2. **Provider detection**: Tenant uses HubSpot
3. **Field mapping**: Load custom field mappings

**Configuration loaded:**
```json
{
  "crm_integration": {
    "provider": "hubspot",
    "credentials_secret": "hubspot-api-key-abc-remodeling",
    "field_mapping": {
      "name": "firstname",
      "phone": "phone",
      "lead_score": "hs_lead_score",
      "project_type": "custom_project_type"
    }
  }
}
```

---

## **LEVEL 13: HUBSPOT INTEGRATION** 🟠

**What happens:**
1. **API authentication**: Use HubSpot API key
2. **Contact search**: Check if contact already exists
3. **Create/update**: Add new contact or update existing
4. **Set properties**: Add lead score, project type, etc.

**HubSpot API call:**
```http
POST https://api.hubapi.com/crm/v3/objects/contacts
Authorization: Bearer sk-hub-abc123...

{
  "properties": {
    "firstname": "John",
    "phone": "+15551234567",
    "hs_lead_score": "85",
    "custom_project_type": "kitchen",
    "lifecyclestage": "lead",
    "lead_source": "phone_call"
  }
}
```

**Result**: Contact created successfully with ID: `12345678` ✅

---

## **LEVEL 14: EMAIL NOTIFICATIONS** 📧

**What happens:**
1. **Check conditions**: Lead score 85 > threshold 30 ✅
2. **Load template**: Kitchen remodel lead notification
3. **Send email**: Alert sales team via SendGrid

**Email sent:**
```
To: sales@abcremodeling.com
Subject: 🔥 High-Quality Kitchen Lead - Score: 85

New kitchen remodeling lead from phone call:
- Customer: (555) 123-4567
- Location: Los Angeles, CA
- Project: Kitchen remodel
- Lead Score: 85/100
- Sentiment: Positive
- Follow-up: Callback requested

HubSpot Contact: https://app.hubspot.com/contacts/12345678
```

---

## **LEVEL 15: DATABASE STORAGE** 🗄️

**What happens:**
1. **Insert main record**: Add to `requests` table
2. **Audio record**: Add to `call_recordings` table
3. **Integration log**: Track CRM success in `crm_integrations`
4. **Webhook audit**: Log in `webhook_events` table

**SQL operations:**
```sql
-- Main request record
INSERT INTO requests (tenant_id, source, call_id, lead_score, ai_analysis, ...)
VALUES ('tenant_abc_remodeling', 'callrail_webhook', 'CAL123456789', 85, {...}, ...)

-- Call recording record
INSERT INTO call_recordings (tenant_id, call_id, storage_url, transcription_status)
VALUES ('tenant_abc_remodeling', 'CAL123456789', 'gs://...', 'completed')

-- CRM integration success
INSERT INTO crm_integrations (tenant_id, request_id, provider, status, external_id)
VALUES ('tenant_abc_remodeling', 'req_987654321', 'hubspot', 'success', '12345678')
```

---

## **LEVEL 16: ANALYTICS UPDATE** 📊

**What happens:**
1. **BigQuery streaming**: Send data to analytics warehouse
2. **Real-time metrics**: Update dashboard counters
3. **Performance tracking**: Log processing time (2.3 seconds)

**BigQuery record:**
```json
{
  "timestamp": "2025-09-13T10:35:00Z",
  "tenant_id": "tenant_abc_remodeling",
  "communication_type": "phone_call",
  "lead_score": 85,
  "processing_time_ms": 2300,
  "crm_success": true,
  "cost_usd": 0.15
}
```

---

## **LEVEL 17: DASHBOARD UPDATE** 📈

**What happens:**
1. **Server-Sent Events**: Push update to live dashboard
2. **Metrics refresh**: Update counters and charts
3. **Alert check**: No alerts triggered (all normal)

**Dashboard update:**
```javascript
// Real-time update sent to dashboard
{
  "type": "new_request",
  "tenant_id": "tenant_abc_remodeling",
  "data": {
    "lead_score": 85,
    "project_type": "kitchen",
    "status": "processed",
    "crm_integrated": true
  }
}
```

---

## **LEVEL 18: COMPLETION** ✅

**What happens:**
1. **HTTP response**: Return 200 OK to CallRail
2. **Processing summary**: Include timing and status
3. **Webhook acknowledged**: CallRail marks as delivered

**Final response:**
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "success",
  "request_id": "req_987654321",
  "processing_time_ms": 2300,
  "lead_score": 85,
  "crm_integrated": true,
  "notifications_sent": 1
}
```

---

## 🎯 **Summary: What Just Happened**

In **2.3 seconds**, your system just:

1. ✅ **Received** a CallRail webhook for a 3-minute phone call
2. ✅ **Authenticated** the tenant and verified the webhook signature
3. ✅ **Downloaded** the call recording and stored it securely
4. ✅ **Transcribed** the audio using AI speech recognition
5. ✅ **Analyzed** the content with Gemini AI for business insights
6. ✅ **Scored** the lead as high-quality (85/100)
7. ✅ **Filtered** out spam (5% likelihood)
8. ✅ **Validated** the customer is in the service area
9. ✅ **Created** a contact in HubSpot with all the insights
10. ✅ **Sent** an email alert to the sales team
11. ✅ **Stored** everything in the database for analytics
12. ✅ **Updated** the real-time dashboard

**Result**: ABC Home Remodeling now has a qualified kitchen remodeling lead in their CRM, their sales team has been notified, and they can follow up immediately with intelligent insights about the customer's needs!

This entire process cost approximately **$0.15** and took **2.3 seconds** to complete automatically.