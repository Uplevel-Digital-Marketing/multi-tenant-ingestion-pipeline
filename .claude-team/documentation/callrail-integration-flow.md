# CallRail Integration Flow - Complete Step-by-Step Documentation

## 🎯 **CallRail Webhook Integration Overview**

### **Key Changes from Original Plan:**
- ❌ **Removed**: API key authentication requirement
- ✅ **Added**: CallRail ID mapping in office settings
- ✅ **Added**: tenant_id in JSON payload for authentication
- ✅ **Added**: Post-call webhook processing with recording download

---

## 📞 **Step-by-Step CallRail Integration Flow**

### **STEP 1: CallRail Webhook Configuration**
**Location**: CallRail Admin Dashboard → Integrations → Webhooks

**Configuration Required:**
```json
{
  "webhook_url": "https://your-domain.com/api/v1/callrail/webhook",
  "events": ["call_completed"],
  "signature_token": "your_webhook_secret_token"
}
```

**Authentication**:
- No API key required in headers
- Uses HMAC signature verification via `x-callrail-signature` header

---

### **STEP 2: Inbound Webhook Reception**
**Service**: Cloud Run (Gemini/Go Agent)
**Endpoint**: `POST /api/v1/callrail/webhook`

**Expected Payload Structure** (Based on CallRail documentation):
```json
{
  "call_id": "CAL123456789",
  "account_id": "AC987654321",
  "company_id": "12345",
  "caller_id": "+15551234567",
  "called_number": "+15559876543",
  "duration": "180",
  "start_time": "2025-09-13T10:30:00Z",
  "end_time": "2025-09-13T10:33:00Z",
  "direction": "inbound",
  "recording_url": "https://api.callrail.com/v3/a/AC987654321/calls/CAL123456789/recording.json",
  "answered": true,
  "first_call": true,
  "value": "unknown",
  "good_call": null,
  "tags": [],
  "note": "",
  "business_phone_number": "+15559876543",
  "customer_name": "",
  "customer_phone_number": "+15551234567",
  "customer_city": "Los Angeles",
  "customer_state": "CA",
  "customer_country": "US",
  "lead_status": "good_lead",
  "tenant_id": "tenant_12345",  // ← Our custom field
  "callrail_company_id": "12345" // ← For mapping to our tenant
}
```

**Processing Steps:**
1. **Verify Webhook Signature**
   ```go
   signature := r.Header.Get("x-callrail-signature")
   if !verifyHMACSignature(payload, signature, webhookSecret) {
       return http.StatusUnauthorized, "Invalid signature"
   }
   ```

2. **Extract Key Fields**
   ```go
   var webhook CallRailWebhook
   json.Unmarshal(payload, &webhook)

   callID := webhook.CallID
   tenantID := webhook.TenantID
   callRailCompanyID := webhook.CallRailCompanyID
   recordingURL := webhook.RecordingURL
   ```

---

### **STEP 3: Tenant Authentication & Mapping**
**Database Query**: Cloud Spanner `offices` table

**SQL Query**:
```sql
SELECT tenant_id, workflow_config, callrail_api_key
FROM offices
WHERE callrail_company_id = @callrail_company_id
AND tenant_id = @tenant_id
AND status = 'active'
```

**Validation Logic**:
```go
if office == nil {
    return errors.New("invalid tenant_id or callrail_company_id mapping")
}

// Load tenant-specific CallRail API credentials
apiKey := office.CallRailAPIKey
workflowConfig := office.WorkflowConfig
```

**Database Schema Update Required**:
```sql
-- Add CallRail mapping to offices table
ALTER TABLE offices ADD COLUMN callrail_company_id STRING(50);
ALTER TABLE offices ADD COLUMN callrail_api_key STRING(100);
CREATE INDEX idx_offices_callrail ON offices(callrail_company_id, tenant_id);
```

---

### **STEP 4: Call Details Retrieval**
**CallRail API Endpoint**: `GET /v3/a/{account_id}/calls/{call_id}.json`
**Authentication**: `Authorization: Token token="YOUR_API_KEY"`

**API Request**:
```go
func getCallDetails(accountID, callID, apiKey string) (*CallDetails, error) {
    url := fmt.Sprintf("https://api.callrail.com/v3/a/%s/calls/%s.json", accountID, callID)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", fmt.Sprintf("Token token=\"%s\"", apiKey))

    resp, err := client.Do(req)
    // Process response...
}
```

**Expected Response Fields**:
```json
{
  "id": "CAL123456789",
  "answered": true,
  "business_phone_number": "+15559876543",
  "caller_id": "+15551234567",
  "company_id": "12345",
  "created_at": "2025-09-13T10:30:00Z",
  "customer_city": "Los Angeles",
  "customer_country": "US",
  "customer_name": "John Smith",
  "customer_phone_number": "+15551234567",
  "customer_state": "CA",
  "direction": "inbound",
  "duration": 180,
  "first_call": true,
  "formatted_business_phone_number": "(555) 987-6543",
  "formatted_customer_location": "Los Angeles, CA",
  "formatted_customer_phone_number": "(555) 123-4567",
  "formatted_duration": "3:00",
  "good_call": null,
  "lead_status": "good_lead",
  "note": "",
  "source": "google_organic",
  "start_time": "2025-09-13T10:30:00Z",
  "tags": ["interested", "kitchen_remodel"],
  "tracking_phone_number": "+15559876543",
  "value": "high",
  "recording": "CAL123456789_recording"
}
```

---

### **STEP 5: Call Recording Download**
**CallRail API Endpoint**: `GET /v3/a/{account_id}/calls/{call_id}/recording.json`

**API Request**:
```go
func getCallRecording(accountID, callID, apiKey string) (*RecordingDetails, error) {
    url := fmt.Sprintf("https://api.callrail.com/v3/a/%s/calls/%s/recording.json", accountID, callID)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", fmt.Sprintf("Token token=\"%s\"", apiKey))

    resp, err := client.Do(req)
    // Process response...
}
```

**Expected Recording Response**:
```json
{
  "call_id": "CAL123456789",
  "recording_url": "https://storage.callrail.com/recordings/CAL123456789.mp3",
  "duration": 180,
  "file_size": 2856432,
  "format": "mp3",
  "created_at": "2025-09-13T10:33:15Z"
}
```

**Recording Download Process**:
```go
func downloadRecording(recordingURL, apiKey string) ([]byte, error) {
    req, _ := http.NewRequest("GET", recordingURL, nil)
    req.Header.Set("Authorization", fmt.Sprintf("Token token=\"%s\"", apiKey))

    resp, err := client.Do(req)
    return ioutil.ReadAll(resp.Body)
}
```

---

### **STEP 6: Audio File Storage**
**Service**: Cloud Storage
**Bucket Structure**: `gs://tenant-audio-files/{tenant_id}/calls/{call_id}.mp3`

**Storage Process**:
```go
func storeAudioFile(tenantID, callID string, audioData []byte) (string, error) {
    bucketName := "tenant-audio-files"
    objectPath := fmt.Sprintf("%s/calls/%s.mp3", tenantID, callID)

    // Upload to Cloud Storage
    ctx := context.Background()
    client, _ := storage.NewClient(ctx)
    bucket := client.Bucket(bucketName)
    obj := bucket.Object(objectPath)

    writer := obj.NewWriter(ctx)
    writer.Write(audioData)
    writer.Close()

    return fmt.Sprintf("gs://%s/%s", bucketName, objectPath), nil
}
```

**Lifecycle Policy**:
```json
{
  "rule": {
    "action": {"type": "SetStorageClass", "storageClass": "COLDLINE"},
    "condition": {"age": 90}
  }
}
```

---

### **STEP 7: Audio Transcription Processing**
**Service**: Speech-to-Text Chirp 3
**Features**: Speaker diarization, punctuation, confidence scoring

**Transcription Request**:
```go
func transcribeAudio(audioFileURL string) (*TranscriptionResult, error) {
    ctx := context.Background()
    client, _ := speech.NewClient(ctx)

    config := &speechpb.RecognitionConfig{
        Encoding:          speechpb.RecognitionConfig_MP3,
        SampleRateHertz:   8000,
        LanguageCode:      "en-US",
        EnableAutomaticPunctuation: true,
        EnableSpeakerDiarization:  true,
        DiarizationSpeakerCount:   2,
        EnableWordTimeOffsets:     true,
        EnableWordConfidence:      true,
    }

    audio := &speechpb.RecognitionAudio{
        AudioSource: &speechpb.RecognitionAudio_Uri{Uri: audioFileURL},
    }

    req := &speechpb.LongRunningRecognizeRequest{
        Config: config,
        Audio:  audio,
    }

    op, err := client.LongRunningRecognize(ctx, req)
    // Process operation...
}
```

**Expected Transcription Output**:
```json
{
  "transcript": "Hi, I'm interested in a kitchen remodel. Can someone call me back to discuss pricing?",
  "confidence": 0.95,
  "speaker_diarization": [
    {"speaker": 1, "start_time": "0.0s", "end_time": "4.2s", "text": "Hi, I'm interested in a kitchen remodel."},
    {"speaker": 2, "start_time": "4.5s", "end_time": "8.1s", "text": "Absolutely! I'd be happy to help you with that."}
  ],
  "word_details": [
    {"word": "kitchen", "start_time": "1.2s", "end_time": "1.8s", "confidence": 0.98}
  ]
}
```

---

### **STEP 8: AI Content Analysis**
**Service**: Vertex AI Gemini 2.5 Flash
**Purpose**: Extract intent, sentiment, project details, urgency

**AI Analysis Prompt**:
```
Analyze this phone call transcription for a home remodeling company:

TRANSCRIPT: {transcription_text}
CALL METADATA: {call_details}

Extract the following information in JSON format:
1. Customer intent (quote_request, information_seeking, appointment_booking, complaint, etc.)
2. Project type (kitchen, bathroom, whole_home, addition, etc.)
3. Timeline urgency (immediate, 1-3_months, 3-6_months, 6+_months, unknown)
4. Budget indicators (high, medium, low, unknown)
5. Customer sentiment (positive, neutral, negative)
6. Lead quality score (1-100)
7. Key details mentioned
8. Follow-up required (yes/no)
9. Appointment requested (yes/no)
```

**AI Response Processing**:
```go
func analyzeCallContent(transcript string, callDetails CallDetails) (*CallAnalysis, error) {
    prompt := buildAnalysisPrompt(transcript, callDetails)

    client := getGeminiClient()
    response, err := client.AnalyzeContent(prompt)

    var analysis CallAnalysis
    json.Unmarshal(response, &analysis)

    return &analysis, nil
}
```

---

### **STEP 9: Enhanced JSON Payload Creation**
**Purpose**: Combine all data sources into unified payload for workflow processing

**Enhanced Payload Structure**:
```json
{
  "request_id": "req_987654321",
  "tenant_id": "tenant_12345",
  "source": "callrail_webhook",
  "request_type": "phone_call",
  "communication_mode": "phone_call",
  "created_at": "2025-09-13T10:33:00Z",

  "original_webhook": {
    "call_id": "CAL123456789",
    "caller_id": "+15551234567",
    "duration": 180,
    "start_time": "2025-09-13T10:30:00Z",
    "answered": true,
    "lead_status": "good_lead"
  },

  "call_details": {
    "customer_name": "John Smith",
    "customer_phone": "+15551234567",
    "customer_location": "Los Angeles, CA",
    "business_phone": "+15559876543",
    "source": "google_organic",
    "tags": ["interested", "kitchen_remodel"],
    "value": "high",
    "good_call": true
  },

  "audio_processing": {
    "recording_url": "gs://tenant-audio-files/tenant_12345/calls/CAL123456789.mp3",
    "transcription": "Hi, I'm interested in a kitchen remodel. Can someone call me back to discuss pricing?",
    "confidence": 0.95,
    "duration": 180,
    "speaker_count": 2
  },

  "ai_analysis": {
    "intent": "quote_request",
    "project_type": "kitchen",
    "timeline": "1-3_months",
    "budget_indicator": "medium",
    "sentiment": "positive",
    "lead_score": 85,
    "urgency": "medium",
    "appointment_requested": false,
    "follow_up_required": true,
    "key_details": [
      "Kitchen remodel interest",
      "Pricing discussion needed",
      "Callback requested"
    ]
  },

  "spam_likelihood": 5,
  "processing_metadata": {
    "processed_at": "2025-09-13T10:35:00Z",
    "processing_time_ms": 2500,
    "gemini_model": "gemini-2.5-flash",
    "speech_model": "chirp-3"
  }
}
```

---

### **STEP 10: Workflow Engine Processing**
**Flow**: Same as original plan but with enhanced call data

**Workflow Steps Executed**:
1. ✅ **Communication Detection**: Already identified as "phone_call"
2. ✅ **Validation**: Spam check with AI analysis (5% likelihood)
3. ✅ **Service Area**: Validate "Los Angeles, CA" against tenant coverage
4. ✅ **CRM Integration**: Push enriched lead data to tenant's CRM
5. ✅ **Email Notifications**: Send lead alert with call summary
6. ✅ **Database Storage**: Insert complete record to `requests` table

---

### **STEP 11: Database Storage**
**Table**: `requests` with enhanced schema

**Required Schema Updates**:
```sql
-- Add CallRail-specific fields to requests table
ALTER TABLE requests ADD COLUMN call_id STRING(50);
ALTER TABLE requests ADD COLUMN recording_url STRING(500);
ALTER TABLE requests ADD COLUMN transcription_data JSON;
ALTER TABLE requests ADD COLUMN ai_analysis JSON;
ALTER TABLE requests ADD COLUMN lead_score INT64;

-- Add indexes for CallRail data
CREATE INDEX idx_requests_call_id ON requests(call_id);
CREATE INDEX idx_requests_lead_score ON requests(tenant_id, lead_score DESC);
```

**Insert Statement**:
```sql
INSERT INTO requests (
  tenant_id, source, request_type, status, data,
  ai_normalized, ai_extracted, call_id, recording_url,
  transcription_data, ai_analysis, lead_score, created_at
) VALUES (
  @tenant_id, 'callrail_webhook', 'phone_call', 'processed',
  @original_payload, @enhanced_data, @ai_analysis_results,
  @call_id, @recording_url, @transcription, @ai_analysis, @lead_score,
  CURRENT_TIMESTAMP()
)
```

---

## 🔄 **Error Handling & Retry Logic**

### **Webhook Processing Failures**:
```go
func processCallRailWebhook(payload []byte) error {
    maxRetries := 3
    backoff := time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := processWebhookInternal(payload)
        if err == nil {
            return nil
        }

        log.Printf("Attempt %d failed: %v", attempt+1, err)
        time.Sleep(backoff)
        backoff *= 2
    }

    // Send to dead letter queue
    return sendToDeadLetterQueue(payload)
}
```

### **API Rate Limiting**:
- CallRail API: 120 requests/minute
- Implement exponential backoff for rate limit errors
- Queue API requests during high volume periods

### **Recording Download Failures**:
- Retry with exponential backoff (1s, 2s, 4s, 8s)
- Store partial data if recording unavailable
- Mark for manual review after max retries

---

## 📊 **Monitoring & Analytics**

### **Key Metrics to Track**:
- Webhook processing time (target: <500ms)
- Recording download success rate (target: >99%)
- Transcription accuracy (target: >95%)
- AI analysis quality scores
- End-to-end processing time (target: <3 minutes)

### **Alerting Conditions**:
- Webhook signature failures
- High spam likelihood calls (>80%)
- Recording download failures
- Transcription processing errors
- CRM push failures

---

## 💰 **Cost Implications**

### **Additional Costs for CallRail Integration**:
- **Speech-to-Text**: $0.024/minute of audio
- **Cloud Storage**: $0.020/GB for audio files
- **Gemini Analysis**: $0.075/1K input tokens
- **Data Transfer**: $0.12/GB for recording downloads

### **Monthly Cost Estimate** (500 calls/month, 3 min average):
- Audio transcription: 1,500 minutes × $0.024 = $36
- Cloud Storage: 50GB × $0.020 = $1
- AI Analysis: ~$25 for call analysis
- **Total Additional**: ~$62/month per tenant

---

## 🚀 **Implementation Timeline**

### **Phase 1** (Week 1-2): Basic Integration
- Webhook endpoint setup
- Signature verification
- Basic tenant mapping

### **Phase 2** (Week 3-4): Audio Processing
- Recording download implementation
- Speech-to-Text integration
- Cloud Storage setup

### **Phase 3** (Week 5-6): AI Enhancement
- Gemini content analysis
- Enhanced payload generation
- Database schema updates

### **Phase 4** (Week 7-8): Testing & Optimization
- End-to-end testing
- Performance optimization
- Error handling refinement

This comprehensive flow ensures secure, efficient processing of CallRail webhooks with full audio transcription and AI-powered analysis.