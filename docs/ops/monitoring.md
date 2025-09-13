# Monitoring and Alerting Guide

## Table of Contents
- [Overview](#overview)
- [Key Metrics](#key-metrics)
- [Dashboards](#dashboards)
- [Alert Policies](#alert-policies)
- [Log Analysis](#log-analysis)
- [Performance Monitoring](#performance-monitoring)
- [Cost Monitoring](#cost-monitoring)
- [Incident Response](#incident-response)

## Overview

This guide covers comprehensive monitoring and alerting for the multi-tenant ingestion pipeline, ensuring high availability, performance, and cost optimization across all tenants and services.

### Monitoring Stack
- **Google Cloud Monitoring**: Metrics, alerts, and dashboards
- **Google Cloud Logging**: Centralized log aggregation and analysis
- **Google Cloud Trace**: Distributed tracing for request flows
- **Google Cloud Profiler**: Application performance profiling
- **Custom Metrics**: Business-specific KPIs and tenant metrics

## Key Metrics

### System Health Metrics

#### Service Availability
```yaml
Metric: cloud_run_revision/request_count
Description: Total requests per minute
Target: > 0 (service is receiving traffic)
Alert Threshold: 0 requests for 5+ minutes

Metric: cloud_run_revision/request_latency
Description: 95th percentile latency
Target: < 2000ms
Alert Threshold: > 5000ms for 2+ minutes

Metric: cloud_run_revision/billable_instance_time
Description: Container instance utilization
Target: 70-85% utilization
Alert Threshold: > 95% for 5+ minutes
```

#### Database Performance
```yaml
Metric: spanner/instance/cpu/smoothed_utilization
Description: Spanner CPU utilization
Target: < 65%
Alert Threshold: > 85% for 3+ minutes

Metric: spanner/instance/storage/used_bytes
Description: Database storage usage
Target: Monitor growth rate
Alert Threshold: > 80% of quota

Metric: spanner/query_stats/total_scan_rows
Description: Query efficiency
Target: < 10,000 rows per query
Alert Threshold: > 100,000 rows consistently
```

#### API Dependencies
```yaml
Metric: speech_api/request_count
Description: Speech-to-Text API calls
Target: Match call volume
Alert Threshold: Error rate > 5%

Metric: vertex_ai/prediction_count
Description: Vertex AI API calls
Target: Match processing volume
Alert Threshold: Error rate > 3%

Metric: storage/bucket/object_count
Description: Audio file storage
Target: Growing with call volume
Alert Threshold: Storage errors > 1%
```

### Business Metrics

#### Call Processing
```yaml
Metric: custom/call_processing_time
Description: End-to-end call processing time
Target: < 3 minutes (95th percentile)
Alert Threshold: > 10 minutes

Metric: custom/transcription_accuracy
Description: Transcription confidence scores
Target: > 90% confidence
Alert Threshold: < 80% confidence for 1+ hour

Metric: custom/lead_score_distribution
Description: AI lead quality scores
Target: Average > 50
Alert Threshold: Average < 30 for 4+ hours
```

#### CRM Integration
```yaml
Metric: custom/crm_push_success_rate
Description: Successful CRM pushes
Target: > 98%
Alert Threshold: < 95% for 10+ minutes

Metric: custom/crm_push_latency
Description: Time to push to CRM
Target: < 30 seconds
Alert Threshold: > 2 minutes

Metric: custom/duplicate_detection_rate
Description: Duplicate contact prevention
Target: < 5% duplicates
Alert Threshold: > 15% duplicates
```

### Tenant-Specific Metrics
```yaml
Metric: custom/tenant_call_volume
Description: Calls per tenant per hour
Target: Baseline ± 50%
Alert Threshold: 80% deviation from baseline

Metric: custom/tenant_webhook_failures
Description: Webhook failures per tenant
Target: 0
Alert Threshold: > 3 failures in 1 hour

Metric: custom/tenant_cost_per_call
Description: Processing cost per call per tenant
Target: < $0.50
Alert Threshold: > $1.00
```

## Dashboards

### Main Operations Dashboard

Create comprehensive dashboard in Google Cloud Monitoring:

```yaml
# Dashboard Configuration
displayName: "Ingestion Pipeline Operations"
dashboardFilters:
  - filterType: RESOURCE_LABEL
    labelKey: service_name
    stringValue: ingestion-pipeline

widgets:
  # System Health Row
  - title: "Request Rate (RPM)"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'resource.type="cloud_run_revision"'
            metric: 'run.googleapis.com/request_count'
          plotType: LINE

  - title: "Request Latency (P95)"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'resource.type="cloud_run_revision"'
            metric: 'run.googleapis.com/request_latencies'
          plotType: LINE

  - title: "Error Rate"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'resource.type="cloud_run_revision"'
            metric: 'run.googleapis.com/request_count'
            groupBy: response_code_class
          plotType: STACKED_AREA

  # Business Metrics Row
  - title: "Call Processing Pipeline"
    scorecard:
      timeSeriesQuery:
        filter: 'metric.type="custom.googleapis.com/call_processing_time"'
      sparkChartView:
        sparkChartType: SPARK_LINE

  - title: "CRM Integration Health"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'metric.type="custom.googleapis.com/crm_push_success_rate"'
          plotType: LINE

  # Resource Utilization Row
  - title: "Spanner CPU Utilization"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'resource.type="spanner_instance"'
            metric: 'spanner.googleapis.com/instance/cpu/smoothed_utilization'
          plotType: LINE

  - title: "Cloud Run Instance Count"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'resource.type="cloud_run_revision"'
            metric: 'run.googleapis.com/container/instance_count'
          plotType: STACKED_AREA
```

### Tenant-Specific Dashboard

```yaml
displayName: "Tenant Performance Dashboard"
dashboardFilters:
  - filterType: METRIC_LABEL
    labelKey: tenant_id
    templateVariable: $tenant_id

widgets:
  - title: "Calls Processed (Last 24h)"
    scorecard:
      timeSeriesQuery:
        filter: 'metric.type="custom.googleapis.com/calls_processed" metric.label.tenant_id="$tenant_id"'

  - title: "Average Lead Score"
    scorecard:
      timeSeriesQuery:
        filter: 'metric.type="custom.googleapis.com/average_lead_score" metric.label.tenant_id="$tenant_id"'

  - title: "Processing Time Distribution"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'metric.type="custom.googleapis.com/processing_time" metric.label.tenant_id="$tenant_id"'
          plotType: STACKED_BAR

  - title: "CRM Push Status"
    pieChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'metric.type="custom.googleapis.com/crm_push_status" metric.label.tenant_id="$tenant_id"'
            groupBy: status
```

### Cost Monitoring Dashboard

```yaml
displayName: "Cost and Usage Analysis"
widgets:
  - title: "Daily Processing Costs"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'metric.type="billing.googleapis.com/billing/billed_cost"'
            groupBy: service
          plotType: STACKED_AREA

  - title: "Cost per Tenant"
    table:
      dataSets:
        - timeSeriesQuery:
            filter: 'metric.type="custom.googleapis.com/tenant_cost"'
            groupBy: tenant_id

  - title: "Speech API Usage"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'metric.type="serviceruntime.googleapis.com/api/request_count" resource.label.service="speech.googleapis.com"'
          plotType: LINE

  - title: "Storage Costs by Tenant"
    xyChart:
      dataSets:
        - timeSeriesQuery:
            filter: 'metric.type="storage.googleapis.com/storage/total_bytes"'
            groupBy: bucket_name
          plotType: STACKED_AREA
```

## Alert Policies

### Critical System Alerts

#### High Error Rate Alert
```yaml
displayName: "High Error Rate - Ingestion Pipeline"
conditions:
  - displayName: "Error rate > 5%"
    conditionThreshold:
      filter: 'resource.type="cloud_run_revision" resource.label.service_name="ingestion-pipeline"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 0.05
      duration: 300s
      aggregations:
        - alignmentPeriod: 60s
          perSeriesAligner: ALIGN_RATE
        - crossSeriesReducer: REDUCE_MEAN
          groupByFields: ["response_code_class"]

documentation:
  content: |
    The ingestion pipeline is experiencing a high error rate.

    ## Immediate Actions:
    1. Check Cloud Run logs for error details
    2. Verify database connectivity
    3. Check external API status (CallRail, Speech-to-Text)
    4. Review recent deployments

    ## Investigation Steps:
    ```bash
    # Check recent errors
    gcloud logging read "severity>=ERROR" --limit=50

    # Check service health
    curl https://api.pipeline.com/v1/health

    # Review metrics
    gcloud monitoring dashboards list
    ```

notificationChannels:
  - "projects/PROJECT_ID/notificationChannels/EMAIL_CHANNEL"
  - "projects/PROJECT_ID/notificationChannels/SLACK_CHANNEL"
  - "projects/PROJECT_ID/notificationChannels/PAGERDUTY_CHANNEL"

alertStrategy:
  autoClose: 1800s
```

#### Database Performance Alert
```yaml
displayName: "Spanner High CPU Utilization"
conditions:
  - displayName: "CPU utilization > 85%"
    conditionThreshold:
      filter: 'resource.type="spanner_instance"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 0.85
      duration: 180s
      aggregations:
        - alignmentPeriod: 60s
          perSeriesAligner: ALIGN_MEAN

documentation:
  content: |
    Spanner instance is experiencing high CPU utilization.

    ## Immediate Actions:
    1. Check for slow queries in Spanner query stats
    2. Review recent schema changes
    3. Consider scaling up processing units

    ## Investigation:
    ```bash
    # Check query performance
    gcloud spanner databases execute-sql pipeline-db \
      --instance=ingestion-db \
      --sql="SELECT * FROM SPANNER_SYS.QUERY_STATS_TOP_10_MINUTE ORDER BY avg_cpu_seconds DESC LIMIT 10"

    # Scale processing units if needed
    gcloud spanner instances update ingestion-db --processing-units=2000
    ```

notificationChannels:
  - "projects/PROJECT_ID/notificationChannels/EMAIL_CHANNEL"
  - "projects/PROJECT_ID/notificationChannels/SLACK_CHANNEL"
```

### Business Logic Alerts

#### CRM Integration Failure
```yaml
displayName: "CRM Push Failure Rate"
conditions:
  - displayName: "CRM push success rate < 95%"
    conditionThreshold:
      filter: 'metric.type="custom.googleapis.com/crm_push_success_rate"'
      comparison: COMPARISON_LESS_THAN
      thresholdValue: 0.95
      duration: 600s

documentation:
  content: |
    CRM integration is failing for multiple tenants.

    ## Troubleshooting Steps:
    1. Check CRM API status
    2. Verify authentication tokens
    3. Review field mapping errors
    4. Check rate limiting

    ```bash
    # Check CRM push logs
    gcloud logging read "jsonPayload.workflow_step=crm_push AND severity>=ERROR" --limit=20

    # Test specific CRM connection
    curl -X GET "https://api.hubapi.com/crm/v3/objects/contacts?limit=1" \
      -H "Authorization: Bearer TENANT_TOKEN"
    ```

notificationChannels:
  - "projects/PROJECT_ID/notificationChannels/EMAIL_CHANNEL"
```

#### Call Volume Anomaly
```yaml
displayName: "Unusual Call Volume"
conditions:
  - displayName: "Call volume deviation > 80%"
    conditionThreshold:
      filter: 'metric.type="custom.googleapis.com/tenant_call_volume"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 0.8
      duration: 1800s

documentation:
  content: |
    A tenant is experiencing unusual call volume patterns.

    This could indicate:
    - Marketing campaign launch
    - System integration issues
    - Potential spam/abuse

    ## Actions:
    1. Contact tenant to verify expected volume
    2. Check for spam patterns
    3. Review webhook authentication
    4. Monitor costs and quotas
```

### Tenant-Specific Alerts

#### Webhook Authentication Failures
```yaml
displayName: "Webhook Authentication Failures"
conditions:
  - displayName: "Auth failures > 3 in 1 hour"
    conditionThreshold:
      filter: 'jsonPayload.error_type="webhook_auth_failure"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 3
      duration: 3600s
      aggregations:
        - crossSeriesReducer: REDUCE_COUNT
          groupByFields: ["jsonPayload.tenant_id"]

documentation:
  content: |
    Repeated webhook authentication failures for a tenant.

    ## Likely Causes:
    - Incorrect webhook secret
    - Clock skew issues
    - Webhook URL misconfiguration

    ## Resolution:
    1. Verify webhook secret in CallRail
    2. Check tenant configuration
    3. Contact tenant for webhook reconfiguration
```

## Log Analysis

### Structured Logging Configuration

Ensure all logs include these standard fields:

```go
// Standard log structure
type LogEntry struct {
    Timestamp    time.Time `json:"timestamp"`
    Severity     string    `json:"severity"`
    Message      string    `json:"message"`
    TenantID     string    `json:"tenant_id,omitempty"`
    RequestID    string    `json:"request_id,omitempty"`
    UserAgent    string    `json:"user_agent,omitempty"`
    Source       string    `json:"source,omitempty"`
    Component    string    `json:"component"`
    TraceID      string    `json:"trace_id,omitempty"`
    Duration     int64     `json:"duration_ms,omitempty"`
    ErrorType    string    `json:"error_type,omitempty"`
    ErrorDetails string    `json:"error_details,omitempty"`
}
```

### Common Log Queries

#### Find Webhook Processing Errors:
```bash
gcloud logging read '
  resource.type="cloud_run_revision" AND
  jsonPayload.component="webhook_processor" AND
  severity>=ERROR
' --limit=50 --format="value(jsonPayload.message, jsonPayload.tenant_id, jsonPayload.error_type)"
```

#### Track Request Processing Flow:
```bash
gcloud logging read '
  jsonPayload.request_id="req_123456"
' --format="table(timestamp, jsonPayload.component, jsonPayload.message)"
```

#### Monitor CRM Integration Health:
```bash
gcloud logging read '
  jsonPayload.workflow_step="crm_push" AND
  timestamp>="2025-09-13T00:00:00Z"
' --format="value(jsonPayload.tenant_id, jsonPayload.status, jsonPayload.error_details)"
```

#### Analyze Call Processing Performance:
```bash
gcloud logging read '
  jsonPayload.component="call_processor" AND
  jsonPayload.duration_ms>5000
' --limit=20 --format="table(timestamp, jsonPayload.tenant_id, jsonPayload.duration_ms)"
```

### Log-Based Metrics

Create metrics from log data:

```yaml
# Call processing time metric
name: "call_processing_time"
description: "Time to process call from webhook to CRM"
filter: 'jsonPayload.component="call_processor" AND jsonPayload.status="completed"'
valueExtractor: "EXTRACT(jsonPayload.duration_ms)"
metricDescriptor:
  metricKind: GAUGE
  valueType: INT64
  unit: "ms"

# Error rate by tenant
name: "errors_by_tenant"
description: "Error count grouped by tenant"
filter: 'severity>=ERROR AND jsonPayload.tenant_id!=""'
metricDescriptor:
  metricKind: CUMULATIVE
  valueType: INT64
labelExtractors:
  tenant_id: "EXTRACT(jsonPayload.tenant_id)"
  error_type: "EXTRACT(jsonPayload.error_type)"
```

## Performance Monitoring

### Application Performance Monitoring

#### Enable Cloud Profiler:
```go
import (
    "cloud.google.com/go/profiler"
)

func main() {
    if err := profiler.Start(profiler.Config{
        Service:        "ingestion-pipeline",
        ServiceVersion: "1.0.0",
        ProjectID:      os.Getenv("PROJECT_ID"),
    }); err != nil {
        log.Printf("Failed to start profiler: %v", err)
    }

    // Application code...
}
```

#### Custom Performance Metrics:
```go
import (
    "contrib.go.opencensus.io/exporter/stackdriver"
    "go.opencensus.io/stats"
    "go.opencensus.io/stats/view"
)

var (
    CallProcessingTime = stats.Float64("call_processing_time", "Time to process call", "ms")
    CRMPushLatency     = stats.Float64("crm_push_latency", "CRM push latency", "ms")
    LeadScore          = stats.Float64("lead_score", "AI lead score", "score")
)

func init() {
    view.Register(&view.View{
        Name:        "call_processing_time",
        Measure:     CallProcessingTime,
        Description: "Distribution of call processing times",
        Aggregation: view.Distribution(0, 1000, 2000, 5000, 10000, 30000),
        TagKeys:     []tag.Key{TenantIDKey},
    })
}
```

### Database Performance Monitoring

#### Query Performance Analysis:
```sql
-- Top 10 slowest queries
SELECT
    text,
    avg_latency_seconds,
    execution_count,
    avg_cpu_seconds
FROM SPANNER_SYS.QUERY_STATS_TOP_HOUR
WHERE interval_end >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
ORDER BY avg_latency_seconds DESC
LIMIT 10;

-- Lock contention analysis
SELECT
    lock_wait_time_seconds,
    query_text,
    execution_count
FROM SPANNER_SYS.LOCK_STATS_TOP_HOUR
WHERE interval_end >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
ORDER BY lock_wait_time_seconds DESC
LIMIT 10;
```

#### Index Usage Analysis:
```sql
-- Unused indexes
SELECT
    table_name,
    index_name,
    has_read_operations,
    has_write_operations
FROM INFORMATION_SCHEMA.INDEX_USAGE
WHERE database_name = 'pipeline-db'
  AND has_read_operations = FALSE
  AND has_write_operations = TRUE;
```

## Cost Monitoring

### Cost Allocation by Tenant

#### Export Billing Data:
```bash
# Create BigQuery export of billing data
bq mk --dataset --location=US ${PROJECT_ID}:billing_export

# Create cost allocation view
bq query --use_legacy_sql=false '
CREATE OR REPLACE VIEW billing_export.tenant_costs AS
SELECT
    service.description as service_name,
    sku.description as sku_description,
    usage_start_time,
    usage_end_time,
    cost,
    currency,
    labels.value as tenant_id
FROM `PROJECT_ID.billing_export.gcp_billing_export_v1_BILLING_ACCOUNT_ID`
LEFT JOIN UNNEST(labels) as labels ON labels.key = "tenant_id"
WHERE labels.value IS NOT NULL
'
```

#### Cost Analysis Queries:
```sql
-- Daily cost by tenant
SELECT
    DATE(usage_start_time) as date,
    tenant_id,
    service_name,
    SUM(cost) as daily_cost
FROM billing_export.tenant_costs
WHERE DATE(usage_start_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
GROUP BY 1, 2, 3
ORDER BY date DESC, daily_cost DESC;

-- Average cost per call by tenant
SELECT
    tc.tenant_id,
    SUM(tc.cost) / COUNT(r.request_id) as cost_per_call,
    COUNT(r.request_id) as total_calls,
    SUM(tc.cost) as total_cost
FROM billing_export.tenant_costs tc
JOIN `PROJECT_ID.spanner_export.requests` r
  ON tc.tenant_id = r.tenant_id
WHERE DATE(tc.usage_start_time) = CURRENT_DATE()
  AND r.request_type = 'phone_call'
GROUP BY tc.tenant_id
ORDER BY cost_per_call DESC;
```

### Cost Optimization Alerts

```yaml
displayName: "High Cost Per Tenant"
conditions:
  - displayName: "Daily cost > $100 for single tenant"
    conditionThreshold:
      filter: 'metric.type="custom.googleapis.com/tenant_daily_cost"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 100
      duration: 300s

documentation:
  content: |
    A tenant has exceeded the expected daily cost threshold.

    ## Investigation Steps:
    1. Check call volume for the tenant
    2. Review Speech API usage
    3. Verify audio file storage patterns
    4. Check for processing loops or errors

    ## Cost Breakdown Query:
    ```sql
    SELECT service_name, SUM(cost) as cost
    FROM billing_export.tenant_costs
    WHERE tenant_id = 'TENANT_ID'
      AND DATE(usage_start_time) = CURRENT_DATE()
    GROUP BY service_name
    ORDER BY cost DESC;
    ```
```

## Incident Response

### Incident Classification

| Priority | Response Time | Description | Examples |
|----------|---------------|-------------|----------|
| P0 | 15 minutes | Complete service outage | API down, database unavailable |
| P1 | 30 minutes | Major functionality impacted | CRM integration down, high error rate |
| P2 | 2 hours | Partial functionality impacted | Single tenant issues, performance degradation |
| P3 | 8 hours | Minor issues | Non-critical errors, monitoring alerts |

### Incident Response Playbooks

#### P0: Complete Service Outage

**Response Steps:**
1. **Acknowledge** (5 minutes)
   - Acknowledge alert in monitoring system
   - Create incident in incident management system
   - Notify on-call team

2. **Assess** (10 minutes)
   ```bash
   # Check service health
   curl https://api.pipeline.com/v1/health

   # Check Cloud Run status
   gcloud run services describe ingestion-pipeline --region=us-central1

   # Check database connectivity
   gcloud spanner databases execute-sql pipeline-db \
     --instance=ingestion-db \
     --sql="SELECT 1"
   ```

3. **Mitigate** (15 minutes)
   - Roll back recent deployments if necessary
   - Scale up resources if needed
   - Implement emergency routing if available

4. **Communicate** (Ongoing)
   - Update status page
   - Notify affected tenants
   - Provide regular updates

#### P1: CRM Integration Failure

**Response Steps:**
1. **Identify Scope**
   ```bash
   # Check which tenants affected
   gcloud logging read 'jsonPayload.workflow_step="crm_push" AND severity>=ERROR' \
     --limit=50 --format="value(jsonPayload.tenant_id)" | sort | uniq -c

   # Check error patterns
   gcloud logging read 'jsonPayload.workflow_step="crm_push" AND severity>=ERROR' \
     --limit=20 --format="value(jsonPayload.error_type, jsonPayload.error_details)"
   ```

2. **Immediate Actions**
   - Disable CRM push for affected tenants if needed
   - Queue failed requests for retry
   - Contact CRM vendor if API issues suspected

3. **Resolution**
   - Fix authentication issues
   - Update field mappings if needed
   - Retry queued requests

### Incident Communication Templates

#### Initial Notification
```
Subject: [P0] Ingestion Pipeline Service Outage - Investigating

We are currently investigating reports of service outage for the ingestion pipeline.
New calls and form submissions may not be processed during this time.

Status: Investigating
Started: 2025-09-13 14:30 UTC
Impact: All tenants affected
ETA: Under investigation

We will provide updates every 15 minutes until resolved.
```

#### Resolution Notification
```
Subject: [RESOLVED] Ingestion Pipeline Service Outage

The service outage has been resolved. All systems are operating normally.

Root Cause: Database connection pool exhaustion due to increased traffic
Resolution: Increased connection pool size and added circuit breaker
Duration: 45 minutes (14:30-15:15 UTC)

Affected Calls: Approximately 150 calls queued and processed successfully
Data Loss: None

Post-Incident Actions:
- Review connection pool sizing
- Implement better traffic surge handling
- Improve monitoring for connection pool metrics
```

### Monitoring Health Checks

Set up comprehensive health checks to validate system components:

```go
type HealthChecker struct {
    spanner   *spanner.Client
    storage   *storage.Client
    speechAPI speech.Client
}

func (h *HealthChecker) CheckHealth() HealthStatus {
    status := HealthStatus{Timestamp: time.Now()}

    // Database check
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := h.spanner.Single().Query(ctx, spanner.Statement{
        SQL: "SELECT 1",
    }).Next()
    status.Database = healthStatus(err)

    // Storage check
    bucket := h.storage.Bucket("audio-files-bucket")
    _, err = bucket.Attrs(ctx)
    status.Storage = healthStatus(err)

    // Speech API check
    _, err = h.speechAPI.Recognize(ctx, &speechpb.RecognizeRequest{
        Config: &speechpb.RecognitionConfig{
            Encoding:        speechpb.RecognitionConfig_LINEAR16,
            SampleRateHertz: 16000,
            LanguageCode:    "en-US",
        },
        Audio: &speechpb.RecognitionAudio{
            AudioSource: &speechpb.RecognitionAudio_Content{
                Content: []byte{}, // Empty test
            },
        },
    })
    status.SpeechAPI = healthStatus(err)

    return status
}
```

Your monitoring and alerting system is now configured to provide comprehensive visibility into the multi-tenant ingestion pipeline, enabling proactive issue detection and rapid incident response.