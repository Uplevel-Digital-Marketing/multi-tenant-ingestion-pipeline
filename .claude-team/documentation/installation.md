# Installation and Configuration Guide

## Table of Contents
- [Prerequisites](#prerequisites)
- [GCP Project Setup](#gcp-project-setup)
- [Service Configuration](#service-configuration)
- [Database Setup](#database-setup)
- [Deployment](#deployment)
- [Environment Configuration](#environment-configuration)
- [Verification](#verification)

## Prerequisites

### Required Tools
- [Google Cloud SDK](https://cloud.google.com/sdk) >= 400.0.0
- [Terraform](https://www.terraform.io) >= 1.5.0
- [Docker](https://www.docker.com) >= 24.0.0
- [Go](https://golang.org) >= 1.21
- [kubectl](https://kubernetes.io/docs/tasks/tools/) (if using GKE)

### Required Accounts
- Google Cloud Platform account with billing enabled
- CallRail account with API access
- CRM system credentials (HubSpot, Salesforce, etc.)

## GCP Project Setup

### 1. Create and Configure Project
```bash
# Set your project ID
export PROJECT_ID="your-ingestion-pipeline"
export REGION="us-central1"

# Create new project (optional)
gcloud projects create $PROJECT_ID

# Set active project
gcloud config set project $PROJECT_ID

# Enable billing (replace with your billing account)
gcloud billing projects link $PROJECT_ID --billing-account=YOUR_BILLING_ACCOUNT_ID
```

### 2. Enable Required APIs
```bash
gcloud services enable \
    cloudrun.googleapis.com \
    cloudbuild.googleapis.com \
    spanner.googleapis.com \
    storage.googleapis.com \
    speech.googleapis.com \
    aiplatform.googleapis.com \
    secretmanager.googleapis.com \
    cloudkms.googleapis.com \
    monitoring.googleapis.com \
    logging.googleapis.com \
    cloudfunctions.googleapis.com \
    pubsub.googleapis.com \
    firestore.googleapis.com
```

### 3. Create Service Account
```bash
# Create service account for the application
gcloud iam service-accounts create ingestion-pipeline \
    --display-name="Ingestion Pipeline Service Account"

# Grant necessary permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/spanner.databaseUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/storage.objectAdmin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/speech.editor"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"

# Download service account key
gcloud iam service-accounts keys create key.json \
    --iam-account=ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com
```

## Service Configuration

### 1. Cloud Spanner Setup
```bash
# Create Spanner instance
gcloud spanner instances create ingestion-db \
    --config=regional-$REGION \
    --description="Multi-tenant ingestion pipeline database" \
    --nodes=1

# Create database
gcloud spanner databases create pipeline-db \
    --instance=ingestion-db
```

### 2. Cloud Storage Setup
```bash
# Create bucket for audio files
gsutil mb -p $PROJECT_ID -c STANDARD -l $REGION gs://$PROJECT_ID-audio-files

# Create bucket for backups
gsutil mb -p $PROJECT_ID -c NEARLINE -l $REGION gs://$PROJECT_ID-backups

# Set lifecycle policy for audio files
cat > lifecycle.json << EOF
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "SetStorageClass", "storageClass": "COLDLINE"},
        "condition": {"age": 90}
      },
      {
        "action": {"type": "Delete"},
        "condition": {"age": 2555}
      }
    ]
  }
}
EOF

gsutil lifecycle set lifecycle.json gs://$PROJECT_ID-audio-files
```

### 3. Secret Manager Setup
```bash
# Create webhook secret
echo -n "your-webhook-secret-key" | gcloud secrets create callrail-webhook-secret --data-file=-

# Create CallRail API key (example for default tenant)
echo -n "your-callrail-api-key" | gcloud secrets create callrail-api-key --data-file=-

# Create database connection string
echo -n "projects/$PROJECT_ID/instances/ingestion-db/databases/pipeline-db" | \
    gcloud secrets create spanner-database --data-file=-

# Create JWT signing key
openssl rand -base64 32 | gcloud secrets create jwt-signing-key --data-file=-
```

## Database Setup

### 1. Create Database Schema
```sql
-- Connect to Spanner database
-- Use Cloud Console SQL editor or gcloud CLI

-- Tenants table
CREATE TABLE tenants (
    tenant_id STRING(100) NOT NULL,
    name STRING(255) NOT NULL,
    status STRING(20) NOT NULL,
    configuration JSON,
    created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (tenant_id);

-- Offices table (for CallRail mapping)
CREATE TABLE offices (
    office_id STRING(100) NOT NULL,
    tenant_id STRING(100) NOT NULL,
    name STRING(255) NOT NULL,
    callrail_company_id STRING(50),
    callrail_api_key STRING(500),
    workflow_config JSON,
    service_area JSON,
    status STRING(20) NOT NULL,
    created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    FOREIGN KEY (tenant_id) REFERENCES tenants (tenant_id),
) PRIMARY KEY (office_id);

-- Requests table
CREATE TABLE requests (
    request_id STRING(100) NOT NULL,
    tenant_id STRING(100) NOT NULL,
    source STRING(50) NOT NULL,
    request_type STRING(50) NOT NULL,
    status STRING(20) NOT NULL,
    data JSON NOT NULL,
    ai_normalized JSON,
    ai_extracted JSON,
    spam_likelihood INT64,
    call_id STRING(50),
    recording_url STRING(500),
    transcription_data JSON,
    ai_analysis JSON,
    lead_score INT64,
    workflow_steps JSON,
    created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    updated_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    FOREIGN KEY (tenant_id) REFERENCES tenants (tenant_id),
) PRIMARY KEY (request_id);

-- Workflow executions table
CREATE TABLE workflow_executions (
    execution_id STRING(100) NOT NULL,
    request_id STRING(100) NOT NULL,
    tenant_id STRING(100) NOT NULL,
    workflow_name STRING(100) NOT NULL,
    status STRING(20) NOT NULL,
    steps JSON,
    error_details JSON,
    started_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
    completed_at TIMESTAMP,
    FOREIGN KEY (request_id) REFERENCES requests (request_id),
    FOREIGN KEY (tenant_id) REFERENCES tenants (tenant_id),
) PRIMARY KEY (execution_id);

-- Create indexes
CREATE INDEX idx_offices_callrail ON offices(callrail_company_id, tenant_id);
CREATE INDEX idx_requests_tenant_created ON requests(tenant_id, created_at DESC);
CREATE INDEX idx_requests_call_id ON requests(call_id);
CREATE INDEX idx_requests_lead_score ON requests(tenant_id, lead_score DESC);
CREATE INDEX idx_requests_status ON requests(tenant_id, status);
CREATE INDEX idx_workflow_executions_request ON workflow_executions(request_id);
```

### 2. Insert Sample Data
```sql
-- Insert sample tenant
INSERT INTO tenants (tenant_id, name, status, configuration, created_at, updated_at)
VALUES (
    'tenant_sample_company',
    'Sample Remodeling Company',
    'active',
    JSON '{
        "crm_type": "hubspot",
        "email_notifications": true,
        "auto_assignment": true,
        "business_hours": {
            "timezone": "America/Los_Angeles",
            "monday": {"start": "08:00", "end": "18:00"},
            "tuesday": {"start": "08:00", "end": "18:00"},
            "wednesday": {"start": "08:00", "end": "18:00"},
            "thursday": {"start": "08:00", "end": "18:00"},
            "friday": {"start": "08:00", "end": "18:00"},
            "saturday": {"start": "09:00", "end": "15:00"},
            "sunday": {"closed": true}
        }
    }',
    PENDING_COMMIT_TIMESTAMP(),
    PENDING_COMMIT_TIMESTAMP()
);

-- Insert sample office
INSERT INTO offices (office_id, tenant_id, name, callrail_company_id, callrail_api_key, workflow_config, service_area, status, created_at, updated_at)
VALUES (
    'office_main_location',
    'tenant_sample_company',
    'Main Office',
    '12345',
    'callrail_api_key_here',
    JSON '{
        "lead_routing": "round_robin",
        "qualification_required": true,
        "appointment_booking": true
    }',
    JSON '{
        "cities": ["Los Angeles", "Beverly Hills", "Santa Monica"],
        "zip_codes": ["90210", "90211", "90212"],
        "radius_miles": 25
    }',
    'active',
    PENDING_COMMIT_TIMESTAMP(),
    PENDING_COMMIT_TIMESTAMP()
);
```

## Deployment

### 1. Using Cloud Build (Recommended)
Create `cloudbuild.yaml`:
```yaml
steps:
  # Build the application image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'build'
      - '-t'
      - 'gcr.io/$PROJECT_ID/ingestion-pipeline:$BUILD_ID'
      - '.'

  # Push to Container Registry
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - 'gcr.io/$PROJECT_ID/ingestion-pipeline:$BUILD_ID'

  # Deploy to Cloud Run
  - name: 'gcr.io/cloud-builders/gcloud'
    args:
      - 'run'
      - 'deploy'
      - 'ingestion-pipeline'
      - '--image'
      - 'gcr.io/$PROJECT_ID/ingestion-pipeline:$BUILD_ID'
      - '--region'
      - '$_REGION'
      - '--platform'
      - 'managed'
      - '--allow-unauthenticated'
      - '--set-env-vars'
      - 'PROJECT_ID=$PROJECT_ID,ENVIRONMENT=production'
      - '--service-account'
      - 'ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com'
      - '--memory'
      - '2Gi'
      - '--cpu'
      - '2'
      - '--min-instances'
      - '1'
      - '--max-instances'
      - '100'
      - '--timeout'
      - '300'

substitutions:
  _REGION: us-central1

options:
  logging: CLOUD_LOGGING_ONLY
```

Deploy with Cloud Build:
```bash
gcloud builds submit --config=cloudbuild.yaml
```

### 2. Using Terraform (Infrastructure as Code)
Create `main.tf`:
```hcl
terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "us-central1"
}

# Cloud Run service
resource "google_cloud_run_service" "ingestion_pipeline" {
  name     = "ingestion-pipeline"
  location = var.region

  template {
    spec {
      service_account_name = google_service_account.ingestion_pipeline.email

      containers {
        image = "gcr.io/${var.project_id}/ingestion-pipeline:latest"

        resources {
          limits = {
            cpu    = "2000m"
            memory = "2Gi"
          }
        }

        env {
          name  = "PROJECT_ID"
          value = var.project_id
        }

        env {
          name  = "ENVIRONMENT"
          value = "production"
        }

        env {
          name = "SPANNER_DATABASE"
          value_from {
            secret_key_ref {
              name = google_secret_manager_secret.spanner_database.secret_id
              key  = "latest"
            }
          }
        }
      }
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/minScale" = "1"
        "autoscaling.knative.dev/maxScale" = "100"
        "run.googleapis.com/cpu-throttling" = "false"
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

# IAM policy for Cloud Run service
data "google_iam_policy" "noauth" {
  binding {
    role = "roles/run.invoker"
    members = [
      "allUsers",
    ]
  }
}

resource "google_cloud_run_service_iam_policy" "noauth" {
  location = google_cloud_run_service.ingestion_pipeline.location
  project  = google_cloud_run_service.ingestion_pipeline.project
  service  = google_cloud_run_service.ingestion_pipeline.name

  policy_data = data.google_iam_policy.noauth.policy_data
}
```

Deploy with Terraform:
```bash
terraform init
terraform plan -var="project_id=$PROJECT_ID"
terraform apply -auto-approve
```

## Environment Configuration

### 1. Development Environment
```bash
# Create .env file for local development
cat > .env << EOF
# Project Configuration
PROJECT_ID=$PROJECT_ID
ENVIRONMENT=development
REGION=$REGION

# Database
SPANNER_DATABASE=projects/$PROJECT_ID/instances/ingestion-db/databases/pipeline-db

# Storage
AUDIO_BUCKET=$PROJECT_ID-audio-files
BACKUP_BUCKET=$PROJECT_ID-backups

# APIs
SPEECH_API_ENDPOINT=speech.googleapis.com
VERTEX_AI_ENDPOINT=aiplatform.googleapis.com

# Security
JWT_SIGNING_KEY=your-jwt-signing-key
CALLRAIL_WEBHOOK_SECRET=your-webhook-secret

# Monitoring
LOG_LEVEL=info
ENABLE_METRICS=true

# External Services
CALLRAIL_API_BASE_URL=https://api.callrail.com/v3
EOF
```

### 2. Production Environment Variables
Set these in Cloud Run or your deployment system:
```bash
# Required environment variables
PROJECT_ID=your-project-id
ENVIRONMENT=production
SPANNER_DATABASE=projects/your-project-id/instances/ingestion-db/databases/pipeline-db
AUDIO_BUCKET=your-project-id-audio-files
JWT_SIGNING_KEY=secret:projects/your-project-id/secrets/jwt-signing-key/versions/latest
CALLRAIL_WEBHOOK_SECRET=secret:projects/your-project-id/secrets/callrail-webhook-secret/versions/latest
```

## Verification

### 1. Health Check
```bash
# Get Cloud Run service URL
SERVICE_URL=$(gcloud run services describe ingestion-pipeline \
    --region=$REGION \
    --format="value(status.url)")

# Test health endpoint
curl $SERVICE_URL/v1/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-09-13T15:30:00Z",
  "version": "1.0.0",
  "checks": {
    "database": "healthy",
    "storage": "healthy",
    "speech_api": "healthy"
  }
}
```

### 2. Test CallRail Webhook
```bash
# Test webhook endpoint (use CallRail test payload)
curl -X POST $SERVICE_URL/v1/callrail/webhook \
  -H "Content-Type: application/json" \
  -H "x-callrail-signature: sha256=test-signature" \
  -d '{
    "call_id": "TEST123",
    "tenant_id": "tenant_sample_company",
    "callrail_company_id": "12345",
    "caller_id": "+15551234567",
    "duration": "120",
    "answered": true
  }'
```

### 3. Check Database Connectivity
```bash
# Test database connection
gcloud spanner databases execute-sql pipeline-db \
    --instance=ingestion-db \
    --sql="SELECT COUNT(*) as tenant_count FROM tenants"
```

### 4. Verify Monitoring
```bash
# Check Cloud Logging
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=ingestion-pipeline" --limit=10

# Check metrics in Cloud Monitoring
gcloud monitoring dashboards list | grep ingestion
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Errors
```bash
# Check Spanner instance status
gcloud spanner instances describe ingestion-db

# Verify service account permissions
gcloud projects get-iam-policy $PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:ingestion-pipeline@$PROJECT_ID.iam.gserviceaccount.com"
```

#### 2. Secret Manager Access Issues
```bash
# Test secret access
gcloud secrets versions access latest --secret="callrail-webhook-secret"

# Check secret permissions
gcloud secrets get-iam-policy callrail-webhook-secret
```

#### 3. Speech API Errors
```bash
# Test Speech-to-Text API
gcloud ml speech recognize --audio-encoding=LINEAR16 \
  --sample-rate=16000 \
  --language-code=en-US \
  gs://cloud-samples-data/speech/brooklyn_bridge.raw
```

### Logs and Monitoring
```bash
# View recent errors
gcloud logging read "severity>=ERROR AND resource.type=cloud_run_revision" --limit=50

# Follow logs in real-time
gcloud logging tail "resource.type=cloud_run_revision AND resource.labels.service_name=ingestion-pipeline"

# Check service metrics
gcloud monitoring metrics list --filter="metric.type:run.googleapis.com"
```

## Next Steps
1. Review the [API Documentation](../api/openapi.yaml)
2. Set up [Tenant Onboarding](../user/tenant-onboarding.md)
3. Configure [Monitoring and Alerting](../ops/monitoring.md)
4. Set up [CRM Integrations](../user/crm-integration.md)