# Security Review Checklist

**Document Version**: 1.0
**Date**: September 13, 2025
**Project**: Multi-Tenant CallRail Integration Pipeline
**Classification**: CONFIDENTIAL

---

## 🎯 **Security Review Overview**

This document provides comprehensive security review guidelines for the multi-tenant ingestion pipeline. All code touching security-sensitive areas must pass this checklist before deployment.

---

## 🔐 **Authentication & Authorization**

### **HMAC Signature Verification**

#### **Implementation Requirements**
- [ ] HMAC-SHA256 is used for webhook signature verification
- [ ] Constant-time comparison prevents timing attacks
- [ ] Signature validation occurs before any data processing
- [ ] Multiple signature formats are supported (hex, base64)
- [ ] Invalid signatures are logged for monitoring

```go
// ✅ Secure Implementation
func VerifyHMACSignature(payload []byte, signature, secret string) error {
    // Handle different signature formats
    expectedSig := strings.TrimPrefix(signature, "sha256=")

    // Compute HMAC-SHA256
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    computedHash := hex.EncodeToString(mac.Sum(nil))

    // Constant-time comparison - CRITICAL for security
    if !hmac.Equal([]byte(expectedSig), []byte(computedHash)) {
        return ErrInvalidSignature
    }

    return nil
}

// ❌ Insecure - Timing attack vulnerability
if expectedSig == computedHash {
    return nil
}
```

#### **Security Checklist**
- [ ] Uses `crypto/hmac` package for verification
- [ ] Uses `hmac.Equal()` for constant-time comparison
- [ ] Secret is retrieved from Google Secret Manager
- [ ] Secret rotation is supported
- [ ] Failed verifications are rate-limited
- [ ] Signature verification timing is consistent

### **Tenant Authentication**

#### **Tenant Isolation Validation**
- [ ] Every database query includes tenant_id in WHERE clause
- [ ] Tenant context is validated before any data access
- [ ] Cross-tenant data access is impossible
- [ ] Admin operations require additional authorization
- [ ] Tenant deactivation immediately blocks access

```go
// ✅ Secure Tenant Validation
func (s *Service) GetCallData(ctx context.Context, tenantID, callID string) (*CallData, error) {
    // Validate tenant context
    if err := ValidateTenantAccess(ctx, tenantID); err != nil {
        return nil, fmt.Errorf("tenant access denied: %w", err)
    }

    // Query with tenant isolation
    stmt := spanner.Statement{
        SQL: `SELECT call_id, tenant_id, transcription_data
              FROM calls
              WHERE tenant_id = @tenant_id AND call_id = @call_id`,
        Params: map[string]interface{}{
            "tenant_id": tenantID,
            "call_id":   callID,
        },
    }
    // Execute query...
}
```

#### **Access Control Matrix**
| Resource | Read | Write | Delete | Admin |
|----------|------|-------|--------|-------|
| Own Tenant Data | ✅ | ✅ | ✅ | ✅ |
| Other Tenant Data | ❌ | ❌ | ❌ | ❌ |
| System Config | ❌ | ❌ | ❌ | ✅ |
| Audit Logs | ❌ | ❌ | ❌ | ✅ |

---

## 🛡️ **Input Validation & Sanitization**

### **Webhook Payload Validation**

#### **Validation Requirements**
- [ ] JSON schema validation for all webhook payloads
- [ ] String length limits enforced
- [ ] Numeric ranges validated
- [ ] Email/phone format validation
- [ ] URL validation for recording URLs
- [ ] Character encoding validation (UTF-8)

```go
// ✅ Comprehensive Validation
type CallRailWebhook struct {
    CallID           string `json:"call_id" validate:"required,min=3,max=50,alphanum"`
    TenantID         string `json:"tenant_id" validate:"required,uuid4"`
    CallRailCompanyID string `json:"callrail_company_id" validate:"required,numeric,max=20"`
    CallerID         string `json:"caller_id" validate:"required,e164"`
    Duration         int    `json:"duration" validate:"min=0,max=86400"`
    RecordingURL     string `json:"recording_url" validate:"required,url,max=500"`
    CallerName       string `json:"caller_name" validate:"max=100"`
    Notes            string `json:"notes" validate:"max=1000"`
}

func ValidateWebhook(webhook *CallRailWebhook) error {
    validate := validator.New()

    // Register custom validators
    validate.RegisterValidation("e164", validateE164)

    if err := validate.Struct(webhook); err != nil {
        return fmt.Errorf("webhook validation failed: %w", err)
    }

    // Additional business logic validation
    if err := validateTenantCallRailMapping(webhook.TenantID, webhook.CallRailCompanyID); err != nil {
        return fmt.Errorf("tenant mapping validation failed: %w", err)
    }

    return nil
}
```

### **SQL Injection Prevention**

#### **Database Query Security**
- [ ] ALL queries use parameterized statements
- [ ] No string concatenation in SQL queries
- [ ] Dynamic queries are avoided or carefully reviewed
- [ ] SQL query logging excludes sensitive parameters
- [ ] Database user has minimal required permissions

```go
// ✅ Secure - Parameterized Query
func (r *Repository) GetTenantCalls(ctx context.Context, tenantID string, startDate time.Time) ([]*Call, error) {
    stmt := spanner.Statement{
        SQL: `SELECT call_id, tenant_id, caller_id, duration, created_at
              FROM calls
              WHERE tenant_id = @tenant_id
              AND created_at >= @start_date
              ORDER BY created_at DESC
              LIMIT 100`,
        Params: map[string]interface{}{
            "tenant_id":   tenantID,
            "start_date":  startDate,
        },
    }
    // Execute safely...
}

// ❌ Vulnerable - SQL Injection Risk
query := fmt.Sprintf("SELECT * FROM calls WHERE tenant_id = '%s'", tenantID)
```

### **File Upload Security**

#### **Audio File Validation**
- [ ] File type validation (audio formats only)
- [ ] File size limits enforced (max 100MB)
- [ ] Virus scanning for uploaded files
- [ ] File content validation (not just extension)
- [ ] Secure file storage with access controls

```go
// ✅ Secure File Validation
func ValidateAudioFile(file io.Reader, filename string) error {
    // Check file extension
    allowedExts := []string{".mp3", ".wav", ".m4a", ".flac"}
    ext := strings.ToLower(filepath.Ext(filename))
    if !contains(allowedExts, ext) {
        return errors.New("invalid file type")
    }

    // Read first few bytes to validate file signature
    header := make([]byte, 512)
    n, err := file.Read(header)
    if err != nil {
        return fmt.Errorf("reading file header: %w", err)
    }

    // Validate MIME type
    mimeType := http.DetectContentType(header[:n])
    if !strings.HasPrefix(mimeType, "audio/") {
        return errors.New("file is not an audio file")
    }

    return nil
}
```

---

## 🔑 **Secret Management**

### **Google Secret Manager Integration**

#### **Secret Handling Requirements**
- [ ] All secrets stored in Google Secret Manager
- [ ] Secrets accessed with least privilege IAM
- [ ] Secret versions are managed properly
- [ ] Secrets are cached with appropriate TTL
- [ ] Secret access is logged and monitored

```go
// ✅ Secure Secret Management
type SecretManager struct {
    client *secretmanager.Client
    cache  *secretCache
}

func (sm *SecretManager) GetSecret(ctx context.Context, secretName string) (string, error) {
    // Check cache first
    if value, found := sm.cache.Get(secretName); found {
        return value, nil
    }

    // Construct full secret path
    name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest",
        sm.projectID, secretName)

    // Access secret
    req := &secretmanagerpb.AccessSecretVersionRequest{Name: name}
    result, err := sm.client.AccessSecretVersion(ctx, req)
    if err != nil {
        return "", fmt.Errorf("accessing secret %s: %w", secretName, err)
    }

    secretValue := string(result.Payload.Data)

    // Cache with TTL
    sm.cache.Set(secretName, secretValue, 15*time.Minute)

    return secretValue, nil
}
```

#### **Secret Security Checklist**
- [ ] Secrets never logged or exposed in errors
- [ ] Secret rotation is automated
- [ ] Development uses separate secrets from production
- [ ] Secret access requires authentication
- [ ] Failed secret access attempts are monitored

### **API Key Management**

#### **CallRail API Key Security**
- [ ] API keys stored encrypted in database
- [ ] API keys retrieved on-demand, not cached long-term
- [ ] API key access logged for audit
- [ ] Invalid API keys trigger alerts
- [ ] API key rotation procedures documented

---

## 🌐 **Network Security**

### **HTTPS/TLS Configuration**

#### **Transport Security Requirements**
- [ ] All external communications use HTTPS/TLS 1.2+
- [ ] Certificate validation is enforced
- [ ] Weak cipher suites are disabled
- [ ] HSTS headers are set
- [ ] Certificate rotation is automated

### **Cloud Run Security**

#### **Service Configuration**
- [ ] Private Google Access is enabled
- [ ] VPC connector configured for internal traffic
- [ ] IAM permissions follow least privilege
- [ ] Service identity is properly configured
- [ ] External access is restricted where possible

```yaml
# ✅ Secure Cloud Run Configuration
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: webhook-processor
  annotations:
    run.googleapis.com/ingress: all
    run.googleapis.com/vpc-access-connector: vpc-connector
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/execution-environment: gen2
        run.googleapis.com/cpu-throttling: "false"
    spec:
      serviceAccountName: webhook-processor@project.iam.gserviceaccount.com
      containers:
      - image: gcr.io/project/webhook-processor
        env:
        - name: GOOGLE_CLOUD_PROJECT
          value: "account-strategy-464106"
        resources:
          limits:
            cpu: "2"
            memory: "2Gi"
```

---

## 📊 **Data Protection**

### **Encryption Standards**

#### **Data at Rest**
- [ ] Database encryption enabled (CMEK preferred)
- [ ] Cloud Storage encryption enabled
- [ ] Application-level encryption for sensitive fields
- [ ] Encryption keys managed securely
- [ ] Key rotation procedures implemented

#### **Data in Transit**
- [ ] TLS 1.2+ for all communications
- [ ] Certificate pinning where applicable
- [ ] API communications encrypted end-to-end
- [ ] Internal service mesh encryption
- [ ] Webhook payloads protected in transit

### **PII/PHI Protection**

#### **Personal Data Handling**
- [ ] PII is identified and classified
- [ ] Data minimization principles applied
- [ ] Retention policies implemented
- [ ] Data anonymization/pseudonymization used
- [ ] GDPR compliance maintained

```go
// ✅ PII Protection Example
type CustomerData struct {
    ID          string    `json:"id"`
    PhoneHash   string    `json:"phone_hash"`   // Hashed, not plain text
    NameEnc     string    `json:"name_enc"`     // Encrypted
    Location    string    `json:"location"`     // City only, not full address
    CreatedAt   time.Time `json:"created_at"`
}

func (c *CustomerData) SetPhone(phone string) {
    // Hash phone number for privacy
    hasher := sha256.New()
    hasher.Write([]byte(phone))
    c.PhoneHash = hex.EncodeToString(hasher.Sum(nil))
}
```

---

## 🚨 **Security Monitoring & Logging**

### **Audit Logging Requirements**

#### **Events to Log**
- [ ] Authentication attempts (success/failure)
- [ ] Authorization failures
- [ ] Data access patterns
- [ ] Configuration changes
- [ ] Admin operations
- [ ] Security incidents

```go
// ✅ Security Event Logging
func (s *SecurityLogger) LogAuthEvent(ctx context.Context, event AuthEvent) {
    fields := []zap.Field{
        zap.String("event_type", "authentication"),
        zap.String("tenant_id", event.TenantID),
        zap.String("source_ip", event.SourceIP),
        zap.String("user_agent", event.UserAgent),
        zap.Bool("success", event.Success),
        zap.String("failure_reason", event.FailureReason),
        zap.Time("timestamp", time.Now()),
    }

    if event.Success {
        s.logger.Info("authentication successful", fields...)
    } else {
        s.logger.Warn("authentication failed", fields...)
        // Trigger security monitoring alert
        s.alertManager.TriggerAlert("auth_failure", event)
    }
}
```

### **Intrusion Detection**

#### **Anomaly Detection**
- [ ] Failed authentication rate monitoring
- [ ] Unusual access patterns detection
- [ ] Data exfiltration monitoring
- [ ] API abuse detection
- [ ] Automated incident response

#### **Security Metrics**
- [ ] Authentication failure rates
- [ ] Invalid signature attempts
- [ ] Cross-tenant access attempts
- [ ] Privilege escalation attempts
- [ ] Suspicious file uploads

---

## 🔍 **Vulnerability Management**

### **Dependency Security**

#### **Third-Party Libraries**
- [ ] Dependency scanning in CI/CD pipeline
- [ ] Known vulnerabilities are tracked
- [ ] Security patches applied promptly
- [ ] License compliance verified
- [ ] Supply chain security maintained

```bash
# ✅ Automated Security Scanning
# Add to CI/CD pipeline
go mod download
gosec ./...
nancy sleuth
govulncheck ./...
```

### **Code Security Scanning**

#### **Static Analysis**
- [ ] `gosec` security scanner integrated
- [ ] Custom security rules configured
- [ ] False positives are documented
- [ ] Security findings are triaged
- [ ] Baseline security metrics established

#### **Dynamic Analysis**
- [ ] Runtime security monitoring
- [ ] Penetration testing scheduled
- [ ] Security regression testing
- [ ] Fuzzing for input validation
- [ ] Performance under attack scenarios

---

## 📋 **Security Review Process**

### **Pre-Deployment Security Gates**

#### **Automated Security Checks**
- [ ] All security scanners pass
- [ ] No high/critical vulnerabilities
- [ ] Dependency vulnerabilities resolved
- [ ] Security test suite passes
- [ ] Configuration security validated

#### **Manual Security Review**
- [ ] Security-sensitive code manually reviewed
- [ ] Threat model updated if needed
- [ ] Security architecture validated
- [ ] Compliance requirements met
- [ ] Security documentation updated

### **Security Incident Response**

#### **Incident Handling**
- [ ] Security incident response plan exists
- [ ] Contact information is current
- [ ] Escalation procedures defined
- [ ] Communication templates prepared
- [ ] Recovery procedures documented

#### **Post-Incident Actions**
- [ ] Root cause analysis conducted
- [ ] Security controls updated
- [ ] Monitoring rules enhanced
- [ ] Training needs identified
- [ ] Process improvements implemented

---

## 🎯 **Security Compliance**

### **Regulatory Requirements**

#### **Data Protection Regulations**
- [ ] GDPR compliance for EU data
- [ ] CCPA compliance for California residents
- [ ] SOC 2 Type II requirements met
- [ ] Industry-specific regulations addressed
- [ ] Data processing agreements in place

#### **Security Frameworks**
- [ ] NIST Cybersecurity Framework alignment
- [ ] ISO 27001 controls implemented
- [ ] CIS Controls applied
- [ ] Cloud security best practices followed
- [ ] Zero-trust principles applied

### **Security Metrics & KPIs**

#### **Security Performance Indicators**
- [ ] Mean time to detect (MTTD) incidents
- [ ] Mean time to respond (MTTR) to incidents
- [ ] Security test coverage percentage
- [ ] Vulnerability remediation time
- [ ] Security awareness training completion

---

## ⚠️ **Critical Security Failures**

### **Immediate Deployment Blockers**
- [ ] Hardcoded secrets or credentials
- [ ] SQL injection vulnerabilities
- [ ] Cross-tenant data exposure
- [ ] Missing authentication on sensitive endpoints
- [ ] Unencrypted sensitive data transmission

### **High Priority Security Issues**
- [ ] Weak encryption algorithms
- [ ] Insufficient input validation
- [ ] Missing security headers
- [ ] Inadequate error handling (information leakage)
- [ ] Insecure direct object references

### **Security Review Sign-off**

**Security Reviewer**: ___________________ **Date**: ___________

**Security Architect**: ___________________ **Date**: ___________

**Approved for Production**: ☐ Yes ☐ No ☐ Conditional

**Conditions (if applicable)**:
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________

---

This security checklist ensures that the multi-tenant ingestion pipeline maintains the highest security standards and protects sensitive customer data throughout the processing workflow.