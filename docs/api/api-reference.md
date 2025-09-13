# API Reference Guide

**Multi-Tenant Ingestion Pipeline**

---

**Version**: 2.0
**Date**: September 13, 2025
**Base URL**: `https://api.yourcompany.com/v1`
**OpenAPI Spec**: [openapi.yaml](openapi.yaml)

---

## Table of Contents

1. [Authentication](#authentication)
2. [Webhook Endpoints](#webhook-endpoints)
3. [Management API](#management-api)
4. [Tenant Administration](#tenant-administration)
5. [Monitoring & Health](#monitoring--health)
6. [Error Handling](#error-handling)
7. [Rate Limiting](#rate-limiting)
8. [SDK Examples](#sdk-examples)

---

## Authentication

The pipeline supports multiple authentication methods depending on the endpoint type:

### Webhook Authentication (HMAC)

CallRail webhooks use HMAC-SHA256 signature verification:

```http
POST /v1/callrail/webhook
Content-Type: application/json
x-callrail-signature: sha256=1a2b3c4d5e6f...
x-timestamp: 1694617200

{
  "call_id": "CAL123456789",
  "tenant_id": "tenant_001"
}
```

**Signature Calculation:**
```python
import hmac
import hashlib

def calculate_signature(payload, secret, timestamp):
    """Calculate CallRail webhook signature"""
    message = f"{timestamp}.{payload}"
    signature = hmac.new(
        secret.encode('utf-8'),
        message.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()
    return f"sha256={signature}"
```

### Management API Authentication (JWT)

Administrative endpoints require JWT tokens:

```http
GET /v1/tenants/tenant_001/requests
Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9...
```

**Token Request:**
```bash
curl -X POST https://api.yourcompany.com/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "your_client_id",
    "client_secret": "your_client_secret",
    "tenant_id": "tenant_001"
  }'
```

**Response:**
```json
{
  "access_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```

---

## Webhook Endpoints

### CallRail Webhook

Receives and processes CallRail webhook events.

#### Endpoint
```http
POST /v1/callrail/webhook
```

#### Headers
| Header | Required | Description |
|--------|----------|-------------|
| `Content-Type` | Yes | Must be `application/json` |
| `x-callrail-signature` | Yes | HMAC-SHA256 signature |
| `x-timestamp` | Yes | Unix timestamp |
| `User-Agent` | No | CallRail webhook user agent |

#### Request Body

```json
{
  "call_id": "CAL123456789",
  "caller_id": "+15551234567",
  "caller_name": "John Smith",
  "duration": "180",
  "answered": true,
  "start_time": "2025-09-13T14:30:00Z",
  "end_time": "2025-09-13T14:33:00Z",
  "recording_url": "https://callrail.com/recordings/CAL123456789.wav",
  "recording_duration": "175",
  "tracking_phone_number": "+15559876543",
  "business_phone_number": "+15555551234",
  "tags": ["bathroom_remodel", "high_intent"],
  "lead_status": "good_lead",
  "value": "potential",
  "company_id": "12345",
  "tenant_id": "tenant_001",
  "custom_fields": {
    "project_type": "bathroom",
    "budget_range": "10000-25000",
    "timeline": "1-3_months"
  }
}
```

#### Response

**Success (200 OK):**
```json
{
  "success": true,
  "request_id": "req_98765432",
  "processing_time_ms": 1250,
  "message": "Call queued for processing",
  "estimated_completion": "2025-09-13T14:35:00Z"
}
```

**Validation Error (400 Bad Request):**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request payload",
    "details": [
      {
        "field": "call_id",
        "issue": "required field missing"
      }
    ]
  },
  "request_id": "req_error_123"
}
```

**Authentication Error (401 Unauthorized):**
```json
{
  "success": false,
  "error": {
    "code": "INVALID_SIGNATURE",
    "message": "Webhook signature verification failed",
    "timestamp": "2025-09-13T14:30:00Z"
  }
}
```

#### cURL Example

```bash
# Calculate signature first
PAYLOAD='{"call_id":"CAL123","tenant_id":"tenant_001"}'
TIMESTAMP=$(date +%s)
SECRET="your_webhook_secret"

# Create signature (in production, use proper HMAC calculation)
SIGNATURE=$(echo -n "${TIMESTAMP}.${PAYLOAD}" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')

curl -X POST https://api.yourcompany.com/v1/callrail/webhook \
  -H "Content-Type: application/json" \
  -H "x-callrail-signature: sha256=${SIGNATURE}" \
  -H "x-timestamp: ${TIMESTAMP}" \
  -d "$PAYLOAD"
```

### Direct API Webhook

Alternative endpoint for direct API submissions (non-CallRail).

#### Endpoint
```http
POST /v1/leads/webhook
```

#### Authentication
Requires API key in header:
```http
x-api-key: your_api_key_here
```

#### Request Body
```json
{
  "tenant_id": "tenant_001",
  "source": "website_form",
  "customer_name": "Jane Doe",
  "customer_phone": "+15551234567",
  "customer_email": "jane@example.com",
  "project_type": "kitchen_remodel",
  "project_description": "Looking to remodel kitchen, approximately 200 sq ft",
  "budget_range": "25000-50000",
  "timeline": "3-6_months",
  "contact_preferences": ["phone", "email"],
  "custom_fields": {
    "source_campaign": "google_ads_kitchen",
    "referral_source": "google",
    "utm_campaign": "spring_kitchen_promo"
  }
}
```

---

## Management API

Administrative endpoints for managing requests, tenants, and system configuration.

### List Requests

Retrieve processing requests for a tenant.

#### Endpoint
```http
GET /v1/tenants/{tenant_id}/requests
```

#### Query Parameters
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 20 | Number of results (1-100) |
| `offset` | integer | 0 | Pagination offset |
| `status` | string | all | Filter by status: `pending`, `processing`, `completed`, `failed` |
| `source` | string | all | Filter by source: `callrail`, `webform`, `api` |
| `start_date` | string | - | ISO 8601 date (YYYY-MM-DD) |
| `end_date` | string | - | ISO 8601 date (YYYY-MM-DD) |
| `sort` | string | created_at | Sort field: `created_at`, `updated_at`, `processing_time` |
| `order` | string | desc | Sort order: `asc`, `desc` |

#### Example Request
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  "https://api.yourcompany.com/v1/tenants/tenant_001/requests?limit=10&status=completed&source=callrail"
```

#### Response
```json
{
  "success": true,
  "data": {
    "requests": [
      {
        "request_id": "req_98765432",
        "tenant_id": "tenant_001",
        "source_type": "callrail",
        "external_id": "CAL123456789",
        "status": "completed",
        "created_at": "2025-09-13T14:30:00Z",
        "updated_at": "2025-09-13T14:33:45Z",
        "processed_at": "2025-09-13T14:33:45Z",
        "processing_time_ms": 225000,
        "call_data": {
          "call_id": "CAL123456789",
          "caller_phone": "+15551234567",
          "duration": 180,
          "recording_url": "gs://bucket/audio/CAL123456789.wav"
        },
        "ai_analysis": {
          "transcript": "Hi, I'm interested in remodeling my kitchen...",
          "intent": "kitchen_remodel",
          "lead_score": 85,
          "confidence": 0.92,
          "project_details": {
            "type": "kitchen",
            "budget_indicator": "medium",
            "timeline": "1-3_months",
            "urgency": "high"
          }
        },
        "crm_integrations": [
          {
            "crm_type": "hubspot",
            "crm_record_id": "12345678",
            "status": "success",
            "sync_time_ms": 850,
            "last_sync_at": "2025-09-13T14:33:40Z"
          }
        ]
      }
    ],
    "pagination": {
      "total": 1250,
      "limit": 10,
      "offset": 0,
      "has_more": true
    }
  },
  "request_id": "req_list_456"
}
```

### Get Request Details

Retrieve detailed information about a specific request.

#### Endpoint
```http
GET /v1/tenants/{tenant_id}/requests/{request_id}
```

#### Response
```json
{
  "success": true,
  "data": {
    "request_id": "req_98765432",
    "tenant_id": "tenant_001",
    "source_type": "callrail",
    "external_id": "CAL123456789",
    "status": "completed",
    "created_at": "2025-09-13T14:30:00Z",
    "updated_at": "2025-09-13T14:33:45Z",
    "processed_at": "2025-09-13T14:33:45Z",
    "processing_steps": [
      {
        "step": "webhook_received",
        "timestamp": "2025-09-13T14:30:00Z",
        "duration_ms": 50,
        "status": "completed"
      },
      {
        "step": "callrail_api_fetch",
        "timestamp": "2025-09-13T14:30:01Z",
        "duration_ms": 1200,
        "status": "completed"
      },
      {
        "step": "audio_download",
        "timestamp": "2025-09-13T14:30:02Z",
        "duration_ms": 2500,
        "status": "completed"
      },
      {
        "step": "speech_to_text",
        "timestamp": "2025-09-13T14:30:05Z",
        "duration_ms": 42000,
        "status": "completed"
      },
      {
        "step": "ai_analysis",
        "timestamp": "2025-09-13T14:30:47Z",
        "duration_ms": 3500,
        "status": "completed"
      },
      {
        "step": "crm_integration",
        "timestamp": "2025-09-13T14:30:51Z",
        "duration_ms": 850,
        "status": "completed"
      }
    ],
    "original_payload": {
      "call_id": "CAL123456789",
      "caller_id": "+15551234567",
      "duration": "180"
    },
    "enriched_data": {
      "callrail_details": {
        "call_id": "CAL123456789",
        "caller_name": "John Smith",
        "caller_location": "Austin, TX",
        "tracking_number": "+15559876543",
        "business_number": "+15555551234",
        "campaign": "Google Ads - Kitchen Remodel",
        "keywords": ["kitchen remodel", "Austin"],
        "gclid": "abc123def456"
      },
      "audio_analysis": {
        "transcript": "Hi, I'm interested in remodeling my kitchen. It's about 200 square feet and I'm looking to do it in the next 3 months. My budget is around $30,000.",
        "speaker_segments": [
          {
            "speaker": "caller",
            "text": "Hi, I'm interested in remodeling my kitchen...",
            "start_time": 0.0,
            "end_time": 15.2
          },
          {
            "speaker": "agent",
            "text": "Great! I'd be happy to help you with that...",
            "start_time": 15.3,
            "end_time": 25.8
          }
        ],
        "sentiment": "positive",
        "confidence": 0.94
      },
      "ai_analysis": {
        "intent": "kitchen_remodel",
        "lead_score": 85,
        "confidence": 0.92,
        "reasoning": "High intent indicated by specific timeline (3 months), defined budget ($30k), and clear project scope (200 sq ft kitchen)",
        "extracted_entities": {
          "project_type": "kitchen_remodel",
          "timeline": "1-3_months",
          "budget_range": "25000-35000",
          "square_footage": "200",
          "urgency": "high",
          "decision_stage": "evaluation"
        },
        "next_actions": [
          "Schedule in-home consultation",
          "Send kitchen portfolio examples",
          "Provide detailed estimate"
        ]
      }
    }
  }
}
```

### Update Request Status

Manually update request status (admin only).

#### Endpoint
```http
PATCH /v1/tenants/{tenant_id}/requests/{request_id}
```

#### Request Body
```json
{
  "status": "failed",
  "error_message": "CRM integration timeout",
  "retry_scheduled": true,
  "retry_at": "2025-09-13T15:00:00Z"
}
```

### Export Request Data

Export request data for compliance or analysis.

#### Endpoint
```http
GET /v1/tenants/{tenant_id}/requests/export
```

#### Query Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| `format` | string | Export format: `json`, `csv`, `xlsx` |
| `start_date` | string | Start date (ISO 8601) |
| `end_date` | string | End date (ISO 8601) |
| `include_pii` | boolean | Include personally identifiable information |
| `include_audio` | boolean | Include audio file URLs |

#### Response
Returns a download URL for the export file:
```json
{
  "success": true,
  "data": {
    "export_id": "exp_789123456",
    "download_url": "https://storage.googleapis.com/exports/tenant_001_export_789123456.json",
    "expires_at": "2025-09-14T14:30:00Z",
    "record_count": 1250,
    "file_size_bytes": 25600000
  }
}
```

---

## Tenant Administration

### Get Tenant Information

#### Endpoint
```http
GET /v1/tenants/{tenant_id}
```

#### Response
```json
{
  "success": true,
  "data": {
    "tenant_id": "tenant_001",
    "name": "Demo Company LLC",
    "status": "active",
    "created_at": "2025-08-01T10:00:00Z",
    "updated_at": "2025-09-13T08:30:00Z",
    "config": {
      "callrail_company_id": "12345",
      "webhook_secret": "***hidden***",
      "crm_config": {
        "hubspot_portal_id": "12345678",
        "hubspot_api_key_secret": "hubspot-api-key",
        "salesforce_instance": "na123",
        "salesforce_credentials_secret": "sf-creds"
      },
      "ai_config": {
        "lead_scoring_threshold": 70,
        "analysis_model": "gemini-2.5-flash",
        "custom_prompts": {
          "lead_scoring": "Analyze this call transcript for lead quality...",
          "intent_detection": "Identify the customer's primary intent..."
        }
      },
      "notification_config": {
        "slack_webhook": "slack-webhook-url",
        "email_alerts": ["manager@company.com"],
        "high_value_threshold": 80
      }
    },
    "usage_stats": {
      "total_requests": 15420,
      "current_month_requests": 892,
      "success_rate": 0.987,
      "average_processing_time_ms": 45000,
      "crm_sync_success_rate": 0.952
    }
  }
}
```

### Update Tenant Configuration

#### Endpoint
```http
PUT /v1/tenants/{tenant_id}/config
```

#### Request Body
```json
{
  "crm_config": {
    "hubspot_portal_id": "87654321",
    "salesforce_instance": "na456"
  },
  "ai_config": {
    "lead_scoring_threshold": 75,
    "custom_prompts": {
      "lead_scoring": "Enhanced lead scoring prompt..."
    }
  },
  "notification_config": {
    "high_value_threshold": 85,
    "email_alerts": ["newmanager@company.com"]
  }
}
```

### Tenant Analytics

#### Endpoint
```http
GET /v1/tenants/{tenant_id}/analytics
```

#### Query Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| `period` | string | Time period: `7d`, `30d`, `90d`, `1y` |
| `metrics` | array | Metrics to include: `volume`, `quality`, `performance`, `costs` |

#### Response
```json
{
  "success": true,
  "data": {
    "period": "30d",
    "tenant_id": "tenant_001",
    "summary": {
      "total_calls": 892,
      "processed_calls": 878,
      "failed_calls": 14,
      "success_rate": 0.984,
      "average_lead_score": 72.5,
      "high_value_leads": 187,
      "crm_sync_success_rate": 0.952
    },
    "daily_breakdown": [
      {
        "date": "2025-09-13",
        "calls": 42,
        "success_rate": 0.976,
        "average_score": 74.2,
        "processing_time_avg_ms": 43500
      }
    ],
    "quality_metrics": {
      "lead_score_distribution": {
        "0-20": 45,
        "21-40": 123,
        "41-60": 267,
        "61-80": 298,
        "81-100": 159
      },
      "project_type_breakdown": {
        "kitchen_remodel": 342,
        "bathroom_remodel": 198,
        "whole_home": 87,
        "additions": 156,
        "other": 109
      }
    },
    "performance_metrics": {
      "average_processing_time_ms": 44500,
      "p50_processing_time_ms": 38000,
      "p95_processing_time_ms": 78000,
      "webhook_response_time_ms": 150
    },
    "cost_analysis": {
      "total_cost": 156.78,
      "cost_per_call": 0.176,
      "cost_breakdown": {
        "speech_to_text": 68.40,
        "vertex_ai": 22.30,
        "storage": 5.25,
        "compute": 45.83,
        "spanner": 15.00
      }
    }
  }
}
```

---

## Monitoring & Health

### System Health Check

#### Endpoint
```http
GET /v1/health
```

#### Response
```json
{
  "status": "healthy",
  "timestamp": "2025-09-13T14:30:00Z",
  "version": "2.0.0",
  "environment": "production",
  "services": {
    "database": {
      "status": "healthy",
      "response_time_ms": 12,
      "connections_active": 8,
      "connections_idle": 12
    },
    "storage": {
      "status": "healthy",
      "response_time_ms": 45
    },
    "speech_api": {
      "status": "healthy",
      "response_time_ms": 89,
      "quota_remaining": 95.2
    },
    "vertex_ai": {
      "status": "healthy",
      "response_time_ms": 234,
      "quota_remaining": 87.6
    },
    "secret_manager": {
      "status": "healthy",
      "response_time_ms": 23
    }
  },
  "metrics": {
    "requests_per_minute": 125,
    "error_rate": 0.012,
    "average_processing_time_ms": 44500
  }
}
```

### Detailed Health Check

#### Endpoint
```http
GET /v1/health/detailed
```

#### Response
```json
{
  "status": "healthy",
  "timestamp": "2025-09-13T14:30:00Z",
  "checks": [
    {
      "name": "database_connectivity",
      "status": "pass",
      "response_time_ms": 12,
      "details": {
        "spanner_instance": "pipeline-prod",
        "database": "pipeline-db",
        "read_query": "SELECT 1",
        "write_test": "passed"
      }
    },
    {
      "name": "speech_api_quota",
      "status": "pass",
      "details": {
        "quota_used_percentage": 4.8,
        "requests_remaining": 285600,
        "reset_time": "2025-09-14T00:00:00Z"
      }
    },
    {
      "name": "vertex_ai_availability",
      "status": "pass",
      "details": {
        "model": "gemini-2.5-flash",
        "region": "us-central1",
        "test_inference": "successful"
      }
    },
    {
      "name": "crm_integrations",
      "status": "warning",
      "details": {
        "hubspot": {
          "status": "healthy",
          "response_time_ms": 456
        },
        "salesforce": {
          "status": "degraded",
          "response_time_ms": 2340,
          "error": "High latency detected"
        }
      }
    }
  ]
}
```

### System Metrics

#### Endpoint
```http
GET /v1/metrics
```

#### Query Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| `timeframe` | string | `1h`, `6h`, `24h`, `7d` |
| `metrics` | array | Specific metrics to return |

#### Response
```json
{
  "success": true,
  "data": {
    "timeframe": "1h",
    "timestamp": "2025-09-13T14:30:00Z",
    "metrics": {
      "requests": {
        "total": 125,
        "successful": 123,
        "failed": 2,
        "success_rate": 0.984
      },
      "performance": {
        "avg_response_time_ms": 150,
        "p50_processing_time_ms": 38000,
        "p95_processing_time_ms": 78000,
        "p99_processing_time_ms": 120000
      },
      "errors": {
        "total": 2,
        "by_type": {
          "validation_error": 1,
          "crm_timeout": 1
        }
      },
      "resources": {
        "cpu_usage_percent": 34.5,
        "memory_usage_percent": 67.2,
        "active_instances": 8,
        "max_instances": 100
      }
    }
  }
}
```

---

## Error Handling

### Error Response Format

All API errors follow a consistent structure:

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {},
    "timestamp": "2025-09-13T14:30:00Z",
    "trace_id": "abc123def456",
    "documentation": "https://docs.api.com/errors/ERROR_CODE"
  },
  "request_id": "req_error_789"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Request validation failed |
| `INVALID_SIGNATURE` | 401 | Webhook signature invalid |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `TENANT_NOT_FOUND` | 404 | Tenant ID not found |
| `REQUEST_NOT_FOUND` | 404 | Request ID not found |
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |
| `DATABASE_ERROR` | 503 | Database connectivity issue |
| `CRM_INTEGRATION_ERROR` | 502 | CRM service error |
| `AI_SERVICE_ERROR` | 502 | AI service unavailable |

### Detailed Error Examples

#### Validation Error
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": {
      "field_errors": [
        {
          "field": "call_id",
          "issue": "required field missing"
        },
        {
          "field": "tenant_id",
          "issue": "invalid format - must be alphanumeric"
        }
      ]
    }
  }
}
```

#### Rate Limit Error
```json
{
  "success": false,
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded for tenant",
    "details": {
      "limit": 100,
      "period": "1 minute",
      "reset_time": "2025-09-13T14:31:00Z",
      "retry_after": 45
    }
  }
}
```

---

## Rate Limiting

### Limits by Endpoint Type

| Endpoint Type | Limit | Window | Burst |
|---------------|-------|--------|-------|
| **Webhooks** | 1000 requests | 1 hour | 50 |
| **Management API** | 500 requests | 1 hour | 25 |
| **Analytics** | 100 requests | 1 hour | 10 |
| **Health Checks** | Unlimited | - | - |

### Rate Limit Headers

All responses include rate limiting information:

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 987
X-RateLimit-Reset: 1694620800
X-RateLimit-Window: 3600
```

### Handling Rate Limits

When rate limited, implement exponential backoff:

```python
import time
import random

def api_request_with_retry(url, headers, data, max_retries=3):
    for attempt in range(max_retries):
        response = requests.post(url, headers=headers, json=data)

        if response.status_code != 429:
            return response

        # Parse retry-after header
        retry_after = int(response.headers.get('X-RateLimit-Reset', 60))

        # Exponential backoff with jitter
        delay = min(2 ** attempt + random.uniform(0, 1), retry_after)
        time.sleep(delay)

    raise Exception("Max retries exceeded")
```

---

## SDK Examples

### Python SDK

```python
from pipeline_client import PipelineClient

# Initialize client
client = PipelineClient(
    base_url="https://api.yourcompany.com/v1",
    api_key="your_api_key",
    tenant_id="tenant_001"
)

# Get tenant requests
requests = client.requests.list(
    limit=10,
    status="completed",
    start_date="2025-09-01"
)

for request in requests:
    print(f"Request {request.id}: {request.status}")
    if request.ai_analysis:
        print(f"  Lead Score: {request.ai_analysis.lead_score}")
        print(f"  Intent: {request.ai_analysis.intent}")

# Get specific request details
request_detail = client.requests.get("req_98765432")
print(f"Processing time: {request_detail.processing_time_ms}ms")

# Export data
export = client.requests.export(
    format="json",
    start_date="2025-09-01",
    end_date="2025-09-13",
    include_pii=False
)
print(f"Download URL: {export.download_url}")
```

### JavaScript/Node.js SDK

```javascript
const { PipelineClient } = require('@yourcompany/pipeline-client');

// Initialize client
const client = new PipelineClient({
  baseUrl: 'https://api.yourcompany.com/v1',
  apiKey: 'your_api_key',
  tenantId: 'tenant_001'
});

// Get tenant analytics
async function getTenantAnalytics() {
  try {
    const analytics = await client.tenants.getAnalytics({
      period: '30d',
      metrics: ['volume', 'quality', 'performance']
    });

    console.log(`Total calls: ${analytics.summary.total_calls}`);
    console.log(`Success rate: ${analytics.summary.success_rate}`);
    console.log(`Average lead score: ${analytics.summary.average_lead_score}`);

    return analytics;
  } catch (error) {
    if (error.code === 'RATE_LIMIT_EXCEEDED') {
      console.log(`Rate limited. Retry after: ${error.retryAfter} seconds`);
    } else {
      console.error('API Error:', error.message);
    }
  }
}

// Set up webhook handler
const express = require('express');
const app = express();

app.post('/webhook', client.webhooks.verifySignature, (req, res) => {
  console.log('Webhook received:', req.body);
  res.json({ success: true, message: 'Webhook processed' });
});
```

### cURL Examples

#### List Requests
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  "https://api.yourcompany.com/v1/tenants/tenant_001/requests?limit=5&status=completed"
```

#### Get Request Details
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  "https://api.yourcompany.com/v1/tenants/tenant_001/requests/req_98765432"
```

#### Get Tenant Analytics
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  "https://api.yourcompany.com/v1/tenants/tenant_001/analytics?period=30d&metrics=volume,quality"
```

#### Health Check
```bash
curl "https://api.yourcompany.com/v1/health"
```

---

## Webhook Integration Examples

### CallRail Configuration

**Webhook URL**: `https://api.yourcompany.com/v1/callrail/webhook`

**Events to Subscribe**:
- `call_completed`
- `form_submission` (if using CallRail forms)

**Custom Fields**:
```json
{
  "tenant_id": "your_tenant_id",
  "project_type": "{{custom_field_project_type}}",
  "budget_range": "{{custom_field_budget}}",
  "timeline": "{{custom_field_timeline}}"
}
```

### Webhook Testing

Test webhook locally using ngrok:

```bash
# Install ngrok
npm install -g ngrok

# Expose local development server
ngrok http 8080

# Use the generated URL for webhook testing
# Example: https://abc123.ngrok.io/v1/callrail/webhook
```

### Signature Verification

Implement proper signature verification:

```python
import hmac
import hashlib
from flask import request, abort

def verify_webhook_signature():
    signature = request.headers.get('x-callrail-signature', '')
    timestamp = request.headers.get('x-timestamp', '')

    if not signature.startswith('sha256='):
        abort(401)

    # Get webhook secret from environment or secret manager
    secret = get_webhook_secret(tenant_id)

    # Calculate expected signature
    payload = request.get_data()
    message = f"{timestamp}.{payload.decode('utf-8')}"
    expected = hmac.new(
        secret.encode('utf-8'),
        message.encode('utf-8'),
        hashlib.sha256
    ).hexdigest()

    if not hmac.compare_digest(signature[7:], expected):
        abort(401)
```

---

## Troubleshooting API Issues

### Common Issues and Solutions

#### 1. Webhook Signature Failures
**Symptoms**: 401 Unauthorized responses on webhook calls
**Solutions**:
- Verify webhook secret is correct
- Check timestamp is within 5-minute window
- Ensure payload is not modified in transit
- Use constant-time comparison for signature verification

#### 2. Rate Limiting
**Symptoms**: 429 Too Many Requests responses
**Solutions**:
- Implement exponential backoff
- Check rate limit headers
- Consider upgrading rate limits
- Batch API calls where possible

#### 3. Timeout Errors
**Symptoms**: 504 Gateway Timeout responses
**Solutions**:
- Increase client timeout settings
- Check service health endpoints
- Verify database connectivity
- Monitor processing pipeline performance

#### 4. Authentication Issues
**Symptoms**: 401 Unauthorized on management API calls
**Solutions**:
- Verify JWT token is valid and not expired
- Check token includes correct tenant_id claim
- Ensure API key has sufficient permissions
- Refresh access token if needed

### Getting Support

For API-related issues:

1. **Check Status Page**: [status.api.yourcompany.com](https://status.api.yourcompany.com)
2. **Review Documentation**: [docs.api.yourcompany.com](https://docs.api.yourcompany.com)
3. **Contact Support**:
   - Email: api-support@yourcompany.com
   - Slack: #api-support
   - Phone: +1-555-API-HELP (enterprise customers)

Include in support requests:
- Request ID from error responses
- Timestamp of the issue
- Complete error response
- Steps to reproduce the issue

---

**For complete OpenAPI specification and interactive documentation, visit: [openapi.yaml](openapi.yaml)**