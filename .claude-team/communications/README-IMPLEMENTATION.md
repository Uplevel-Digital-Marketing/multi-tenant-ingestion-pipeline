# Multi-Tenant Ingestion Pipeline - Go Implementation

## 🚀 Overview

This implementation provides production-ready Go microservices for the multi-tenant ingestion pipeline, specifically designed to process CallRail webhooks with AI-powered analysis.

## 📁 Project Structure

```
├── cmd/                           # Main applications
│   ├── webhook-processor/         # CallRail webhook handler
│   ├── audio-processor/           # Speech-to-Text service (planned)
│   ├── ai-analyzer/              # Gemini content analysis (planned)
│   ├── workflow-engine/          # Task orchestration (planned)
│   └── api-gateway/              # Main API server
├── internal/                     # Private application code
│   ├── auth/                     # Tenant authentication & HMAC verification
│   ├── callrail/                 # CallRail API client with rate limiting
│   ├── spanner/                  # Cloud Spanner database operations
│   ├── storage/                  # Cloud Storage operations
│   ├── ai/                       # Vertex AI & Speech-to-Text integration
│   └── workflow/                 # Workflow processing (planned)
├── pkg/                          # Public shared libraries
│   ├── models/                   # Data structures & models
│   ├── config/                   # Configuration management
│   └── utils/                    # Shared utilities (planned)
├── deployments/                  # Deployment configurations
│   ├── docker/                   # Dockerfiles
│   ├── k8s/                      # Kubernetes manifests (planned)
│   └── terraform/                # Infrastructure as Code
└── scripts/                      # Setup and deployment scripts
```

## 🔧 Core Services Implemented

### 1. Webhook Processor (`cmd/webhook-processor`)
- **Purpose**: Processes CallRail webhooks with HMAC verification
- **Features**:
  - Webhook signature verification using HMAC-SHA256
  - Tenant authentication via CallRail company ID mapping
  - Async processing of audio downloads and AI analysis
  - Cloud Spanner integration for data persistence
  - Structured logging and error handling

### 2. API Gateway (`cmd/api-gateway`)
- **Purpose**: RESTful API for tenant data access
- **Features**:
  - Tenant-based authentication middleware
  - Request management endpoints
  - Analytics calculation
  - Pagination support
  - Health checks

### 3. Authentication Service (`internal/auth`)
- **Purpose**: Handle tenant authentication and authorization
- **Features**:
  - HMAC webhook signature verification
  - Tenant-to-CallRail company mapping
  - Workflow configuration management
  - Access control validation

### 4. CallRail Client (`internal/callrail`)
- **Purpose**: Interact with CallRail API
- **Features**:
  - Rate limiting (120 requests/minute)
  - Exponential backoff retry logic
  - Recording download capability
  - Error handling for authentication issues

### 5. Cloud Spanner Repository (`internal/spanner`)
- **Purpose**: Database operations for multi-tenant data
- **Features**:
  - Office and tenant management
  - Request tracking and analytics
  - Webhook event logging
  - Call recording metadata

### 6. Cloud Storage Service (`internal/storage`)
- **Purpose**: Audio file storage and management
- **Features**:
  - Tenant-isolated storage paths
  - Lifecycle policies for cost optimization
  - Signed URL generation
  - Storage statistics tracking

### 7. AI Service (`internal/ai`)
- **Purpose**: Speech-to-Text and Vertex AI integration
- **Features**:
  - Chirp 3 speech recognition with diarization
  - Gemini 2.5 Flash content analysis
  - Spam detection
  - Lead scoring and sentiment analysis

## 🛠️ Configuration

### Environment Variables
```bash
# Required for all services
GOOGLE_CLOUD_PROJECT=account-strategy-464106
GOOGLE_CLOUD_LOCATION=us-central1
SPANNER_INSTANCE=upai-customers
SPANNER_DATABASE=agent_platform

# AI Services
VERTEX_AI_PROJECT=account-strategy-464106
VERTEX_AI_LOCATION=us-central1
VERTEX_AI_MODEL=gemini-2.5-flash
SPEECH_TO_TEXT_MODEL=chirp-3
SPEECH_LANGUAGE=en-US
ENABLE_DIARIZATION=true

# Storage
AUDIO_STORAGE_BUCKET=tenant-audio-files-account-strategy-464106
STORAGE_LOCATION=us-central1
RETENTION_DAYS=2555

# Security
CALLRAIL_WEBHOOK_SECRET_NAME=callrail-webhook-secret

# Service Configuration
PORT=8080
ENVIRONMENT=production
```

## 🚀 Deployment

### Quick Start
```bash
# Set up development environment
make setup-dev

# Build all services
make build

# Run tests
make test

# Deploy to production
make deploy
```

### Infrastructure Setup
```bash
# Initialize Terraform
make terraform-init

# Plan infrastructure changes
make terraform-plan

# Apply infrastructure
make terraform-apply

# Apply database schema
make db-update

# Create storage bucket
make create-bucket
```

### Docker Deployment
```bash
# Build Docker images
make docker-build

# Run services locally
make run-local
```

## 📊 API Endpoints

### Webhook Processor
- `POST /api/v1/callrail/webhook` - CallRail webhook endpoint
- `GET /health` - Service health check

### API Gateway
- `GET /api/v1/requests` - List requests (with pagination)
- `GET /api/v1/requests/:request_id` - Get specific request
- `GET /api/v1/tenants/:tenant_id/requests` - Tenant-specific requests
- `GET /api/v1/tenants/:tenant_id/analytics` - Tenant analytics
- `GET /health` - Service health check

## 🔒 Security Features

### Authentication
- HMAC-SHA256 webhook signature verification
- Tenant isolation via CallRail company ID mapping
- Row-level security in Cloud Spanner
- Service account-based GCP authentication

### Data Protection
- Tenant-isolated storage paths
- Encrypted storage at rest
- Secure secret management via Secret Manager
- Audit logging for all operations

## 📈 Monitoring & Observability

### Logging
- Structured JSON logging
- Request tracing with correlation IDs
- Error tracking and alerting
- Performance metrics

### Health Checks
```bash
# Check service health
make health-check

# Stream logs
make logs-webhook
make logs-api
```

## 🧪 Testing

### Unit Tests
```bash
# Run unit tests
go test ./... -v

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests
```bash
# Start emulators for testing
gcloud emulators spanner start

# Run integration tests
SPANNER_EMULATOR_HOST=localhost:9010 go test ./test/integration/...
```

## 🔄 CallRail Integration Flow

### Step-by-Step Process
1. **Webhook Reception**: Receive CallRail webhook at `/api/v1/callrail/webhook`
2. **Signature Verification**: Validate HMAC signature using stored secret
3. **Tenant Authentication**: Map CallRail company ID to tenant
4. **Call Data Retrieval**: Fetch detailed call information from CallRail API
5. **Recording Download**: Download and store audio file in Cloud Storage
6. **Audio Transcription**: Process audio with Speech-to-Text Chirp 3
7. **AI Analysis**: Analyze content with Gemini 2.5 Flash
8. **Data Storage**: Store enhanced payload in Cloud Spanner
9. **Workflow Trigger**: Initiate downstream processing (planned)

### Sample Webhook Payload
```json
{
  "call_id": "CAL123456789",
  "tenant_id": "tenant_12345",
  "callrail_company_id": "12345",
  "caller_id": "+15551234567",
  "duration": "180",
  "recording_url": "https://api.callrail.com/...",
  "customer_name": "John Smith",
  "customer_city": "Los Angeles",
  "customer_state": "CA"
}
```

## 🎯 AI-Powered Features

### Speech-to-Text Processing
- **Model**: Chirp 3 (Google's latest)
- **Features**: Speaker diarization, punctuation, confidence scoring
- **Languages**: English (US) with expansion capability
- **Output**: Detailed transcription with speaker segments

### Content Analysis
- **Model**: Gemini 2.5 Flash
- **Analysis**: Intent, project type, sentiment, urgency
- **Lead Scoring**: 1-100 scale based on multiple factors
- **Spam Detection**: AI-powered spam likelihood assessment

### Sample AI Analysis Output
```json
{
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
}
```

## 📦 Dependencies

### Core Dependencies
- `cloud.google.com/go/spanner` - Cloud Spanner client
- `cloud.google.com/go/aiplatform` - Vertex AI client
- `cloud.google.com/go/speech` - Speech-to-Text client
- `cloud.google.com/go/storage` - Cloud Storage client
- `cloud.google.com/go/secretmanager` - Secret Manager client
- `github.com/gin-gonic/gin` - HTTP web framework
- `github.com/golang-jwt/jwt/v5` - JWT handling
- `github.com/google/uuid` - UUID generation

### Development Dependencies
- `github.com/stretchr/testify` - Testing framework
- Go 1.21+ required

## 💰 Cost Optimization

### Storage Lifecycle
- **Standard**: 0-90 days
- **Coldline**: 90-365 days
- **Archive**: 365+ days
- **Delete**: After 2555 days (7 years)

### AI Services Usage
- **Speech-to-Text**: Chirp 3 model for accuracy
- **Vertex AI**: Gemini 2.5 Flash for cost-effective analysis
- **Optimized Prompts**: Minimal token usage for analysis

## 🚨 Error Handling

### Retry Logic
- Exponential backoff for API calls
- Rate limiting compliance
- Dead letter queues for failed processing

### Graceful Degradation
- Continue processing without recording if download fails
- Store partial data if AI analysis fails
- Comprehensive error logging for debugging

## 📝 Next Steps

### Planned Services
1. **Audio Processor**: Dedicated service for audio processing
2. **AI Analyzer**: Standalone AI analysis service
3. **Workflow Engine**: Cloud Tasks-based orchestration
4. **Monitoring Dashboard**: Real-time metrics and alerts

### Production Readiness Checklist
- [x] HMAC webhook verification
- [x] Tenant authentication and isolation
- [x] Audio storage and transcription
- [x] AI-powered content analysis
- [x] Database operations with error handling
- [x] Docker containerization
- [x] Terraform infrastructure
- [x] Health checks and monitoring
- [ ] Comprehensive test suite
- [ ] Performance optimization
- [ ] Security audit
- [ ] Documentation completion

## 🤝 Contributing

1. Follow Go best practices and patterns
2. Add tests for new functionality
3. Update documentation for API changes
4. Use structured logging for observability
5. Implement proper error handling

This implementation provides a solid foundation for the multi-tenant ingestion pipeline with CallRail integration, ready for production deployment and scalable growth.