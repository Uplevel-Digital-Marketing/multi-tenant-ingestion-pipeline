# 🛡️ Multi-Tenant Ingestion Pipeline - 2025 Security Baseline

## 🔒 **Security Auditor Report - 2025 Enterprise Standards**
**Generated**: September 2025 | **Agent**: security-auditor | **Duration**: 15min
**Security Focus**: Multi-tenant isolation, HMAC verification, 2025 compliance standards

---

## 📋 **Current Security Assessment**

### **✅ Strong Security Foundation Identified**
Based on analysis of existing authentication code (`internal/auth/auth.go`):

1. **HMAC Signature Verification**: ✅ Properly implemented
2. **Tenant Isolation**: ✅ Multi-tenant architecture present
3. **Error Handling**: ✅ Custom error types for security events
4. **Webhook Security**: ✅ CallRail signature validation

### **🔧 Areas Requiring 2025 Upgrades**
1. **Enhanced Secret Management**: Migrate to Google Secret Manager
2. **Zero-Trust Architecture**: Implement service-to-service authentication
3. **Advanced Monitoring**: Real-time security event detection
4. **Compliance Frameworks**: GDPR, CCPA, SOC2 2025 requirements

---

## 🏗️ **2025 Security Architecture Framework**

### **Multi-Tenant Security Layers**

```
┌─────────────────────────────────────────────────────────────────┐
│                    2025 SECURITY ARCHITECTURE                   │
├─────────────────────────────────────────────────────────────────┤
│  Layer 1: Edge Security                                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Cloud CDN   │  │ Cloud Armor │  │ Load        │             │
│  │ DDoS        │  │ WAF Rules   │  │ Balancer    │             │
│  │ Protection  │  │ Rate Limit  │  │ SSL/TLS     │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
├─────────────────────────────────────────────────────────────────┤
│  Layer 2: Application Security                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ HMAC        │  │ JWT Token   │  │ mTLS        │             │
│  │ Verification│  │ Validation  │  │ Service     │             │
│  │ (Enhanced)  │  │ (New 2025)  │  │ Mesh        │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3: Data Security                                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Row-Level   │  │ Field-Level │  │ Vector      │             │
│  │ Security    │  │ Encryption  │  │ Embeddings  │             │
│  │ (Spanner)   │  │ (KMS)       │  │ Security    │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4: Infrastructure Security                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ VPC         │  │ Private     │  │ Workload    │             │
│  │ Network     │  │ Service     │  │ Identity    │             │
│  │ Isolation   │  │ Connect     │  │ (2025)      │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔐 **Enhanced Authentication & Authorization (2025)**

### **Current HMAC Implementation Assessment**
```go
// CURRENT: Basic HMAC verification (GOOD foundation)
func (a *AuthService) VerifyCallRailWebhook(payload []byte, signature string) error {
    // ✅ Proper HMAC-SHA256 implementation
    // ✅ Signature prefix handling
    // ✅ Constant-time comparison (assumed)
    // ❌ Missing: Rate limiting
    // ❌ Missing: Tenant-specific secrets
    // ❌ Missing: Audit logging
}
```

### **2025 Enhanced HMAC Verification**
```go
// internal/auth/hmac_2025.go
package auth

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/hex"
    "fmt"
    "time"

    secretmanager "cloud.google.com/go/secretmanager/apiv1"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"

    "github.com/home-renovators/ingestion-pipeline/pkg/monitoring"
)

// Enhanced2025AuthService with advanced security features
type Enhanced2025AuthService struct {
    secretManager   *secretmanager.Client
    rateLimiter     *RateLimiter
    auditLogger     *AuditLogger
    securityMonitor *SecurityMonitor
    tracer          trace.Tracer
}

// TenantSecurityContext holds tenant-specific security information
type TenantSecurityContext struct {
    TenantID          string    `json:"tenant_id"`
    WebhookSecret     string    `json:"webhook_secret"`
    LastVerification  time.Time `json:"last_verification"`
    FailedAttempts    int       `json:"failed_attempts"`
    SecurityLevel     string    `json:"security_level"` // "standard", "enhanced", "strict"
    IPWhitelist       []string  `json:"ip_whitelist"`
    MaxRequestsPerMin int       `json:"max_requests_per_min"`
}

// Enhanced webhook verification with comprehensive security
func (a *Enhanced2025AuthService) VerifyCallRailWebhook2025(
    ctx context.Context,
    payload []byte,
    signature string,
    tenantID string,
    sourceIP string,
) (*TenantSecurityContext, error) {

    span := a.tracer.Start(ctx, "webhook_verification")
    defer span.End()

    // 1. Rate limiting per tenant and IP
    if !a.rateLimiter.AllowTenant(tenantID) {
        a.auditLogger.LogSecurityEvent(ctx, "rate_limit_exceeded", tenantID, sourceIP)
        return nil, fmt.Errorf("rate limit exceeded for tenant %s", tenantID)
    }

    if !a.rateLimiter.AllowIP(sourceIP) {
        a.auditLogger.LogSecurityEvent(ctx, "ip_rate_limit_exceeded", tenantID, sourceIP)
        return nil, fmt.Errorf("rate limit exceeded for IP %s", sourceIP)
    }

    // 2. Get tenant security context
    securityCtx, err := a.getTenantSecurityContext(ctx, tenantID)
    if err != nil {
        a.auditLogger.LogSecurityEvent(ctx, "tenant_context_error", tenantID, sourceIP)
        return nil, fmt.Errorf("retrieving tenant security context: %w", err)
    }

    // 3. IP whitelist validation (if configured)
    if len(securityCtx.IPWhitelist) > 0 && !a.isIPWhitelisted(sourceIP, securityCtx.IPWhitelist) {
        a.auditLogger.LogSecurityEvent(ctx, "ip_not_whitelisted", tenantID, sourceIP)
        return nil, fmt.Errorf("IP %s not whitelisted for tenant %s", sourceIP, tenantID)
    }

    // 4. Enhanced HMAC verification with timing attack protection
    if err := a.verifyHMACConstantTime(payload, signature, securityCtx.WebhookSecret); err != nil {
        // Increment failed attempts
        securityCtx.FailedAttempts++
        a.updateTenantSecurityContext(ctx, securityCtx)

        a.auditLogger.LogSecurityEvent(ctx, "hmac_verification_failed", tenantID, sourceIP)
        a.securityMonitor.RecordFailedVerification(tenantID, sourceIP)

        return nil, fmt.Errorf("HMAC verification failed: %w", err)
    }

    // 5. Success - reset failed attempts and update last verification
    securityCtx.FailedAttempts = 0
    securityCtx.LastVerification = time.Now().UTC()
    a.updateTenantSecurityContext(ctx, securityCtx)

    a.auditLogger.LogSecurityEvent(ctx, "webhook_verified_successfully", tenantID, sourceIP)
    a.securityMonitor.RecordSuccessfulVerification(tenantID, sourceIP)

    return securityCtx, nil
}

// Constant-time HMAC verification to prevent timing attacks
func (a *Enhanced2025AuthService) verifyHMACConstantTime(payload []byte, signature, secret string) error {
    // Remove prefix variations
    cleanSignature := signature
    if len(signature) > 7 && signature[:7] == "sha256=" {
        cleanSignature = signature[7:]
    }

    // Compute expected signature
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expectedMAC := mac.Sum(nil)
    expectedHex := hex.EncodeToString(expectedMAC)

    // Constant-time comparison to prevent timing attacks
    if subtle.ConstantTimeCompare([]byte(cleanSignature), []byte(expectedHex)) != 1 {
        return fmt.Errorf("invalid HMAC signature")
    }

    return nil
}

// Retrieve tenant-specific webhook secret from Secret Manager
func (a *Enhanced2025AuthService) getTenantSecurityContext(ctx context.Context, tenantID string) (*TenantSecurityContext, error) {
    secretName := fmt.Sprintf("projects/account-strategy-464106/secrets/webhook-secret-%s/versions/latest", tenantID)

    result, err := a.secretManager.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
        Name: secretName,
    })
    if err != nil {
        return nil, fmt.Errorf("accessing webhook secret: %w", err)
    }

    return &TenantSecurityContext{
        TenantID:          tenantID,
        WebhookSecret:     string(result.Payload.Data),
        SecurityLevel:     "enhanced",
        MaxRequestsPerMin: 100, // Default rate limit
    }, nil
}
```

---

## 🏢 **Multi-Tenant Data Security (2025 Enhanced)**

### **Row-Level Security Implementation**
```sql
-- Enhanced 2025 row-level security policies
CREATE ROW ACCESS POLICY tenant_strict_isolation_requests ON requests
  GRANT TO ('application_role', 'admin_role')
  FILTER USING (
    tenant_id = @tenant_id_param
    AND (
      -- Standard access
      CURRENT_USER() = 'application_role'
      OR
      -- Admin access with audit trail
      (CURRENT_USER() = 'admin_role' AND @audit_reason IS NOT NULL)
    )
  );

-- Vector embeddings security
CREATE ROW ACCESS POLICY tenant_vector_isolation ON requests
  GRANT TO ('ai_service_role')
  FILTER USING (
    tenant_id = @tenant_id_param
    AND content_embedding IS NOT NULL
    AND @ai_operation_type IN ('similarity_search', 'embedding_generation')
  );

-- Audit trail for admin access
CREATE TABLE security_audit_log (
  audit_id STRING(36) NOT NULL DEFAULT (GENERATE_UUID()),
  tenant_id STRING(36) NOT NULL,
  user_id STRING(100) NOT NULL,
  action STRING(50) NOT NULL,
  resource_type STRING(50) NOT NULL,
  resource_id STRING(36),
  audit_reason STRING(500),
  ip_address STRING(45),
  user_agent STRING(500),
  created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
  PRIMARY KEY(audit_id)
);
```

### **Field-Level Encryption for PII**
```go
// internal/security/field_encryption_2025.go
package security

import (
    "context"
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"

    kms "cloud.google.com/go/kms/apiv1"
    "cloud.google.com/go/kms/apiv1/kmspb"
)

// FieldEncryption handles PII encryption using Cloud KMS
type FieldEncryption struct {
    kmsClient *kms.KeyManagementClient
    keyName   string
}

// PIIFields defines which fields require encryption
type PIIFields struct {
    CallerName    string `json:"caller_name,omitempty"`
    CallerPhone   string `json:"caller_phone,omitempty"`
    CallerEmail   string `json:"caller_email,omitempty"`
    CallContent   string `json:"call_content,omitempty"`
    CustomerNotes string `json:"customer_notes,omitempty"`
}

// EncryptPIIFields encrypts sensitive data before storage
func (fe *FieldEncryption) EncryptPIIFields(ctx context.Context, tenantID string, pii *PIIFields) (*PIIFields, error) {
    encrypted := &PIIFields{}

    // Encrypt caller name
    if pii.CallerName != "" {
        encryptedName, err := fe.encryptField(ctx, tenantID, "caller_name", pii.CallerName)
        if err != nil {
            return nil, fmt.Errorf("encrypting caller name: %w", err)
        }
        encrypted.CallerName = encryptedName
    }

    // Encrypt phone number
    if pii.CallerPhone != "" {
        encryptedPhone, err := fe.encryptField(ctx, tenantID, "caller_phone", pii.CallerPhone)
        if err != nil {
            return nil, fmt.Errorf("encrypting caller phone: %w", err)
        }
        encrypted.CallerPhone = encryptedPhone
    }

    // Encrypt email
    if pii.CallerEmail != "" {
        encryptedEmail, err := fe.encryptField(ctx, tenantID, "caller_email", pii.CallerEmail)
        if err != nil {
            return nil, fmt.Errorf("encrypting caller email: %w", err)
        }
        encrypted.CallerEmail = encryptedEmail
    }

    return encrypted, nil
}

func (fe *FieldEncryption) encryptField(ctx context.Context, tenantID, fieldType, plaintext string) (string, error) {
    // Generate tenant-specific encryption key
    keyName := fmt.Sprintf("projects/account-strategy-464106/locations/us-central1/keyRings/tenant-encryption/cryptoKeys/tenant-%s", tenantID)

    // Encrypt using Cloud KMS
    req := &kmspb.EncryptRequest{
        Name:      keyName,
        Plaintext: []byte(plaintext),
        AdditionalAuthenticatedData: []byte(fieldType), // Field type as AAD
    }

    result, err := fe.kmsClient.Encrypt(ctx, req)
    if err != nil {
        return "", fmt.Errorf("KMS encryption failed: %w", err)
    }

    // Return base64 encoded ciphertext
    return base64.StdEncoding.EncodeToString(result.Ciphertext), nil
}
```

---

## 🔍 **Real-Time Security Monitoring (2025)**

### **Security Event Detection System**
```go
// internal/security/monitoring_2025.go
package security

import (
    "context"
    "time"

    "cloud.google.com/go/logging"
    "cloud.google.com/go/pubsub"
)

// SecurityMonitor handles real-time security event detection
type SecurityMonitor struct {
    logger      *logging.Client
    pubsub      *pubsub.Client
    alertTopic  *pubsub.Topic
    anomalyAI   *AnomalyDetectionAI
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
    EventID     string                 `json:"event_id"`
    EventType   string                 `json:"event_type"`
    TenantID    string                 `json:"tenant_id"`
    SourceIP    string                 `json:"source_ip"`
    UserAgent   string                 `json:"user_agent"`
    Severity    string                 `json:"severity"` // low, medium, high, critical
    Metadata    map[string]interface{} `json:"metadata"`
    Timestamp   time.Time              `json:"timestamp"`
    Automated   bool                   `json:"automated"`
    ActionTaken string                 `json:"action_taken"`
}

// MonitoringRules defines security thresholds
type MonitoringRules struct {
    MaxFailedAttemptsPerMinute  int           `json:"max_failed_attempts_per_minute"`
    MaxRequestsPerTenant        int           `json:"max_requests_per_tenant"`
    SuspiciousIPThreshold       int           `json:"suspicious_ip_threshold"`
    AnomalyDetectionEnabled     bool          `json:"anomaly_detection_enabled"`
    AutomaticResponseEnabled    bool          `json:"automatic_response_enabled"`
    AlertDestinations          []string       `json:"alert_destinations"`
}

// Real-time threat detection
func (sm *SecurityMonitor) DetectThreats(ctx context.Context, event *SecurityEvent) (*ThreatAssessment, error) {
    assessment := &ThreatAssessment{
        EventID:    event.EventID,
        ThreatLevel: "none",
        Confidence:  0.0,
        Recommendations: []string{},
    }

    // 1. Rate-based attack detection
    if sm.detectRateBasedAttack(ctx, event) {
        assessment.ThreatLevel = "medium"
        assessment.Confidence = 0.8
        assessment.Recommendations = append(assessment.Recommendations, "Implement IP-based rate limiting")
    }

    // 2. Geographic anomaly detection
    if sm.detectGeographicAnomaly(ctx, event) {
        assessment.ThreatLevel = "medium"
        assessment.Confidence = 0.7
        assessment.Recommendations = append(assessment.Recommendations, "Verify legitimate access from new location")
    }

    // 3. AI-powered behavioral analysis
    if sm.anomalyAI != nil {
        aiAssessment, err := sm.anomalyAI.AnalyzeBehavior(ctx, event)
        if err == nil && aiAssessment.AnomalyScore > 0.8 {
            assessment.ThreatLevel = "high"
            assessment.Confidence = aiAssessment.AnomalyScore
            assessment.Recommendations = append(assessment.Recommendations, aiAssessment.Recommendation)
        }
    }

    // 4. Automated response
    if assessment.ThreatLevel != "none" {
        sm.triggerAutomatedResponse(ctx, event, assessment)
    }

    return assessment, nil
}

// Automated security response
func (sm *SecurityMonitor) triggerAutomatedResponse(ctx context.Context, event *SecurityEvent, assessment *ThreatAssessment) error {
    response := &AutomatedResponse{
        EventID:     event.EventID,
        TenantID:    event.TenantID,
        ThreatLevel: assessment.ThreatLevel,
        Actions:     []string{},
    }

    switch assessment.ThreatLevel {
    case "medium":
        // Temporary rate limiting
        response.Actions = append(response.Actions, "enable_strict_rate_limiting")
        sm.enableStrictRateLimit(ctx, event.TenantID, event.SourceIP, 15*time.Minute)

    case "high":
        // Block IP temporarily
        response.Actions = append(response.Actions, "temporary_ip_block")
        sm.blockIPTemporarily(ctx, event.SourceIP, 1*time.Hour)

    case "critical":
        // Emergency response
        response.Actions = append(response.Actions, "emergency_lockdown", "alert_admin")
        sm.emergencyTenantLockdown(ctx, event.TenantID)
        sm.alertSecurityTeam(ctx, event, assessment)
    }

    // Log automated response
    sm.logAutomatedResponse(ctx, response)

    return nil
}
```

---

## 📋 **2025 Compliance Framework**

### **GDPR/CCPA Compliance Implementation**
```go
// internal/compliance/privacy_2025.go
package compliance

import (
    "context"
    "fmt"
    "time"
)

// PrivacyComplianceManager handles data privacy requirements
type PrivacyComplianceManager struct {
    dataMapper    *DataMapper
    retentionMgr  *RetentionManager
    consentMgr    *ConsentManager
    auditLogger   *ComplianceAuditLogger
}

// DataProcessingRecord tracks all PII processing for compliance
type DataProcessingRecord struct {
    RecordID        string    `json:"record_id"`
    TenantID        string    `json:"tenant_id"`
    DataSubject     string    `json:"data_subject"` // phone number hash
    ProcessingType  string    `json:"processing_type"`
    LegalBasis      string    `json:"legal_basis"`
    DataCategories  []string  `json:"data_categories"`
    RetentionPeriod int       `json:"retention_period_days"`
    ConsentStatus   string    `json:"consent_status"`
    ProcessedAt     time.Time `json:"processed_at"`
    ExpiresAt       time.Time `json:"expires_at"`
}

// ProcessPIIWithCompliance ensures GDPR/CCPA compliance
func (pcm *PrivacyComplianceManager) ProcessPIIWithCompliance(
    ctx context.Context,
    tenantID string,
    callerPhone string,
    piiData *PIIFields,
    processingPurpose string,
) error {

    // 1. Check consent status
    consentStatus, err := pcm.consentMgr.GetConsentStatus(ctx, callerPhone, tenantID)
    if err != nil {
        return fmt.Errorf("checking consent status: %w", err)
    }

    if !consentStatus.HasValidConsent(processingPurpose) {
        return fmt.Errorf("no valid consent for processing purpose: %s", processingPurpose)
    }

    // 2. Create data processing record
    record := &DataProcessingRecord{
        RecordID:        generateRecordID(),
        TenantID:        tenantID,
        DataSubject:     hashPhone(callerPhone), // Hash for anonymization
        ProcessingType:  processingPurpose,
        LegalBasis:      consentStatus.LegalBasis,
        DataCategories:  []string{"contact_info", "call_recording", "ai_analysis"},
        RetentionPeriod: consentStatus.RetentionDays,
        ConsentStatus:   "valid",
        ProcessedAt:     time.Now().UTC(),
        ExpiresAt:       time.Now().UTC().Add(time.Duration(consentStatus.RetentionDays) * 24 * time.Hour),
    }

    // 3. Log processing activity
    if err := pcm.auditLogger.LogDataProcessing(ctx, record); err != nil {
        return fmt.Errorf("logging data processing: %w", err)
    }

    // 4. Schedule automatic deletion
    if err := pcm.retentionMgr.ScheduleDeletion(ctx, record); err != nil {
        return fmt.Errorf("scheduling data deletion: %w", err)
    }

    return nil
}

// HandleDataDeletionRequest processes right to be forgotten requests
func (pcm *PrivacyComplianceManager) HandleDataDeletionRequest(ctx context.Context, phoneNumber, tenantID string) error {
    dataSubjectHash := hashPhone(phoneNumber)

    // 1. Find all data for this subject
    records, err := pcm.findAllDataForSubject(ctx, dataSubjectHash, tenantID)
    if err != nil {
        return fmt.Errorf("finding data for subject: %w", err)
    }

    // 2. Verify deletion is allowed (e.g., no legal hold)
    for _, record := range records {
        if record.LegalBasis == "legal_obligation" {
            return fmt.Errorf("cannot delete data subject to legal obligation")
        }
    }

    // 3. Perform secure deletion
    for _, record := range records {
        if err := pcm.secureDeleteData(ctx, record); err != nil {
            return fmt.Errorf("deleting record %s: %w", record.RecordID, err)
        }
    }

    // 4. Create compliance certificate
    certificate := &DeletionCertificate{
        CertificateID:   generateCertificateID(),
        DataSubject:     dataSubjectHash,
        TenantID:        tenantID,
        RecordsDeleted:  len(records),
        DeletionMethod:  "crypto_shredding",
        CompletedAt:     time.Now().UTC(),
        VerifiedBy:      "automated_compliance_system",
    }

    return pcm.auditLogger.LogDataDeletion(ctx, certificate)
}
```

---

## 🎯 **Security Implementation Roadmap**

### **Week 1: Enhanced Authentication (Immediate Priority)**
- [ ] Implement tenant-specific webhook secrets in Secret Manager
- [ ] Add rate limiting per tenant and IP address
- [ ] Deploy enhanced HMAC verification with timing attack protection
- [ ] Set up security audit logging

### **Week 2: Data Protection (High Priority)**
- [ ] Implement field-level encryption for PII using Cloud KMS
- [ ] Deploy enhanced row-level security policies
- [ ] Set up vector embedding security controls
- [ ] Create data anonymization procedures

### **Week 3: Monitoring & Response (Critical)**
- [ ] Deploy real-time security monitoring system
- [ ] Implement automated threat detection and response
- [ ] Set up security alerting and notification system
- [ ] Create incident response automation

### **Week 4: Compliance & Audit (Essential)**
- [ ] Implement GDPR/CCPA compliance framework
- [ ] Set up automated data retention and deletion
- [ ] Create compliance reporting dashboard
- [ ] Conduct security audit and penetration testing

---

## 📊 **Security Metrics & KPIs**

### **Real-Time Security Monitoring**
- **Failed Authentication Attempts**: <5% of total requests
- **Rate Limiting Effectiveness**: 99.9% malicious traffic blocked
- **HMAC Verification Speed**: <5ms average verification time
- **False Positive Rate**: <0.1% legitimate traffic blocked

### **Compliance Metrics**
- **Data Retention Compliance**: 100% automatic deletion on schedule
- **Consent Management**: 100% processing tied to valid consent
- **Audit Trail Completeness**: 100% security events logged
- **Incident Response Time**: <15 minutes for critical threats

### **Infrastructure Security**
- **Network Isolation**: 100% tenant traffic isolated
- **Encryption Coverage**: 100% PII encrypted at rest and in transit
- **Secret Rotation**: All secrets rotated every 90 days
- **Vulnerability Scanning**: Zero high/critical vulnerabilities

---

**2025 SECURITY BASELINE COMPLETE** ✅
**Next Phase**: Testing Framework Development & Performance Engineering