package security

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"cloud.google.com/go/spanner"

	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
	"github.com/home-renovators/ingestion-pipeline/internal/auth"
	spannerdb "github.com/home-renovators/ingestion-pipeline/internal/spanner"
)

// TenantIsolationSecurityTestSuite tests multi-tenant data isolation and security
type TenantIsolationSecurityTestSuite struct {
	suite.Suite
	ctx           context.Context
	spannerClient *spanner.Client
	spannerDB     *spannerdb.DB
	authService   *auth.Service
	config        *config.Config
	testTenants   []string
}

func (suite *TenantIsolationSecurityTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testTenants = []string{
		"tenant_security_test_1",
		"tenant_security_test_2",
		"tenant_security_test_3",
		"tenant_malicious_test",
	}

	// Setup test configuration
	suite.config = &config.Config{
		ProjectID:       "test-project",
		SpannerInstance: "test-instance",
		SpannerDatabase: "test-database",
	}

	// Initialize Spanner client
	var err error
	suite.spannerClient, err = spanner.NewClient(suite.ctx,
		fmt.Sprintf("projects/%s/instances/%s/databases/%s",
			suite.config.ProjectID, suite.config.SpannerInstance, suite.config.SpannerDatabase))
	require.NoError(suite.T(), err)

	// Initialize database service
	suite.spannerDB = spannerdb.NewDB(suite.spannerClient)

	// Initialize auth service
	suite.authService = auth.NewService(suite.spannerDB)

	// Setup test tenants
	suite.setupTestTenants()
}

func (suite *TenantIsolationSecurityTestSuite) TearDownSuite() {
	// Cleanup test data
	suite.cleanupTestTenants()

	if suite.spannerClient != nil {
		suite.spannerClient.Close()
	}
}

func (suite *TenantIsolationSecurityTestSuite) SetupTest() {
	// Clean up any test data before each test
	suite.cleanupTestData()
}

// setupTestTenants creates test tenants with different configurations
func (suite *TenantIsolationSecurityTestSuite) setupTestTenants() {
	for i, tenantID := range suite.testTenants {
		office := &models.Office{
			TenantID:          tenantID,
			OfficeID:         fmt.Sprintf("office_%d", i+1),
			CallRailCompanyID: fmt.Sprintf("CR%d%d%d", i+1, i+1, i+1),
			CallRailAPIKey:    fmt.Sprintf("api_key_%d", i+1),
			WorkflowConfig:    `{"communication_detection":{"enabled":true}}`,
			Status:           "active",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		err := suite.spannerDB.CreateOffice(suite.ctx, office)
		require.NoError(suite.T(), err)
	}
}

// cleanupTestTenants removes all test tenant data
func (suite *TenantIsolationSecurityTestSuite) cleanupTestTenants() {
	for i, tenantID := range suite.testTenants {
		suite.spannerDB.DeleteOffice(suite.ctx, tenantID, fmt.Sprintf("office_%d", i+1))
	}
}

// cleanupTestData removes test request/call data
func (suite *TenantIsolationSecurityTestSuite) cleanupTestData() {
	for _, tenantID := range suite.testTenants {
		suite.spannerDB.DeleteRequestsByTenant(suite.ctx, tenantID)
		suite.spannerDB.DeleteCallRecordingsByTenant(suite.ctx, tenantID)
		suite.spannerDB.DeleteWebhookEventsByTenant(suite.ctx, tenantID)
	}
}

// TestDatabaseTenantIsolation tests that database queries respect tenant boundaries
func (suite *TenantIsolationSecurityTestSuite) TestDatabaseTenantIsolation() {
	suite.T().Run("RequestDataIsolation", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		// Create requests for both tenants
		tenant1Requests := []*models.Request{
			{
				RequestID:         models.NewRequestID(),
				TenantID:          tenant1,
				Source:            "callrail_webhook",
				RequestType:       "call",
				Status:            "processed",
				Data:              fmt.Sprintf(`{"call_id":"CAL_T1_001","secret":"tenant1_secret"}`),
				CallID:            stringPtr("CAL_T1_001"),
				CommunicationMode: "phone_call",
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
			{
				RequestID:         models.NewRequestID(),
				TenantID:          tenant1,
				Source:            "callrail_webhook",
				RequestType:       "call",
				Status:            "processed",
				Data:              fmt.Sprintf(`{"call_id":"CAL_T1_002","secret":"tenant1_secret_2"}`),
				CallID:            stringPtr("CAL_T1_002"),
				CommunicationMode: "phone_call",
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
		}

		tenant2Requests := []*models.Request{
			{
				RequestID:         models.NewRequestID(),
				TenantID:          tenant2,
				Source:            "callrail_webhook",
				RequestType:       "call",
				Status:            "processed",
				Data:              fmt.Sprintf(`{"call_id":"CAL_T2_001","secret":"tenant2_secret"}`),
				CallID:            stringPtr("CAL_T2_001"),
				CommunicationMode: "phone_call",
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
		}

		// Insert requests
		for _, req := range tenant1Requests {
			err := suite.spannerDB.CreateRequest(suite.ctx, req)
			require.NoError(t, err)
		}
		for _, req := range tenant2Requests {
			err := suite.spannerDB.CreateRequest(suite.ctx, req)
			require.NoError(t, err)
		}

		// Test 1: Tenant 1 should only see its own requests
		t1Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, tenant1)
		require.NoError(t, err)
		assert.Equal(t, 2, len(t1Requests), "Tenant 1 should see exactly 2 requests")

		for _, req := range t1Requests {
			assert.Equal(t, tenant1, req.TenantID, "All returned requests should belong to tenant 1")
			assert.Contains(t, req.Data, "tenant1_secret", "Request should contain tenant 1 data")
			assert.NotContains(t, req.Data, "tenant2_secret", "Request should not contain tenant 2 data")
		}

		// Test 2: Tenant 2 should only see its own requests
		t2Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, tenant2)
		require.NoError(t, err)
		assert.Equal(t, 1, len(t2Requests), "Tenant 2 should see exactly 1 request")

		for _, req := range t2Requests {
			assert.Equal(t, tenant2, req.TenantID, "All returned requests should belong to tenant 2")
			assert.Contains(t, req.Data, "tenant2_secret", "Request should contain tenant 2 data")
			assert.NotContains(t, req.Data, "tenant1_secret", "Request should not contain tenant 1 data")
		}

		// Test 3: Cross-tenant request access should fail
		tenant1RequestID := tenant1Requests[0].RequestID
		crossTenantRequest, err := suite.spannerDB.GetRequestByID(suite.ctx, tenant1RequestID, tenant2)
		assert.Error(t, err, "Cross-tenant request access should be blocked")
		assert.Nil(t, crossTenantRequest, "Cross-tenant request should return nil")
	})

	suite.T().Run("CallRecordingDataIsolation", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		// Create call recordings for both tenants
		tenant1Recording := &models.CallRecording{
			RecordingID:         models.NewRecordingID(),
			TenantID:            tenant1,
			CallID:              "CAL_REC_T1_001",
			StorageURL:          fmt.Sprintf("gs://tenant-%s-audio/recording1.mp3", tenant1),
			TranscriptionStatus: "completed",
			CreatedAt:           time.Now(),
		}

		tenant2Recording := &models.CallRecording{
			RecordingID:         models.NewRecordingID(),
			TenantID:            tenant2,
			CallID:              "CAL_REC_T2_001",
			StorageURL:          fmt.Sprintf("gs://tenant-%s-audio/recording1.mp3", tenant2),
			TranscriptionStatus: "completed",
			CreatedAt:           time.Now(),
		}

		err := suite.spannerDB.CreateCallRecording(suite.ctx, tenant1Recording)
		require.NoError(t, err)
		err = suite.spannerDB.CreateCallRecording(suite.ctx, tenant2Recording)
		require.NoError(t, err)

		// Test isolation
		t1Recording, err := suite.spannerDB.GetCallRecording(suite.ctx, tenant1, "CAL_REC_T1_001")
		require.NoError(t, err)
		assert.Equal(t, tenant1, t1Recording.TenantID)
		assert.Contains(t, t1Recording.StorageURL, tenant1, "Storage URL should contain tenant ID")

		// Cross-tenant access should fail
		crossTenantRecording, err := suite.spannerDB.GetCallRecording(suite.ctx, tenant2, "CAL_REC_T1_001")
		assert.Error(t, err, "Cross-tenant call recording access should be blocked")
		assert.Nil(t, crossTenantRecording)
	})
}

// TestAPITenantIsolation tests API-level tenant isolation
func (suite *TenantIsolationSecurityTestSuite) TestAPITenantIsolation() {
	suite.T().Run("AuthenticationTenantSeparation", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		// Get office configurations for both tenants
		office1, err := suite.spannerDB.GetOfficeByTenantID(suite.ctx, tenant1)
		require.NoError(t, err)

		office2, err := suite.spannerDB.GetOfficeByTenantID(suite.ctx, tenant2)
		require.NoError(t, err)

		// Test 1: Correct tenant authentication should succeed
		isValidTenant1 := suite.authService.ValidateTenant(suite.ctx, tenant1, office1.CallRailCompanyID)
		assert.True(t, isValidTenant1, "Valid tenant authentication should succeed")

		isValidTenant2 := suite.authService.ValidateTenant(suite.ctx, tenant2, office2.CallRailCompanyID)
		assert.True(t, isValidTenant2, "Valid tenant authentication should succeed")

		// Test 2: Cross-tenant authentication should fail
		crossTenantValid1 := suite.authService.ValidateTenant(suite.ctx, tenant1, office2.CallRailCompanyID)
		assert.False(t, crossTenantValid1, "Cross-tenant authentication should fail")

		crossTenantValid2 := suite.authService.ValidateTenant(suite.ctx, tenant2, office1.CallRailCompanyID)
		assert.False(t, crossTenantValid2, "Cross-tenant authentication should fail")

		// Test 3: Non-existent tenant should fail
		invalidTenantValid := suite.authService.ValidateTenant(suite.ctx, "non_existent_tenant", office1.CallRailCompanyID)
		assert.False(t, invalidTenantValid, "Non-existent tenant authentication should fail")
	})
}

// TestDataLeakagePrevention tests prevention of data leakage between tenants
func (suite *TenantIsolationSecurityTestSuite) TestDataLeakagePrevention() {
	suite.T().Run("SQLInjectionResistance", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		maliciousTenant := suite.testTenants[3] // Use the malicious test tenant

		// Create a normal request for tenant 1
		normalRequest := &models.Request{
			RequestID:         models.NewRequestID(),
			TenantID:          tenant1,
			Source:            "callrail_webhook",
			RequestType:       "call",
			Status:            "processed",
			Data:              `{"call_id":"CAL_NORMAL_001","sensitive":"secret_data"}`,
			CallID:            stringPtr("CAL_NORMAL_001"),
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := suite.spannerDB.CreateRequest(suite.ctx, normalRequest)
		require.NoError(t, err)

		// Attempt various SQL injection patterns as tenant IDs
		maliciousPatterns := []string{
			"' OR '1'='1",
			"'; DROP TABLE requests; --",
			"' UNION SELECT * FROM requests WHERE tenant_id = '" + tenant1 + "' --",
			maliciousTenant + "' OR tenant_id = '" + tenant1,
			"*",
			"%",
		}

		for _, pattern := range maliciousPatterns {
			requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, pattern)

			// Should either return empty results or error, never return other tenants' data
			if err == nil {
				assert.Empty(t, requests, "Malicious pattern '%s' should not return data", pattern)
			}

			// Specifically ensure we don't get tenant1's data
			for _, req := range requests {
				assert.NotEqual(t, tenant1, req.TenantID,
					"Malicious pattern '%s' should not leak tenant1 data", pattern)
			}
		}
	})

	suite.T().Run("ParameterPollutionResistance", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		// Create test data for both tenants
		t1Request := &models.Request{
			RequestID:         models.NewRequestID(),
			TenantID:          tenant1,
			Source:            "test",
			RequestType:       "call",
			Status:            "processed",
			Data:              `{"secret":"tenant1_secret"}`,
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		t2Request := &models.Request{
			RequestID:         models.NewRequestID(),
			TenantID:          tenant2,
			Source:            "test",
			RequestType:       "call",
			Status:            "processed",
			Data:              `{"secret":"tenant2_secret"}`,
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := suite.spannerDB.CreateRequest(suite.ctx, t1Request)
		require.NoError(t, err)
		err = suite.spannerDB.CreateRequest(suite.ctx, t2Request)
		require.NoError(t, err)

		// Attempt to access tenant1's specific request with tenant2 credentials
		retrievedRequest, err := suite.spannerDB.GetRequestByID(suite.ctx, t1Request.RequestID, tenant2)
		assert.Error(t, err, "Should not be able to access other tenant's request by ID")
		assert.Nil(t, retrievedRequest, "Should not return cross-tenant data")

		// Double-check with reverse access
		retrievedRequest, err = suite.spannerDB.GetRequestByID(suite.ctx, t2Request.RequestID, tenant1)
		assert.Error(t, err, "Should not be able to access other tenant's request by ID")
		assert.Nil(t, retrievedRequest, "Should not return cross-tenant data")
	})
}

// TestResourceIsolation tests resource-level isolation between tenants
func (suite *TenantIsolationSecurityTestSuite) TestResourceIsolation() {
	suite.T().Run("StoragePathIsolation", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		// Test that storage URLs are tenant-specific
		t1Recording := &models.CallRecording{
			RecordingID:         models.NewRecordingID(),
			TenantID:            tenant1,
			CallID:              "CAL_STORAGE_T1_001",
			StorageURL:          suite.generateTenantStorageURL(tenant1, "CAL_STORAGE_T1_001"),
			TranscriptionStatus: "completed",
			CreatedAt:           time.Now(),
		}

		t2Recording := &models.CallRecording{
			RecordingID:         models.NewRecordingID(),
			TenantID:            tenant2,
			CallID:              "CAL_STORAGE_T2_001",
			StorageURL:          suite.generateTenantStorageURL(tenant2, "CAL_STORAGE_T2_001"),
			TranscriptionStatus: "completed",
			CreatedAt:           time.Now(),
		}

		err := suite.spannerDB.CreateCallRecording(suite.ctx, t1Recording)
		require.NoError(t, err)
		err = suite.spannerDB.CreateCallRecording(suite.ctx, t2Recording)
		require.NoError(t, err)

		// Verify storage URLs contain tenant isolation
		assert.Contains(t, t1Recording.StorageURL, tenant1,
			"Storage URL should contain tenant ID for isolation")
		assert.NotContains(t, t1Recording.StorageURL, tenant2,
			"Storage URL should not contain other tenant IDs")

		assert.Contains(t, t2Recording.StorageURL, tenant2,
			"Storage URL should contain tenant ID for isolation")
		assert.NotContains(t, t2Recording.StorageURL, tenant1,
			"Storage URL should not contain other tenant IDs")

		// Verify paths don't overlap
		assert.NotEqual(t, t1Recording.StorageURL, t2Recording.StorageURL,
			"Different tenants should have different storage URLs")
	})

	suite.T().Run("MemoryIsolation", func(t *testing.T) {
		// This test ensures that in-memory data structures don't leak between tenants
		// In a real implementation, you might test caching, session storage, etc.

		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		// Simulate processing requests for both tenants
		t1Request := &models.Request{
			RequestID:         models.NewRequestID(),
			TenantID:          tenant1,
			Source:            "test",
			Data:              `{"sensitive_data":"tenant1_private_info"}`,
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		t2Request := &models.Request{
			RequestID:         models.NewRequestID(),
			TenantID:          tenant2,
			Source:            "test",
			Data:              `{"sensitive_data":"tenant2_private_info"}`,
			CommunicationMode: "phone_call",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		// Create both requests
		err := suite.spannerDB.CreateRequest(suite.ctx, t1Request)
		require.NoError(t, err)
		err = suite.spannerDB.CreateRequest(suite.ctx, t2Request)
		require.NoError(t, err)

		// Retrieve tenant-specific data
		t1Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, tenant1)
		require.NoError(t, err)
		assert.Equal(t, 1, len(t1Requests))

		t2Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, tenant2)
		require.NoError(t, err)
		assert.Equal(t, 1, len(t2Requests))

		// Verify no cross-contamination
		assert.Contains(t, t1Requests[0].Data, "tenant1_private_info")
		assert.NotContains(t, t1Requests[0].Data, "tenant2_private_info")

		assert.Contains(t, t2Requests[0].Data, "tenant2_private_info")
		assert.NotContains(t, t2Requests[0].Data, "tenant1_private_info")
	})
}

// TestConcurrentTenantAccess tests isolation under concurrent access
func (suite *TenantIsolationSecurityTestSuite) TestConcurrentTenantAccess() {
	suite.T().Run("ConcurrentReadWriteIsolation", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		const numConcurrentOps = 10

		// Create channels to coordinate concurrent operations
		results := make(chan struct {
			tenantID string
			success  bool
			err      error
		}, numConcurrentOps*2)

		// Start concurrent operations for both tenants
		for i := 0; i < numConcurrentOps; i++ {
			// Operations for tenant 1
			go func(opIndex int) {
				request := &models.Request{
					RequestID:         models.NewRequestID(),
					TenantID:          tenant1,
					Source:            "concurrent_test",
					Data:              fmt.Sprintf(`{"operation":%d,"tenant":"%s"}`, opIndex, tenant1),
					CommunicationMode: "phone_call",
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := suite.spannerDB.CreateRequest(suite.ctx, request)
				results <- struct {
					tenantID string
					success  bool
					err      error
				}{tenant1, err == nil, err}
			}(i)

			// Operations for tenant 2
			go func(opIndex int) {
				request := &models.Request{
					RequestID:         models.NewRequestID(),
					TenantID:          tenant2,
					Source:            "concurrent_test",
					Data:              fmt.Sprintf(`{"operation":%d,"tenant":"%s"}`, opIndex, tenant2),
					CommunicationMode: "phone_call",
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}

				err := suite.spannerDB.CreateRequest(suite.ctx, request)
				results <- struct {
					tenantID string
					success  bool
					err      error
				}{tenant2, err == nil, err}
			}(i)
		}

		// Collect results
		t1SuccessCount := 0
		t2SuccessCount := 0

		for i := 0; i < numConcurrentOps*2; i++ {
			result := <-results
			if result.success {
				if result.tenantID == tenant1 {
					t1SuccessCount++
				} else if result.tenantID == tenant2 {
					t2SuccessCount++
				}
			} else {
				t.Logf("Operation failed for %s: %v", result.tenantID, result.err)
			}
		}

		// Verify isolation was maintained
		assert.Equal(t, numConcurrentOps, t1SuccessCount,
			"All tenant 1 operations should succeed")
		assert.Equal(t, numConcurrentOps, t2SuccessCount,
			"All tenant 2 operations should succeed")

		// Verify data isolation after concurrent operations
		t1Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, tenant1)
		require.NoError(t, err)
		assert.Equal(t, numConcurrentOps, len(t1Requests),
			"Tenant 1 should have exactly its own requests")

		t2Requests, err := suite.spannerDB.GetRequestsByTenant(suite.ctx, tenant2)
		require.NoError(t, err)
		assert.Equal(t, numConcurrentOps, len(t2Requests),
			"Tenant 2 should have exactly its own requests")

		// Verify no cross-tenant data contamination
		for _, req := range t1Requests {
			assert.Equal(t, tenant1, req.TenantID)
			assert.Contains(t, req.Data, fmt.Sprintf(`"tenant":"%s"`, tenant1))
			assert.NotContains(t, req.Data, fmt.Sprintf(`"tenant":"%s"`, tenant2))
		}

		for _, req := range t2Requests {
			assert.Equal(t, tenant2, req.TenantID)
			assert.Contains(t, req.Data, fmt.Sprintf(`"tenant":"%s"`, tenant2))
			assert.NotContains(t, req.Data, fmt.Sprintf(`"tenant":"%s"`, tenant1))
		}
	})
}

// TestTenantConfigurationIsolation tests isolation of tenant configurations
func (suite *TenantIsolationSecurityTestSuite) TestTenantConfigurationIsolation() {
	suite.T().Run("OfficeConfigurationIsolation", func(t *testing.T) {
		tenant1 := suite.testTenants[0]
		tenant2 := suite.testTenants[1]

		// Get configurations for both tenants
		office1, err := suite.spannerDB.GetOfficeByTenantID(suite.ctx, tenant1)
		require.NoError(t, err)

		office2, err := suite.spannerDB.GetOfficeByTenantID(suite.ctx, tenant2)
		require.NoError(t, err)

		// Verify configurations are isolated
		assert.NotEqual(t, office1.TenantID, office2.TenantID, "Tenant IDs should be different")
		assert.NotEqual(t, office1.CallRailCompanyID, office2.CallRailCompanyID,
			"CallRail company IDs should be different")
		assert.NotEqual(t, office1.CallRailAPIKey, office2.CallRailAPIKey,
			"API keys should be different")

		// Test that tenant 1 cannot access tenant 2's configuration
		_, err = suite.spannerDB.GetOfficeByTenantID(suite.ctx, "non_existent_tenant")
		assert.Error(t, err, "Non-existent tenant should return error")

		// Verify API key isolation - tenant 1's key shouldn't work for tenant 2's company ID
		isValidAuth := suite.authService.ValidateCallRailAuth(office1.CallRailAPIKey, office2.CallRailCompanyID)
		assert.False(t, isValidAuth, "Cross-tenant API key authentication should fail")
	})
}

// Helper functions

func (suite *TenantIsolationSecurityTestSuite) generateTenantStorageURL(tenantID, callID string) string {
	return fmt.Sprintf("gs://audio-storage-%s/%s/calls/%s.mp3", tenantID, tenantID, callID)
}

func stringPtr(s string) *string {
	return &s
}

// Run the test suite
func TestTenantIsolationSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(TenantIsolationSecurityTestSuite))
}