# CRM Integration Setup Guide

## Table of Contents
- [Overview](#overview)
- [Supported CRM Systems](#supported-crm-systems)
- [HubSpot Integration](#hubspot-integration)
- [Salesforce Integration](#salesforce-integration)
- [Pipedrive Integration](#pipedrive-integration)
- [Custom CRM Integration](#custom-crm-integration)
- [Field Mapping Configuration](#field-mapping-configuration)
- [Testing and Troubleshooting](#testing-and-troubleshooting)

## Overview

The multi-tenant ingestion pipeline supports seamless integration with popular CRM systems, automatically pushing enriched lead data from phone calls and form submissions directly into your CRM workflow.

### Key Features
- **Automatic Lead Creation**: Contacts and deals created automatically
- **Rich Data Mapping**: Phone call transcripts, AI analysis, and lead scores
- **Real-time Sync**: Leads appear in CRM within seconds
- **Custom Field Support**: Map any call data to CRM custom properties
- **Duplicate Detection**: Smart matching prevents duplicate contacts
- **Workflow Triggers**: Configurable actions based on lead quality

### Data Flow
```
CallRail Webhook → AI Analysis → CRM Integration
       ↓
   - Call Details
   - Audio Transcript
   - AI Intent Analysis
   - Lead Quality Score
   - Project Type
   - Timeline/Urgency
```

## Supported CRM Systems

| CRM System | API Version | Authentication | Features |
|------------|-------------|----------------|----------|
| HubSpot | v3 | Private App Token | Full integration |
| Salesforce | v54.0 | OAuth 2.0 / JWT | Full integration |
| Pipedrive | v1 | API Token | Full integration |
| Monday.com | 2024-01 | API Token | Basic integration |
| Zoho CRM | v2 | OAuth 2.0 | Basic integration |
| Custom REST API | Any | Configurable | Custom fields |

## HubSpot Integration

### Step 1: Create HubSpot Private App

1. **Access Developer Settings**:
   - Go to HubSpot account → Settings → Integrations → Private Apps
   - Click "Create a private app"

2. **Configure App Settings**:
   ```json
   {
     "name": "Ingestion Pipeline Integration",
     "description": "Automated lead ingestion from phone calls and forms"
   }
   ```

3. **Set Required Scopes**:
   ```
   crm.objects.contacts.read
   crm.objects.contacts.write
   crm.objects.deals.read
   crm.objects.deals.write
   crm.objects.companies.read
   crm.objects.companies.write
   crm.schemas.contacts.read
   crm.schemas.deals.read
   ```

4. **Generate Access Token**:
   - Copy the private app access token
   - Store securely in Secret Manager

### Step 2: Configure Integration

#### Store HubSpot Credentials:
```bash
# Store access token
echo -n "pat-na1-your-token-here" | \
    gcloud secrets create hubspot-token-tenant-${TENANT_ID} \
    --data-file=-

# Store portal ID (optional)
echo -n "12345678" | \
    gcloud secrets create hubspot-portal-tenant-${TENANT_ID} \
    --data-file=-
```

#### Update Tenant Configuration:
```sql
UPDATE tenants
SET configuration = JSON_SET(
    configuration,
    '$.crm_config',
    JSON '{
        "type": "hubspot",
        "api_token": "secret:projects/PROJECT_ID/secrets/hubspot-token-tenant-TENANT_ID/versions/latest",
        "portal_id": "12345678",
        "pipeline_id": "default",
        "deal_stage": "appointmentscheduled",
        "contact_owner": "owner@company.com",
        "company_domain_mapping": true,
        "duplicate_detection": {
            "enabled": true,
            "match_fields": ["phone", "email"]
        },
        "field_mappings": {
            "phone": "phone",
            "email": "email",
            "firstname": "customer_name_first",
            "lastname": "customer_name_last",
            "city": "customer_city",
            "state": "customer_state",
            "lead_source": "source",
            "lead_source_detail": "call_id"
        },
        "custom_properties": [
            {
                "name": "call_recording_url",
                "value": "recording_url",
                "type": "string"
            },
            {
                "name": "call_transcript",
                "value": "transcription",
                "type": "string"
            },
            {
                "name": "ai_project_type",
                "value": "ai_analysis.project_type",
                "type": "enumeration"
            },
            {
                "name": "ai_timeline",
                "value": "ai_analysis.timeline",
                "type": "enumeration"
            },
            {
                "name": "ai_lead_score",
                "value": "ai_analysis.lead_score",
                "type": "number"
            },
            {
                "name": "call_duration",
                "value": "call_details.duration",
                "type": "number"
            }
        ],
        "workflow_triggers": {
            "high_value_lead": {
                "condition": "ai_analysis.lead_score > 80",
                "actions": ["send_notification", "assign_owner"]
            },
            "immediate_timeline": {
                "condition": "ai_analysis.timeline == 'immediate'",
                "actions": ["create_task", "send_sms"]
            }
        }
    }'
)
WHERE tenant_id = 'tenant_TENANT_ID';
```

### Step 3: Create Custom Properties in HubSpot

Create these custom properties in HubSpot to store call data:

```javascript
// Use HubSpot API to create custom properties
const properties = [
    {
        "name": "call_recording_url",
        "label": "Call Recording URL",
        "type": "string",
        "fieldType": "text",
        "groupName": "call_information"
    },
    {
        "name": "call_transcript",
        "label": "Call Transcript",
        "type": "string",
        "fieldType": "textarea",
        "groupName": "call_information"
    },
    {
        "name": "ai_project_type",
        "label": "AI Project Type",
        "type": "enumeration",
        "fieldType": "select",
        "options": [
            {"label": "Kitchen", "value": "kitchen"},
            {"label": "Bathroom", "value": "bathroom"},
            {"label": "Whole Home", "value": "whole_home"},
            {"label": "Addition", "value": "addition"},
            {"label": "Unknown", "value": "unknown"}
        ],
        "groupName": "call_information"
    },
    {
        "name": "ai_timeline",
        "label": "Project Timeline",
        "type": "enumeration",
        "fieldType": "select",
        "options": [
            {"label": "Immediate", "value": "immediate"},
            {"label": "1-3 Months", "value": "1-3_months"},
            {"label": "3-6 Months", "value": "3-6_months"},
            {"label": "6+ Months", "value": "6+_months"},
            {"label": "Unknown", "value": "unknown"}
        ],
        "groupName": "call_information"
    },
    {
        "name": "ai_lead_score",
        "label": "AI Lead Score",
        "type": "number",
        "fieldType": "number",
        "groupName": "call_information"
    }
];

// Create properties via API
properties.forEach(property => {
    fetch('https://api.hubapi.com/crm/v3/properties/contacts', {
        method: 'POST',
        headers: {
            'Authorization': 'Bearer YOUR_ACCESS_TOKEN',
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(property)
    });
});
```

## Salesforce Integration

### Step 1: Set Up Salesforce Connected App

1. **Create Connected App**:
   - Setup → Apps → App Manager → New Connected App
   - Enable OAuth Settings
   - Add scopes: `api`, `refresh_token`, `offline_access`

2. **Generate Certificates** (for JWT Bearer Flow):
   ```bash
   # Generate private key
   openssl genrsa -out salesforce_private_key.pem 2048

   # Generate certificate
   openssl req -new -x509 -key salesforce_private_key.pem -out salesforce_cert.crt -days 365
   ```

3. **Configure JWT Bearer Flow**:
   - Upload certificate to Connected App
   - Enable "Use digital signatures"
   - Set callback URL (not needed for JWT)

### Step 2: Store Salesforce Credentials

```bash
# Store private key
cat salesforce_private_key.pem | \
    gcloud secrets create sf-private-key-tenant-${TENANT_ID} \
    --data-file=-

# Store consumer key (Client ID)
echo -n "3MVG9A2kN3Bn17hs..." | \
    gcloud secrets create sf-consumer-key-tenant-${TENANT_ID} \
    --data-file=-

# Store username
echo -n "api@company.com" | \
    gcloud secrets create sf-username-tenant-${TENANT_ID} \
    --data-file=-
```

### Step 3: Configure Salesforce Integration

```sql
UPDATE tenants
SET configuration = JSON_SET(
    configuration,
    '$.crm_config',
    JSON '{
        "type": "salesforce",
        "instance_url": "https://company.my.salesforce.com",
        "consumer_key": "secret:projects/PROJECT_ID/secrets/sf-consumer-key-tenant-TENANT_ID/versions/latest",
        "private_key": "secret:projects/PROJECT_ID/secrets/sf-private-key-tenant-TENANT_ID/versions/latest",
        "username": "secret:projects/PROJECT_ID/secrets/sf-username-tenant-TENANT_ID/versions/latest",
        "api_version": "v54.0",
        "object_mappings": {
            "lead": {
                "Phone": "customer_phone",
                "Email": "customer_email",
                "FirstName": "customer_name_first",
                "LastName": "customer_name_last",
                "City": "customer_city",
                "State": "customer_state",
                "LeadSource": "source",
                "Rating": "ai_analysis.budget_indicator",
                "Call_Recording_URL__c": "recording_url",
                "Call_Transcript__c": "transcription",
                "AI_Project_Type__c": "ai_analysis.project_type",
                "AI_Timeline__c": "ai_analysis.timeline",
                "AI_Lead_Score__c": "ai_analysis.lead_score"
            },
            "task": {
                "Subject": "Follow up on call",
                "Priority": "Normal",
                "Status": "Open",
                "ActivityDate": "computed:today+1",
                "Description": "transcription"
            }
        },
        "workflow_rules": {
            "high_score_lead": {
                "condition": "ai_analysis.lead_score > 85",
                "actions": ["create_task", "assign_to_top_rep"]
            }
        }
    }'
)
WHERE tenant_id = 'tenant_TENANT_ID';
```

### Step 4: Create Custom Fields in Salesforce

```sql
-- Create custom fields on Lead object via Salesforce Setup
-- Or use Metadata API:

<?xml version="1.0" encoding="UTF-8"?>
<CustomField xmlns="http://soap.sforce.com/2006/04/metadata">
    <fullName>Call_Recording_URL__c</fullName>
    <description>URL to the call recording</description>
    <label>Call Recording URL</label>
    <length>500</length>
    <required>false</required>
    <trackFeedHistory>false</trackFeedHistory>
    <type>Url</type>
</CustomField>

<CustomField xmlns="http://soap.sforce.com/2006/04/metadata">
    <fullName>AI_Lead_Score__c</fullName>
    <description>AI-generated lead quality score</description>
    <label>AI Lead Score</label>
    <precision>3</precision>
    <required>false</required>
    <scale>0</scale>
    <trackFeedHistory>false</trackFeedHistory>
    <type>Number</type>
</CustomField>

<CustomField xmlns="http://soap.sforce.com/2006/04/metadata">
    <fullName>AI_Project_Type__c</fullName>
    <description>AI-detected project type</description>
    <label>AI Project Type</label>
    <required>false</required>
    <trackFeedHistory>false</trackFeedHistory>
    <type>Picklist</type>
    <valueSet>
        <valueSetDefinition>
            <value><fullName>Kitchen</fullName><default>false</default></value>
            <value><fullName>Bathroom</fullName><default>false</default></value>
            <value><fullName>Whole Home</fullName><default>false</default></value>
            <value><fullName>Addition</fullName><default>false</default></value>
            <value><fullName>Unknown</fullName><default>false</default></value>
        </valueSetDefinition>
    </valueSet>
</CustomField>
```

## Pipedrive Integration

### Step 1: Generate API Token

1. **Get API Token**:
   - Pipedrive → Settings → Personal → API
   - Generate new token
   - Copy token value

### Step 2: Configure Pipedrive Integration

```bash
# Store API token
echo -n "your-pipedrive-api-token" | \
    gcloud secrets create pipedrive-token-tenant-${TENANT_ID} \
    --data-file=-
```

```sql
UPDATE tenants
SET configuration = JSON_SET(
    configuration,
    '$.crm_config',
    JSON '{
        "type": "pipedrive",
        "api_token": "secret:projects/PROJECT_ID/secrets/pipedrive-token-tenant-TENANT_ID/versions/latest",
        "company_domain": "company-name.pipedrive.com",
        "pipeline_id": 1,
        "stage_id": 2,
        "default_owner": 123456,
        "field_mappings": {
            "name": "customer_name",
            "phone": "customer_phone",
            "email": "customer_email",
            "address_locality": "customer_city",
            "address_admin_area_level_1": "customer_state"
        },
        "custom_fields": {
            "call_recording_url": "recording_url",
            "call_transcript": "transcription",
            "ai_project_type": "ai_analysis.project_type",
            "ai_lead_score": "ai_analysis.lead_score"
        },
        "deal_settings": {
            "title_template": "{{customer_name}} - {{ai_analysis.project_type}} Project",
            "currency": "USD",
            "value_estimation": {
                "kitchen": 25000,
                "bathroom": 15000,
                "whole_home": 100000,
                "addition": 50000,
                "default": 20000
            }
        }
    }'
)
WHERE tenant_id = 'tenant_TENANT_ID';
```

## Custom CRM Integration

### Step 1: Define Custom API Configuration

For CRMs not directly supported, configure a custom REST API integration:

```sql
UPDATE tenants
SET configuration = JSON_SET(
    configuration,
    '$.crm_config',
    JSON '{
        "type": "custom",
        "api_base_url": "https://api.yourcrm.com/v1",
        "authentication": {
            "type": "bearer",
            "token": "secret:projects/PROJECT_ID/secrets/custom-crm-token-tenant-TENANT_ID/versions/latest"
        },
        "endpoints": {
            "create_contact": {
                "method": "POST",
                "path": "/contacts",
                "headers": {
                    "Content-Type": "application/json",
                    "Authorization": "Bearer {{auth.token}}"
                }
            },
            "create_opportunity": {
                "method": "POST",
                "path": "/opportunities",
                "headers": {
                    "Content-Type": "application/json",
                    "Authorization": "Bearer {{auth.token}}"
                }
            }
        },
        "payload_templates": {
            "contact": {
                "first_name": "{{customer_name_first}}",
                "last_name": "{{customer_name_last}}",
                "phone": "{{customer_phone}}",
                "email": "{{customer_email}}",
                "address": {
                    "city": "{{customer_city}}",
                    "state": "{{customer_state}}"
                },
                "custom_fields": {
                    "lead_source": "phone_call",
                    "call_recording": "{{recording_url}}",
                    "ai_score": "{{ai_analysis.lead_score}}"
                }
            }
        }
    }'
)
WHERE tenant_id = 'tenant_TENANT_ID';
```

## Field Mapping Configuration

### Standard Field Mappings

| Pipeline Field | HubSpot | Salesforce | Pipedrive | Description |
|----------------|---------|------------|-----------|-------------|
| `customer_name` | `firstname`, `lastname` | `FirstName`, `LastName` | `name` | Customer full name |
| `customer_phone` | `phone` | `Phone` | `phone` | Primary phone number |
| `customer_email` | `email` | `Email` | `email` | Email address |
| `customer_city` | `city` | `City` | `address_locality` | Customer city |
| `customer_state` | `state` | `State` | `address_admin_area_level_1` | Customer state/region |
| `source` | `hs_lead_source` | `LeadSource` | `lead_source` | Lead source |

### AI Analysis Field Mappings

| Pipeline Field | Type | Possible Values | Description |
|----------------|------|-----------------|-------------|
| `ai_analysis.intent` | Enumeration | `quote_request`, `information_seeking`, `appointment_booking` | Customer intent |
| `ai_analysis.project_type` | Enumeration | `kitchen`, `bathroom`, `whole_home`, `addition` | Type of project |
| `ai_analysis.timeline` | Enumeration | `immediate`, `1-3_months`, `3-6_months`, `6+_months` | Project timeline |
| `ai_analysis.budget_indicator` | Enumeration | `high`, `medium`, `low`, `unknown` | Budget indication |
| `ai_analysis.sentiment` | Enumeration | `positive`, `neutral`, `negative` | Customer sentiment |
| `ai_analysis.lead_score` | Number | 0-100 | AI-generated lead quality score |
| `ai_analysis.urgency` | Enumeration | `high`, `medium`, `low` | Urgency level |

### Advanced Mapping Examples

#### Conditional Mapping:
```json
{
  "field_mappings": {
    "priority": {
      "type": "conditional",
      "conditions": [
        {
          "if": "ai_analysis.lead_score > 80",
          "then": "High"
        },
        {
          "if": "ai_analysis.urgency == 'high'",
          "then": "High"
        },
        {
          "default": "Normal"
        }
      ]
    }
  }
}
```

#### Computed Fields:
```json
{
  "field_mappings": {
    "estimated_value": {
      "type": "computed",
      "formula": "project_value_map[ai_analysis.project_type] || 20000"
    },
    "follow_up_date": {
      "type": "computed",
      "formula": "addDays(today(), ai_analysis.timeline == 'immediate' ? 1 : 7)"
    }
  }
}
```

## Testing and Troubleshooting

### Test CRM Integration

#### 1. Test API Connectivity:
```bash
# HubSpot test
curl -X GET "https://api.hubapi.com/crm/v3/objects/contacts?limit=1" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Salesforce test (after getting access token)
curl -X GET "https://your-instance.salesforce.com/services/data/v54.0/sobjects/" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Pipedrive test
curl -X GET "https://api.pipedrive.com/v1/users/me?api_token=YOUR_API_TOKEN"
```

#### 2. Test Field Mapping:
```bash
# Send test webhook with known data
curl -X POST https://api.pipeline.com/v1/callrail/webhook \
  -H "Content-Type: application/json" \
  -H "x-callrail-signature: sha256=VALID_SIGNATURE" \
  -d '{
    "call_id": "TEST_CRM_001",
    "tenant_id": "tenant_your_company",
    "callrail_company_id": "12345",
    "caller_id": "+15551234567",
    "customer_name": "Jane Doe",
    "customer_city": "Los Angeles",
    "customer_state": "CA",
    "duration": "180",
    "answered": true
  }'

# Check CRM for created contact
# Verify all fields mapped correctly
```

### Common Issues and Solutions

#### Issue 1: Authentication Failures

**HubSpot "Unauthorized" Error:**
```bash
# Check token validity
curl -X GET "https://api.hubapi.com/oauth/v1/access-tokens/YOUR_TOKEN"

# Verify scopes
curl -X GET "https://api.hubapi.com/oauth/v1/access-tokens/YOUR_TOKEN" | \
  jq '.scopes'
```

**Salesforce JWT Error:**
```bash
# Verify connected app configuration
# Check certificate upload
# Validate username in JWT payload

# Test JWT generation manually
python3 -c "
import jwt
import datetime

private_key = open('salesforce_private_key.pem').read()
payload = {
    'iss': 'CONSUMER_KEY',
    'sub': 'USERNAME',
    'aud': 'https://login.salesforce.com',
    'exp': datetime.datetime.utcnow() + datetime.timedelta(minutes=5)
}
token = jwt.encode(payload, private_key, algorithm='RS256')
print(token)
"
```

#### Issue 2: Field Mapping Errors

**Missing Custom Fields:**
```bash
# Check if custom fields exist in CRM
# For HubSpot:
curl -X GET "https://api.hubapi.com/crm/v3/properties/contacts" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" | \
  jq '.results[] | select(.name | contains("call_"))'

# For Salesforce:
curl -X GET "https://your-instance.salesforce.com/services/data/v54.0/sobjects/Lead/describe/" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" | \
  jq '.fields[] | select(.name | contains("Call_"))'
```

**Data Type Mismatches:**
```json
{
  "error": "Invalid value for field ai_lead_score: expected number, got string",
  "solution": "Ensure AI analysis returns numeric values for score fields"
}
```

#### Issue 3: Duplicate Detection Issues

**Duplicates Being Created:**
```sql
-- Check CRM configuration for duplicate detection
UPDATE tenants
SET configuration = JSON_SET(
    configuration,
    '$.crm_config.duplicate_detection',
    JSON '{
        "enabled": true,
        "match_fields": ["phone", "email"],
        "match_strategy": "exact",
        "action": "update_existing"
    }'
)
WHERE tenant_id = 'tenant_your_company';
```

### Monitoring CRM Integration Health

#### Key Metrics to Monitor:
```bash
# CRM push success rate
gcloud logging read \
  "resource.type=cloud_run_revision AND
   jsonPayload.workflow_step=crm_push AND
   jsonPayload.status=success" \
  --limit=100

# CRM push failures
gcloud logging read \
  "resource.type=cloud_run_revision AND
   jsonPayload.workflow_step=crm_push AND
   severity>=ERROR" \
  --limit=50

# Field mapping errors
gcloud logging read \
  "resource.type=cloud_run_revision AND
   jsonPayload.error_type=field_mapping_error" \
  --limit=20
```

#### Set Up Alerting:
```yaml
# Cloud Monitoring alert policy
displayName: "CRM Integration Failure Rate"
conditions:
  - displayName: "CRM push failure rate > 5%"
    conditionThreshold:
      filter: 'resource.type="cloud_run_revision" jsonPayload.workflow_step="crm_push"'
      comparison: COMPARISON_GREATER_THAN
      thresholdValue: 0.05
notificationChannels:
  - "projects/PROJECT_ID/notificationChannels/CHANNEL_ID"
```

Your CRM integration is now fully configured and ready to automatically push enriched lead data from phone calls and forms directly into your sales workflow!