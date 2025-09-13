# Tenant Onboarding Guide

## Table of Contents
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Step-by-Step Onboarding](#step-by-step-onboarding)
- [CallRail Integration](#callrail-integration)
- [CRM Configuration](#crm-configuration)
- [Testing and Verification](#testing-and-verification)
- [Common Issues](#common-issues)

## Overview

This guide walks through the complete process of onboarding a new tenant to the multi-tenant ingestion pipeline. Each tenant represents a separate business (e.g., a remodeling company) with their own data isolation, configurations, and integrations.

### What You'll Set Up
- Tenant account and configuration
- CallRail webhook integration
- CRM system connection
- Workflow automation rules
- Monitoring and notifications

## Prerequisites

### Information Required
Before starting, collect the following information from the new tenant:

#### Business Information
- Company name and contact details
- Primary business phone number
- Service area (cities, zip codes, radius)
- Business hours and timezone
- Primary contact email

#### CallRail Account Details
- CallRail account ID
- CallRail company ID
- CallRail API key
- Webhook secret token
- Phone numbers to track

#### CRM System Information
- CRM platform (HubSpot, Salesforce, Pipedrive, etc.)
- API credentials or access tokens
- Pipeline/stage configuration
- Field mappings

#### Notification Preferences
- Email addresses for lead notifications
- Slack webhook URLs (if applicable)
- SMS notification preferences

## Step-by-Step Onboarding

### Step 1: Create Tenant Account

#### Using Admin API
```bash
# Create new tenant
curl -X POST https://api.pipeline.com/v1/admin/tenants \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant_acme_remodeling",
    "name": "ACME Remodeling Company",
    "contact_email": "admin@acmeremodeling.com",
    "phone": "+15551234567",
    "status": "active",
    "configuration": {
      "timezone": "America/Los_Angeles",
      "currency": "USD",
      "business_hours": {
        "monday": {"start": "08:00", "end": "18:00"},
        "tuesday": {"start": "08:00", "end": "18:00"},
        "wednesday": {"start": "08:00", "end": "18:00"},
        "thursday": {"start": "08:00", "end": "18:00"},
        "friday": {"start": "08:00", "end": "18:00"},
        "saturday": {"start": "09:00", "end": "15:00"},
        "sunday": {"closed": true}
      },
      "notification_settings": {
        "email_notifications": true,
        "sms_notifications": false,
        "slack_notifications": true
      },
      "data_retention": {
        "recordings_days": 2555,
        "transcripts_days": 2555,
        "analytics_days": 1095
      }
    }
  }'
```

#### Using Database Direct Insert
```sql
INSERT INTO tenants (
    tenant_id,
    name,
    status,
    configuration,
    created_at,
    updated_at
) VALUES (
    'tenant_acme_remodeling',
    'ACME Remodeling Company',
    'active',
    JSON '{
        "contact_email": "admin@acmeremodeling.com",
        "phone": "+15551234567",
        "timezone": "America/Los_Angeles",
        "business_hours": {
            "monday": {"start": "08:00", "end": "18:00"},
            "tuesday": {"start": "08:00", "end": "18:00"},
            "wednesday": {"start": "08:00", "end": "18:00"},
            "thursday": {"start": "08:00", "end": "18:00"},
            "friday": {"start": "08:00", "end": "18:00"},
            "saturday": {"start": "09:00", "end": "15:00"},
            "sunday": {"closed": true}
        },
        "crm_type": "hubspot",
        "email_notifications": true,
        "auto_assignment": true
    }',
    PENDING_COMMIT_TIMESTAMP(),
    PENDING_COMMIT_TIMESTAMP()
);
```

### Step 2: Create Office Configuration

```sql
INSERT INTO offices (
    office_id,
    tenant_id,
    name,
    callrail_company_id,
    callrail_api_key,
    workflow_config,
    service_area,
    status,
    created_at,
    updated_at
) VALUES (
    'office_acme_main',
    'tenant_acme_remodeling',
    'ACME Main Office',
    '67890',  -- CallRail company ID
    'sk_live_abc123xyz789',  -- CallRail API key
    JSON '{
        "lead_routing": "round_robin",
        "qualification_required": true,
        "appointment_booking": true,
        "auto_follow_up": true,
        "follow_up_delay_minutes": 15,
        "business_hours_only": true,
        "spam_filter_enabled": true,
        "minimum_call_duration": 30
    }',
    JSON '{
        "primary_city": "Los Angeles",
        "cities": [
            "Los Angeles",
            "Beverly Hills",
            "Santa Monica",
            "West Hollywood",
            "Culver City"
        ],
        "zip_codes": [
            "90210", "90211", "90212", "90213",
            "90401", "90402", "90403",
            "90028", "90046", "90048"
        ],
        "radius_miles": 25,
        "exclude_areas": ["Downtown LA", "South LA"]
    }',
    'active',
    PENDING_COMMIT_TIMESTAMP(),
    PENDING_COMMIT_TIMESTAMP()
);
```

### Step 3: Set Up Secure Credentials

```bash
# Store CallRail API key securely
echo -n "sk_live_abc123xyz789" | \
    gcloud secrets create callrail-api-key-tenant-acme \
    --data-file=-

# Store webhook secret
echo -n "webhook_secret_acme_xyz123" | \
    gcloud secrets create webhook-secret-tenant-acme \
    --data-file=-

# Store CRM credentials (example for HubSpot)
echo -n "pat-na1-abc123-xyz789" | \
    gcloud secrets create hubspot-token-tenant-acme \
    --data-file=-
```

## CallRail Integration

### Step 1: Configure CallRail Webhook

#### In CallRail Dashboard:
1. Navigate to **Integrations** → **Webhooks**
2. Click **Add Webhook**
3. Configure webhook settings:

```json
{
  "name": "ACME Remodeling Pipeline Webhook",
  "webhook_url": "https://api.pipeline.com/v1/callrail/webhook",
  "events": ["call_completed"],
  "signature_token": "webhook_secret_acme_xyz123",
  "custom_fields": {
    "tenant_id": "tenant_acme_remodeling",
    "callrail_company_id": "67890"
  }
}
```

#### Test Webhook Configuration:
```bash
# CallRail provides a test webhook feature
# Use this payload to test:
{
  "call_id": "TEST123456",
  "account_id": "AC123456",
  "company_id": "67890",
  "caller_id": "+15551234567",
  "called_number": "+15559876543",
  "duration": "180",
  "start_time": "2025-09-13T10:30:00Z",
  "end_time": "2025-09-13T10:33:00Z",
  "direction": "inbound",
  "answered": true,
  "tenant_id": "tenant_acme_remodeling",
  "callrail_company_id": "67890"
}
```

### Step 2: Verify Integration

```bash
# Check webhook processing logs
gcloud logging read \
  "resource.type=cloud_run_revision AND
   jsonPayload.tenant_id=tenant_acme_remodeling AND
   jsonPayload.source=callrail_webhook" \
  --limit=10

# Query processed requests
gcloud spanner databases execute-sql pipeline-db \
  --instance=ingestion-db \
  --sql="
    SELECT request_id, status, created_at, lead_score
    FROM requests
    WHERE tenant_id = 'tenant_acme_remodeling'
    ORDER BY created_at DESC
    LIMIT 5"
```

## CRM Configuration

### Step 1: Configure CRM Integration

#### HubSpot Example:
```json
{
  "crm_config": {
    "type": "hubspot",
    "api_token": "secret:projects/PROJECT_ID/secrets/hubspot-token-tenant-acme/versions/latest",
    "pipeline_id": "default",
    "deal_stage": "appointmentscheduled",
    "contact_owner": "user@acmeremodeling.com",
    "field_mappings": {
      "phone": "phone",
      "email": "email",
      "firstname": "customer_name",
      "lastname": "customer_name",
      "city": "customer_city",
      "state": "customer_state",
      "lead_source": "source",
      "project_type": "ai_analysis.project_type",
      "timeline": "ai_analysis.timeline",
      "budget": "ai_analysis.budget_indicator",
      "lead_score": "ai_analysis.lead_score"
    },
    "custom_properties": [
      {
        "name": "call_recording_url",
        "value": "recording_url"
      },
      {
        "name": "call_transcript",
        "value": "transcription"
      },
      {
        "name": "ai_intent",
        "value": "ai_analysis.intent"
      }
    ]
  }
}
```

#### Salesforce Example:
```json
{
  "crm_config": {
    "type": "salesforce",
    "instance_url": "https://acmeremodeling.salesforce.com",
    "client_id": "3MVG9A2kN3Bn17hsbc123",
    "client_secret": "secret:projects/PROJECT_ID/secrets/sf-client-secret-tenant-acme/versions/latest",
    "username": "api@acmeremodeling.com",
    "password": "secret:projects/PROJECT_ID/secrets/sf-password-tenant-acme/versions/latest",
    "object_mappings": {
      "lead": {
        "Phone": "phone",
        "Email": "email",
        "FirstName": "customer_name",
        "LastName": "customer_name",
        "City": "customer_city",
        "State": "customer_state",
        "LeadSource": "source",
        "Rating": "ai_analysis.lead_score"
      }
    }
  }
}
```

### Step 2: Update Tenant Configuration

```sql
UPDATE tenants
SET configuration = JSON_SET(
    configuration,
    '$.crm_config',
    JSON '{
        "type": "hubspot",
        "api_token": "secret:projects/PROJECT_ID/secrets/hubspot-token-tenant-acme/versions/latest",
        "pipeline_id": "default",
        "field_mappings": {
            "phone": "phone",
            "email": "email",
            "firstname": "customer_name",
            "city": "customer_city",
            "state": "customer_state",
            "lead_source": "source"
        }
    }'
)
WHERE tenant_id = 'tenant_acme_remodeling';
```

## Testing and Verification

### Step 1: End-to-End Test

#### Simulate CallRail Webhook:
```bash
# Create test webhook payload
cat > test_webhook.json << EOF
{
  "call_id": "TEST_ACME_001",
  "account_id": "AC123456",
  "company_id": "67890",
  "caller_id": "+15551234567",
  "called_number": "+15559876543",
  "duration": "180",
  "start_time": "2025-09-13T10:30:00Z",
  "end_time": "2025-09-13T10:33:00Z",
  "direction": "inbound",
  "recording_url": "https://api.callrail.com/v3/a/AC123456/calls/TEST_ACME_001/recording.json",
  "answered": true,
  "customer_name": "John Smith",
  "customer_phone_number": "+15551234567",
  "customer_city": "Los Angeles",
  "customer_state": "CA",
  "lead_status": "good_lead",
  "tenant_id": "tenant_acme_remodeling",
  "callrail_company_id": "67890"
}
EOF

# Generate HMAC signature
SIGNATURE=$(echo -n "$(cat test_webhook.json)" | \
    openssl dgst -sha256 -hmac "webhook_secret_acme_xyz123" -binary | \
    base64)

# Send test webhook
curl -X POST https://api.pipeline.com/v1/callrail/webhook \
  -H "Content-Type: application/json" \
  -H "x-callrail-signature: sha256=$SIGNATURE" \
  -d @test_webhook.json
```

### Step 2: Verify Processing

#### Check Request Processing:
```sql
SELECT
    request_id,
    status,
    source,
    ai_analysis,
    lead_score,
    created_at
FROM requests
WHERE tenant_id = 'tenant_acme_remodeling'
  AND call_id = 'TEST_ACME_001';
```

#### Check Workflow Execution:
```sql
SELECT
    execution_id,
    workflow_name,
    status,
    steps,
    error_details
FROM workflow_executions
WHERE tenant_id = 'tenant_acme_remodeling'
ORDER BY started_at DESC
LIMIT 5;
```

### Step 3: Verify CRM Integration

#### Check CRM Push Logs:
```bash
gcloud logging read \
  "resource.type=cloud_run_revision AND
   jsonPayload.tenant_id=tenant_acme_remodeling AND
   jsonPayload.workflow_step=crm_push" \
  --limit=10
```

#### Manual CRM Verification:
1. Log into tenant's CRM system
2. Search for contact created from test call
3. Verify all fields populated correctly
4. Check custom properties and call data

## Common Issues

### Issue 1: Webhook Signature Verification Fails

**Symptoms:**
- 401 Unauthorized responses
- "Invalid signature" errors in logs

**Solutions:**
```bash
# Verify webhook secret is correct
gcloud secrets versions access latest --secret=webhook-secret-tenant-acme

# Check signature generation in CallRail
# Ensure webhook URL is exactly: https://api.pipeline.com/v1/callrail/webhook

# Test signature generation locally
echo -n '{"test":"payload"}' | \
  openssl dgst -sha256 -hmac "your_webhook_secret" -binary | \
  base64
```

### Issue 2: Tenant/CallRail Mapping Not Found

**Symptoms:**
- "Invalid tenant_id or callrail_company_id mapping" errors
- Requests not processed

**Solutions:**
```sql
-- Verify office record exists
SELECT * FROM offices
WHERE tenant_id = 'tenant_acme_remodeling'
  AND callrail_company_id = '67890';

-- Check webhook payload includes correct IDs
-- Ensure CallRail custom fields are configured
```

### Issue 3: CRM Integration Fails

**Symptoms:**
- Leads not appearing in CRM
- "CRM push failed" in workflow logs

**Solutions:**
```bash
# Test CRM credentials
curl -X GET "https://api.hubapi.com/contacts/v1/lists/all/contacts/all" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Check secret manager access
gcloud secrets versions access latest --secret=hubspot-token-tenant-acme

# Verify field mappings in tenant configuration
```

### Issue 4: Audio Transcription Fails

**Symptoms:**
- Empty transcription fields
- Speech API errors in logs

**Solutions:**
```bash
# Check Speech API quota
gcloud logging read "protoPayload.serviceName=speech.googleapis.com" --limit=10

# Verify audio file storage
gsutil ls gs://PROJECT_ID-audio-files/tenant_acme_remodeling/calls/

# Test manual transcription
gcloud ml speech recognize \
  --include-word-time-offsets \
  --language-code=en-US \
  gs://PROJECT_ID-audio-files/tenant_acme_remodeling/calls/TEST_ACME_001.mp3
```

## Post-Onboarding Checklist

### ✅ Verification Checklist
- [ ] Tenant created in database
- [ ] Office configuration complete
- [ ] CallRail webhook configured and tested
- [ ] CRM integration working
- [ ] Test call processed end-to-end
- [ ] Lead appears in CRM with correct data
- [ ] Audio recording stored and transcribed
- [ ] AI analysis completed
- [ ] Notifications sent to correct recipients
- [ ] Dashboard access configured

### 📋 Documentation to Provide
- [ ] Webhook endpoint URL
- [ ] Tenant ID for reference
- [ ] Dashboard login credentials
- [ ] Support contact information
- [ ] Escalation procedures

### 🔄 Ongoing Monitoring
- [ ] Set up alerting for webhook failures
- [ ] Monitor CRM integration health
- [ ] Review lead quality scores weekly
- [ ] Check audio storage costs monthly
- [ ] Validate transcription accuracy

The tenant onboarding process is now complete! The new tenant should start receiving and processing calls automatically through the pipeline.