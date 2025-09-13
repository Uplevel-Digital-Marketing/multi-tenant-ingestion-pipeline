# Troubleshooting Guide

**Multi-Tenant Ingestion Pipeline**

---

**Version**: 2.0
**Date**: September 13, 2025
**Audience**: Technical Support, System Administrators, End Users

---

## Table of Contents

1. [Quick Diagnostics](#quick-diagnostics)
2. [Webhook Processing Issues](#webhook-processing-issues)
3. [CRM Integration Problems](#crm-integration-problems)
4. [Performance Issues](#performance-issues)
5. [Authentication & Security](#authentication--security)
6. [Data Quality Issues](#data-quality-issues)
7. [System Health Problems](#system-health-problems)
8. [Network and Connectivity](#network-and-connectivity)
9. [Database Issues](#database-issues)
10. [Emergency Procedures](#emergency-procedures)

---

## Quick Diagnostics

### System Health Check

Before diving into specific issues, run these quick health checks:

#### 1. Overall System Status

```bash
# Check main API endpoint
curl -f https://api.company.com/v1/health

# Expected response:
{
  "status": "healthy",
  "timestamp": "2025-09-13T14:30:00Z",
  "services": {
    "database": "healthy",
    "storage": "healthy",
    "speech_api": "healthy",
    "vertex_ai": "healthy"
  }
}
```

#### 2. Dashboard Quick Check

**✅ Green Indicators (All Good)**
- System status: 🟢 Operational
- Processing queue: 0 pending
- Error rate: < 1%
- CRM sync: > 95%

**⚠️ Yellow Indicators (Attention Needed)**
- Processing delays: > 2 minutes average
- Error rate: 1-5%
- CRM sync: 90-95%

**❌ Red Indicators (Action Required)**
- System status: 🔴 Degraded/Down
- Processing queue: > 50 pending
- Error rate: > 5%
- CRM sync: < 90%

#### 3. Recent Error Summary

```bash
# Check last 10 errors
gcloud logging read 'resource.type="cloud_run_revision" severity>=ERROR' \
  --limit=10 --format="table(timestamp,resource.labels.service_name,textPayload)"
```

### Common Error Patterns

| Error Pattern | Likely Issue | Quick Fix |
|---------------|--------------|-----------|
| **"connection refused"** | Service down | Check Cloud Run status |
| **"timeout"** | Performance issue | Check resource limits |
| **"unauthorized"** | Auth problem | Verify API keys/tokens |
| **"rate limit exceeded"** | Too many requests | Implement backoff |
| **"invalid signature"** | Webhook auth issue | Check webhook secret |

---

## Webhook Processing Issues

### Issue: Calls Not Appearing in Dashboard

#### Symptoms
- CallRail shows call completed
- No corresponding record in pipeline dashboard
- Webhook delivery appears successful in CallRail logs

#### Diagnostic Steps

**1. Check CallRail Webhook Configuration**

```bash
# Verify webhook URL is correct
Expected: https://api.company.com/v1/callrail/webhook
Common mistakes:
❌ http:// instead of https://
❌ Missing /v1/ in path
❌ Wrong domain name
```

**2. Check Webhook Delivery Logs**

In CallRail dashboard:
1. Go to Settings → Integrations → Webhooks
2. Click on your webhook
3. Review "Recent Deliveries" tab
4. Look for HTTP status codes:
   - ✅ 200: Success
   - ❌ 4xx: Client error (auth, validation)
   - ❌ 5xx: Server error

**3. Verify Webhook Payload**

```json
// Minimum required payload
{
  "call_id": "CAL123456789",
  "tenant_id": "your_tenant_id",  // Required!
  "caller_id": "+15551234567",
  "duration": "180"
}
```

#### Solutions

**Solution 1: Fix Webhook Configuration**
```bash
# Update CallRail webhook settings
URL: https://api.company.com/v1/callrail/webhook
Method: POST
Content-Type: application/json
Events: ☑️ call_completed

# Required custom fields
tenant_id: your_actual_tenant_id
```

**Solution 2: Test Webhook Manually**
```bash
# Test webhook with curl
curl -X POST https://api.company.com/v1/callrail/webhook \
  -H "Content-Type: application/json" \
  -H "x-callrail-signature: sha256=test_signature" \
  -H "x-timestamp: $(date +%s)" \
  -d '{
    "call_id": "TEST123",
    "tenant_id": "your_tenant_id",
    "caller_id": "+15551234567",
    "duration": "120"
  }'
```

**Solution 3: Check Signature Verification**
```python
import hmac
import hashlib
import time

# Verify signature calculation
webhook_secret = "your_webhook_secret"
timestamp = str(int(time.time()))
payload = '{"call_id":"TEST123"}'

message = f"{timestamp}.{payload}"
signature = hmac.new(
    webhook_secret.encode('utf-8'),
    message.encode('utf-8'),
    hashlib.sha256
).hexdigest()

print(f"Expected signature: sha256={signature}")
```

### Issue: Processing Stuck in "Pending" Status

#### Symptoms
- Calls appear in dashboard but stay in "Processing" status
- No AI analysis results
- CRM sync never occurs

#### Diagnostic Steps

**1. Check Processing Queue**
```bash
# Check Pub/Sub subscription backlog
gcloud pubsub subscriptions describe webhook-processing-subscription \
  --format="value(numUndeliveredMessages)"

# Check Cloud Run processing service
gcloud run services describe audio-processor --region=us-central1
```

**2. Check Service Logs**
```bash
# Check for processing errors
gcloud logging read 'resource.type="cloud_run_revision"
  resource.labels.service_name="audio-processor" severity>=ERROR' \
  --limit=20
```

#### Solutions

**Solution 1: Clear Queue Backlog**
```bash
# Scale up processing service
gcloud run services update audio-processor \
  --region=us-central1 \
  --min-instances=5 \
  --max-instances=50

# Check queue processing rate
watch "gcloud pubsub subscriptions describe webhook-processing-subscription \
  --format='value(numUndeliveredMessages)'"
```

**Solution 2: Restart Processing Services**
```bash
# Deploy new revision to restart services
gcloud run deploy audio-processor \
  --image=gcr.io/$PROJECT_ID/pipeline:current \
  --region=us-central1
```

**Solution 3: Manual Reprocess**
```bash
# Requeue stuck messages (admin only)
python3 scripts/reprocess_stuck_calls.py --call-id=CAL123456789
```

### Issue: Audio Processing Failures

#### Symptoms
- Calls process but no transcript generated
- Error message: "Audio download failed" or "STT processing error"

#### Diagnostic Steps

**1. Verify Audio File Access**
```bash
# Test audio file URL directly
curl -I "https://callrail-recordings.s3.amazonaws.com/call123.wav"

# Should return HTTP 200 with audio content type
```

**2. Check Speech-to-Text Quotas**
```bash
# Check STT API quota usage
gcloud logging read 'resource.type="consumed_api"
  resource.labels.service="speech.googleapis.com"' \
  --limit=10
```

#### Solutions

**Solution 1: Audio Format Issues**
```python
# Check supported audio formats
supported_formats = [
    'audio/wav', 'audio/mp3', 'audio/flac',
    'audio/amr', 'audio/ogg'
]

# Convert if necessary using ffmpeg
import subprocess
subprocess.run([
    'ffmpeg', '-i', 'input.mp3',
    '-ar', '16000', '-ac', '1', 'output.wav'
])
```

**Solution 2: Increase STT Quotas**
```bash
# Request quota increase in Google Cloud Console
# Or implement queuing for high-volume periods
```

---

## CRM Integration Problems

### Issue: HubSpot Sync Failures

#### Symptoms
- Error: "HubSpot API key invalid"
- Error: "Rate limit exceeded"
- Error: "Required property missing"

#### Diagnostic Steps

**1. Test API Connection**
```bash
# Test HubSpot API directly
curl -H "Authorization: Bearer YOUR_API_KEY" \
  "https://api.hubapi.com/contacts/v1/lists/all/contacts/all?count=1"
```

**2. Check Required Properties**
```bash
# Get HubSpot contact properties
curl -H "Authorization: Bearer YOUR_API_KEY" \
  "https://api.hubapi.com/properties/v1/contacts/properties" \
  | jq '.[] | select(.required == true)'
```

#### Solutions

**Solution 1: Update API Key**
```bash
# Generate new API key in HubSpot
1. Go to Settings → Integrations → API Key
2. Generate new key
3. Update in Pipeline Settings → CRM Integration
4. Test connection
```

**Solution 2: Fix Field Mappings**
```json
// Ensure all required HubSpot fields are mapped
{
  "email": "{{caller_email}}",
  "firstname": "{{caller_name.first}}",
  "lastname": "{{caller_name.last}}",
  "phone": "{{caller_phone}}",
  "lifecyclestage": "lead"  // Required by HubSpot
}
```

**Solution 3: Handle Rate Limits**
```python
# Implement exponential backoff
import time
import random

def hubspot_api_call_with_retry(api_call, max_retries=3):
    for attempt in range(max_retries):
        try:
            return api_call()
        except RateLimitException as e:
            if attempt == max_retries - 1:
                raise

            # Exponential backoff with jitter
            delay = (2 ** attempt) + random.uniform(0, 1)
            time.sleep(delay)
```

### Issue: Salesforce Integration Failures

#### Symptoms
- Error: "INVALID_LOGIN: Invalid username, password, security token"
- Error: "Required field missing"
- Error: "Duplicate rule violation"

#### Diagnostic Steps

**1. Test Salesforce Authentication**
```python
# Test Salesforce login
import requests
import xml.etree.ElementTree as ET

login_soap = """<?xml version="1.0" encoding="utf-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:urn="urn:enterprise.soap.sforce.com">
  <soapenv:Body>
    <urn:login>
      <urn:username>your_username</urn:username>
      <urn:password>your_password_and_token</urn:password>
    </urn:login>
  </soapenv:Body>
</soapenv:Envelope>"""

response = requests.post(
    'https://login.salesforce.com/services/Soap/c/54.0',
    data=login_soap,
    headers={'Content-Type': 'text/xml; charset=utf-8'}
)
```

**2. Check Required Fields**
```bash
# Use Salesforce CLI to check Lead object
sfdx force:schema:sobject:describe -s Lead -u your_org
```

#### Solutions

**Solution 1: Fix Authentication**
```bash
# Reset security token in Salesforce
1. Go to Setup → Personal Setup → My Personal Information → Reset My Security Token
2. Combine password + security token
3. Update credentials in Pipeline settings
```

**Solution 2: Handle Duplicate Rules**
```json
// Configure duplicate handling
{
  "duplicate_rule_header": "DuplicateRuleHeader",
  "allow_save_on_duplicate_rule_warning": true,
  "include_duplicates_header": "IncludeDuplicatesHeader",
  "duplicate_rule_id": "your_rule_id"
}
```

### Issue: Multiple CRM Sync Conflicts

#### Symptoms
- Same lead created multiple times
- Inconsistent data across CRMs
- Sync timing conflicts

#### Solutions

**Solution 1: Implement Sync Ordering**
```yaml
# Configure sync sequence
sync_order:
  - hubspot     # Primary CRM first
  - salesforce  # Secondary CRM second
  - pipedrive   # Additional CRM last

sync_delay: 5  # seconds between syncs
```

**Solution 2: Master Record Strategy**
```python
# Designate primary CRM as master
primary_crm = "hubspot"
secondary_crms = ["salesforce", "pipedrive"]

def sync_to_crms(lead_data):
    # Sync to primary first
    primary_id = sync_to_primary(lead_data)

    # Add primary CRM ID to secondary sync
    lead_data["primary_crm_id"] = primary_id

    # Sync to secondary CRMs
    for crm in secondary_crms:
        sync_to_secondary(crm, lead_data)
```

---

## Performance Issues

### Issue: High Latency Response Times

#### Symptoms
- Webhook response times > 5 seconds
- Dashboard loading slowly
- API timeouts

#### Diagnostic Steps

**1. Check Service Performance**
```bash
# Check Cloud Run metrics
gcloud monitoring metrics list \
  --filter="metric.type:run.googleapis.com/request_latencies"

# Check current instance count
gcloud run services describe webhook-processor --region=us-central1 \
  --format="value(status.traffic[0].percent,status.traffic[0].latestRevision)"
```

**2. Identify Bottlenecks**
```bash
# Check database query performance
gcloud logging read 'resource.type="spanner_instance" severity>=WARNING' \
  --filter="textPayload:\"slow query\"" --limit=10

# Check memory/CPU usage
gcloud monitoring metrics list \
  --filter="metric.type:run.googleapis.com/container/cpu/utilizations"
```

#### Solutions

**Solution 1: Scale Up Services**
```bash
# Increase Cloud Run resources
gcloud run services update webhook-processor \
  --region=us-central1 \
  --memory=8Gi \
  --cpu=4 \
  --min-instances=10 \
  --max-instances=200

# Reduce cold starts
gcloud run services update webhook-processor \
  --region=us-central1 \
  --min-instances=20
```

**Solution 2: Optimize Database Queries**
```sql
-- Add indexes for common queries
CREATE INDEX idx_tenant_created_at
ON processing_requests (tenant_id, created_at);

CREATE INDEX idx_call_status
ON call_records (tenant_id, status, created_at);
```

**Solution 3: Implement Caching**
```python
# Add Redis caching for frequent queries
import redis

cache = redis.Redis(host='redis-host', port=6379)

def get_tenant_config(tenant_id):
    cache_key = f"tenant_config:{tenant_id}"
    cached = cache.get(cache_key)

    if cached:
        return json.loads(cached)

    # Fetch from database
    config = fetch_from_db(tenant_id)

    # Cache for 5 minutes
    cache.setex(cache_key, 300, json.dumps(config))
    return config
```

### Issue: Memory Leaks and High Resource Usage

#### Symptoms
- Services restarting frequently
- Out of memory errors
- High CPU usage with low load

#### Diagnostic Steps

**1. Monitor Resource Usage**
```bash
# Check memory trends
gcloud monitoring metrics list \
  --filter="metric.type:run.googleapis.com/container/memory/utilizations"

# Check for memory-related restarts
gcloud logging read 'resource.type="cloud_run_revision"
  textPayload:"killed" OR textPayload:"OOMKilled"' --limit=20
```

**2. Analyze Memory Patterns**
```bash
# Check goroutine count (for Go services)
curl https://api.company.com/debug/pprof/goroutine?debug=1

# Check heap usage
curl https://api.company.com/debug/pprof/heap?debug=1
```

#### Solutions

**Solution 1: Optimize Memory Usage**
```go
// Close HTTP response bodies
defer response.Body.Close()

// Use connection pooling
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     30 * time.Second,
}
```

**Solution 2: Implement Resource Limits**
```yaml
# Cloud Run resource configuration
resources:
  limits:
    memory: "4Gi"
    cpu: "2"
  requests:
    memory: "2Gi"
    cpu: "1"
```

---

## Authentication & Security

### Issue: Webhook Signature Failures

#### Symptoms
- Error: "Invalid webhook signature"
- HTTP 401 responses to webhooks
- CallRail shows successful delivery, but processing fails

#### Diagnostic Steps

**1. Verify Signature Calculation**
```python
import hmac
import hashlib

def verify_signature(payload, timestamp, signature, secret):
    message = f"{timestamp}.{payload}"
    expected = hmac.new(
        secret.encode('utf-8'),
        message.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()

    return hmac.compare_digest(signature, f"sha256={expected}")

# Test with actual values
payload = '{"call_id":"TEST123"}'
timestamp = "1694617200"
signature = "sha256=actual_signature_from_header"
secret = "your_webhook_secret"

print(verify_signature(payload, timestamp, signature, secret))
```

**2. Check Timestamp Window**
```python
import time

def is_timestamp_valid(timestamp_str, window_seconds=300):
    try:
        timestamp = int(timestamp_str)
        now = int(time.time())
        return abs(now - timestamp) <= window_seconds
    except ValueError:
        return False
```

#### Solutions

**Solution 1: Update Webhook Secret**
```bash
# Update secret in both systems
1. Generate new secure secret: openssl rand -hex 32
2. Update in CallRail webhook settings
3. Update in Pipeline secret manager
4. Test webhook delivery
```

**Solution 2: Fix Timestamp Issues**
```bash
# Ensure server time is synchronized
sudo ntpdate -s time.nist.gov

# Check system time
date -u
```

### Issue: JWT Token Authentication Failures

#### Symptoms
- Error: "Invalid token" or "Token expired"
- Unable to access management API
- Dashboard login failures

#### Diagnostic Steps

**1. Verify Token Format**
```python
import jwt
import json

def decode_token_without_verification(token):
    # Decode without verification to inspect claims
    try:
        header = jwt.get_unverified_header(token)
        payload = jwt.decode(token, options={"verify_signature": False})
        print(f"Header: {json.dumps(header, indent=2)}")
        print(f"Payload: {json.dumps(payload, indent=2)}")
    except jwt.InvalidTokenError as e:
        print(f"Token decode error: {e}")
```

**2. Check Token Expiration**
```python
import time

def check_token_expiry(token):
    try:
        payload = jwt.decode(token, options={"verify_signature": False})
        exp = payload.get('exp', 0)
        now = int(time.time())

        if exp < now:
            print(f"Token expired {now - exp} seconds ago")
        else:
            print(f"Token valid for {exp - now} seconds")
    except:
        print("Cannot parse token expiration")
```

#### Solutions

**Solution 1: Refresh Expired Tokens**
```python
# Implement token refresh logic
def refresh_access_token(refresh_token):
    response = requests.post('https://api.company.com/v1/auth/refresh', {
        'refresh_token': refresh_token
    })

    if response.status_code == 200:
        return response.json()['access_token']
    else:
        raise AuthenticationError("Token refresh failed")
```

**Solution 2: Fix Token Claims**
```python
# Ensure required claims are present
def create_jwt_token(user_id, tenant_id):
    payload = {
        'sub': user_id,
        'tenant_id': tenant_id,
        'iat': int(time.time()),
        'exp': int(time.time()) + 3600,  # 1 hour
        'iss': 'pipeline-api',
        'aud': 'pipeline-dashboard'
    }

    return jwt.encode(payload, secret_key, algorithm='RS256')
```

---

## Data Quality Issues

### Issue: Poor Lead Scoring Accuracy

#### Symptoms
- High-quality leads getting low scores
- Low-quality leads getting high scores
- Inconsistent scoring across similar calls

#### Diagnostic Steps

**1. Review Sample Transcripts**
```sql
-- Get sample of high and low scored calls
SELECT
  call_id,
  ai_analysis.lead_score,
  ai_analysis.transcript,
  ai_analysis.confidence
FROM call_records
WHERE tenant_id = 'your_tenant_id'
AND created_at >= CURRENT_DATE()
ORDER BY ai_analysis.lead_score DESC
LIMIT 10;
```

**2. Check Audio Quality**
```python
# Analyze audio file properties
import librosa

def analyze_audio_quality(audio_file):
    y, sr = librosa.load(audio_file)

    # Check signal-to-noise ratio
    signal_power = np.mean(y**2)
    noise_power = np.mean(y[:1000]**2)  # First second as noise baseline
    snr = 10 * np.log10(signal_power / noise_power)

    # Check duration
    duration = len(y) / sr

    return {
        'duration': duration,
        'sample_rate': sr,
        'snr_db': snr,
        'quality': 'good' if snr > 10 and duration > 30 else 'poor'
    }
```

#### Solutions

**Solution 1: Update Scoring Criteria**
```python
# Customize lead scoring weights
scoring_config = {
    'project_intent_weight': 0.3,
    'timeline_urgency_weight': 0.25,
    'budget_indicators_weight': 0.2,
    'call_duration_weight': 0.15,
    'customer_engagement_weight': 0.1
}

# Minimum thresholds
quality_thresholds = {
    'min_call_duration': 60,  # seconds
    'min_confidence_score': 0.7,
    'required_project_intent': True
}
```

**Solution 2: Improve Audio Processing**
```python
# Pre-process audio for better transcription
def enhance_audio_for_stt(audio_file):
    # Noise reduction
    y, sr = librosa.load(audio_file)
    y_reduced = nr.reduce_noise(y=y, sr=sr)

    # Normalize volume
    y_normalized = librosa.util.normalize(y_reduced)

    # Resample to 16kHz (optimal for STT)
    y_resampled = librosa.resample(y_normalized, orig_sr=sr, target_sr=16000)

    return y_resampled, 16000
```

### Issue: Incomplete Contact Information

#### Symptoms
- Missing phone numbers or names
- Invalid email addresses
- Inconsistent data formatting

#### Diagnostic Steps

**1. Analyze Data Completeness**
```sql
-- Check data completeness rates
SELECT
  COUNT(*) as total_records,
  COUNT(customer_name) as has_name,
  COUNT(customer_phone) as has_phone,
  COUNT(customer_email) as has_email,
  ROUND(COUNT(customer_name) * 100.0 / COUNT(*), 2) as name_completion_rate,
  ROUND(COUNT(customer_phone) * 100.0 / COUNT(*), 2) as phone_completion_rate,
  ROUND(COUNT(customer_email) * 100.0 / COUNT(*), 2) as email_completion_rate
FROM call_records
WHERE tenant_id = 'your_tenant_id'
AND created_at >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY);
```

**2. Check Data Validation Rules**
```python
import re

def validate_contact_data(record):
    errors = []

    # Phone validation
    phone = record.get('customer_phone', '')
    if not re.match(r'^\+?1?[0-9]{10}$', re.sub(r'[^\d]', '', phone)):
        errors.append('Invalid phone number format')

    # Email validation
    email = record.get('customer_email', '')
    if email and not re.match(r'^[^@]+@[^@]+\.[^@]+$', email):
        errors.append('Invalid email format')

    # Name validation
    name = record.get('customer_name', '')
    if len(name.strip()) < 2:
        errors.append('Name too short or missing')

    return errors
```

#### Solutions

**Solution 1: Enhance Data Extraction**
```python
# Improve AI extraction prompts
extraction_prompt = """
Analyze this call transcript and extract:
1. Customer name (first and last if available)
2. Phone number (format as +1XXXXXXXXXX)
3. Email address (if mentioned)
4. Project type and details
5. Timeline and urgency indicators
6. Budget range or indicators

If information is not clearly stated, mark as "not provided" rather than guessing.
"""
```

**Solution 2: Implement Data Validation**
```python
def clean_and_validate_data(raw_data):
    cleaned = {}

    # Clean phone number
    phone = raw_data.get('customer_phone', '')
    phone_digits = re.sub(r'[^\d]', '', phone)
    if len(phone_digits) == 10:
        cleaned['customer_phone'] = f"+1{phone_digits}"
    elif len(phone_digits) == 11 and phone_digits[0] == '1':
        cleaned['customer_phone'] = f"+{phone_digits}"

    # Clean name
    name = raw_data.get('customer_name', '').strip()
    if len(name) >= 2:
        cleaned['customer_name'] = name.title()

    # Validate email
    email = raw_data.get('customer_email', '').lower().strip()
    if re.match(r'^[^@]+@[^@]+\.[^@]+$', email):
        cleaned['customer_email'] = email

    return cleaned
```

---

## System Health Problems

### Issue: Database Connection Failures

#### Symptoms
- Error: "connection refused" or "timeout"
- Intermittent database errors
- High database latency

#### Diagnostic Steps

**1. Check Spanner Instance Status**
```bash
# Check instance health
gcloud spanner instances describe pipeline-prod

# Check current processing units
gcloud spanner instances describe pipeline-prod \
  --format="value(processingUnits,state)"

# Check recent operations
gcloud spanner operations list --instance=pipeline-prod --limit=10
```

**2. Monitor Database Metrics**
```bash
# Check CPU utilization
gcloud monitoring metrics list \
  --filter="resource.type=spanner_instance
  metric.type=spanner.googleapis.com/instance/cpu/utilization"

# Check connection count
gcloud logging read 'resource.type="spanner_instance"
  textPayload:"connection"' --limit=20
```

#### Solutions

**Solution 1: Scale Database Resources**
```bash
# Increase processing units
gcloud spanner instances update pipeline-prod \
  --processing-units=2000

# Monitor scaling progress
gcloud spanner operations list --instance=pipeline-prod \
  --filter="done=false"
```

**Solution 2: Optimize Connection Management**
```go
// Implement proper connection pooling
config := spanner.ClientConfig{
    NumChannels: 10,
}

client, err := spanner.NewClientWithConfig(ctx, database, config)
if err != nil {
    return err
}
defer client.Close()
```

**Solution 3: Implement Circuit Breaker**
```python
from circuit_breaker import CircuitBreaker

db_breaker = CircuitBreaker(
    failure_threshold=5,
    timeout=30,
    expected_exception=DatabaseError
)

@db_breaker
def database_operation():
    return execute_query()
```

### Issue: Storage Access Problems

#### Symptoms
- Audio files not accessible
- Upload/download failures
- Storage permission errors

#### Diagnostic Steps

**1. Test Storage Access**
```bash
# Test bucket access
gsutil ls gs://your-bucket-name/

# Test file download
gsutil cp gs://your-bucket-name/test-file.wav /tmp/

# Check bucket permissions
gsutil iam get gs://your-bucket-name/
```

**2. Check Service Account Permissions**
```bash
# Check service account roles
gcloud projects get-iam-policy $PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:pipeline-service@$PROJECT_ID.iam.gserviceaccount.com"
```

#### Solutions

**Solution 1: Fix Storage Permissions**
```bash
# Grant storage access to service account
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:pipeline-service@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"

# Set bucket-specific permissions
gsutil iam ch serviceAccount:pipeline-service@$PROJECT_ID.iam.gserviceaccount.com:objectAdmin \
  gs://your-bucket-name
```

**Solution 2: Implement Retry Logic**
```python
import time
import random
from google.api_core import retry

@retry.Retry(predicate=retry.if_exception_type(Exception))
def download_with_retry(bucket_name, file_name, destination):
    client = storage.Client()
    bucket = client.bucket(bucket_name)
    blob = bucket.blob(file_name)
    blob.download_to_filename(destination)
```

---

## Network and Connectivity

### Issue: External API Timeouts

#### Symptoms
- CallRail API timeouts
- CRM API connection failures
- Speech-to-Text service timeouts

#### Diagnostic Steps

**1. Test External Connectivity**
```bash
# Test CallRail API
curl -w "%{time_total}" -o /dev/null -s "https://api.callrail.com/v3/calls.json"

# Test HubSpot API
curl -w "%{time_total}" -o /dev/null -s "https://api.hubapi.com/contacts/v1/lists/all/contacts/all?count=1"

# Test Google Speech API
curl -w "%{time_total}" -o /dev/null -s "https://speech.googleapis.com/v1/speech:recognize"
```

**2. Check Network Latency**
```bash
# Check latency to key endpoints
ping -c 4 api.callrail.com
ping -c 4 api.hubapi.com
ping -c 4 speech.googleapis.com
```

#### Solutions

**Solution 1: Implement Timeouts and Retries**
```python
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

# Configure retry strategy
retry_strategy = Retry(
    total=3,
    backoff_factor=1,
    status_forcelist=[429, 500, 502, 503, 504],
)

adapter = HTTPAdapter(max_retries=retry_strategy)
session = requests.Session()
session.mount("http://", adapter)
session.mount("https://", adapter)

# Set reasonable timeouts
response = session.get(url, timeout=(5, 30))  # (connect, read)
```

**Solution 2: Use Connection Pooling**
```python
# Reuse connections for better performance
session = requests.Session()
session.mount('https://', HTTPAdapter(
    pool_connections=10,
    pool_maxsize=20
))
```

### Issue: DNS Resolution Problems

#### Symptoms
- Intermittent connection failures
- "Name or service not known" errors
- Load balancer routing issues

#### Solutions

**Solution 1: Configure Custom DNS**
```bash
# Use Google DNS for better reliability
echo "nameserver 8.8.8.8" >> /etc/resolv.conf
echo "nameserver 8.8.4.4" >> /etc/resolv.conf
```

**Solution 2: Implement DNS Caching**
```python
import socket

# Enable DNS caching
socket.setdefaulttimeout(10)

# Use IP addresses for critical services
CRITICAL_IPS = {
    'api.hubapi.com': '52.84.123.456',
    'api.callrail.com': '104.16.123.456'
}
```

---

## Emergency Procedures

### Critical System Outage

#### Immediate Actions (0-5 minutes)

**1. Acknowledge and Assess**
```bash
# Quick system check
curl -f https://api.company.com/v1/health || echo "SYSTEM DOWN"

# Check all services
gcloud run services list --region=us-central1 --filter="status.conditions.status=False"

# Check recent deployments
gcloud run revisions list --service=webhook-processor --region=us-central1 --limit=5
```

**2. Emergency Communication**
```bash
# Slack notification template
curl -X POST -H 'Content-type: application/json' \
  --data '{"text":"🚨 CRITICAL: Pipeline system outage detected. Investigating immediately."}' \
  $SLACK_WEBHOOK_URL

# Update status page
curl -X POST https://api.statuspage.io/v1/pages/PAGE_ID/incidents \
  -H "Authorization: OAuth YOUR_TOKEN" \
  -d '{"incident": {"name": "System Outage", "status": "investigating"}}'
```

**3. Immediate Mitigation**
```bash
# Rollback to previous version if recent deployment
PREVIOUS_REV=$(gcloud run revisions list --service=webhook-processor \
  --region=us-central1 --limit=2 --format="value(metadata.name)" | tail -n1)

gcloud run services update-traffic webhook-processor \
  --region=us-central1 --to-revisions=$PREVIOUS_REV=100

# Scale up if resource issue
gcloud run services update webhook-processor \
  --region=us-central1 --min-instances=10 --max-instances=500
```

### Data Recovery Procedures

#### Database Corruption Recovery

**1. Assess Damage**
```bash
# Check data integrity
gcloud spanner databases execute-sql pipeline-db --instance=pipeline-prod \
  --sql="SELECT COUNT(*) FROM tenants WHERE tenant_id IS NULL"

# Identify corruption timeframe
gcloud spanner databases execute-sql pipeline-db --instance=pipeline-prod \
  --sql="SELECT MIN(created_at), MAX(created_at) FROM processing_requests
         WHERE status = 'corrupted' OR tenant_id IS NULL"
```

**2. Stop Processing**
```bash
# Scale down to prevent further damage
gcloud run services update webhook-processor \
  --region=us-central1 --min-instances=0 --max-instances=0
```

**3. Restore from Backup**
```bash
# Find appropriate backup
gcloud spanner backups list --instance=pipeline-prod \
  --filter="state=READY AND createTime<'2025-09-13T12:00:00Z'" \
  --sort-by="~createTime" --limit=3

# Restore to new database
BACKUP_NAME="backup-2025-09-13-06-00"
gcloud spanner databases restore \
  --source-backup=projects/$PROJECT_ID/instances/pipeline-prod/backups/$BACKUP_NAME \
  --target-database=pipeline-db-restored \
  --target-instance=pipeline-prod

# Verify restored data
gcloud spanner databases execute-sql pipeline-db-restored --instance=pipeline-prod \
  --sql="SELECT COUNT(*) FROM tenants; SELECT MAX(created_at) FROM processing_requests"
```

### Security Incident Response

#### Suspected Breach

**1. Immediate Isolation**
```bash
# Block suspicious IP ranges
gcloud compute firewall-rules create emergency-block-suspicious \
  --action=DENY \
  --rules=tcp:80,tcp:443 \
  --source-ranges=SUSPICIOUS_IP/32,ANOTHER_IP/32 \
  --priority=100

# Rotate all API keys immediately
./scripts/emergency-credential-rotation.sh
```

**2. Evidence Preservation**
```bash
# Backup current logs
gsutil -m cp -r gs://$PROJECT_ID-logs/$(date +%Y/%m/%d)/ \
  gs://$PROJECT_ID-incident-$(date +%Y%m%d)/

# Enable additional audit logging
gcloud logging sinks create security-incident-sink \
  storage.googleapis.com/$PROJECT_ID-security-audit \
  --log-filter='severity>=INFO'
```

**3. Damage Assessment**
```bash
# Check for unauthorized access
gcloud logging read 'protoPayload.authenticationInfo.principalEmail!~".*@yourdomain.com$"
  AND severity>=WARNING' --limit=100

# Check for data access patterns
gcloud logging read 'resource.type="spanner_instance"
  protoPayload.methodName="google.spanner.v1.Spanner.ExecuteSql"' --limit=50
```

### Contact Information

#### Emergency Escalation

**Primary On-Call: +1-555-ONCALL1**
**Backup On-Call: +1-555-ONCALL2**
**Management Escalation: +1-555-MANAGER**
**Executive Escalation: +1-555-EXEC**

#### External Support

**Google Cloud Support: +1-855-836-3987**
- Have project ID ready: $PROJECT_ID
- Severity: Production Critical

**CallRail Emergency: +1-404-CALLRAIL**
**HubSpot Support: Available via help.hubspot.com**
**Salesforce Support: Available via help.salesforce.com**

---

## Support and Resources

### Getting Additional Help

**Internal Support:**
- Slack: #pipeline-support
- Email: support-internal@company.com
- Wiki: https://wiki.company.com/pipeline

**Customer Support:**
- Email: support@company.com
- Phone: +1-555-SUPPORT
- Chat: Available in dashboard

### Documentation Links

- **API Documentation**: https://docs.company.com/api
- **User Guide**: https://docs.company.com/user-guide
- **Architecture Guide**: https://docs.company.com/architecture
- **Security Guide**: https://docs.company.com/security

### Useful Commands Reference

```bash
# System health check
curl https://api.company.com/v1/health | jq '.'

# Service status
gcloud run services list --region=us-central1

# Recent errors
gcloud logging read 'severity>=ERROR' --limit=20 --freshness=1h

# Scale service
gcloud run services update SERVICE_NAME --min-instances=N --max-instances=M

# Database status
gcloud spanner instances describe pipeline-prod

# Emergency rollback
gcloud run services update-traffic SERVICE_NAME --to-revisions=REVISION=100

# Create backup
gcloud spanner backups create emergency-$(date +%s) --instance=pipeline-prod --database=pipeline-db
```

---

**Remember**: When troubleshooting, always start with the basics (health checks, recent changes, logs) before diving into complex solutions. Document any fixes for future reference and update this guide as needed.