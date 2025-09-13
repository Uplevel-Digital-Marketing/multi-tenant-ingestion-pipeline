# Troubleshooting Guide

## Table of Contents
- [Common Issues](#common-issues)
- [Webhook Problems](#webhook-problems)
- [Database Issues](#database-issues)
- [Audio Processing Problems](#audio-processing-problems)
- [CRM Integration Issues](#crm-integration-issues)
- [Performance Problems](#performance-problems)
- [Authentication Errors](#authentication-errors)
- [Diagnostic Tools](#diagnostic-tools)

## Common Issues

### Issue: Service Returns 503 Unavailable

**Symptoms:**
- All requests returning 503 errors
- Health check endpoint failing
- No requests being processed

**Possible Causes:**
1. Database connection failures
2. Insufficient Cloud Run capacity
3. Dependency service outages
4. Resource quota exhaustion

**Diagnosis Steps:**
```bash
# Check Cloud Run service status
gcloud run services describe ingestion-pipeline --region=us-central1

# Check recent deployments
gcloud run revisions list --service=ingestion-pipeline --region=us-central1

# Check logs for errors
gcloud logging read "severity>=ERROR" --limit=20

# Test database connectivity
gcloud spanner databases execute-sql pipeline-db \
  --instance=ingestion-db \
  --sql="SELECT 1"
```

**Resolution Steps:**
1. **Scale up Cloud Run instances:**
   ```bash
   gcloud run services update ingestion-pipeline \
     --region=us-central1 \
     --min-instances=3 \
     --max-instances=200
   ```

2. **Check and restart database connections:**
   ```bash
   # Restart the service to reset connections
   gcloud run services replace-traffic ingestion-pipeline \
     --to-latest --region=us-central1
   ```

3. **Verify resource quotas:**
   ```bash
   gcloud compute project-info describe --project=$PROJECT_ID
   ```

### Issue: High Latency (>5 seconds)

**Symptoms:**
- Slow response times
- Timeouts from CallRail webhooks
- Processing delays

**Diagnosis:**
```bash
# Check Cloud Run metrics
gcloud monitoring timeseries list \
  --filter='metric.type="run.googleapis.com/request_latencies"' \
  --interval.end-time=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --interval.start-time=$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)

# Check Spanner query performance
gcloud spanner databases execute-sql pipeline-db \
  --instance=ingestion-db \
  --sql="
    SELECT text, avg_latency_seconds, execution_count
    FROM SPANNER_SYS.QUERY_STATS_TOP_HOUR
    ORDER BY avg_latency_seconds DESC
    LIMIT 10"

# Check for slow external API calls
gcloud logging read 'jsonPayload.duration_ms>5000' --limit=20
```

**Resolution:**
1. **Optimize database queries:**
   ```sql
   -- Add missing indexes
   CREATE INDEX idx_requests_tenant_status ON requests(tenant_id, status);
   CREATE INDEX idx_requests_created_desc ON requests(created_at DESC);
   ```

2. **Increase CPU allocation:**
   ```bash
   gcloud run services update ingestion-pipeline \
     --cpu=2 --memory=4Gi --region=us-central1
   ```

3. **Implement request timeouts:**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

## Webhook Problems

### Issue: CallRail Webhook Signature Verification Fails

**Symptoms:**
- 401 Unauthorized responses
- "Invalid signature" errors in logs
- CallRail webhooks not processing

**Diagnosis:**
```bash
# Check recent webhook attempts
gcloud logging read '
  resource.type="cloud_run_revision" AND
  jsonPayload.component="webhook_processor" AND
  jsonPayload.error_type="signature_verification"
' --limit=10

# Verify webhook secret configuration
gcloud secrets versions access latest --secret=callrail-webhook-secret-tenant-TENANT_ID

# Check webhook endpoint configuration in CallRail
curl -X GET "https://api.callrail.com/v3/a/ACCOUNT_ID/webhooks.json" \
  -H "Authorization: Token token=\"CALLRAIL_API_KEY\""
```

**Common Fixes:**

1. **Verify webhook secret matches:**
   ```bash
   # Compare secrets
   echo "CallRail configured secret: [check CallRail dashboard]"
   gcloud secrets versions access latest --secret=callrail-webhook-secret-tenant-TENANT_ID
   ```

2. **Check signature calculation:**
   ```bash
   # Test signature generation locally
   PAYLOAD='{"test":"data"}'
   SECRET="your-webhook-secret"
   SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" -binary | base64)
   echo "sha256=$SIGNATURE"
   ```

3. **Validate webhook URL:**
   ```bash
   # Ensure URL is exactly: https://your-domain.com/v1/callrail/webhook
   # No trailing slashes or extra parameters
   ```

### Issue: Tenant ID Mapping Failures

**Symptoms:**
- "Invalid tenant_id or callrail_company_id mapping" errors
- Webhooks received but not processed
- No CRM data updates

**Diagnosis:**
```sql
-- Check tenant/office mapping
SELECT
    o.office_id,
    o.tenant_id,
    o.callrail_company_id,
    t.status as tenant_status,
    o.status as office_status
FROM offices o
JOIN tenants t ON o.tenant_id = t.tenant_id
WHERE o.callrail_company_id = 'COMPANY_ID_FROM_WEBHOOK';

-- Check recent webhook payloads
SELECT
    request_id,
    JSON_EXTRACT_SCALAR(data, '$.callrail_company_id') as company_id,
    JSON_EXTRACT_SCALAR(data, '$.tenant_id') as tenant_id,
    status,
    created_at
FROM requests
WHERE source = 'callrail_webhook'
ORDER BY created_at DESC
LIMIT 10;
```

**Resolution:**
1. **Verify office configuration:**
   ```sql
   -- Update office with correct CallRail company ID
   UPDATE offices
   SET callrail_company_id = 'CORRECT_COMPANY_ID'
   WHERE tenant_id = 'tenant_example'
     AND office_id = 'office_main';
   ```

2. **Check CallRail webhook configuration:**
   - Ensure custom fields include `tenant_id` and `callrail_company_id`
   - Verify values match database records

### Issue: Webhook Timeouts

**Symptoms:**
- CallRail retrying webhooks
- Partial processing
- "Request timeout" errors

**Diagnosis:**
```bash
# Check processing times
gcloud logging read '
  jsonPayload.component="webhook_processor" AND
  jsonPayload.duration_ms>30000
' --limit=20 --format="table(timestamp, jsonPayload.tenant_id, jsonPayload.duration_ms)"

# Check for external API delays
gcloud logging read '
  jsonPayload.api_call="speech_to_text" OR
  jsonPayload.api_call="vertex_ai" OR
  jsonPayload.api_call="crm_push"
' --format="table(timestamp, jsonPayload.api_call, jsonPayload.duration_ms)"
```

**Resolution:**
1. **Implement async processing:**
   ```go
   // Process webhook immediately, queue heavy work
   func processWebhook(w http.ResponseWriter, r *http.Request) {
       // Quick validation and response
       w.WriteHeader(http.StatusOK)
       w.Write([]byte(`{"success": true, "queued": true}`))

       // Queue for background processing
       go processCallAsync(webhookData)
   }
   ```

2. **Optimize external API calls:**
   ```go
   // Add timeouts and retries
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()

   client := &http.Client{Timeout: 30 * time.Second}
   ```

## Database Issues

### Issue: Spanner CPU Utilization > 85%

**Symptoms:**
- Slow query performance
- Lock contention
- Request timeouts

**Diagnosis:**
```sql
-- Check current CPU utilization
SELECT
    interval_end,
    avg_cpu_seconds,
    text as query_text
FROM SPANNER_SYS.QUERY_STATS_TOP_10_MINUTE
ORDER BY interval_end DESC, avg_cpu_seconds DESC
LIMIT 20;

-- Check lock contention
SELECT
    sample_time,
    lock_wait_time_seconds,
    query_text
FROM SPANNER_SYS.LOCK_STATS_TOP_10_MINUTE
WHERE lock_wait_time_seconds > 1
ORDER BY sample_time DESC
LIMIT 10;

-- Check hot partitions
SELECT
    read_columns,
    where_clause,
    execution_count,
    avg_cpu_seconds
FROM SPANNER_SYS.QUERY_STATS_TOP_HOUR
WHERE execution_count > 1000
ORDER BY avg_cpu_seconds DESC;
```

**Resolution:**
1. **Scale processing units:**
   ```bash
   gcloud spanner instances update ingestion-db --processing-units=2000
   ```

2. **Optimize queries:**
   ```sql
   -- Add covering indexes
   CREATE INDEX idx_requests_covering ON requests(
       tenant_id,
       status,
       created_at DESC
   ) STORING (request_id, source, request_type);

   -- Use query hints for large scans
   SELECT * FROM requests@{FORCE_INDEX=idx_requests_tenant_created}
   WHERE tenant_id = @tenant_id
   ORDER BY created_at DESC
   LIMIT 100;
   ```

3. **Implement connection pooling:**
   ```go
   config := spanner.ClientConfig{
       SessionPoolConfig: spanner.SessionPoolConfig{
           MinOpened:       10,
           MaxOpened:       100,
           MaxIdle:         50,
           WriteSessions:   0.2,
           TrackSessionHandles: true,
       },
   }
   ```

### Issue: Database Connection Pool Exhaustion

**Symptoms:**
- "No sessions available" errors
- Intermittent database failures
- Connection timeout errors

**Diagnosis:**
```bash
# Check session pool metrics
gcloud logging read '
  jsonPayload.error_type="session_pool_exhausted" OR
  jsonPayload.error_type="connection_timeout"
' --limit=20

# Monitor active sessions
gcloud spanner operations list --instance=ingestion-db --type=SESSION_CREATE
```

**Resolution:**
```go
// Optimize session configuration
config := spanner.ClientConfig{
    SessionPoolConfig: spanner.SessionPoolConfig{
        MinOpened:     25,    // Increased from default
        MaxOpened:     200,   // Increased from default
        MaxIdle:       100,   // Increased from default
        WriteSessions: 0.2,   // 20% write sessions
        TrackSessionHandles: true,
        HealthCheckWorkers: 10,
        HealthCheckInterval: time.Minute * 5,
    },
}

// Implement proper session cleanup
func processRequest(ctx context.Context, client *spanner.Client) error {
    // Use ReadOnlyTransaction for read operations
    tx := client.ReadOnlyTransaction()
    defer tx.Close()

    // Use Single() for simple reads
    row, err := client.Single().ReadRow(ctx, "requests", key, []string{"data"})
    return err
}
```

## Audio Processing Problems

### Issue: Speech-to-Text API Failures

**Symptoms:**
- Empty transcription fields
- "Speech API quota exceeded" errors
- Low transcription confidence scores

**Diagnosis:**
```bash
# Check Speech API usage
gcloud logging read '
  resource.type="cloud_run_revision" AND
  jsonPayload.api_call="speech_to_text" AND
  severity>=ERROR
' --limit=20

# Check quota usage
gcloud logging read '
  protoPayload.serviceName="speech.googleapis.com" AND
  protoPayload.authenticationInfo.principalEmail!=""
' --limit=10

# Check audio file access
gsutil ls gs://$PROJECT_ID-audio-files/tenant_*/calls/ | head -10
```

**Common Issues:**

1. **Quota Exceeded:**
   ```bash
   # Check current quota
   gcloud services quota list --service=speech.googleapis.com

   # Request quota increase
   gcloud services quota update \
     --consumer=projects/$PROJECT_ID \
     --service=speech.googleapis.com \
     --quota-id=SpeechAPIRequestsPerDay \
     --value=100000
   ```

2. **Audio Format Issues:**
   ```go
   // Ensure correct audio configuration
   config := &speechpb.RecognitionConfig{
       Encoding:                   speechpb.RecognitionConfig_MP3,
       SampleRateHertz:           8000,  // CallRail uses 8kHz
       LanguageCode:              "en-US",
       EnableAutomaticPunctuation: true,
       EnableSpeakerDiarization:  true,
       DiarizationSpeakerCount:   2,
       EnableWordTimeOffsets:     true,
       EnableWordConfidence:      true,
       AudioChannelCount:         1,     // Mono audio
   }
   ```

3. **Large File Handling:**
   ```go
   // Use long-running recognition for files > 1 minute
   if audioDuration > 60*time.Second {
       op, err := speechClient.LongRunningRecognize(ctx, &speechpb.LongRunningRecognizeRequest{
           Config: config,
           Audio:  &speechpb.RecognitionAudio{
               AudioSource: &speechpb.RecognitionAudio_Uri{Uri: audioURI},
           },
       })

       // Poll for completion
       resp, err := op.Wait(ctx)
   }
   ```

### Issue: Audio File Download Failures

**Symptoms:**
- "Recording not available" errors
- 404 errors from CallRail API
- Empty audio files in storage

**Diagnosis:**
```bash
# Check CallRail API responses
gcloud logging read '
  jsonPayload.api_call="callrail_recording_download" AND
  jsonPayload.http_status!=200
' --limit=20

# Check storage upload failures
gcloud logging read '
  jsonPayload.component="audio_storage" AND
  severity>=ERROR
' --limit=20

# Test CallRail API access
curl -X GET "https://api.callrail.com/v3/a/ACCOUNT_ID/calls/CALL_ID/recording.json" \
  -H "Authorization: Token token=\"CALLRAIL_API_KEY\""
```

**Resolution:**
1. **Implement retry logic:**
   ```go
   func downloadRecording(callID, apiKey string) ([]byte, error) {
       maxRetries := 3
       backoff := 2 * time.Second

       for attempt := 0; attempt < maxRetries; attempt++ {
           data, err := attemptDownload(callID, apiKey)
           if err == nil {
               return data, nil
           }

           if attempt < maxRetries-1 {
               time.Sleep(backoff)
               backoff *= 2
           }
       }
       return nil, fmt.Errorf("failed after %d attempts", maxRetries)
   }
   ```

2. **Handle recording delays:**
   ```go
   // CallRail recordings may not be immediately available
   // Implement delayed processing
   if strings.Contains(err.Error(), "recording not ready") {
       // Queue for retry in 5 minutes
       return scheduleRetry(callID, time.Now().Add(5*time.Minute))
   }
   ```

## CRM Integration Issues

### Issue: HubSpot API Authentication Failures

**Symptoms:**
- 401 Unauthorized from HubSpot API
- "Invalid access token" errors
- Leads not appearing in HubSpot

**Diagnosis:**
```bash
# Test HubSpot token
HUBSPOT_TOKEN=$(gcloud secrets versions access latest --secret=hubspot-token-tenant-TENANT_ID)

curl -X GET "https://api.hubapi.com/crm/v3/objects/contacts?limit=1" \
  -H "Authorization: Bearer $HUBSPOT_TOKEN"

# Check token scopes
curl -X GET "https://api.hubapi.com/oauth/v1/access-tokens/$HUBSPOT_TOKEN"

# Check recent CRM push attempts
gcloud logging read '
  jsonPayload.workflow_step="crm_push" AND
  jsonPayload.crm_type="hubspot" AND
  severity>=ERROR
' --limit=20
```

**Resolution:**
1. **Verify token scopes:**
   ```bash
   # Required scopes for HubSpot:
   # - crm.objects.contacts.read
   # - crm.objects.contacts.write
   # - crm.objects.deals.read
   # - crm.objects.deals.write
   ```

2. **Update token if expired:**
   ```bash
   # Generate new private app token in HubSpot
   echo -n "new-hubspot-token" | \
       gcloud secrets versions add hubspot-token-tenant-TENANT_ID --data-file=-
   ```

### Issue: Salesforce JWT Authentication Failures

**Symptoms:**
- "Invalid JWT" errors
- "User not found" errors
- OAuth token request failures

**Diagnosis:**
```bash
# Check JWT configuration
gcloud secrets versions access latest --secret=sf-private-key-tenant-TENANT_ID
gcloud secrets versions access latest --secret=sf-consumer-key-tenant-TENANT_ID
gcloud secrets versions access latest --secret=sf-username-tenant-TENANT_ID

# Test JWT generation
python3 << EOF
import jwt
import datetime
import json

private_key = open('sf_private_key.pem').read()
payload = {
    'iss': 'CONSUMER_KEY',
    'sub': 'USERNAME',
    'aud': 'https://login.salesforce.com',
    'exp': datetime.datetime.utcnow() + datetime.timedelta(minutes=5)
}
token = jwt.encode(payload, private_key, algorithm='RS256')
print(token)
EOF
```

**Resolution:**
1. **Verify Connected App configuration:**
   - Digital certificates uploaded correctly
   - JWT Bearer Flow enabled
   - User has API access

2. **Check user permissions:**
   ```bash
   # User must have "API Enabled" permission
   # Connected App must be approved for user's profile
   ```

### Issue: Field Mapping Errors

**Symptoms:**
- "Invalid field" errors in CRM logs
- Partial data in CRM records
- Data type conversion errors

**Diagnosis:**
```sql
-- Check field mapping configuration
SELECT
    tenant_id,
    JSON_EXTRACT(configuration, '$.crm_config.field_mappings') as field_mappings,
    JSON_EXTRACT(configuration, '$.crm_config.custom_properties') as custom_properties
FROM tenants
WHERE tenant_id = 'tenant_example';
```

**Resolution:**
1. **Validate field names:**
   ```bash
   # HubSpot - check available properties
   curl -X GET "https://api.hubapi.com/crm/v3/properties/contacts" \
     -H "Authorization: Bearer $HUBSPOT_TOKEN" | \
     jq '.results[] | select(.name | contains("custom")) | {name, type}'

   # Salesforce - check field metadata
   curl -X GET "https://instance.salesforce.com/services/data/v54.0/sobjects/Lead/describe/" \
     -H "Authorization: Bearer $SF_TOKEN" | \
     jq '.fields[] | select(.custom==true) | {name, type}'
   ```

2. **Fix data type mismatches:**
   ```go
   // Ensure proper type conversion
   func mapFieldValue(value interface{}, targetType string) interface{} {
       switch targetType {
       case "number":
           if str, ok := value.(string); ok {
               if num, err := strconv.ParseFloat(str, 64); err == nil {
                   return num
               }
           }
       case "boolean":
           if str, ok := value.(string); ok {
               return strings.ToLower(str) == "true"
           }
       }
       return value
   }
   ```

## Performance Problems

### Issue: Memory Leaks

**Symptoms:**
- Increasing memory usage over time
- Out of memory errors
- Container restarts

**Diagnosis:**
```bash
# Enable profiling
go tool pprof http://localhost:8080/debug/pprof/heap

# Check Cloud Run memory metrics
gcloud monitoring timeseries list \
  --filter='metric.type="run.googleapis.com/container/memory/utilizations"' \
  --interval.end-time=$(date -u +%Y-%m-%dT%H:%M:%SZ)

# Look for memory-related errors
gcloud logging read 'jsonPayload.error_type="out_of_memory"' --limit=20
```

**Resolution:**
1. **Review resource cleanup:**
   ```go
   // Ensure proper cleanup of resources
   func processCall(ctx context.Context) error {
       defer func() {
           // Clean up resources
           if recorder != nil {
               recorder.Close()
           }
           if httpResp != nil {
               httpResp.Body.Close()
           }
       }()

       // Process logic...
   }
   ```

2. **Optimize memory usage:**
   ```go
   // Stream large files instead of loading into memory
   func processLargeAudio(url string) error {
       resp, err := http.Get(url)
       if err != nil {
           return err
       }
       defer resp.Body.Close()

       // Stream directly to storage
       writer := storageClient.Bucket("bucket").Object("file").NewWriter(ctx)
       defer writer.Close()

       _, err = io.Copy(writer, resp.Body)
       return err
   }
   ```

## Authentication Errors

### Issue: JWT Token Validation Failures

**Symptoms:**
- "Invalid token" errors
- "Token expired" errors
- API authentication failures

**Diagnosis:**
```bash
# Check JWT signing key
gcloud secrets versions access latest --secret=jwt-signing-key

# Validate token format
echo "JWT_TOKEN" | cut -d'.' -f2 | base64 -d | jq .

# Check token expiration
gcloud logging read 'jsonPayload.error_type="jwt_expired"' --limit=10
```

**Resolution:**
```go
// Implement proper JWT validation
func validateJWT(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtKey, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, fmt.Errorf("invalid token")
}
```

## Diagnostic Tools

### Health Check Script

```bash
#!/bin/bash
# health_check.sh - Comprehensive system health check

PROJECT_ID="your-project-id"
SERVICE_URL="https://api.pipeline.com"

echo "=== Ingestion Pipeline Health Check ==="
echo "Timestamp: $(date)"
echo

# 1. Service Health
echo "1. Checking service health..."
HEALTH_RESPONSE=$(curl -s "$SERVICE_URL/v1/health" | jq -r '.status')
echo "Service Status: $HEALTH_RESPONSE"

# 2. Database Health
echo "2. Checking database health..."
DB_RESULT=$(gcloud spanner databases execute-sql pipeline-db \
  --instance=ingestion-db \
  --sql="SELECT COUNT(*) as count FROM tenants" \
  --format="value(count)")
echo "Database: Connected (Tenants: $DB_RESULT)"

# 3. External APIs
echo "3. Checking external APIs..."

# Speech API
SPEECH_STATUS=$(gcloud services list --enabled --filter="name:speech.googleapis.com" --format="value(name)")
if [ -n "$SPEECH_STATUS" ]; then
    echo "Speech API: Enabled"
else
    echo "Speech API: Not enabled"
fi

# 4. Recent Errors
echo "4. Recent errors (last hour)..."
ERROR_COUNT=$(gcloud logging read "severity>=ERROR timestamp>=\"$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)\"" --format="value(timestamp)" | wc -l)
echo "Error count: $ERROR_COUNT"

# 5. Processing Metrics
echo "5. Processing metrics (last hour)..."
WEBHOOK_COUNT=$(gcloud logging read "jsonPayload.component=\"webhook_processor\" timestamp>=\"$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)\"" --format="value(timestamp)" | wc -l)
echo "Webhooks processed: $WEBHOOK_COUNT"

echo "=== Health Check Complete ==="
```

### Performance Analysis Script

```bash
#!/bin/bash
# performance_analysis.sh - Analyze system performance

echo "=== Performance Analysis ==="

# 1. Cloud Run Metrics
echo "1. Cloud Run Performance..."
gcloud monitoring timeseries list \
  --filter='metric.type="run.googleapis.com/request_latencies"' \
  --interval.end-time=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  --interval.start-time=$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ) \
  --format="table(points.value.distribution_value.mean)"

# 2. Database Performance
echo "2. Database Performance..."
gcloud spanner databases execute-sql pipeline-db \
  --instance=ingestion-db \
  --sql="
    SELECT
      text,
      ROUND(avg_latency_seconds, 3) as avg_latency,
      execution_count
    FROM SPANNER_SYS.QUERY_STATS_TOP_HOUR
    ORDER BY avg_latency_seconds DESC
    LIMIT 5"

# 3. Error Analysis
echo "3. Error Analysis..."
gcloud logging read "severity>=ERROR timestamp>=\"$(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%SZ)\"" \
  --format="value(jsonPayload.error_type)" | sort | uniq -c | sort -nr

echo "=== Analysis Complete ==="
```

### Tenant Diagnostic Script

```bash
#!/bin/bash
# tenant_diagnostics.sh - Diagnose tenant-specific issues

TENANT_ID="$1"

if [ -z "$TENANT_ID" ]; then
    echo "Usage: $0 <tenant_id>"
    exit 1
fi

echo "=== Tenant Diagnostics: $TENANT_ID ==="

# 1. Tenant Configuration
echo "1. Tenant Configuration..."
gcloud spanner databases execute-sql pipeline-db \
  --instance=ingestion-db \
  --sql="SELECT status, configuration FROM tenants WHERE tenant_id = '$TENANT_ID'"

# 2. Recent Requests
echo "2. Recent Requests (last 24h)..."
gcloud spanner databases execute-sql pipeline-db \
  --instance=ingestion-db \
  --sql="
    SELECT
      request_id,
      source,
      status,
      created_at
    FROM requests
    WHERE tenant_id = '$TENANT_ID'
      AND created_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
    ORDER BY created_at DESC
    LIMIT 10"

# 3. Error Analysis
echo "3. Error Analysis..."
gcloud logging read "jsonPayload.tenant_id=\"$TENANT_ID\" AND severity>=ERROR" \
  --limit=10 \
  --format="table(timestamp, jsonPayload.error_type, jsonPayload.message)"

# 4. CRM Integration Status
echo "4. CRM Integration..."
gcloud logging read "jsonPayload.tenant_id=\"$TENANT_ID\" AND jsonPayload.workflow_step=\"crm_push\"" \
  --limit=5 \
  --format="table(timestamp, jsonPayload.status, jsonPayload.crm_type)"

echo "=== Diagnostics Complete ==="
```

These troubleshooting tools and procedures should help you quickly identify and resolve issues in the multi-tenant ingestion pipeline. Remember to always check logs first, validate configurations, and test external dependencies when diagnosing problems.