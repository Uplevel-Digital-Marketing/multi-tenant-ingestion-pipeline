# Operations Runbook

**Multi-Tenant Ingestion Pipeline**

---

**Version**: 2.0
**Date**: September 13, 2025
**Environment**: Production
**On-Call Team**: Platform Operations

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Emergency Procedures](#emergency-procedures)
3. [Common Operations](#common-operations)
4. [Incident Response](#incident-response)
5. [Monitoring and Alerting](#monitoring-and-alerting)
6. [Maintenance Procedures](#maintenance-procedures)
7. [Disaster Recovery](#disaster-recovery)
8. [Performance Tuning](#performance-tuning)
9. [Security Operations](#security-operations)
10. [Troubleshooting Guide](#troubleshooting-guide)

---

## System Overview

### Architecture Components

| Component | Technology | Purpose | Health Check |
|-----------|------------|---------|--------------|
| **Load Balancer** | Cloud Load Balancer | Traffic distribution | `curl https://api.company.com/v1/health` |
| **API Gateway** | Cloud Run | Webhook processing | Check Cloud Run console |
| **Database** | Cloud Spanner | Multi-tenant data | `gcloud spanner databases list` |
| **AI Processing** | Vertex AI | Content analysis | Check AI Platform console |
| **Audio Storage** | Cloud Storage | Recording storage | `gsutil ls gs://bucket-name` |
| **Message Queue** | Pub/Sub | Async processing | Check Pub/Sub console |
| **Monitoring** | Cloud Operations | System observability | Check monitoring dashboards |

### Key Metrics and SLOs

| Metric | Target (SLO) | Alert Threshold | Critical Threshold |
|--------|--------------|-----------------|-------------------|
| **Availability** | 99.9% | < 99.5% | < 99% |
| **Webhook Latency** | P95 < 2s | P95 > 3s | P95 > 5s |
| **Processing Time** | P95 < 3min | P95 > 5min | P95 > 10min |
| **Error Rate** | < 1% | > 2% | > 5% |
| **CRM Sync Success** | > 98% | < 95% | < 90% |

### Production Environment Details

```bash
PROJECT_ID="ingestion-pipeline-prod"
REGION="us-central1"
ENVIRONMENT="production"
DOMAIN="api.company.com"

# Key service URLs
WEBHOOK_URL="https://api.company.com/v1/callrail/webhook"
HEALTH_URL="https://api.company.com/v1/health"
METRICS_URL="https://api.company.com/v1/metrics"

# Database details
SPANNER_INSTANCE="pipeline-prod"
SPANNER_DATABASE="pipeline-db"
```

---

## Emergency Procedures

### 🚨 CRITICAL OUTAGE - Complete Service Down

**Immediate Actions (0-5 minutes)**

1. **Acknowledge the alert** in PagerDuty
2. **Create incident channel** in Slack: `#incident-YYYY-MM-DD-HH`
3. **Check system status**:
   ```bash
   # Quick health check
   curl -f https://api.company.com/v1/health || echo "API DOWN"

   # Check Cloud Run services
   gcloud run services list --region=us-central1

   # Check load balancer
   gcloud compute forwarding-rules list --global
   ```

4. **Notify stakeholders**:
   - Post in `#general`: "🚨 System outage detected, investigating"
   - Update status page: https://status.company.com

**Incident Response Actions (5-15 minutes)**

5. **Identify the root cause**:
   ```bash
   # Check recent deployments
   gcloud run revisions list --service=webhook-processor --region=us-central1

   # Check error logs
   gcloud logging read 'resource.type="cloud_run_revision" severity>=ERROR' \
     --limit=50 --format="table(timestamp,resource.labels.service_name,textPayload)"

   # Check resource utilization
   gcloud monitoring metrics list --filter="metric.type:compute"
   ```

6. **Apply immediate fix**:
   ```bash
   # Option A: Rollback to previous revision
   PREVIOUS_REV=$(gcloud run revisions list --service=webhook-processor \
     --region=us-central1 --limit=2 --format="value(metadata.name)" | tail -n1)

   gcloud run services update-traffic webhook-processor \
     --region=us-central1 --to-revisions=$PREVIOUS_REV=100

   # Option B: Scale up if resource constrained
   gcloud run services update webhook-processor \
     --region=us-central1 --min-instances=5 --max-instances=200
   ```

**Recovery Actions (15-30 minutes)**

7. **Verify service restoration**:
   ```bash
   # Test webhook endpoint
   curl -X POST https://api.company.com/v1/callrail/webhook \
     -H "Content-Type: application/json" \
     -H "x-callrail-signature: sha256=test" \
     -d '{"test": true}'

   # Check all health endpoints
   curl https://api.company.com/v1/health | jq '.'
   ```

8. **Monitor for stability** (30 minutes)
9. **Update stakeholders** on resolution
10. **Schedule post-mortem** within 24 hours

### 🟡 DEGRADED PERFORMANCE - High Latency/Errors

**Quick Assessment (0-2 minutes)**

```bash
# Check current performance metrics
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/request_latencies"

# Check error rates
gcloud logging read 'resource.type="cloud_run_revision" severity>=ERROR' \
  --freshness=5m --format="table(timestamp,resource.labels.service_name)"

# Check auto-scaling status
gcloud run services describe webhook-processor --region=us-central1 \
  --format="value(status.traffic[0].latestRevision,spec.template.metadata.annotations)"
```

**Mitigation Actions**

1. **Scale up resources**:
   ```bash
   # Increase minimum instances
   gcloud run services update webhook-processor \
     --region=us-central1 --min-instances=10

   # Increase memory/CPU if needed
   gcloud run services update webhook-processor \
     --region=us-central1 --memory=8Gi --cpu=4
   ```

2. **Enable circuit breaker** (if available):
   ```bash
   # Update service with circuit breaker configuration
   kubectl apply -f configs/circuit-breaker-config.yaml
   ```

3. **Check downstream dependencies**:
   ```bash
   # Test Spanner connectivity
   gcloud spanner databases execute-sql pipeline-db \
     --instance=pipeline-prod --sql="SELECT 1"

   # Test external API connectivity
   curl -o /dev/null -s -w "%{http_code}\n" https://api.callrail.com/v3/calls.json
   ```

---

## Common Operations

### Daily Operations Checklist

**Morning Checklist (9:00 AM PST)**

- [ ] **Check overnight alerts** in PagerDuty and Slack
- [ ] **Review system health**:
  ```bash
  curl https://api.company.com/v1/health | jq '.services'
  ```
- [ ] **Check processing backlog**:
  ```bash
  gcloud pubsub topics list-subscriptions webhook-processing-topic
  ```
- [ ] **Review error rates** from past 24 hours
- [ ] **Check cost alerts** in billing console
- [ ] **Verify backup completion**:
  ```bash
  gcloud spanner backups list --instance=pipeline-prod
  ```

**Evening Checklist (6:00 PM PST)**

- [ ] **Review daily metrics** and create summary
- [ ] **Check for any capacity alerts**
- [ ] **Review and acknowledge any non-critical alerts**
- [ ] **Ensure on-call rotation** is updated
- [ ] **Check upcoming maintenance windows**

### Deployment Operations

#### Standard Deployment Process

```bash
#!/bin/bash
# deploy-production.sh

set -e

echo "Starting production deployment..."

# 1. Pre-deployment checks
echo "Running pre-deployment checks..."
curl -f https://api.company.com/v1/health || (echo "Health check failed" && exit 1)

# 2. Deploy with gradual rollout
echo "Deploying new revision..."
gcloud run deploy webhook-processor \
  --image=gcr.io/$PROJECT_ID/pipeline:$NEW_VERSION \
  --region=us-central1 \
  --no-traffic

# 3. Get new revision name
NEW_REVISION=$(gcloud run revisions list --service=webhook-processor \
  --region=us-central1 --limit=1 --format="value(metadata.name)")

echo "New revision: $NEW_REVISION"

# 4. Gradual traffic split
echo "Starting gradual rollout..."
gcloud run services update-traffic webhook-processor \
  --region=us-central1 --to-revisions=$NEW_REVISION=10

sleep 300  # Wait 5 minutes

# 5. Check metrics for new revision
echo "Checking health of new revision..."
# Add health checks here

# 6. Continue rollout if healthy
gcloud run services update-traffic webhook-processor \
  --region=us-central1 --to-revisions=$NEW_REVISION=50

sleep 300  # Wait 5 minutes

# 7. Complete rollout
gcloud run services update-traffic webhook-processor \
  --region=us-central1 --to-revisions=$NEW_REVISION=100

echo "Deployment complete!"
```

#### Emergency Rollback

```bash
#!/bin/bash
# emergency-rollback.sh

echo "🚨 EMERGENCY ROLLBACK INITIATED"

# Get previous revision
PREVIOUS_REV=$(gcloud run revisions list --service=webhook-processor \
  --region=us-central1 --limit=2 --format="value(metadata.name)" | tail -n1)

echo "Rolling back to: $PREVIOUS_REV"

# Immediate rollback
gcloud run services update-traffic webhook-processor \
  --region=us-central1 --to-revisions=$PREVIOUS_REV=100

# Verify rollback
echo "Verifying rollback..."
curl -f https://api.company.com/v1/health && echo "✅ Rollback successful" || echo "❌ Rollback failed"
```

### Database Operations

#### Spanner Maintenance

```bash
# Check database performance
gcloud spanner operations list --instance=pipeline-prod --filter="done=false"

# Monitor query performance
gcloud logging read 'resource.type="spanner_instance" severity>=WARNING' --limit=20

# Check storage utilization
gcloud spanner databases describe pipeline-db --instance=pipeline-prod \
  --format="value(sizeBytes,state)"

# Create manual backup
gcloud spanner backups create manual-backup-$(date +%Y%m%d) \
  --instance=pipeline-prod \
  --database=pipeline-db \
  --retention-period=7d
```

#### Database Schema Updates

```bash
# Apply schema migration
gcloud spanner databases ddl update pipeline-db \
  --instance=pipeline-prod \
  --ddl-file=migrations/002_add_new_column.sql

# Monitor migration progress
gcloud spanner operations list --instance=pipeline-prod \
  --filter="metadata.@type:type.googleapis.com/google.spanner.admin.database.v1.UpdateDatabaseDdlMetadata"
```

### Scaling Operations

#### Manual Scaling

```bash
# Scale up for high traffic
gcloud run services update webhook-processor \
  --region=us-central1 \
  --min-instances=20 \
  --max-instances=500 \
  --concurrency=100

# Scale down after traffic spike
gcloud run services update webhook-processor \
  --region=us-central1 \
  --min-instances=2 \
  --max-instances=100 \
  --concurrency=80

# Check current scaling status
gcloud run services describe webhook-processor --region=us-central1 \
  --format="yaml(spec.template.metadata.annotations,status.traffic)"
```

#### Database Scaling

```bash
# Scale up Spanner processing units
gcloud spanner instances update pipeline-prod \
  --processing-units=2000

# Monitor scaling operation
gcloud spanner operations list --instance=pipeline-prod --filter="done=false"

# Check performance after scaling
gcloud monitoring metrics list --filter="resource.label.instance_id=pipeline-prod"
```

---

## Incident Response

### Incident Classification

#### Severity Levels

**🔴 P1 - Critical**
- Complete service outage
- Data corruption or loss
- Security breach
- **Response Time**: 15 minutes
- **Escalation**: Immediate to management

**🟡 P2 - High**
- Significant performance degradation (>5s latency)
- Partial feature unavailability
- Error rate >5%
- **Response Time**: 1 hour
- **Escalation**: Within 2 hours

**🟠 P3 - Medium**
- Minor performance issues
- Non-critical feature degradation
- Error rate 1-5%
- **Response Time**: 4 hours
- **Escalation**: Next business day

**🟢 P4 - Low**
- Cosmetic issues
- Documentation requests
- Enhancement requests
- **Response Time**: 24 hours
- **Escalation**: Not required

### Incident Response Workflow

#### Phase 1: Detection and Triage (0-15 minutes)

1. **Alert received** via PagerDuty/monitoring
2. **Acknowledge alert** within 5 minutes
3. **Initial assessment**:
   ```bash
   # Quick status check
   curl https://api.company.com/v1/health

   # Check recent logs
   gcloud logging read 'severity>=ERROR' --limit=10 --freshness=10m

   # Check service status
   gcloud run services list --region=us-central1
   ```

4. **Determine severity** using classification above
5. **Create incident** in tracking system
6. **Initial communication**:
   - Slack: `#incidents`
   - Status page update (if P1/P2)
   - Stakeholder notification (if P1)

#### Phase 2: Investigation and Mitigation (15-60 minutes)

1. **Gather detailed information**:
   ```bash
   # Service metrics
   gcloud monitoring metrics list --filter="resource.type=cloud_run_revision"

   # Error analysis
   gcloud logging read 'resource.type="cloud_run_revision" severity>=ERROR' \
     --limit=100 --format="table(timestamp,resource.labels.service_name,textPayload)"

   # Resource utilization
   gcloud compute instances list
   gcloud run services describe webhook-processor --region=us-central1
   ```

2. **Identify root cause**:
   - Recent deployments
   - Configuration changes
   - External dependency issues
   - Resource constraints
   - Code bugs

3. **Apply mitigation**:
   - Rollback if deployment-related
   - Scale resources if capacity issue
   - Enable circuit breakers
   - Failover to backup systems

4. **Monitor effectiveness** of mitigation

#### Phase 3: Resolution and Recovery (60+ minutes)

1. **Implement permanent fix**
2. **Verify full service restoration**
3. **Monitor for stability** (minimum 30 minutes)
4. **Update stakeholders** on resolution
5. **Close incident** in tracking system

#### Phase 4: Post-Incident (24-72 hours)

1. **Conduct post-mortem** meeting
2. **Document lessons learned**
3. **Create action items** for prevention
4. **Update runbooks** and procedures

### Communication Templates

#### Initial Incident Notification

```
🚨 INCIDENT P1 - Service Outage
Time: 2025-09-13 14:30 PST
Impact: Complete API unavailability
ETA: Investigating
Lead: @oncall-engineer
Channel: #incident-2025-09-13-14-30
Status Page: Updated
```

#### Incident Update

```
📊 INCIDENT UPDATE P1
Time: 2025-09-13 14:45 PST
Status: Mitigation in progress
Action: Rolling back to previous revision
ETA: 15 minutes
Next Update: 15:00 PST
```

#### Incident Resolution

```
✅ INCIDENT RESOLVED P1
Time: 2025-09-13 15:10 PST
Duration: 40 minutes
Root Cause: Deployment issue with new revision
Resolution: Rollback to stable revision
Post-Mortem: Scheduled for 2025-09-14 10:00 PST
```

---

## Monitoring and Alerting

### Critical Alerts Configuration

#### System Health Alerts

```yaml
# Cloud Monitoring Alert Policies
alert_policies:
  - name: "API Availability"
    condition: "uptime_check < 99%"
    duration: "2 minutes"
    severity: "CRITICAL"
    notification: ["pagerduty", "slack-critical"]

  - name: "High Error Rate"
    condition: "error_rate > 5%"
    duration: "5 minutes"
    severity: "HIGH"
    notification: ["pagerduty", "slack-alerts"]

  - name: "Database Connectivity"
    condition: "spanner_connection_errors > 10"
    duration: "1 minute"
    severity: "CRITICAL"
    notification: ["pagerduty", "slack-critical"]

  - name: "Processing Latency"
    condition: "p95_latency > 5000ms"
    duration: "10 minutes"
    severity: "MEDIUM"
    notification: ["slack-alerts"]
```

#### Business Metrics Alerts

```yaml
business_alerts:
  - name: "High Value Lead"
    condition: "lead_score > 90"
    duration: "immediate"
    severity: "INFO"
    notification: ["slack-sales"]

  - name: "CRM Sync Failure"
    condition: "crm_sync_success_rate < 90%"
    duration: "15 minutes"
    severity: "HIGH"
    notification: ["slack-alerts", "email-ops"]

  - name: "Processing Backlog"
    condition: "pubsub_queue_depth > 100"
    duration: "20 minutes"
    severity: "MEDIUM"
    notification: ["slack-alerts"]
```

### Dashboard URLs

**Primary Dashboards**:
- System Overview: https://console.cloud.google.com/monitoring/dashboards/custom/system-overview
- Application Performance: https://console.cloud.google.com/monitoring/dashboards/custom/app-performance
- Business Metrics: https://console.cloud.google.com/monitoring/dashboards/custom/business-metrics
- Cost Analysis: https://console.cloud.google.com/billing/reports

### Key Metrics to Monitor

#### System Metrics
```bash
# Request volume and latency
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/request_count"
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/request_latencies"

# Error rates
gcloud logging read 'resource.type="cloud_run_revision" severity>=ERROR' \
  --format="table(timestamp,resource.labels.service_name)" --limit=20

# Resource utilization
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/container/cpu/utilizations"
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/container/memory/utilizations"
```

#### Business Metrics
```bash
# Lead processing metrics
gcloud spanner databases execute-sql pipeline-db --instance=pipeline-prod \
  --sql="SELECT
    COUNT(*) as total_requests,
    AVG(processing_time_ms) as avg_processing_time,
    COUNTIF(status = 'completed') / COUNT(*) as success_rate
  FROM processing_requests
  WHERE created_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)"

# CRM integration success rates
gcloud spanner databases execute-sql pipeline-db --instance=pipeline-prod \
  --sql="SELECT
    crm_type,
    COUNT(*) as total_syncs,
    COUNTIF(status = 'success') / COUNT(*) as success_rate
  FROM crm_integrations
  WHERE created_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)
  GROUP BY crm_type"
```

---

## Maintenance Procedures

### Regular Maintenance Schedule

#### Daily Tasks (Automated)
- Health checks and metric collection
- Log rotation and cleanup
- Backup verification
- Security scan execution

#### Weekly Tasks
- [ ] **Review system performance** trends
- [ ] **Analyze cost optimization** opportunities
- [ ] **Check security alerts** and patches
- [ ] **Review and rotate** access keys (if needed)
- [ ] **Capacity planning** review based on growth trends

#### Monthly Tasks
- [ ] **Security audit** and vulnerability assessment
- [ ] **Disaster recovery** drill execution
- [ ] **Performance testing** and baseline updates
- [ ] **Documentation review** and updates
- [ ] **Cost analysis** and optimization review

### System Updates

#### Cloud Run Service Updates

```bash
# Update service with zero downtime
gcloud run deploy webhook-processor \
  --image=gcr.io/$PROJECT_ID/pipeline:new-version \
  --region=us-central1 \
  --no-traffic

# Gradual rollout
NEW_REV=$(gcloud run revisions list --service=webhook-processor \
  --region=us-central1 --limit=1 --format="value(metadata.name)")

# 10% traffic
gcloud run services update-traffic webhook-processor \
  --region=us-central1 --to-revisions=$NEW_REV=10

# Monitor and continue rollout...
```

#### Database Maintenance

```bash
# Schedule maintenance window
echo "Scheduling database maintenance for $(date)"

# Create pre-maintenance backup
gcloud spanner backups create pre-maintenance-$(date +%Y%m%d) \
  --instance=pipeline-prod \
  --database=pipeline-db \
  --retention-period=14d

# Apply any pending schema updates
for migration in migrations/pending/*.sql; do
  echo "Applying $migration"
  gcloud spanner databases ddl update pipeline-db \
    --instance=pipeline-prod \
    --ddl-file=$migration
done

# Verify health post-maintenance
curl https://api.company.com/v1/health
```

### Secret Rotation

```bash
#!/bin/bash
# rotate-secrets.sh

echo "Starting secret rotation..."

# Rotate API keys
NEW_CALLRAIL_KEY=$(generate_new_callrail_key)
echo "$NEW_CALLRAIL_KEY" | gcloud secrets versions add callrail-api-key --data-file=-

NEW_HUBSPOT_KEY=$(generate_new_hubspot_key)
echo "$NEW_HUBSPOT_KEY" | gcloud secrets versions add hubspot-api-key --data-file=-

# Update service to use new secrets (triggers restart)
gcloud run services update webhook-processor \
  --region=us-central1 \
  --update-env-vars="SECRET_VERSION=$(date +%s)"

# Verify new secrets work
sleep 60
curl https://api.company.com/v1/health | jq '.services'

echo "Secret rotation complete"
```

---

## Disaster Recovery

### Backup Verification

```bash
#!/bin/bash
# verify-backups.sh

echo "Verifying backup integrity..."

# Check Spanner backups
LATEST_BACKUP=$(gcloud spanner backups list --instance=pipeline-prod \
  --filter="state=READY" --sort-by="~createTime" --limit=1 \
  --format="value(name)")

if [ -n "$LATEST_BACKUP" ]; then
  echo "✅ Latest Spanner backup: $LATEST_BACKUP"
else
  echo "❌ No recent Spanner backups found"
  exit 1
fi

# Check Cloud Storage backups
gsutil ls -l gs://$PROJECT_ID-backups/ | head -10

# Test backup restore (to temporary database)
TEST_DB="pipeline-db-test-$(date +%s)"
gcloud spanner databases restore \
  --source-backup=$LATEST_BACKUP \
  --target-database=$TEST_DB \
  --target-instance=pipeline-prod

# Verify restored data
gcloud spanner databases execute-sql $TEST_DB --instance=pipeline-prod \
  --sql="SELECT COUNT(*) FROM tenants"

# Cleanup test database
gcloud spanner databases delete $TEST_DB --instance=pipeline-prod --quiet

echo "Backup verification complete"
```

### Disaster Recovery Procedures

#### Scenario 1: Database Corruption

```bash
# 1. Identify corruption extent
gcloud spanner databases execute-sql pipeline-db --instance=pipeline-prod \
  --sql="SELECT COUNT(*) FROM tenants WHERE status IS NULL"

# 2. Stop all processing
gcloud run services update webhook-processor \
  --region=us-central1 --min-instances=0 --max-instances=0

# 3. Restore from backup
BACKUP_NAME="backup-2025-09-13"
gcloud spanner databases restore \
  --source-backup=projects/$PROJECT_ID/instances/pipeline-prod/backups/$BACKUP_NAME \
  --target-database=pipeline-db-restored \
  --target-instance=pipeline-prod

# 4. Verify restored data
gcloud spanner databases execute-sql pipeline-db-restored --instance=pipeline-prod \
  --sql="SELECT COUNT(*) FROM tenants"

# 5. Switch to restored database (update connection strings)
# 6. Resume processing
```

#### Scenario 2: Region Outage

```bash
# 1. Activate secondary region
gcloud config set compute/region us-east1

# 2. Deploy services to secondary region
gcloud run deploy webhook-processor \
  --image=gcr.io/$PROJECT_ID/pipeline:latest \
  --region=us-east1 \
  --min-instances=5

# 3. Update DNS to point to secondary region
gcloud dns record-sets transaction start --zone=production-zone
gcloud dns record-sets transaction remove --zone=production-zone \
  --name=api.company.com. --type=A --ttl=300 "OLD_IP"
gcloud dns record-sets transaction add --zone=production-zone \
  --name=api.company.com. --type=A --ttl=300 "NEW_IP"
gcloud dns record-sets transaction execute --zone=production-zone

# 4. Verify failover
curl https://api.company.com/v1/health
```

### Recovery Time Objectives (RTO)

| Scenario | RTO Target | Steps |
|----------|------------|-------|
| **Service Instance Failure** | < 2 minutes | Auto-restart, health checks |
| **Database Connection Loss** | < 5 minutes | Connection pool reset, retry logic |
| **Single Service Failure** | < 15 minutes | Auto-scaling, load balancer rerouting |
| **Database Corruption** | < 30 minutes | Backup restoration |
| **Regional Outage** | < 60 minutes | Multi-region failover |
| **Complete Data Loss** | < 4 hours | Full system restoration |

---

## Performance Tuning

### Monitoring Performance

```bash
# Check current performance baselines
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/request_latencies" \
  --format="table(metric.type,resourceType)"

# Analyze slow queries
gcloud logging read 'resource.type="spanner_instance" severity>=WARNING' \
  --filter="textPayload:\"slow query\"" --limit=10

# Check memory/CPU utilization
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/container/cpu/utilizations"
```

### Optimization Strategies

#### Database Performance

```sql
-- Check query performance statistics
SELECT
  query_hash,
  query_text,
  avg_latency_seconds,
  execution_count,
  avg_rows_scanned
FROM SPANNER_SYS.QUERY_STATS_TOP_10M
ORDER BY avg_latency_seconds DESC
LIMIT 10;

-- Check for missing indexes
SELECT
  table_name,
  query_text,
  index_recommendation
FROM SPANNER_SYS.INDEX_RECOMMENDATIONS
WHERE recommendation_type = 'ADD_INDEX';
```

#### Application Performance

```bash
# Optimize Cloud Run configuration
gcloud run services update webhook-processor \
  --region=us-central1 \
  --cpu=2 \
  --memory=4Gi \
  --concurrency=100 \
  --min-instances=5 \
  --max-instances=1000

# Configure CPU allocation
gcloud run services update webhook-processor \
  --region=us-central1 \
  --cpu-throttling=false
```

#### Load Testing

```bash
# Install load testing tools
go install github.com/tsenart/vegeta@latest

# Create test payload
cat > webhook-test.json << EOF
{
  "call_id": "LOAD_TEST_$(date +%s)",
  "tenant_id": "load-test-tenant",
  "caller_id": "+15551234567",
  "duration": "120"
}
EOF

# Run load test
echo "POST https://api.company.com/v1/callrail/webhook" | \
  vegeta attack -duration=60s -rate=100 -header="Content-Type: application/json" \
  -body=webhook-test.json | vegeta report

# Analyze results
vegeta report -type=text
vegeta report -type=json > load-test-results.json
```

---

## Security Operations

### Security Monitoring

```bash
# Check for security events
gcloud logging read 'protoPayload.serviceName="iam.googleapis.com"' \
  --limit=50 --format="table(timestamp,protoPayload.authenticationInfo.principalEmail,protoPayload.methodName)"

# Monitor failed authentication attempts
gcloud logging read 'resource.type="cloud_run_revision" textPayload:"unauthorized"' \
  --limit=20

# Check for unusual access patterns
gcloud logging read 'resource.type="cloud_run_revision" httpRequest.status>=400' \
  --limit=50 --format="table(timestamp,httpRequest.remoteIp,httpRequest.status)"
```

### Security Incident Response

#### Suspected Breach

```bash
# 1. Immediately isolate affected systems
gcloud compute firewall-rules create emergency-block \
  --action=DENY \
  --rules=tcp:443,tcp:80 \
  --source-ranges=SUSPICIOUS_IP/32 \
  --priority=100

# 2. Rotate all credentials
./scripts/emergency-credential-rotation.sh

# 3. Enable additional logging
gcloud logging sinks create security-incident-sink \
  bigquery.googleapis.com/projects/$PROJECT_ID/datasets/security_logs \
  --log-filter='severity>=INFO'

# 4. Preserve evidence
gsutil -m cp -r gs://$PROJECT_ID-logs/$(date +%Y/%m/%d)/ \
  gs://$PROJECT_ID-incident-evidence/$(date +%Y%m%d)/
```

#### Access Review

```bash
# Review IAM bindings
gcloud projects get-iam-policy $PROJECT_ID \
  --format="table(bindings.role,bindings.members.flatten())" \
  > iam-audit-$(date +%Y%m%d).txt

# Check service account usage
gcloud logging read 'protoPayload.authenticationInfo.principalEmail~"@.*\.iam\.gserviceaccount\.com"' \
  --format="table(timestamp,protoPayload.authenticationInfo.principalEmail,protoPayload.methodName)" \
  --limit=100

# Review secret access
gcloud logging read 'resource.type="secretmanager.googleapis.com/Secret"' \
  --format="table(timestamp,protoPayload.authenticationInfo.principalEmail,protoPayload.resourceName)" \
  --limit=50
```

---

## Troubleshooting Guide

### Common Issues and Solutions

#### Issue: High Webhook Latency

**Symptoms**: P95 latency > 5 seconds, timeout errors

**Diagnosis**:
```bash
# Check current latency metrics
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/request_latencies"

# Check instance scaling
gcloud run services describe webhook-processor --region=us-central1 \
  --format="value(status.traffic[0].latestRevision,spec.template.spec.containerConcurrency)"

# Check CPU/memory utilization
gcloud logging read 'resource.type="cloud_run_revision" resource.labels.service_name="webhook-processor"' \
  --filter="textPayload:\"memory\" OR textPayload:\"cpu\"" --limit=10
```

**Solutions**:
```bash
# 1. Scale up resources
gcloud run services update webhook-processor \
  --region=us-central1 \
  --memory=8Gi \
  --cpu=4 \
  --min-instances=10

# 2. Reduce concurrency
gcloud run services update webhook-processor \
  --region=us-central1 \
  --concurrency=50

# 3. Check for cold starts
gcloud run services update webhook-processor \
  --region=us-central1 \
  --min-instances=20
```

#### Issue: Database Connection Errors

**Symptoms**: "connection refused", "too many connections"

**Diagnosis**:
```bash
# Check Spanner instance status
gcloud spanner instances describe pipeline-prod

# Check connection pool metrics
gcloud logging read 'resource.type="spanner_instance" textPayload:"connection"' \
  --limit=20

# Check current processing units
gcloud spanner instances describe pipeline-prod \
  --format="value(config,processingUnits)"
```

**Solutions**:
```bash
# 1. Scale up Spanner processing units
gcloud spanner instances update pipeline-prod --processing-units=2000

# 2. Optimize connection pooling
# Update application configuration for connection pool size

# 3. Check for long-running transactions
gcloud logging read 'resource.type="spanner_instance" severity>=WARNING' \
  --filter="textPayload:\"transaction\"" --limit=10
```

#### Issue: CRM Integration Failures

**Symptoms**: CRM sync success rate < 95%

**Diagnosis**:
```bash
# Check CRM-specific error logs
gcloud logging read 'resource.type="cloud_run_revision" textPayload:"crm"' \
  --filter="severity>=ERROR" --limit=20

# Check API rate limiting
gcloud logging read 'textPayload:"rate limit" OR textPayload:"429"' \
  --limit=10

# Test CRM connectivity
curl -o /dev/null -s -w "%{http_code}\n" https://api.hubapi.com/
curl -o /dev/null -s -w "%{http_code}\n" https://api.salesforce.com/
```

**Solutions**:
```bash
# 1. Implement exponential backoff
# Update application code for retry logic

# 2. Check API quotas and limits
# Review CRM API documentation for limits

# 3. Enable circuit breaker
# Deploy circuit breaker configuration
```

#### Issue: High Memory Usage

**Symptoms**: Out of memory errors, pod restarts

**Diagnosis**:
```bash
# Check memory metrics
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com/container/memory/utilizations"

# Check for memory leaks
gcloud logging read 'resource.type="cloud_run_revision" textPayload:"memory"' \
  --filter="severity>=WARNING" --limit=20

# Check container restarts
gcloud run revisions describe webhook-processor-latest --region=us-central1 \
  --format="value(status.conditions)"
```

**Solutions**:
```bash
# 1. Increase memory allocation
gcloud run services update webhook-processor \
  --region=us-central1 \
  --memory=8Gi

# 2. Optimize application memory usage
# Review application code for memory leaks

# 3. Reduce concurrency
gcloud run services update webhook-processor \
  --region=us-central1 \
  --concurrency=50
```

### Emergency Contact Information

#### On-Call Escalation

**Level 1 - Platform Team**
- PagerDuty: +1-555-ONCALL1
- Slack: @oncall-platform
- Primary: engineer1@company.com
- Secondary: engineer2@company.com

**Level 2 - Management**
- Engineering Manager: manager@company.com
- Platform Lead: lead@company.com
- Phone: +1-555-MGMT

**Level 3 - Executive**
- VP Engineering: vp@company.com
- CTO: cto@company.com
- Emergency Line: +1-555-EXEC

#### External Support

**Google Cloud Support**
- Console: https://console.cloud.google.com/support
- Phone: +1-855-836-3987
- Case Priority: Production Critical

**Third-Party Services**
- CallRail Support: support@callrail.com, +1-404-CALLRAIL
- HubSpot Support: https://help.hubspot.com/
- Salesforce Support: https://help.salesforce.com/

### Useful Commands Reference

```bash
# Quick health check
curl https://api.company.com/v1/health | jq '.'

# Check all services
gcloud run services list --region=us-central1

# View recent errors
gcloud logging read 'severity>=ERROR' --limit=20 --freshness=1h

# Check scaling status
gcloud run services describe webhook-processor --region=us-central1 \
  --format="value(spec.template.metadata.annotations)"

# Manual scale up
gcloud run services update webhook-processor \
  --region=us-central1 --min-instances=10

# Emergency rollback
PREV=$(gcloud run revisions list --service=webhook-processor \
  --region=us-central1 --limit=2 --format="value(metadata.name)" | tail -n1)
gcloud run services update-traffic webhook-processor \
  --region=us-central1 --to-revisions=$PREV=100

# Check database status
gcloud spanner instances describe pipeline-prod

# Create manual backup
gcloud spanner backups create emergency-backup-$(date +%s) \
  --instance=pipeline-prod --database=pipeline-db --retention-period=7d
```

---

**Remember**: When in doubt, escalate early. It's better to involve more people than needed than to let an incident continue without proper attention.

**This runbook should be reviewed quarterly and updated after every major incident.**