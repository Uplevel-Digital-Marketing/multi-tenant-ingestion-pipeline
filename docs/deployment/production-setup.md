# Production Deployment Guide

**Multi-Tenant Ingestion Pipeline**

---

**Version**: 2.0
**Date**: September 13, 2025
**Target Environment**: Google Cloud Platform Production
**Estimated Deployment Time**: 2-4 hours

---

## Table of Contents

1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Infrastructure Setup](#infrastructure-setup)
3. [Security Configuration](#security-configuration)
4. [Application Deployment](#application-deployment)
5. [Database Setup](#database-setup)
6. [Monitoring Configuration](#monitoring-configuration)
7. [Testing and Validation](#testing-and-validation)
8. [Post-Deployment Tasks](#post-deployment-tasks)
9. [Rollback Procedures](#rollback-procedures)
10. [Troubleshooting](#troubleshooting)

---

## Pre-Deployment Checklist

### Prerequisites Verification

Before beginning deployment, verify you have:

- [ ] **Google Cloud Project**: Production project with billing enabled
- [ ] **IAM Permissions**: Project Owner or Editor role
- [ ] **CLI Tools Installed**:
  ```bash
  # Verify installations
  gcloud --version  # >= 400.0.0
  terraform --version  # >= 1.5.0 (if using Terraform)
  kubectl --version  # >= 1.27.0
  ```

### Resource Planning

| Component | Specification | Monthly Cost (1K calls) |
|-----------|---------------|--------------------------|
| **Cloud Run** | 2-100 instances, 2 vCPU, 4GB RAM | $80 |
| **Cloud Spanner** | Regional, 1000 Processing Units | $650 |
| **Speech-to-Text** | 50 hours/month | $720 |
| **Vertex AI** | 1000 requests/month | $250 |
| **Cloud Storage** | 25GB storage + operations | $25 |
| **Load Balancer** | Global HTTPS LB | $18 |
| **Networking** | Egress and inter-region | $100 |
| **Total Estimated** | | **$1,843/month** |

### Environment Variables

Create a secure environment configuration:

```bash
# Production environment setup
export PROJECT_ID="your-production-project-id"
export REGION="us-central1"
export ZONE="us-central1-a"
export ENVIRONMENT="production"
export DOMAIN="api.yourcompany.com"
export SERVICE_NAME="ingestion-pipeline"

# Set GCP project
gcloud config set project $PROJECT_ID
gcloud config set compute/region $REGION
gcloud config set compute/zone $ZONE
```

---

## Infrastructure Setup

### Step 1: Enable Required APIs

Enable all necessary Google Cloud APIs:

```bash
# Core compute and database APIs
gcloud services enable \
    cloudrun.googleapis.com \
    spanner.googleapis.com \
    cloudbuild.googleapis.com \
    storage-component.googleapis.com \
    storage.googleapis.com

# AI and ML APIs
gcloud services enable \
    speech.googleapis.com \
    aiplatform.googleapis.com \
    ml.googleapis.com

# Security and monitoring APIs
gcloud services enable \
    secretmanager.googleapis.com \
    cloudkms.googleapis.com \
    logging.googleapis.com \
    monitoring.googleapis.com \
    cloudtrace.googleapis.com \
    clouddebugger.googleapis.com

# Networking and load balancing
gcloud services enable \
    compute.googleapis.com \
    dns.googleapis.com \
    certificatemanager.googleapis.com
```

### Step 2: Create Core Infrastructure

#### Cloud Spanner Database

Create the multi-tenant database instance:

```bash
# Create Spanner instance (regional for production)
gcloud spanner instances create pipeline-prod \
    --config=regional-${REGION} \
    --description="Production ingestion pipeline database" \
    --processing-units=1000

# Create database
gcloud spanner databases create pipeline-db \
    --instance=pipeline-prod \
    --ddl-file=sql/schema.sql
```

#### Cloud Storage Buckets

```bash
# Create buckets for audio storage and backups
gsutil mb -p $PROJECT_ID -c STANDARD -l $REGION gs://${PROJECT_ID}-audio-files
gsutil mb -p $PROJECT_ID -c NEARLINE -l $REGION gs://${PROJECT_ID}-backups
gsutil mb -p $PROJECT_ID -c STANDARD -l $REGION gs://${PROJECT_ID}-terraform-state

# Set bucket policies
gsutil iam ch serviceAccount:pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com:objectAdmin \
    gs://${PROJECT_ID}-audio-files

# Enable versioning for backup bucket
gsutil versioning set on gs://${PROJECT_ID}-backups
```

#### Cloud KMS Setup

Set up encryption keys for sensitive data:

```bash
# Create key ring
gcloud kms keyrings create pipeline-keys --location=global

# Create encryption keys
gcloud kms keys create pii-encryption-key \
    --location=global \
    --keyring=pipeline-keys \
    --purpose=encryption

gcloud kms keys create audio-encryption-key \
    --location=global \
    --keyring=pipeline-keys \
    --purpose=encryption
```

### Step 3: Network Configuration

#### VPC and Subnets

```bash
# Create VPC network
gcloud compute networks create pipeline-vpc \
    --subnet-mode=custom \
    --bgp-routing-mode=regional

# Create subnet
gcloud compute networks subnets create pipeline-subnet \
    --network=pipeline-vpc \
    --range=10.1.0.0/24 \
    --region=$REGION
```

#### Cloud Load Balancer

```bash
# Reserve static IP
gcloud compute addresses create pipeline-ip \
    --global

# Create SSL certificate (managed)
gcloud compute ssl-certificates create pipeline-ssl-cert \
    --domains=$DOMAIN \
    --global
```

---

## Security Configuration

### Service Account Creation

Create dedicated service accounts with minimal required permissions:

```bash
# Main pipeline service account
gcloud iam service-accounts create pipeline-service \
    --display-name="Pipeline Main Service" \
    --description="Primary service account for ingestion pipeline"

# Set required IAM roles
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/spanner.databaseUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/storage.objectAdmin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/speech.client"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"
```

### Secret Management

Store sensitive configuration in Secret Manager:

```bash
# Create secrets for API keys and configuration
gcloud secrets create callrail-api-key --data-file=secrets/callrail-key.txt
gcloud secrets create hubspot-api-key --data-file=secrets/hubspot-key.txt
gcloud secrets create salesforce-credentials --data-file=secrets/sf-creds.json
gcloud secrets create jwt-signing-key --data-file=secrets/jwt-key.pem
gcloud secrets create database-connection --data-file=secrets/db-config.json

# Grant access to service account
for SECRET in callrail-api-key hubspot-api-key salesforce-credentials jwt-signing-key database-connection; do
    gcloud secrets add-iam-policy-binding $SECRET \
        --member="serviceAccount:pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com" \
        --role="roles/secretmanager.secretAccessor"
done
```

### Row-Level Security Configuration

Enable multi-tenant data isolation in Spanner:

```sql
-- Connect to Spanner and run these commands
-- Enable RLS on all tenant tables
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE processing_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE call_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm_integrations ENABLE ROW LEVEL SECURITY;

-- Activate tenant isolation policies
CREATE ROW ACCESS POLICY tenant_isolation_requests
ON processing_requests
GRANT TO ('pipeline-service@PROJECT_ID.iam.gserviceaccount.com')
FILTER USING (tenant_id = @tenant_id);

CREATE ROW ACCESS POLICY tenant_isolation_calls
ON call_records
GRANT TO ('pipeline-service@PROJECT_ID.iam.gserviceaccount.com')
FILTER USING (tenant_id = @tenant_id);

CREATE ROW ACCESS POLICY tenant_isolation_crm
ON crm_integrations
GRANT TO ('pipeline-service@PROJECT_ID.iam.gserviceaccount.com')
FILTER USING (tenant_id = @tenant_id);
```

---

## Application Deployment

### Step 1: Build Container Image

Build and push the application container to Google Container Registry:

```bash
# Build container image
gcloud builds submit \
    --tag gcr.io/$PROJECT_ID/ingestion-pipeline:latest \
    --machine-type=e2-highcpu-8 \
    --timeout=20m

# Tag for production
gcloud container images add-tag \
    gcr.io/$PROJECT_ID/ingestion-pipeline:latest \
    gcr.io/$PROJECT_ID/ingestion-pipeline:prod-v1.0.0
```

### Step 2: Deploy to Cloud Run

Deploy the main application service:

```bash
# Deploy primary service
gcloud run deploy ingestion-pipeline \
    --image gcr.io/$PROJECT_ID/ingestion-pipeline:prod-v1.0.0 \
    --platform managed \
    --region $REGION \
    --service-account pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com \
    --set-env-vars "PROJECT_ID=${PROJECT_ID},ENVIRONMENT=production,REGION=${REGION}" \
    --memory 4Gi \
    --cpu 2 \
    --concurrency 80 \
    --max-instances 100 \
    --min-instances 2 \
    --port 8080 \
    --timeout 300 \
    --no-allow-unauthenticated

# Deploy webhook processing service
gcloud run deploy webhook-processor \
    --image gcr.io/$PROJECT_ID/ingestion-pipeline:prod-v1.0.0 \
    --platform managed \
    --region $REGION \
    --service-account pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com \
    --set-env-vars "PROJECT_ID=${PROJECT_ID},ENVIRONMENT=production,SERVICE_TYPE=webhook" \
    --memory 2Gi \
    --cpu 1 \
    --concurrency 50 \
    --max-instances 50 \
    --min-instances 1 \
    --port 8080 \
    --timeout 60 \
    --allow-unauthenticated

# Deploy audio processor service
gcloud run deploy audio-processor \
    --image gcr.io/$PROJECT_ID/ingestion-pipeline:prod-v1.0.0 \
    --platform managed \
    --region $REGION \
    --service-account pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com \
    --set-env-vars "PROJECT_ID=${PROJECT_ID},ENVIRONMENT=production,SERVICE_TYPE=audio" \
    --memory 8Gi \
    --cpu 4 \
    --concurrency 10 \
    --max-instances 25 \
    --min-instances 0 \
    --timeout 900 \
    --no-allow-unauthenticated
```

### Step 3: Configure Load Balancer

Set up the global load balancer:

```bash
# Create backend service for webhooks
gcloud compute backend-services create webhook-backend \
    --global \
    --protocol HTTPS \
    --health-checks webhook-health-check \
    --timeout 30s

# Add Cloud Run backend
gcloud compute backend-services add-backend webhook-backend \
    --global \
    --backend-service-type SERVERLESS \
    --backend-service webhook-processor

# Create URL map
gcloud compute url-maps create pipeline-url-map \
    --default-service webhook-backend

# Add path rules for different services
gcloud compute url-maps add-path-matcher pipeline-url-map \
    --path-matcher-name pipeline-matcher \
    --default-service webhook-backend \
    --backend-service-path-rules "/v1/callrail/*=webhook-backend,/v1/health=webhook-backend"

# Create HTTPS proxy
gcloud compute target-https-proxies create pipeline-https-proxy \
    --ssl-certificates pipeline-ssl-cert \
    --url-map pipeline-url-map

# Create forwarding rule
gcloud compute forwarding-rules create pipeline-forwarding-rule \
    --global \
    --target-https-proxy pipeline-https-proxy \
    --ports 443 \
    --address pipeline-ip
```

---

## Database Setup

### Initialize Database Schema

Deploy the production database schema:

```bash
# Run schema migrations
gcloud spanner databases execute-sql pipeline-db \
    --instance=pipeline-prod \
    --sql-file=sql/001_initial_schema.sql

gcloud spanner databases execute-sql pipeline-db \
    --instance=pipeline-prod \
    --sql-file=sql/002_tenant_isolation.sql

gcloud spanner databases execute-sql pipeline-db \
    --instance=pipeline-prod \
    --sql-file=sql/003_indexes.sql
```

### Create Initial Tenant

Set up the first production tenant:

```sql
-- Create demo/test tenant
INSERT INTO tenants (
    tenant_id,
    name,
    status,
    config,
    created_at
) VALUES (
    'demo-tenant-001',
    'Demo Company LLC',
    'active',
    JSON '{
        "callrail_company_id": "12345",
        "webhook_secret": "demo_webhook_secret_change_in_prod",
        "crm_config": {
            "hubspot_api_key_secret": "hubspot-api-key",
            "hubspot_portal_id": "12345678"
        },
        "ai_config": {
            "lead_scoring_threshold": 70,
            "analysis_model": "gemini-pro"
        }
    }',
    CURRENT_TIMESTAMP()
);
```

---

## Monitoring Configuration

### Cloud Operations Setup

Configure comprehensive monitoring and alerting:

```bash
# Create custom dashboard
gcloud logging sinks create pipeline-errors-sink \
    bigquery.googleapis.com/projects/$PROJECT_ID/datasets/pipeline_logs \
    --log-filter='resource.type="cloud_run_revision" severity>=ERROR'

# Create notification channels
gcloud alpha monitoring channels create \
    --display-name="Pipeline Alerts Slack" \
    --type=slack \
    --channel-labels=channel_name=#alerts,url=YOUR_SLACK_WEBHOOK_URL

gcloud alpha monitoring channels create \
    --display-name="Pipeline Alerts Email" \
    --type=email \
    --channel-labels=email_address=ops@yourcompany.com
```

### Critical Alert Policies

```yaml
# alerting-policy.yaml
policies:
  - displayName: "High Error Rate"
    conditions:
      - displayName: "Error rate > 5%"
        conditionThreshold:
          filter: 'resource.type="cloud_run_revision" resource.label.service_name="webhook-processor"'
          comparison: COMPARISON_GREATER_THAN
          thresholdValue: 0.05
          duration: 300s
    alertStrategy:
      autoClose: 86400s
    notificationChannels:
      - "projects/PROJECT_ID/notificationChannels/CHANNEL_ID"

  - displayName: "Service Unavailable"
    conditions:
      - displayName: "Availability < 99%"
        conditionThreshold:
          filter: 'resource.type="cloud_run_revision"'
          comparison: COMPARISON_LESS_THAN
          thresholdValue: 0.99
          duration: 120s
```

### Performance Monitoring

Set up SLI/SLO monitoring:

```bash
# Create SLO for webhook latency
gcloud alpha monitoring slos create \
    --service=webhook-processor \
    --slo-id=webhook-latency-slo \
    --display-name="Webhook 95th percentile latency < 2s" \
    --goal=0.995 \
    --calendar-period=30d \
    --request-based-sli-distribution-cut-range-max=2000
```

---

## Testing and Validation

### Pre-Production Testing

Execute comprehensive testing before going live:

#### 1. Health Check Validation

```bash
# Test all service endpoints
WEBHOOK_URL=$(gcloud run services describe webhook-processor \
    --region=$REGION --format="value(status.url)")

curl -f "$WEBHOOK_URL/v1/health" | jq '.'

# Expected response:
# {
#   "status": "healthy",
#   "timestamp": "2025-09-13T10:00:00Z",
#   "services": {
#     "database": "healthy",
#     "storage": "healthy",
#     "speech_api": "healthy",
#     "vertex_ai": "healthy"
#   }
# }
```

#### 2. Database Connectivity Test

```bash
# Test Spanner connectivity
gcloud spanner databases execute-sql pipeline-db \
    --instance=pipeline-prod \
    --sql="SELECT COUNT(*) as tenant_count FROM tenants WHERE status = 'active'"
```

#### 3. Load Testing

```bash
# Install load testing tool
go install github.com/tsenart/vegeta@latest

# Create test payload
cat > test-webhook.json << EOF
{
  "call_id": "TEST123456789",
  "caller_id": "+15551234567",
  "duration": "180",
  "answered": true,
  "tenant_id": "demo-tenant-001",
  "callrail_company_id": "12345",
  "recording_url": "https://example.com/test-recording.wav",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

# Run load test (100 requests over 1 minute)
echo "POST $WEBHOOK_URL/v1/callrail/webhook" | \
  vegeta attack -duration=60s -rate=100 -header="Content-Type: application/json" \
  -body=test-webhook.json | vegeta report
```

#### 4. End-to-End Integration Test

```bash
# Test complete workflow
python3 tests/e2e/test_production_flow.py \
    --project-id=$PROJECT_ID \
    --webhook-url=$WEBHOOK_URL \
    --tenant-id=demo-tenant-001
```

---

## Post-Deployment Tasks

### Security Hardening

Complete security configuration:

```bash
# Enable audit logging
gcloud logging sinks create security-audit-sink \
    storage.googleapis.com/${PROJECT_ID}-security-audit-logs \
    --log-filter='protoPayload.serviceName="spanner.googleapis.com" OR
                  protoPayload.serviceName="secretmanager.googleapis.com" OR
                  resource.type="cloud_run_revision"'

# Configure VPC Service Controls (if required)
gcloud access-context-manager perimeters create pipeline-perimeter \
    --policy=YOUR_POLICY_ID \
    --title="Pipeline Security Perimeter" \
    --resources=projects/$PROJECT_ID \
    --restricted-services=spanner.googleapis.com,secretmanager.googleapis.com
```

### Backup Configuration

Set up automated backups:

```bash
# Schedule Spanner backups
gcloud spanner backup-schedules create pipeline-daily-backup \
    --instance=pipeline-prod \
    --database=pipeline-db \
    --cron="0 2 * * *" \
    --retention-period=30d \
    --backup-type=FULL

# Configure storage lifecycle
gsutil lifecycle set lifecycle-config.json gs://${PROJECT_ID}-audio-files
```

### Documentation Updates

Update production documentation:

```bash
# Create production runbook
cp docs/operations/runbook-template.md docs/operations/production-runbook.md

# Update with production-specific values:
# - Service URLs
# - Database instance names
# - Monitoring dashboard links
# - Alert escalation procedures
```

---

## Rollback Procedures

### Emergency Rollback Plan

If issues are discovered post-deployment:

#### 1. Traffic Rollback (< 5 minutes)

```bash
# Rollback to previous Cloud Run revision
PREVIOUS_REVISION=$(gcloud run revisions list \
    --service=webhook-processor \
    --region=$REGION \
    --limit=2 --format="value(metadata.name)" | tail -n1)

gcloud run services update-traffic webhook-processor \
    --region=$REGION \
    --to-revisions=$PREVIOUS_REVISION=100
```

#### 2. Database Schema Rollback

```bash
# If schema changes need rollback
gcloud spanner databases execute-sql pipeline-db \
    --instance=pipeline-prod \
    --sql-file=sql/rollback/rollback-v1.0.0.sql
```

#### 3. Full Environment Rollback

```bash
# Restore from backup (if needed)
gcloud spanner databases restore \
    --source-backup=projects/$PROJECT_ID/instances/pipeline-prod/backups/BACKUP_NAME \
    --target-database=pipeline-db-restored \
    --target-instance=pipeline-prod
```

---

## Troubleshooting

### Common Issues and Solutions

#### 1. Service Startup Failures

**Symptom**: Cloud Run services fail to start
```bash
# Check service logs
gcloud run services logs read webhook-processor --region=$REGION --limit=100

# Common fixes:
# - Verify service account permissions
# - Check secret manager access
# - Validate environment variables
```

#### 2. Database Connection Issues

**Symptom**: "connection refused" or "permission denied" errors
```bash
# Verify Spanner instance status
gcloud spanner instances describe pipeline-prod

# Check service account IAM bindings
gcloud projects get-iam-policy $PROJECT_ID \
    --flatten="bindings[].members" \
    --filter="bindings.members:pipeline-service@${PROJECT_ID}.iam.gserviceaccount.com"
```

#### 3. SSL Certificate Issues

**Symptom**: HTTPS not working or certificate errors
```bash
# Check certificate status
gcloud compute ssl-certificates describe pipeline-ssl-cert --global

# Certificate provisioning can take 15-60 minutes
# Verify DNS points to correct IP address
nslookup $DOMAIN
```

#### 4. High Latency Issues

**Symptom**: Response times > 2 seconds
```bash
# Check Cloud Run cold starts
gcloud run services describe webhook-processor \
    --region=$REGION \
    --format="value(status.traffic[0].latestRevision)"

# Increase minimum instances if needed
gcloud run services update webhook-processor \
    --region=$REGION \
    --min-instances=3
```

### Support Escalation

For critical production issues:

1. **Level 1**: Check service health and logs (15 minutes)
2. **Level 2**: Engage platform team (30 minutes)
3. **Level 3**: Contact Google Cloud support (1 hour)
4. **Level 4**: Execute emergency rollback (immediate)

### Health Check Commands

Quick system health verification:

```bash
#!/bin/bash
# production-health-check.sh

echo "=== Production Health Check ==="
echo "Timestamp: $(date)"
echo

# 1. Service Status
echo "Cloud Run Services:"
gcloud run services list --region=$REGION --filter="metadata.name:pipeline OR metadata.name:webhook OR metadata.name:audio"

# 2. Database Status
echo -e "\nSpanner Database:"
gcloud spanner instances describe pipeline-prod --format="value(state,displayName)"

# 3. Load Balancer Status
echo -e "\nLoad Balancer:"
gcloud compute forwarding-rules describe pipeline-forwarding-rule --global --format="value(IPAddress,target)"

# 4. Certificate Status
echo -e "\nSSL Certificate:"
gcloud compute ssl-certificates describe pipeline-ssl-cert --global --format="value(managed.status)"

# 5. Recent Errors
echo -e "\nRecent Error Count (last hour):"
gcloud logging read 'resource.type="cloud_run_revision" severity>=ERROR timestamp>="2025-09-13T09:00:00Z"' --format="value(timestamp,resource.labels.service_name,textPayload)" --limit=10

echo -e "\n=== Health Check Complete ==="
```

---

## Production Deployment Checklist

Final verification before marking deployment complete:

### Infrastructure ✅
- [ ] All GCP APIs enabled and functioning
- [ ] Spanner instance created and accessible
- [ ] Cloud Storage buckets created with proper IAM
- [ ] KMS keys created and accessible
- [ ] VPC and networking configured
- [ ] Load balancer configured with SSL certificate
- [ ] Static IP assigned and DNS updated

### Security ✅
- [ ] Service accounts created with minimal permissions
- [ ] Secrets stored in Secret Manager
- [ ] Row-level security enabled and tested
- [ ] Audit logging configured
- [ ] VPC Service Controls enabled (if required)
- [ ] Binary authorization configured (if required)

### Application ✅
- [ ] Container images built and tagged
- [ ] All Cloud Run services deployed and healthy
- [ ] Environment variables configured correctly
- [ ] Health checks responding successfully
- [ ] Load balancer routing traffic correctly

### Database ✅
- [ ] Schema deployed successfully
- [ ] Initial tenant created and tested
- [ ] Row-level security policies active
- [ ] Backup schedule configured
- [ ] Connection pooling optimized

### Monitoring ✅
- [ ] Cloud Operations configured
- [ ] Alert policies created and tested
- [ ] Notification channels configured
- [ ] SLI/SLO monitoring active
- [ ] Custom dashboards deployed

### Testing ✅
- [ ] Health checks passing
- [ ] End-to-end integration test successful
- [ ] Load testing completed with acceptable performance
- [ ] Security testing passed
- [ ] Rollback procedures tested

### Documentation ✅
- [ ] Production runbook updated
- [ ] Architecture documentation current
- [ ] API documentation published
- [ ] Troubleshooting guide updated
- [ ] Team notified of deployment

---

**Deployment Complete!** 🎉

Your multi-tenant ingestion pipeline is now running in production. Monitor the system closely for the first 24 hours and ensure all alerts are configured properly.

**Next Steps:**
1. Set up tenant onboarding process
2. Configure production CRM integrations
3. Schedule security audit review
4. Plan capacity scaling for growth

For ongoing operations, refer to the [Operations Runbook](../operations/runbook.md).