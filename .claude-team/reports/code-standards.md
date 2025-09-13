# Multi-Tenant Ingestion Pipeline - Code Standards

**Document Version**: 1.0
**Date**: September 13, 2025
**Project**: Multi-Tenant CallRail Integration Pipeline
**Team**: Backend Engineering

---

## 🎯 **Overview**

This document establishes comprehensive code quality standards for the multi-tenant ingestion pipeline built with Go microservices, Google Cloud Platform, and CallRail integration. These standards ensure production-ready, secure, and performant code that meets enterprise requirements.

---

## 🏗️ **Go Code Quality Standards**

### **1. Code Structure & Organization**

#### **Project Layout** (Follow Go Standard Project Layout)
```
cmd/
├── webhook-processor/     # CallRail webhook handler
├── audio-processor/       # Speech-to-Text service
├── ai-analyzer/          # Gemini content analysis
├── workflow-engine/       # Task orchestration
└── api-gateway/          # Main API server

internal/
├── auth/                 # Tenant authentication
├── callrail/            # CallRail API client
├── spanner/             # Database operations
├── storage/             # Cloud Storage operations
├── ai/                  # Vertex AI integration
└── workflow/            # Workflow processing

pkg/
├── models/              # Data structures
├── config/              # Configuration management
└── utils/               # Shared utilities
```

#### **File Naming Conventions**
- Use snake_case for file names: `webhook_handler.go`, `tenant_auth.go`
- Test files: `webhook_handler_test.go`
- Interface files: `repository.go`, `service.go`
- Implementation files: `spanner_repository.go`, `callrail_service.go`

#### **Package Naming**
- Single word, lowercase: `auth`, `spanner`, `workflow`
- Descriptive and concise: `callrail` not `callrailintegration`
- Avoid generic names: `models` not `data` or `types`

### **2. Error Handling Standards**

#### **Error Types**
```go
// Custom error types for better error handling
type CallRailError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    CallID  string `json:"call_id,omitempty"`
}

func (e *CallRailError) Error() string {
    return fmt.Sprintf("CallRail error [%s]: %s", e.Code, e.Message)
}

// Sentinel errors for known conditions
var (
    ErrTenantNotFound     = errors.New("tenant not found")
    ErrInvalidSignature   = errors.New("invalid HMAC signature")
    ErrAudioDownloadFailed = errors.New("audio download failed")
)
```

#### **Error Wrapping**
```go
// Always wrap errors with context
func (s *CallRailService) ProcessWebhook(ctx context.Context, payload []byte) error {
    webhook, err := s.parseWebhook(payload)
    if err != nil {
        return fmt.Errorf("parsing webhook payload: %w", err)
    }

    if err := s.validateTenant(ctx, webhook.TenantID); err != nil {
        return fmt.Errorf("validating tenant %s: %w", webhook.TenantID, err)
    }

    return nil
}
```

#### **Logging Standards**
```go
import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// Structured logging with context
logger.Error("webhook processing failed",
    zap.String("tenant_id", tenantID),
    zap.String("call_id", callID),
    zap.Error(err),
    zap.Duration("processing_time", time.Since(start)),
)
```

### **3. Context Usage**

#### **Mandatory Context Propagation**
```go
// All functions that perform I/O must accept context
func (r *SpannerRepository) GetTenantByCallRailID(
    ctx context.Context,
    callrailID, tenantID string,
) (*models.Tenant, error) {
    // Implementation with context timeout
}

// Set appropriate timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

#### **Context Values** (Use Sparingly)
```go
type contextKey string

const (
    tenantIDKey contextKey = "tenant_id"
    requestIDKey contextKey = "request_id"
)

// Helper functions for context values
func WithTenantID(ctx context.Context, tenantID string) context.Context {
    return context.WithValue(ctx, tenantIDKey, tenantID)
}

func TenantIDFromContext(ctx context.Context) (string, bool) {
    tenantID, ok := ctx.Value(tenantIDKey).(string)
    return tenantID, ok
}
```

### **4. Interface Design**

#### **Small, Focused Interfaces**
```go
// Repository interface - single responsibility
type TenantRepository interface {
    GetByCallRailID(ctx context.Context, callrailID, tenantID string) (*models.Tenant, error)
    UpdateWorkflowConfig(ctx context.Context, tenantID string, config *models.WorkflowConfig) error
}

// Service interface - business logic
type WebhookProcessor interface {
    ProcessCallRailWebhook(ctx context.Context, payload []byte, signature string) error
    ValidateSignature(payload []byte, signature, secret string) error
}
```

#### **Mock Generation**
```go
//go:generate mockgen -source=repository.go -destination=mocks/mock_repository.go

// Use dependency injection for testability
type CallRailService struct {
    tenantRepo TenantRepository
    audioStore AudioStorage
    logger     *zap.Logger
}
```

### **5. Concurrency Patterns**

#### **Worker Pool for Audio Processing**
```go
type AudioProcessor struct {
    workers   int
    workQueue chan AudioJob
    wg        sync.WaitGroup
}

func (p *AudioProcessor) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(ctx)
    }
}

func (p *AudioProcessor) worker(ctx context.Context) {
    defer p.wg.Done()
    for {
        select {
        case job := <-p.workQueue:
            p.processAudio(ctx, job)
        case <-ctx.Done():
            return
        }
    }
}
```

#### **Graceful Shutdown**
```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    server := &http.Server{Addr: ":8080", Handler: handler}

    go func() {
        <-ctx.Done()
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer shutdownCancel()
        server.Shutdown(shutdownCtx)
    }()

    server.ListenAndServe()
}
```

---

## 🔐 **Security Standards**

### **1. HMAC Signature Verification**

#### **Implementation Requirements**
```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func VerifyHMACSignature(payload []byte, signature, secret string) error {
    // Remove 'sha256=' prefix if present
    expectedSignature := strings.TrimPrefix(signature, "sha256=")

    // Compute HMAC-SHA256
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    computedHash := hex.EncodeToString(mac.Sum(nil))

    // Constant-time comparison to prevent timing attacks
    if !hmac.Equal([]byte(expectedSignature), []byte(computedHash)) {
        return ErrInvalidSignature
    }

    return nil
}
```

#### **Security Checklist**
- [ ] Use constant-time comparison (`hmac.Equal`)
- [ ] Validate signature before any processing
- [ ] Log signature verification failures
- [ ] Use strong secrets (minimum 32 bytes)
- [ ] Store secrets in Google Secret Manager

### **2. Input Validation**

#### **Request Validation**
```go
import "github.com/go-playground/validator/v10"

type CallRailWebhook struct {
    CallID           string `json:"call_id" validate:"required,min=3,max=50"`
    TenantID         string `json:"tenant_id" validate:"required,uuid4"`
    CallRailCompanyID string `json:"callrail_company_id" validate:"required,numeric"`
    CallerID         string `json:"caller_id" validate:"required,e164"`
    Duration         int    `json:"duration" validate:"min=0,max=86400"`
    RecordingURL     string `json:"recording_url" validate:"required,url"`
}

func ValidateWebhook(webhook *CallRailWebhook) error {
    validate := validator.New()
    if err := validate.Struct(webhook); err != nil {
        return fmt.Errorf("webhook validation failed: %w", err)
    }
    return nil
}
```

#### **SQL Injection Prevention**
```go
// ALWAYS use parameterized queries
func (r *SpannerRepository) GetTenantByCallRailID(ctx context.Context, callrailID, tenantID string) (*models.Tenant, error) {
    stmt := spanner.Statement{
        SQL: `SELECT tenant_id, workflow_config, callrail_api_key
              FROM offices
              WHERE callrail_company_id = @callrail_id
              AND tenant_id = @tenant_id
              AND status = 'active'`,
        Params: map[string]interface{}{
            "callrail_id": callrailID,
            "tenant_id":   tenantID,
        },
    }
    // Execute query...
}
```

### **3. Tenant Isolation**

#### **Row-Level Security Enforcement**
```go
// Always include tenant_id in queries
func (r *SpannerRepository) GetRequestsByTenant(ctx context.Context, tenantID string, limit int) ([]*models.Request, error) {
    stmt := spanner.Statement{
        SQL: `SELECT request_id, tenant_id, call_id, transcription_data, ai_analysis
              FROM requests
              WHERE tenant_id = @tenant_id
              ORDER BY created_at DESC
              LIMIT @limit`,
        Params: map[string]interface{}{
            "tenant_id": tenantID,
            "limit":     limit,
        },
    }
    // Implementation...
}
```

#### **Tenant Context Validation**
```go
func ValidateTenantAccess(ctx context.Context, requestedTenantID string) error {
    // Extract tenant ID from authentication context
    authenticatedTenantID, ok := TenantIDFromContext(ctx)
    if !ok {
        return errors.New("no tenant context found")
    }

    if authenticatedTenantID != requestedTenantID {
        return fmt.Errorf("tenant access denied: authenticated=%s, requested=%s",
            authenticatedTenantID, requestedTenantID)
    }

    return nil
}
```

---

## 🚀 **Performance Standards**

### **1. Database Optimization**

#### **Connection Pooling**
```go
import "cloud.google.com/go/spanner"

func NewSpannerClient(ctx context.Context) (*spanner.Client, error) {
    config := spanner.ClientConfig{
        SessionPoolConfig: spanner.SessionPoolConfig{
            MinOpened:     10,    // Minimum sessions
            MaxOpened:     100,   // Maximum sessions
            MaxIdle:       50,    // Maximum idle sessions
            WriteSessions: 0.2,   // 20% write sessions
        },
    }

    return spanner.NewClientWithConfig(ctx, "projects/account-strategy-464106/instances/upai-customers/databases/agent_platform", config)
}
```

#### **Query Optimization**
```go
// Use appropriate indexes and limit results
func (r *SpannerRepository) GetRecentRequestsByTenant(ctx context.Context, tenantID string, limit int) ([]*models.Request, error) {
    // This query uses idx_requests_lead_score index
    stmt := spanner.Statement{
        SQL: `SELECT request_id, tenant_id, call_id, lead_score, created_at
              FROM requests@{FORCE_INDEX=idx_requests_lead_score}
              WHERE tenant_id = @tenant_id
              ORDER BY lead_score DESC, created_at DESC
              LIMIT @limit`,
        Params: map[string]interface{}{
            "tenant_id": tenantID,
            "limit":     limit,
        },
    }
    // Implementation...
}
```

### **2. Memory Management**

#### **Audio Processing**
```go
// Stream audio processing to avoid loading entire file in memory
func (s *AudioService) ProcessAudioStream(ctx context.Context, storageURL string) (*models.Transcription, error) {
    reader, err := s.storage.NewReader(ctx, storageURL)
    if err != nil {
        return nil, fmt.Errorf("creating audio reader: %w", err)
    }
    defer reader.Close()

    // Process in chunks
    const chunkSize = 64 * 1024 // 64KB chunks
    buffer := make([]byte, chunkSize)

    for {
        n, err := reader.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, fmt.Errorf("reading audio chunk: %w", err)
        }

        // Process chunk
        if err := s.processAudioChunk(ctx, buffer[:n]); err != nil {
            return nil, fmt.Errorf("processing audio chunk: %w", err)
        }
    }

    return s.getTranscriptionResult(ctx)
}
```

### **3. Caching Strategies**

#### **Configuration Caching**
```go
import (
    "github.com/patrickmn/go-cache"
    "time"
)

type ConfigCache struct {
    cache *cache.Cache
}

func NewConfigCache() *ConfigCache {
    return &ConfigCache{
        cache: cache.New(15*time.Minute, 30*time.Minute), // 15min TTL, 30min cleanup
    }
}

func (c *ConfigCache) GetWorkflowConfig(ctx context.Context, tenantID string) (*models.WorkflowConfig, error) {
    if config, found := c.cache.Get(tenantID); found {
        return config.(*models.WorkflowConfig), nil
    }

    // Fetch from database
    config, err := c.repo.GetWorkflowConfig(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    // Cache the result
    c.cache.Set(tenantID, config, cache.DefaultExpiration)
    return config, nil
}
```

---

## 🏛️ **Architecture Standards**

### **1. Dependency Injection**

#### **Service Construction**
```go
type Services struct {
    TenantRepo      TenantRepository
    AudioProcessor  AudioProcessor
    AIAnalyzer      AIAnalyzer
    Logger          *zap.Logger
    Config          *config.Config
}

func NewServices(cfg *config.Config) (*Services, error) {
    logger, err := zap.NewProduction()
    if err != nil {
        return nil, fmt.Errorf("creating logger: %w", err)
    }

    spannerClient, err := NewSpannerClient(context.Background())
    if err != nil {
        return nil, fmt.Errorf("creating spanner client: %w", err)
    }

    return &Services{
        TenantRepo:     NewSpannerTenantRepository(spannerClient),
        AudioProcessor: NewSpeechToTextProcessor(cfg.SpeechToText),
        AIAnalyzer:     NewGeminiAnalyzer(cfg.VertexAI),
        Logger:         logger,
        Config:         cfg,
    }, nil
}
```

### **2. Configuration Management**

#### **Environment-based Configuration**
```go
import (
    "github.com/kelseyhightower/envconfig"
)

type Config struct {
    ProjectID     string `envconfig:"GOOGLE_CLOUD_PROJECT" required:"true"`
    Location      string `envconfig:"GOOGLE_CLOUD_LOCATION" default:"us-central1"`

    Spanner struct {
        Instance string `envconfig:"SPANNER_INSTANCE" required:"true"`
        Database string `envconfig:"SPANNER_DATABASE" required:"true"`
    }

    VertexAI struct {
        Project  string `envconfig:"VERTEX_AI_PROJECT" required:"true"`
        Location string `envconfig:"VERTEX_AI_LOCATION" default:"us-central1"`
        Model    string `envconfig:"VERTEX_AI_MODEL" default:"gemini-2.5-flash"`
    }

    SpeechToText struct {
        Project  string `envconfig:"SPEECH_TO_TEXT_PROJECT" required:"true"`
        Language string `envconfig:"SPEECH_LANGUAGE" default:"en-US"`
    }

    CallRail struct {
        WebhookSecretName string `envconfig:"CALLRAIL_WEBHOOK_SECRET_NAME" required:"true"`
    }

    Storage struct {
        AudioBucket string `envconfig:"AUDIO_STORAGE_BUCKET" required:"true"`
    }
}

func LoadConfig() (*Config, error) {
    var cfg Config
    if err := envconfig.Process("", &cfg); err != nil {
        return nil, fmt.Errorf("loading config: %w", err)
    }
    return &cfg, nil
}
```

### **3. Clean Interface Boundaries**

#### **Repository Pattern**
```go
// Domain models (no external dependencies)
type Tenant struct {
    ID                string
    CallRailCompanyID string
    WorkflowConfig    *WorkflowConfig
    APIKey           string
    Status           string
    CreatedAt        time.Time
}

// Repository interface (domain layer)
type TenantRepository interface {
    GetByCallRailID(ctx context.Context, callrailID, tenantID string) (*Tenant, error)
    UpdateWorkflowConfig(ctx context.Context, tenantID string, config *WorkflowConfig) error
}

// Spanner implementation (infrastructure layer)
type SpannerTenantRepository struct {
    client *spanner.Client
}

func (r *SpannerTenantRepository) GetByCallRailID(ctx context.Context, callrailID, tenantID string) (*Tenant, error) {
    // Spanner-specific implementation
}
```

---

## 📋 **Code Review Requirements**

### **Mandatory Reviews**
- [ ] All production code requires peer review
- [ ] Security-sensitive code requires security team review
- [ ] Database schema changes require DBA review
- [ ] Performance-critical code requires performance review

### **Review Criteria**
- [ ] Follows Go idioms and conventions
- [ ] Proper error handling and logging
- [ ] Context usage and timeout handling
- [ ] Thread-safe operations
- [ ] Multi-tenant isolation maintained
- [ ] Security best practices followed
- [ ] Performance considerations addressed
- [ ] Tests provide adequate coverage
- [ ] Documentation is clear and complete

### **Automated Checks**
- [ ] `go fmt` formatting
- [ ] `go vet` static analysis
- [ ] `golangci-lint` comprehensive linting
- [ ] Unit test coverage >80%
- [ ] Security scanning (gosec)
- [ ] Dependency vulnerability scanning

---

## 📚 **Documentation Standards**

### **Code Documentation**
```go
// WebhookProcessor handles CallRail webhook events and processes them
// through the multi-tenant ingestion pipeline.
//
// It performs the following operations:
// 1. Validates HMAC signature for security
// 2. Authenticates tenant access
// 3. Downloads and processes audio recordings
// 4. Runs AI analysis on call content
// 5. Triggers configured workflows
type WebhookProcessor struct {
    tenantRepo TenantRepository
    logger     *zap.Logger
}

// ProcessCallRailWebhook processes an incoming CallRail webhook event.
//
// The method validates the HMAC signature, authenticates the tenant,
// and initiates the call processing workflow. Processing is done
// asynchronously using Cloud Tasks.
//
// Parameters:
//   - ctx: Request context with timeout and cancellation
//   - payload: Raw webhook payload bytes
//   - signature: HMAC signature from x-callrail-signature header
//
// Returns an error if validation fails or processing cannot be initiated.
func (p *WebhookProcessor) ProcessCallRailWebhook(ctx context.Context, payload []byte, signature string) error {
    // Implementation...
}
```

### **API Documentation**
- Use OpenAPI/Swagger specifications
- Document all endpoints, parameters, and responses
- Include example requests and responses
- Document error codes and messages

---

This comprehensive code standards document ensures consistent, secure, and maintainable code across the multi-tenant ingestion pipeline project.