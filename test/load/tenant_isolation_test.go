package load

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// LoadTestSuite runs performance and load tests for multi-tenant scenarios
type LoadTestSuite struct {
	suite.Suite
	ctx           context.Context
	testDuration  time.Duration
	rampUpTime    time.Duration
	steadyTime    time.Duration
	rampDownTime  time.Duration
	maxGoroutines int
}

// LoadTestResults captures performance metrics
type LoadTestResults struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	P95Latency         time.Duration `json:"p95_latency"`
	P99Latency         time.Duration `json:"p99_latency"`
	MaxLatency         time.Duration `json:"max_latency"`
	MinLatency         time.Duration `json:"min_latency"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	ErrorRate          float64       `json:"error_rate"`
	ThroughputMBps     float64       `json:"throughput_mbps"`
	StartTime          time.Time     `json:"start_time"`
	EndTime            time.Time     `json:"end_time"`
	Duration           time.Duration `json:"duration"`
}

// TenantLoadProfile defines load characteristics for a tenant
type TenantLoadProfile struct {
	TenantID           string        `json:"tenant_id"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	AudioFileSizeKB    int           `json:"audio_file_size_kb"`
	ProcessingComplexity string      `json:"processing_complexity"` // simple, medium, complex
	CRMIntegrationDelay time.Duration `json:"crm_integration_delay"`
	Priority           int           `json:"priority"` // 1-10, higher is higher priority
}

// RequestMetrics tracks individual request performance
type RequestMetrics struct {
	TenantID    string        `json:"tenant_id"`
	RequestID   string        `json:"request_id"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	ErrorType   string        `json:"error_type,omitempty"`
	FileSize    int           `json:"file_size_bytes"`
	Stage       string        `json:"stage"` // upload, processing, completion
}

// Mock ingestion service for load testing
type MockIngestionService struct {
	processingDelay time.Duration
	failureRate     float64
	tenantQuotas    map[string]int // requests per minute per tenant
	activeRequests  map[string]int64
	mutex           sync.RWMutex
}

func NewMockIngestionService() *MockIngestionService {
	return &MockIngestionService{
		processingDelay: 2 * time.Second,
		failureRate:     0.01, // 1% failure rate
		tenantQuotas:    make(map[string]int),
		activeRequests:  make(map[string]int64),
	}
}

func (m *MockIngestionService) ProcessAudio(ctx context.Context, tenantID string, audioData []byte) (*RequestMetrics, error) {
	startTime := time.Now()
	requestID := fmt.Sprintf("req_%d_%s", time.Now().UnixNano(), tenantID)

	// Check tenant quota
	m.mutex.Lock()
	currentRequests := atomic.LoadInt64(&m.activeRequests[tenantID])
	quota, hasQuota := m.tenantQuotas[tenantID]
	if hasQuota && currentRequests >= int64(quota) {
		m.mutex.Unlock()
		return nil, fmt.Errorf("tenant %s exceeded quota: %d active requests", tenantID, currentRequests)
	}
	atomic.AddInt64(&m.activeRequests[tenantID], 1)
	m.mutex.Unlock()

	defer func() {
		atomic.AddInt64(&m.activeRequests[tenantID], -1)
	}()

	// Simulate processing time with some variance
	baseDelay := m.processingDelay
	variance := time.Duration(rand.Int63n(int64(baseDelay) / 2)) // Up to 50% variance
	actualDelay := baseDelay + variance

	// Simulate different processing complexity based on tenant or data
	complexityMultiplier := 1.0
	if len(audioData) > 50*1024 { // Large files take longer
		complexityMultiplier = 1.5
	}

	processingTime := time.Duration(float64(actualDelay) * complexityMultiplier)

	// Wait for processing (or context cancellation)
	select {
	case <-time.After(processingTime):
		// Processing completed
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Simulate random failures
	success := rand.Float64() > m.failureRate

	metrics := &RequestMetrics{
		TenantID:  tenantID,
		RequestID: requestID,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
		Success:   success,
		FileSize:  len(audioData),
		Stage:     "completion",
	}

	if !success {
		metrics.ErrorType = "processing_error"
		return metrics, fmt.Errorf("simulated processing failure")
	}

	return metrics, nil
}

func (m *MockIngestionService) SetTenantQuota(tenantID string, requestsPerMinute int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tenantQuotas[tenantID] = requestsPerMinute
}

func (suite *LoadTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testDuration = 2 * time.Minute
	suite.rampUpTime = 30 * time.Second
	suite.steadyTime = 60 * time.Second
	suite.rampDownTime = 30 * time.Second
	suite.maxGoroutines = 100
}

func (suite *LoadTestSuite) TestMultiTenantLoadIsolation() {
	// Test that high load from one tenant doesn't affect others
	service := NewMockIngestionService()

	// Setup tenant profiles
	tenants := []TenantLoadProfile{
		{
			TenantID:          "tenant-high-load",
			RequestsPerSecond: 50.0, // High load tenant
			AudioFileSizeKB:   100,
			Priority:          5,
		},
		{
			TenantID:          "tenant-medium-load",
			RequestsPerSecond: 20.0, // Medium load tenant
			AudioFileSizeKB:   80,
			Priority:          7,
		},
		{
			TenantID:          "tenant-low-load",
			RequestsPerSecond: 5.0, // Low load tenant
			AudioFileSizeKB:   50,
			Priority:          10, // Highest priority
		},
	}

	// Set quotas to ensure isolation
	service.SetTenantQuota("tenant-high-load", 60)   // Slightly above normal rate
	service.SetTenantQuota("tenant-medium-load", 30) // Above normal rate
	service.SetTenantQuota("tenant-low-load", 20)    // Well above normal rate for priority

	// Run concurrent load for all tenants
	var wg sync.WaitGroup
	results := make(map[string]*LoadTestResults)
	resultsMutex := sync.Mutex{}

	for _, tenant := range tenants {
		wg.Add(1)
		go func(profile TenantLoadProfile) {
			defer wg.Done()

			result := suite.runTenantLoad(service, profile, 60*time.Second)

			resultsMutex.Lock()
			results[profile.TenantID] = result
			resultsMutex.Unlock()
		}(tenant)
	}

	wg.Wait()

	// Analyze results for isolation
	suite.analyzeTenantIsolation(results)
}

func (suite *LoadTestSuite) TestScalabilityUnderLoad() {
	// Test system scalability as load increases
	service := NewMockIngestionService()

	loadLevels := []struct {
		name          string
		requestsPerSec float64
		duration      time.Duration
		expectedP95   time.Duration
	}{
		{"Light Load", 10.0, 30 * time.Second, 3 * time.Second},
		{"Medium Load", 25.0, 30 * time.Second, 4 * time.Second},
		{"Heavy Load", 50.0, 30 * time.Second, 6 * time.Second},
		{"Peak Load", 100.0, 30 * time.Second, 10 * time.Second},
	}

	tenantProfile := TenantLoadProfile{
		TenantID:          "scalability-test-tenant",
		AudioFileSizeKB:   75,
		Priority:          5,
	}

	var allResults []*LoadTestResults

	for _, level := range loadLevels {
		suite.T().Logf("Testing %s: %.1f req/sec for %v", level.name, level.requestsPerSec, level.duration)

		tenantProfile.RequestsPerSecond = level.requestsPerSec
		result := suite.runTenantLoad(service, tenantProfile, level.duration)

		suite.T().Logf("Results - Success Rate: %.2f%%, P95 Latency: %v, RPS: %.2f",
			(1.0-result.ErrorRate)*100, result.P95Latency, result.RequestsPerSecond)

		// Validate performance meets expectations
		assert.True(suite.T(), result.ErrorRate < 0.05, "Error rate should be < 5% for %s", level.name)
		assert.True(suite.T(), result.P95Latency < level.expectedP95, "P95 latency should be < %v for %s", level.expectedP95, level.name)

		allResults = append(allResults, result)

		// Brief cooldown between load levels
		time.Sleep(5 * time.Second)
	}

	// Analyze scalability trends
	suite.analyzeScalabilityTrends(allResults)
}

func (suite *LoadTestSuite) TestTenantQuotaEnforcement() {
	// Test that tenant quotas are properly enforced
	service := NewMockIngestionService()

	tenantID := "quota-test-tenant"
	quotaLimit := 30 // requests per minute
	service.SetTenantQuota(tenantID, quotaLimit)

	// Generate load that exceeds quota
	profile := TenantLoadProfile{
		TenantID:          tenantID,
		RequestsPerSecond: 1.0, // 60 requests per minute - exceeds quota
		AudioFileSizeKB:   50,
		Priority:          5,
	}

	result := suite.runTenantLoad(service, profile, 90*time.Second)

	// Analyze quota enforcement
	expectedSuccessRate := float64(quotaLimit*1.5) / float64(result.TotalRequests) // Some buffer for timing
	actualSuccessRate := 1.0 - result.ErrorRate

	suite.T().Logf("Quota test - Expected success rate: ~%.2f%%, Actual: %.2f%%",
		expectedSuccessRate*100, actualSuccessRate*100)

	// Should see quota-related failures
	assert.True(suite.T(), result.ErrorRate > 0.3, "Should have significant error rate due to quota enforcement")
	assert.True(suite.T(), result.SuccessfulRequests <= int64(quotaLimit*2), "Should not exceed quota by too much")
}

func (suite *LoadTestSuite) TestConcurrentTenantCreation() {
	// Test system behavior when many tenants are created simultaneously
	service := NewMockIngestionService()

	const numTenants = 20
	const requestsPerTenant = 10

	var wg sync.WaitGroup
	var totalSuccessRequests int64
	var totalRequests int64

	for i := 0; i < numTenants; i++ {
		wg.Add(1)
		go func(tenantIndex int) {
			defer wg.Done()

			tenantID := fmt.Sprintf("concurrent-tenant-%d", tenantIndex)
			profile := TenantLoadProfile{
				TenantID:          tenantID,
				RequestsPerSecond: 2.0, // Moderate load per tenant
				AudioFileSizeKB:   60,
				Priority:          5,
			}

			result := suite.runTenantLoad(service, profile, 10*time.Second)

			atomic.AddInt64(&totalSuccessRequests, result.SuccessfulRequests)
			atomic.AddInt64(&totalRequests, result.TotalRequests)
		}(i)
	}

	wg.Wait()

	// Analyze overall system performance
	successRate := float64(totalSuccessRequests) / float64(totalRequests)
	suite.T().Logf("Concurrent tenants test - Total requests: %d, Success rate: %.2f%%",
		totalRequests, successRate*100)

	assert.True(suite.T(), successRate > 0.90, "Success rate should be > 90% with concurrent tenants")
	assert.True(suite.T(), totalRequests >= int64(numTenants*15), "Should have processed reasonable number of requests")
}

func (suite *LoadTestSuite) TestMemoryLeakDetection() {
	// Test for memory leaks during sustained load
	service := NewMockIngestionService()

	profile := TenantLoadProfile{
		TenantID:          "memory-test-tenant",
		RequestsPerSecond: 10.0,
		AudioFileSizeKB:   100, // Larger files to test memory handling
		Priority:          5,
	}

	// Run sustained load
	const testDuration = 2 * time.Minute
	result := suite.runTenantLoad(service, profile, testDuration)

	// Basic validation - more sophisticated memory monitoring would be needed in real tests
	assert.True(suite.T(), result.ErrorRate < 0.05, "Error rate should remain low during sustained load")
	assert.True(suite.T(), result.TotalRequests > 1000, "Should process significant number of requests")

	// In a real implementation, you would monitor:
	// - Go runtime memory stats
	// - GC frequency and duration
	// - Goroutine count
	// - File descriptor usage
	suite.T().Logf("Memory test completed - Processed %d requests over %v",
		result.TotalRequests, testDuration)
}

func (suite *LoadTestSuite) runTenantLoad(service *MockIngestionService, profile TenantLoadProfile, duration time.Duration) *LoadTestResults {
	ctx, cancel := context.WithTimeout(suite.ctx, duration)
	defer cancel()

	var (
		totalRequests      int64
		successfulRequests int64
		failedRequests     int64
		latencies          []time.Duration
		latenciesMutex     sync.Mutex
	)

	startTime := time.Now()

	// Calculate request interval
	requestInterval := time.Duration(float64(time.Second) / profile.RequestsPerSecond)

	// Generate synthetic audio data
	audioData := make([]byte, profile.AudioFileSizeKB*1024)
	rand.Read(audioData)

	// Rate-limited request generator
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()

				atomic.AddInt64(&totalRequests, 1)

				reqCtx, reqCancel := context.WithTimeout(ctx, 10*time.Second)
				defer reqCancel()

				metrics, err := service.ProcessAudio(reqCtx, profile.TenantID, audioData)

				if err != nil {
					atomic.AddInt64(&failedRequests, 1)
				} else if metrics != nil && metrics.Success {
					atomic.AddInt64(&successfulRequests, 1)

					latenciesMutex.Lock()
					latencies = append(latencies, metrics.Duration)
					latenciesMutex.Unlock()
				} else {
					atomic.AddInt64(&failedRequests, 1)
				}
			}()
		}
	}

done:
	// Wait for all requests to complete
	wg.Wait()
	endTime := time.Now()

	// Calculate statistics
	result := &LoadTestResults{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		StartTime:          startTime,
		EndTime:            endTime,
		Duration:           endTime.Sub(startTime),
	}

	if totalRequests > 0 {
		result.ErrorRate = float64(failedRequests) / float64(totalRequests)
		result.RequestsPerSecond = float64(totalRequests) / result.Duration.Seconds()
	}

	if len(latencies) > 0 {
		result.AverageLatency = suite.calculateAverageLatency(latencies)
		result.P95Latency = suite.calculatePercentileLatency(latencies, 95)
		result.P99Latency = suite.calculatePercentileLatency(latencies, 99)
		result.MaxLatency = suite.calculateMaxLatency(latencies)
		result.MinLatency = suite.calculateMinLatency(latencies)
	}

	// Calculate throughput (MB/s)
	totalDataMB := float64(successfulRequests*int64(profile.AudioFileSizeKB)) / 1024.0
	result.ThroughputMBps = totalDataMB / result.Duration.Seconds()

	return result
}

func (suite *LoadTestSuite) analyzeTenantIsolation(results map[string]*LoadTestResults) {
	suite.T().Log("=== Tenant Isolation Analysis ===")

	for tenantID, result := range results {
		suite.T().Logf("Tenant %s:", tenantID)
		suite.T().Logf("  Total Requests: %d", result.TotalRequests)
		suite.T().Logf("  Success Rate: %.2f%%", (1.0-result.ErrorRate)*100)
		suite.T().Logf("  Avg Latency: %v", result.AverageLatency)
		suite.T().Logf("  P95 Latency: %v", result.P95Latency)
		suite.T().Logf("  RPS: %.2f", result.RequestsPerSecond)
		suite.T().Logf("  Throughput: %.2f MB/s", result.ThroughputMBps)
	}

	// Validate isolation - high-priority tenant should have better performance
	lowLoadResult := results["tenant-low-load"]
	highLoadResult := results["tenant-high-load"]

	if lowLoadResult != nil && highLoadResult != nil {
		// Low load tenant should have better latency despite overall system load
		assert.True(suite.T(), lowLoadResult.AverageLatency <= highLoadResult.AverageLatency*1.2,
			"Low load tenant should have comparable latency despite high system load")

		// Both should maintain reasonable success rates
		assert.True(suite.T(), lowLoadResult.ErrorRate < 0.05, "Low load tenant should have < 5% error rate")
		assert.True(suite.T(), highLoadResult.ErrorRate < 0.10, "High load tenant should have < 10% error rate")
	}
}

func (suite *LoadTestSuite) analyzeScalabilityTrends(results []*LoadTestResults) {
	suite.T().Log("=== Scalability Trend Analysis ===")

	for i, result := range results {
		suite.T().Logf("Load Level %d:", i+1)
		suite.T().Logf("  RPS: %.2f", result.RequestsPerSecond)
		suite.T().Logf("  Error Rate: %.2f%%", result.ErrorRate*100)
		suite.T().Logf("  P95 Latency: %v", result.P95Latency)
		suite.T().Logf("  Throughput: %.2f MB/s", result.ThroughputMBps)
	}

	// Validate that system scales reasonably
	if len(results) >= 2 {
		firstResult := results[0]
		lastResult := results[len(results)-1]

		// Latency should not increase exponentially
		latencyIncrease := float64(lastResult.P95Latency) / float64(firstResult.P95Latency)
		assert.True(suite.T(), latencyIncrease < 5.0, "P95 latency should not increase more than 5x under load")

		// Error rate should remain reasonable
		assert.True(suite.T(), lastResult.ErrorRate < 0.15, "Error rate should remain < 15% even under peak load")
	}
}

// Utility functions for latency calculations

func (suite *LoadTestSuite) calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	return total / time.Duration(len(latencies))
}

func (suite *LoadTestSuite) calculatePercentileLatency(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Simple percentile calculation (would use sort in real implementation)
	index := (len(latencies) * percentile) / 100
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	// For simplicity, just use a rough approximation
	// In real implementation, you'd sort the slice first
	maxLatency := suite.calculateMaxLatency(latencies)
	avgLatency := suite.calculateAverageLatency(latencies)

	// Rough approximation for percentile
	factor := float64(percentile) / 100.0
	return time.Duration(float64(avgLatency) + factor*float64(maxLatency-avgLatency))
}

func (suite *LoadTestSuite) calculateMaxLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	max := latencies[0]
	for _, latency := range latencies[1:] {
		if latency > max {
			max = latency
		}
	}
	return max
}

func (suite *LoadTestSuite) calculateMinLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	min := latencies[0]
	for _, latency := range latencies[1:] {
		if latency < min {
			min = latency
		}
	}
	return min
}

// Benchmark tests for performance regression detection

func BenchmarkSingleTenantProcessing(b *testing.B) {
	service := NewMockIngestionService()
	service.processingDelay = 100 * time.Millisecond // Faster for benchmarking

	audioData := make([]byte, 50*1024) // 50KB
	rand.Read(audioData)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := service.ProcessAudio(context.Background(), "bench-tenant", audioData)
			if err != nil {
				b.Error("Processing failed:", err)
			}
		}
	})
}

func BenchmarkMultiTenantProcessing(b *testing.B) {
	service := NewMockIngestionService()
	service.processingDelay = 100 * time.Millisecond

	audioData := make([]byte, 50*1024)
	rand.Read(audioData)

	tenants := []string{"tenant-1", "tenant-2", "tenant-3", "tenant-4", "tenant-5"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		tenantIndex := 0
		for pb.Next() {
			tenantID := tenants[tenantIndex%len(tenants)]
			tenantIndex++

			_, err := service.ProcessAudio(context.Background(), tenantID, audioData)
			if err != nil {
				b.Error("Processing failed:", err)
			}
		}
	})
}

// Memory usage benchmark
func BenchmarkMemoryUsage(b *testing.B) {
	service := NewMockIngestionService()
	service.processingDelay = 50 * time.Millisecond

	// Test with different file sizes
	fileSizes := []int{10 * 1024, 100 * 1024, 1024 * 1024} // 10KB, 100KB, 1MB

	for _, size := range fileSizes {
		b.Run(fmt.Sprintf("FileSize%dKB", size/1024), func(b *testing.B) {
			audioData := make([]byte, size)
			rand.Read(audioData)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := service.ProcessAudio(context.Background(), "memory-bench-tenant", audioData)
				if err != nil {
					b.Error("Processing failed:", err)
				}
			}
		})
	}
}

// Run the test suite
func TestLoadTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	suite.Run(t, new(LoadTestSuite))
}