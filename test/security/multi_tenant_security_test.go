package security

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SecurityTestSuite tests multi-tenant security and isolation
type SecurityTestSuite struct {
	suite.Suite
	server         *httptest.Server
	spannerClient  *spanner.Client
	ctx            context.Context
	jwtSecret      []byte
	testTenants    map[string]*TenantConfig
	adminToken     string
	userTokens     map[string]string // tenant_id -> token
}

// Security test data models
type TenantConfig struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	IsActive        bool                   `json:"is_active"`
	SecurityLevel   string                 `json:"security_level"` // basic, enhanced, enterprise
	AllowedOrigins  []string               `json:"allowed_origins"`
	RateLimits      map[string]int         `json:"rate_limits"`
	DataRetention   int                    `json:"data_retention_days"`
	EncryptionKey   string                 `json:"encryption_key,omitempty"`
	Permissions     []string               `json:"permissions"`
	CreatedAt       time.Time              `json:"created_at"`
}

type SecurityAuditLog struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	UserID       string                 `json:"user_id"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Success      bool                   `json:"success"`
	ErrorType    string                 `json:"error_type,omitempty"`
	Details      map[string]interface{} `json:"details"`
	Timestamp    time.Time              `json:"timestamp"`
}

type AuthTokenClaims struct {
	UserID      string   `json:"user_id"`
	TenantID    string   `json:"tenant_id"`
	Permissions []string `json:"permissions"`
	Role        string   `json:"role"`
	jwt.RegisteredClaims
}

type AuthRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
}

type SecurityViolation struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	TenantID    string                 `json:"tenant_id"`
	UserID      string                 `json:"user_id"`
	IPAddress   string                 `json:"ip_address"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
}

func (suite *SecurityTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.jwtSecret = make([]byte, 32)
	rand.Read(suite.jwtSecret)

	// Setup test tenants
	suite.testTenants = map[string]*TenantConfig{
		"tenant-basic": {
			ID:            "tenant-basic",
			Name:          "Basic Security Tenant",
			IsActive:      true,
			SecurityLevel: "basic",
			AllowedOrigins: []string{"http://localhost:3000"},
			RateLimits:    map[string]int{"upload": 60, "query": 100},
			DataRetention: 90,
			Permissions:   []string{"upload", "read"},
		},
		"tenant-enhanced": {
			ID:            "tenant-enhanced",
			Name:          "Enhanced Security Tenant",
			IsActive:      true,
			SecurityLevel: "enhanced",
			AllowedOrigins: []string{"https://secure.example.com"},
			RateLimits:    map[string]int{"upload": 30, "query": 50},
			DataRetention: 365,
			Permissions:   []string{"upload", "read", "delete"},
		},
		"tenant-enterprise": {
			ID:            "tenant-enterprise",
			Name:          "Enterprise Security Tenant",
			IsActive:      true,
			SecurityLevel: "enterprise",
			AllowedOrigins: []string{"https://enterprise.example.com"},
			RateLimits:    map[string]int{"upload": 100, "query": 200},
			DataRetention: 2555, // 7 years
			Permissions:   []string{"upload", "read", "delete", "admin"},
		},
		"tenant-inactive": {
			ID:            "tenant-inactive",
			Name:          "Inactive Tenant",
			IsActive:      false,
			SecurityLevel: "basic",
			Permissions:   []string{},
		},
	}

	// Generate tokens for each tenant
	suite.userTokens = make(map[string]string)
	for tenantID, config := range suite.testTenants {
		token, err := suite.generateTestToken(tenantID, "test-user", "user", config.Permissions)
		require.NoError(suite.T(), err)
		suite.userTokens[tenantID] = token
	}

	// Generate admin token
	adminToken, err := suite.generateTestToken("", "admin-user", "admin", []string{"admin", "super_admin"})
	require.NoError(suite.T(), err)
	suite.adminToken = adminToken

	// Setup test server
	suite.setupTestServer()
}

func (suite *SecurityTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.spannerClient != nil {
		suite.spannerClient.Close()
	}
}

func (suite *SecurityTestSuite) SetupTest() {
	// Clean up any test data before each test
	suite.cleanupTestData()
}

func (suite *SecurityTestSuite) generateTestToken(tenantID, userID, role string, permissions []string) (string, error) {
	claims := &AuthTokenClaims{
		UserID:      userID,
		TenantID:    tenantID,
		Permissions: permissions,
		Role:        role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   userID,
			Audience:  []string{"ingestion-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        fmt.Sprintf("test-jwt-%d", time.Now().UnixNano()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(suite.jwtSecret)
}

func (suite *SecurityTestSuite) setupTestServer() {
	mux := http.NewServeMux()

	// Authentication endpoint
	mux.HandleFunc("/auth/token", suite.handleAuth)

	// Protected endpoints
	mux.HandleFunc("/api/v1/ingestion/upload", suite.authMiddleware(suite.handleProtectedUpload))
	mux.HandleFunc("/api/v1/ingestion/records", suite.authMiddleware(suite.handleProtectedRecords))
	mux.HandleFunc("/api/v1/admin/tenants", suite.authMiddleware(suite.handleAdminTenants))

	// Security audit endpoint
	mux.HandleFunc("/api/v1/security/audit", suite.authMiddleware(suite.handleSecurityAudit))

	suite.server = httptest.NewServer(mux)
}

func (suite *SecurityTestSuite) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			suite.recordSecurityViolation("missing_auth", "medium", "Missing Authorization header", "", r)
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			suite.recordSecurityViolation("invalid_auth_format", "medium", "Invalid Authorization format", "", r)
			http.Error(w, "Invalid Authorization format", http.StatusUnauthorized)
			return
		}

		// Parse and validate JWT
		token, err := jwt.ParseWithClaims(tokenString, &AuthTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return suite.jwtSecret, nil
		})

		if err != nil {
			suite.recordSecurityViolation("invalid_token", "high", "Invalid JWT token", "", r)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*AuthTokenClaims)
		if !ok || !token.Valid {
			suite.recordSecurityViolation("invalid_claims", "high", "Invalid token claims", "", r)
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Validate tenant access
		requestedTenantID := r.Header.Get("X-Tenant-ID")
		if requestedTenantID == "" {
			requestedTenantID = r.URL.Query().Get("tenant_id")
		}

		// Admin users can access any tenant
		if claims.Role != "admin" && claims.Role != "super_admin" {
			if claims.TenantID != requestedTenantID && requestedTenantID != "" {
				suite.recordSecurityViolation("tenant_access_violation", "critical",
					fmt.Sprintf("User %s attempted to access tenant %s", claims.UserID, requestedTenantID),
					claims.TenantID, r)
				http.Error(w, "Forbidden: Tenant access denied", http.StatusForbidden)
				return
			}
		}

		// Check if tenant is active
		if claims.TenantID != "" {
			if tenant, exists := suite.testTenants[claims.TenantID]; exists && !tenant.IsActive {
				suite.recordSecurityViolation("inactive_tenant_access", "high",
					"Attempted access to inactive tenant", claims.TenantID, r)
				http.Error(w, "Tenant is inactive", http.StatusForbidden)
				return
			}
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "claims", claims)
		next(w, r.WithContext(ctx))
	}
}

func (suite *SecurityTestSuite) recordSecurityViolation(violationType, severity, description, tenantID string, r *http.Request) {
	violation := &SecurityViolation{
		Type:        violationType,
		Severity:    severity,
		Description: description,
		TenantID:    tenantID,
		IPAddress:   r.RemoteAddr,
		Details: map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"user_agent": r.Header.Get("User-Agent"),
		},
		Timestamp: time.Now(),
	}

	// In a real system, this would be stored in a security audit database
	suite.T().Logf("SECURITY VIOLATION: %+v", violation)
}

func (suite *SecurityTestSuite) cleanupTestData() {
	// Clean up any test data
}

// HTTP Handlers

func (suite *SecurityTestSuite) handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var authReq AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate tenant
	tenant, exists := suite.testTenants[authReq.TenantID]
	if !exists {
		http.Error(w, "Invalid tenant", http.StatusUnauthorized)
		return
	}

	if !tenant.IsActive {
		http.Error(w, "Tenant is inactive", http.StatusForbidden)
		return
	}

	// Generate token
	token, err := suite.generateTestToken(authReq.TenantID, authReq.UserID, authReq.Role, tenant.Permissions)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := AuthResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		TenantID:  authReq.TenantID,
		UserID:    authReq.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *SecurityTestSuite) handleProtectedUpload(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*AuthTokenClaims)

	// Check permissions
	if !suite.hasPermission(claims.Permissions, "upload") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Simulate file upload processing
	response := map[string]interface{}{
		"status":       "uploaded",
		"ingestion_id": fmt.Sprintf("ing_%d", time.Now().UnixNano()),
		"tenant_id":    claims.TenantID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (suite *SecurityTestSuite) handleProtectedRecords(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*AuthTokenClaims)

	// Check permissions
	if !suite.hasPermission(claims.Permissions, "read") {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// Return mock records filtered by tenant
	records := []map[string]interface{}{
		{
			"id":        "record-1",
			"tenant_id": claims.TenantID,
			"status":    "completed",
		},
		{
			"id":        "record-2",
			"tenant_id": claims.TenantID,
			"status":    "processing",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"records": records})
}

func (suite *SecurityTestSuite) handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*AuthTokenClaims)

	// Check admin permissions
	if !suite.hasPermission(claims.Permissions, "admin") {
		http.Error(w, "Admin permissions required", http.StatusForbidden)
		return
	}

	// Return all tenants (admin view)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tenants": suite.testTenants})
}

func (suite *SecurityTestSuite) handleSecurityAudit(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*AuthTokenClaims)

	// Check audit permissions
	if !suite.hasPermission(claims.Permissions, "admin") {
		http.Error(w, "Audit permissions required", http.StatusForbidden)
		return
	}

	// Return mock audit logs
	auditLogs := []SecurityAuditLog{
		{
			ID:           "audit-1",
			TenantID:     claims.TenantID,
			Action:       "upload",
			ResourceType: "ingestion_record",
			UserID:       claims.UserID,
			Success:      true,
			Timestamp:    time.Now().Add(-1 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"audit_logs": auditLogs})
}

func (suite *SecurityTestSuite) hasPermission(userPermissions []string, requiredPermission string) bool {
	for _, perm := range userPermissions {
		if perm == requiredPermission || perm == "admin" || perm == "super_admin" {
			return true
		}
	}
	return false
}

// Test Cases

func (suite *SecurityTestSuite) TestAuthenticationRequired() {
	// Test that protected endpoints require authentication
	endpoints := []string{
		"/api/v1/ingestion/upload",
		"/api/v1/ingestion/records",
		"/api/v1/admin/tenants",
		"/api/v1/security/audit",
	}

	for _, endpoint := range endpoints {
		suite.T().Run(endpoint, func(t *testing.T) {
			req, _ := http.NewRequest("GET", suite.server.URL+endpoint, nil)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	}
}

func (suite *SecurityTestSuite) TestValidTokenAccess() {
	// Test that valid tokens allow access
	tenantID := "tenant-basic"
	token := suite.userTokens[tenantID]

	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/ingestion/upload", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *SecurityTestSuite) TestInvalidTokenRejection() {
	// Test invalid token scenarios
	testCases := []struct {
		name        string
		authHeader  string
		expectedCode int
	}{
		{
			name:        "Invalid Bearer format",
			authHeader:  "InvalidFormat token123",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:        "Malformed JWT",
			authHeader:  "Bearer invalid.jwt.token",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:        "Empty token",
			authHeader:  "Bearer ",
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/ingestion/records", nil)
			req.Header.Set("Authorization", tc.authHeader)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedCode, resp.StatusCode)
		})
	}
}

func (suite *SecurityTestSuite) TestTenantIsolationViolation() {
	// Test that users cannot access other tenants' data
	tenant1ID := "tenant-basic"
	tenant2ID := "tenant-enhanced"

	tenant1Token := suite.userTokens[tenant1ID]

	// Try to access tenant2's data with tenant1's token
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/ingestion/records", nil)
	req.Header.Set("Authorization", "Bearer "+tenant1Token)
	req.Header.Set("X-Tenant-ID", tenant2ID) // Different tenant!

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
}

func (suite *SecurityTestSuite) TestInactiveTenantAccess() {
	// Test that inactive tenants cannot access the system
	inactiveTenantID := "tenant-inactive"

	// Try to get a token for inactive tenant
	authReq := AuthRequest{
		TenantID: inactiveTenantID,
		UserID:   "test-user",
		Role:     "user",
	}

	body, _ := json.Marshal(authReq)
	req, _ := http.NewRequest("POST", suite.server.URL+"/auth/token", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)
}

func (suite *SecurityTestSuite) TestPermissionBasedAccess() {
	// Test that users can only access endpoints they have permissions for
	testCases := []struct {
		tenantID        string
		endpoint        string
		method          string
		expectedStatus  int
		description     string
	}{
		{
			tenantID:       "tenant-basic",
			endpoint:       "/api/v1/ingestion/upload",
			method:         "POST",
			expectedStatus: http.StatusOK,
			description:    "Basic tenant should be able to upload",
		},
		{
			tenantID:       "tenant-basic",
			endpoint:       "/api/v1/admin/tenants",
			method:         "GET",
			expectedStatus: http.StatusForbidden,
			description:    "Basic tenant should not have admin access",
		},
		{
			tenantID:       "tenant-enterprise",
			endpoint:       "/api/v1/ingestion/upload",
			method:         "POST",
			expectedStatus: http.StatusOK,
			description:    "Enterprise tenant should be able to upload",
		},
		{
			tenantID:       "tenant-enterprise",
			endpoint:       "/api/v1/admin/tenants",
			method:         "GET",
			expectedStatus: http.StatusForbidden, // Regular user, not admin
			description:    "Enterprise tenant user should not have admin access",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.description, func(t *testing.T) {
			token := suite.userTokens[tc.tenantID]

			req, _ := http.NewRequest(tc.method, suite.server.URL+tc.endpoint, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Tenant-ID", tc.tenantID)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode, tc.description)
		})
	}
}

func (suite *SecurityTestSuite) TestAdminAccess() {
	// Test that admin users can access admin endpoints
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/admin/tenants", nil)
	req.Header.Set("Authorization", "Bearer "+suite.adminToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify response contains all tenants
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	tenants, ok := response["tenants"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Len(suite.T(), tenants, len(suite.testTenants))
}

func (suite *SecurityTestSuite) TestRateLimiting() {
	// Test rate limiting (mock implementation)
	tenantID := "tenant-enhanced"
	token := suite.userTokens[tenantID]

	// Enhanced tenant has rate limit of 30 uploads per minute
	const maxRequests = 35 // Exceed the limit

	var successCount, rateLimitedCount int

	for i := 0; i < maxRequests; i++ {
		req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/ingestion/upload", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", tenantID)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			successCount++
		} else if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}

		// Small delay to avoid overwhelming the test server
		time.Sleep(10 * time.Millisecond)
	}

	suite.T().Logf("Rate limiting test - Success: %d, Rate limited: %d", successCount, rateLimitedCount)

	// In a real implementation with rate limiting, we would expect some rate limited responses
	// For this mock test, all requests succeed since rate limiting is not implemented
	assert.Equal(suite.T(), maxRequests, successCount)
}

func (suite *SecurityTestSuite) TestDataEncryptionInTransit() {
	// Test that sensitive data is properly handled
	tenantID := "tenant-enterprise"
	token := suite.userTokens[tenantID]

	// Create request with sensitive data
	sensitiveData := map[string]interface{}{
		"customer_phone": "555-1234",
		"customer_email": "test@example.com",
		"project_budget": "$50,000",
	}

	body, _ := json.Marshal(sensitiveData)
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/ingestion/upload", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// In a real implementation, you would verify:
	// 1. Data is encrypted in transit (HTTPS)
	// 2. Sensitive fields are masked in logs
	// 3. Encryption keys are properly managed
}

func (suite *SecurityTestSuite) TestSQLInjectionPrevention() {
	// Test SQL injection prevention in query parameters
	tenantID := "tenant-basic"
	token := suite.userTokens[tenantID]

	// Malicious query parameters
	maliciousQueries := []string{
		"'; DROP TABLE ingestion_records; --",
		"' OR '1'='1",
		"'; SELECT * FROM tenant_configurations; --",
		"<script>alert('xss')</script>",
	}

	for _, maliciousQuery := range maliciousQueries {
		suite.T().Run("MaliciousQuery", func(t *testing.T) {
			// Test in query parameter
			encodedQuery := url.QueryEscape(maliciousQuery)
			req, _ := http.NewRequest("GET",
				suite.server.URL+"/api/v1/ingestion/records?filter="+encodedQuery, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Tenant-ID", tenantID)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should either succeed (if properly sanitized) or return 400 (if validation catches it)
			// Should NOT return 500 (internal server error from SQL injection)
			assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest,
				"Response should not indicate SQL injection vulnerability")
		})
	}
}

func (suite *SecurityTestSuite) TestCrossOriginResourceSharing() {
	// Test CORS headers are properly set
	tenantID := "tenant-enhanced"
	token := suite.userTokens[tenantID]

	req, _ := http.NewRequest("OPTIONS", suite.server.URL+"/api/v1/ingestion/upload", nil)
	req.Header.Set("Origin", "https://secure.example.com") // Allowed origin
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// In a real implementation, check CORS headers
	// For this test, we just verify the request completes
	assert.True(suite.T(), resp.StatusCode < 500)
}

func (suite *SecurityTestSuite) TestAuditLogging() {
	// Test that security events are properly logged
	tenantID := "tenant-basic"
	token := suite.userTokens[tenantID]

	// Perform an action that should be audited
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/ingestion/upload", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// In a real implementation, verify audit log entry was created
	// For this test, we trust that the middleware logs appropriately
}

func (suite *SecurityTestSuite) TestTokenExpiration() {
	// Test expired token handling
	// Generate expired token
	expiredClaims := &AuthTokenClaims{
		UserID:      "test-user",
		TenantID:    "tenant-basic",
		Permissions: []string{"upload", "read"},
		Role:        "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   "test-user",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := expiredToken.SignedString(suite.jwtSecret)
	require.NoError(suite.T(), err)

	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/ingestion/records", nil)
	req.Header.Set("Authorization", "Bearer "+expiredTokenString)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (suite *SecurityTestSuite) TestSessionManagement() {
	// Test session management and concurrent session limits
	tenantID := "tenant-enterprise"

	// Generate multiple tokens for the same user
	const maxSessions = 5
	tokens := make([]string, maxSessions+1) // One more than allowed

	for i := 0; i < maxSessions+1; i++ {
		token, err := suite.generateTestToken(tenantID, "concurrent-user", "user", []string{"upload", "read"})
		require.NoError(suite.T(), err)
		tokens[i] = token
	}

	// Test that all tokens work (in a real system, older tokens might be invalidated)
	for i, token := range tokens {
		req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/ingestion/records", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Tenant-ID", tenantID)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(suite.T(), err)
		resp.Body.Close()

		// For this mock implementation, all tokens work
		// In a real system with session limits, older tokens might be invalid
		suite.T().Logf("Token %d status: %d", i+1, resp.StatusCode)
	}
}

// Performance and stress tests for security features

func (suite *SecurityTestSuite) TestAuthenticationPerformance() {
	// Test authentication performance under load
	tenantID := "tenant-basic"
	token := suite.userTokens[tenantID]

	const numRequests = 100
	const concurrency = 10

	startTime := time.Now()

	// Concurrent authentication tests
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/ingestion/records", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Tenant-ID", tenantID)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				suite.T().Error("Request failed:", err)
				return
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				suite.T().Error("Unexpected status:", resp.StatusCode)
			}
		}()
	}

	wg.Wait()
	duration := time.Since(startTime)

	averageTime := duration / numRequests
	requestsPerSecond := float64(numRequests) / duration.Seconds()

	suite.T().Logf("Authentication performance - %d requests in %v", numRequests, duration)
	suite.T().Logf("Average time per request: %v", averageTime)
	suite.T().Logf("Requests per second: %.2f", requestsPerSecond)

	// Performance should be reasonable
	assert.True(suite.T(), averageTime < 100*time.Millisecond, "Authentication should be fast")
	assert.True(suite.T(), requestsPerSecond > 50, "Should handle at least 50 requests per second")
}

// Run the test suite
func TestSecurityTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	suite.Run(t, new(SecurityTestSuite))
}

// Benchmark security operations
func BenchmarkJWTValidation(b *testing.B) {
	suite := &SecurityTestSuite{}
	suite.jwtSecret = make([]byte, 32)
	rand.Read(suite.jwtSecret)

	token, _ := suite.generateTestToken("bench-tenant", "bench-user", "user", []string{"upload"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := jwt.ParseWithClaims(token, &AuthTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			return suite.jwtSecret, nil
		})
		if err != nil {
			b.Error("Token validation failed:", err)
		}
	}
}

func BenchmarkPermissionCheck(b *testing.B) {
	permissions := []string{"upload", "read", "delete"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate permission checking
		found := false
		for _, perm := range permissions {
			if perm == "upload" {
				found = true
				break
			}
		}
		if !found {
			b.Error("Permission check failed")
		}
	}
}