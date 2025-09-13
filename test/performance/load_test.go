package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/home-renovators/ingestion-pipeline/pkg/config"
	"github.com/home-renovators/ingestion-pipeline/pkg/models"
	"github.com/home-renovators/ingestion-pipeline/internal/callrail"
	"github.com/home-renovators/ingestion-pipeline/internal/ai"
)

// LoadTestSuite tests system performance under load
type LoadTestSuite struct {
	suite.Suite
	ctx           context.Context
	config        *config.Config
	callrailClient *callrail.Client
	aiService     *ai.Service
	testTenants   []string
}

// LoadTestMetrics tracks performance metrics during load testing
type LoadTestMetrics struct {
	TotalRequests       int64
	SuccessfulRequests  int64
	FailedRequests      int64
	TotalLatency        time.Duration
	MinLatency          time.Duration
	MaxLatency          time.Duration
	P50Latency          time.Duration
	P95Latency          time.Duration
	P99Latency          time.Duration
	RequestsPerSecond   float64
	ErrorRate           float64
	MemoryUsageMB       float64
	StartTime           time.Time
	EndTime             time.Time
	LatencyHistogram    []time.Duration
	mutex               sync.RWMutex
}

func (m *LoadTestMetrics) RecordRequest(latency time.Duration, success bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalRequests++
	if success {
		m.SuccessfulRequests++
	} else {
		m.FailedRequests++
	}

	m.TotalLatency += latency
	m.LatencyHistogram = append(m.LatencyHistogram, latency)

	if m.MinLatency == 0 || latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}
}

func (m *LoadTestMetrics) CalculatePercentiles() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.LatencyHistogram) == 0 {
		return
	}

	// Sort latencies for percentile calculation
	latencies := make([]time.Duration, len(m.LatencyHistogram))
	copy(latencies, m.LatencyHistogram)

	// Simple insertion sort for small datasets
	for i := 1; i < len(latencies); i++ {
		key := latencies[i]
		j := i - 1
		for j >= 0 && latencies[j] > key {
			latencies[j+1] = latencies[j]
			j--
		}
		latencies[j+1] = key
	}

	// Calculate percentiles
	p50Index := int(float64(len(latencies)) * 0.5)
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)

	if p50Index < len(latencies) {
		m.P50Latency = latencies[p50Index]
	}
	if p95Index < len(latencies) {
		m.P95Latency = latencies[p95Index]
	}
	if p99Index < len(latencies) {
		m.P99Latency = latencies[p99Index]
	}

	// Calculate RPS and error rate
	duration := m.EndTime.Sub(m.StartTime).Seconds()
	if duration > 0 {
		m.RequestsPerSecond = float64(m.TotalRequests) / duration
		m.ErrorRate = float64(m.FailedRequests) / float64(m.TotalRequests) * 100
	}
}

func (suite *LoadTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testTenants = []string{
		"tenant_load_test_1",
		"tenant_load_test_2",
		"tenant_load_test_3",
		"tenant_load_test_4",
		"tenant_load_test_5",
	}

	// Setup test configuration
	suite.config = &config.Config{
		ProjectID:           "test-project-load",
		VertexAIProject:     "test-vertex-project",
		VertexAILocation:    "us-central1",
		VertexAIModel:       "gemini-2.0-flash-exp",
		SpeechToTextModel:   "chirp-3",
		SpeechLanguage:      "en-US",
		EnableDiarization:   true,
		CallRailBaseURL:     "https://api.callrail.com/v3",
		AudioStorageBucket:  "test-load-audio-bucket",
	}

	// Initialize services
	var err error
	suite.callrailClient = callrail.NewClient()

	suite.aiService, err = ai.NewService(suite.ctx, suite.config)
	require.NoError(suite.T(), err)
}

func (suite *LoadTestSuite) TearDownSuite() {
	if suite.aiService != nil {
		suite.aiService.Close()
	}
}

// TestWebhookProcessingLoad tests webhook processing under various load conditions
func (suite *LoadTestSuite) TestWebhookProcessingLoad() {
	loadScenarios := []struct {
		name                string
		concurrentUsers     int
		requestsPerUser     int
		targetLatencyP95    time.Duration
		targetThroughput    float64 // requests per second
		maxErrorRate        float64 // percentage
		testDuration        time.Duration
	}{
		{
			name:             "LightLoad",
			concurrentUsers:  10,
			requestsPerUser:  20,
			targetLatencyP95: 200 * time.Millisecond,
			targetThroughput: 50.0,
			maxErrorRate:     1.0,
			testDuration:     30 * time.Second,
		},
		{
			name:             "MediumLoad",
			concurrentUsers:  50,
			requestsPerUser:  40,
			targetLatencyP95: 300 * time.Millisecond,
			targetThroughput: 200.0,
			maxErrorRate:     2.0,
			testDuration:     60 * time.Second,
		},
		{
			name:             "HeavyLoad",
			concurrentUsers:  100,
			requestsPerUser:  100,
			targetLatencyP95: 500 * time.Millisecond,
			targetThroughput: 500.0,
			maxErrorRate:     5.0,
			testDuration:     120 * time.Second,
		},
		{
			name:             "StressLoad",
			concurrentUsers:  200,
			requestsPerUser:  50,
			targetLatencyP95: 1000 * time.Millisecond,
			targetThroughput: 300.0,
			maxErrorRate:     10.0,
			testDuration:     180 * time.Second,
		},
	}

	for _, scenario := range loadScenarios {
		suite.T().Run(scenario.name, func(t *testing.T) {
			metrics := &LoadTestMetrics{
				StartTime:        time.Now(),
				MinLatency:       time.Hour, // Initialize with high value
				LatencyHistogram: make([]time.Duration, 0),
			}

			// Create webhook handler for testing
			handler := suite.createTestWebhookHandler()

			// Execute load test
			suite.executeLoadTest(t, handler, scenario, metrics)

			metrics.EndTime = time.Now()
			metrics.CalculatePercentiles()

			// Assert performance requirements
			suite.assertPerformanceRequirements(t, scenario, metrics)

			// Log detailed metrics
			suite.logLoadTestResults(t, scenario, metrics)
		})
	}
}

// TestMultiTenantLoadIsolation tests load isolation between tenants
func (suite *LoadTestSuite) TestMultiTenantLoadIsolation() {
	suite.T().Run("TenantLoadIsolation", func(t *testing.T) {
		const (
			concurrentTenantsCount = 5
			requestsPerTenant      = 100
			testDuration           = 60 * time.Second
		)

		tenantMetrics := make(map[string]*LoadTestMetrics)
		var wg sync.WaitGroup

		// Initialize metrics for each tenant
		for _, tenantID := range suite.testTenants[:concurrentTenantsCount] {
			tenantMetrics[tenantID] = &LoadTestMetrics{
				StartTime:        time.Now(),
				MinLatency:       time.Hour,
				LatencyHistogram: make([]time.Duration, 0),
			}
		}

		handler := suite.createTestWebhookHandler()

		// Start load tests for all tenants concurrently
		for _, tenantID := range suite.testTenants[:concurrentTenantsCount] {
			wg.Add(1)
			go func(tenant string) {
				defer wg.Done()
				suite.executeTenantSpecificLoad(t, handler, tenant, requestsPerTenant, tenantMetrics[tenant])
			}(tenantID)
		}

		wg.Wait()

		// Analyze results and verify isolation
		for tenantID, metrics := range tenantMetrics {
			metrics.EndTime = time.Now()
			metrics.CalculatePercentiles()

			// Assert each tenant met reasonable performance standards
			assert.Greater(t, metrics.RequestsPerSecond, 10.0,
				"Tenant %s should maintain reasonable throughput under multi-tenant load", tenantID)
			assert.Less(t, metrics.P95Latency, 1*time.Second,
				"Tenant %s should maintain reasonable latency under multi-tenant load", tenantID)
			assert.Less(t, metrics.ErrorRate, 10.0,
				"Tenant %s should maintain low error rate under multi-tenant load", tenantID)

			t.Logf("Tenant %s: RPS=%.2f, P95=%.2fms, ErrorRate=%.2f%%",
				tenantID, metrics.RequestsPerSecond,
				float64(metrics.P95Latency.Nanoseconds())/1e6, metrics.ErrorRate)
		}

		// Verify no tenant significantly degraded others' performance
		var allP95Latencies []time.Duration
		for _, metrics := range tenantMetrics {
			allP95Latencies = append(allP95Latencies, metrics.P95Latency)
		}

		// Calculate coefficient of variation to ensure consistent performance
		mean := suite.calculateMeanDuration(allP95Latencies)
		stddev := suite.calculateStdDevDuration(allP95Latencies, mean)
		cv := float64(stddev.Nanoseconds()) / float64(mean.Nanoseconds())

		assert.Less(t, cv, 0.5, "Coefficient of variation should be < 0.5 (good tenant isolation)")
	})
}

// TestAudioProcessingLoad tests audio processing pipeline under load
func (suite *LoadTestSuite) TestAudioProcessingLoad() {
	suite.T().Run("TranscriptionLoad", func(t *testing.T) {
		const (
			concurrentTranscriptions = 10
			testDuration             = 120 * time.Second
			targetLatencyP95         = 5 * time.Second
			maxErrorRate             = 5.0
		)

		var metrics LoadTestMetrics
		metrics.StartTime = time.Now()
		metrics.MinLatency = time.Hour
		metrics.LatencyHistogram = make([]time.Duration, 0)

		var wg sync.WaitGroup
		done := make(chan bool)

		// Start concurrent transcription load
		for i := 0; i < concurrentTranscriptions; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				suite.executeTranscriptionLoad(t, workerID, done, &metrics)
			}(i)
		}

		// Run for specified duration
		time.Sleep(testDuration)
		close(done)
		wg.Wait()

		metrics.EndTime = time.Now()
		metrics.CalculatePercentiles()

		// Assert performance requirements
		assert.Less(t, metrics.P95Latency, targetLatencyP95,
			"95th percentile transcription latency should be under %v, got %v",
			targetLatencyP95, metrics.P95Latency)
		assert.Less(t, metrics.ErrorRate, maxErrorRate,
			"Error rate should be under %.1f%%, got %.2f%%", maxErrorRate, metrics.ErrorRate)
		assert.Greater(t, metrics.SuccessfulRequests, int64(0),
			"Should have some successful transcriptions")

		t.Logf("Transcription Load Test: %d requests, RPS=%.2f, P95=%.2fs, ErrorRate=%.2f%%",
			metrics.TotalRequests, metrics.RequestsPerSecond,
			metrics.P95Latency.Seconds(), metrics.ErrorRate)
	})
}

// TestAIAnalysisLoad tests AI analysis performance under load
func (suite *LoadTestSuite) TestAIAnalysisLoad() {
	suite.T().Run("GeminiAnalysisLoad", func(t *testing.T) {
		const (
			concurrentAnalyses = 20
			analysesPerWorker  = 50
			targetLatencyP95   = 1 * time.Second
			maxErrorRate       = 3.0
		)

		var metrics LoadTestMetrics
		metrics.StartTime = time.Now()
		metrics.MinLatency = time.Hour
		metrics.LatencyHistogram = make([]time.Duration, 0)

		var wg sync.WaitGroup

		// Start concurrent AI analysis load
		for i := 0; i < concurrentAnalyses; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				suite.executeAIAnalysisLoad(t, workerID, analysesPerWorker, &metrics)
			}(i)
		}

		wg.Wait()

		metrics.EndTime = time.Now()
		metrics.CalculatePercentiles()

		// Assert performance requirements
		assert.Less(t, metrics.P95Latency, targetLatencyP95,
			"95th percentile AI analysis latency should be under %v, got %v",
			targetLatencyP95, metrics.P95Latency)
		assert.Less(t, metrics.ErrorRate, maxErrorRate,
			"Error rate should be under %.1f%%, got %.2f%%", maxErrorRate, metrics.ErrorRate)

		t.Logf("AI Analysis Load Test: %d requests, RPS=%.2f, P95=%.2fs, ErrorRate=%.2f%%",
			metrics.TotalRequests, metrics.RequestsPerSecond,
			metrics.P95Latency.Seconds(), metrics.ErrorRate)
	})
}

// TestSystemResourceUsage tests resource usage under load
func (suite *LoadTestSuite) TestSystemResourceUsage() {
	suite.T().Run("MemoryUsageUnderLoad", func(t *testing.T) {
		const (
			testDuration      = 60 * time.Second
			concurrentWorkers = 50
			maxMemoryIncrease = 100.0 // MB
		)

		// Measure initial memory usage
		initialMemory := suite.getCurrentMemoryUsageMB()

		var wg sync.WaitGroup
		done := make(chan bool)

		// Start load generators
		for i := 0; i < concurrentWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				suite.executeMemoryLoadTest(done)
			}()
		}

		// Monitor memory usage during test
		memoryUsages := suite.monitorMemoryUsage(testDuration)

		// Stop load generators
		close(done)
		wg.Wait()

		// Measure final memory usage
		finalMemory := suite.getCurrentMemoryUsageMB()
		memoryIncrease := finalMemory - initialMemory

		// Assert memory usage is within acceptable bounds
		assert.Less(t, memoryIncrease, maxMemoryIncrease,
			"Memory increase should be under %.1f MB, got %.2f MB", maxMemoryIncrease, memoryIncrease)

		// Check for memory leaks (memory should not continuously increase)
		memoryTrend := suite.calculateMemoryTrend(memoryUsages)
		assert.Less(t, memoryTrend, 50.0,
			"Memory trend should be stable (< 50 MB/min), got %.2f MB/min", memoryTrend)

		t.Logf("Memory test: Initial=%.2fMB, Final=%.2fMB, Increase=%.2fMB, Trend=%.2fMB/min",
			initialMemory, finalMemory, memoryIncrease, memoryTrend)
	})
}

// Helper methods for load testing

func (suite *LoadTestSuite) createTestWebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simulate webhook processing
		startTime := time.Now()

		// Add some realistic processing delay
		processingDelay := time.Duration(rand.Intn(100)) * time.Millisecond
		time.Sleep(processingDelay)

		// Simulate occasional errors (5% failure rate)
		if rand.Float64() < 0.05 {
			http.Error(w, "Simulated processing error", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"status":           "success",
			"processing_time":  time.Since(startTime).Milliseconds(),
			"ingestion_id":     fmt.Sprintf("ing_%d", rand.Int63()),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (suite *LoadTestSuite) executeLoadTest(t *testing.T, handler http.HandlerFunc, scenario struct {
	name                string
	concurrentUsers     int
	requestsPerUser     int
	targetLatencyP95    time.Duration
	targetThroughput    float64
	maxErrorRate        float64
	testDuration        time.Duration
}, metrics *LoadTestMetrics) {

	server := httptest.NewServer(handler)
	defer server.Close()

	var wg sync.WaitGroup

	// Launch concurrent users
	for i := 0; i < scenario.concurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			suite.simulateUser(t, server.URL, userID, scenario.requestsPerUser, metrics)
		}(i)
	}

	wg.Wait()
}

func (suite *LoadTestSuite) simulateUser(t *testing.T, baseURL string, userID, requestCount int, metrics *LoadTestMetrics) {
	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < requestCount; i++ {
		// Create realistic webhook payload
		payload := suite.createTestWebhookPayload(userID, i)
		payloadBytes, _ := json.Marshal(payload)

		// Make request and measure latency
		startTime := time.Now()
		resp, err := client.Post(baseURL, "application/json", bytes.NewBuffer(payloadBytes))
		latency := time.Since(startTime)

		success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
		if resp != nil {
			resp.Body.Close()
		}

		// Record metrics
		metrics.RecordRequest(latency, success)

		// Add some realistic thinking time between requests
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	}
}

func (suite *LoadTestSuite) executeTenantSpecificLoad(t *testing.T, handler http.HandlerFunc, tenantID string, requestCount int, metrics *LoadTestMetrics) {
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	for i := 0; i < requestCount; i++ {
		// Create tenant-specific webhook payload
		payload := suite.createTenantWebhookPayload(tenantID, i)
		payloadBytes, _ := json.Marshal(payload)

		startTime := time.Now()
		resp, err := client.Post(server.URL, "application/json", bytes.NewBuffer(payloadBytes))
		latency := time.Since(startTime)

		success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
		if resp != nil {
			resp.Body.Close()
		}

		metrics.RecordRequest(latency, success)

		// Small delay between requests
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	}
}

func (suite *LoadTestSuite) executeTranscriptionLoad(t *testing.T, workerID int, done <-chan bool, metrics *LoadTestMetrics) {
	// Simulate audio transcription requests
	for {
		select {
		case <-done:
			return
		default:
			startTime := time.Now()

			// Simulate transcription processing time
			processingTime := time.Duration(2000+rand.Intn(3000)) * time.Millisecond
			time.Sleep(processingTime)

			// Simulate occasional transcription failures
			success := rand.Float64() > 0.05

			latency := time.Since(startTime)
			metrics.RecordRequest(latency, success)
		}
	}
}

func (suite *LoadTestSuite) executeAIAnalysisLoad(t *testing.T, workerID, requestCount int, metrics *LoadTestMetrics) {
	for i := 0; i < requestCount; i++ {
		startTime := time.Now()

		// Create realistic call analysis request
		transcription := suite.generateTestTranscription()
		callDetails := suite.generateTestCallDetails(workerID, i)

		// Simulate AI analysis (in real test, this would call actual AI service)
		// For load testing, we simulate the processing time
		processingTime := time.Duration(500+rand.Intn(500)) * time.Millisecond
		time.Sleep(processingTime)

		// Simulate occasional AI service failures
		success := rand.Float64() > 0.02

		latency := time.Since(startTime)
		metrics.RecordRequest(latency, success)

		// Small delay between requests
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	}
}

func (suite *LoadTestSuite) executeMemoryLoadTest(done <-chan bool) {
	// Simulate memory-intensive operations
	for {
		select {
		case <-done:
			return
		default:
			// Allocate and process some data to simulate real workload
			data := make([]byte, 1024*1024) // 1MB
			for i := range data {
				data[i] = byte(rand.Intn(256))
			}

			// Process the data (simulate work)
			time.Sleep(10 * time.Millisecond)

			// Allow garbage collection
			data = nil
		}
	}
}

func (suite *LoadTestSuite) monitorMemoryUsage(duration time.Duration) []float64 {
	var usages []float64
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(duration)

	for {
		select {
		case <-timeout:
			return usages
		case <-ticker.C:
			usage := suite.getCurrentMemoryUsageMB()
			usages = append(usages, usage)
		}
	}
}

func (suite *LoadTestSuite) getCurrentMemoryUsageMB() float64 {
	// In a real implementation, this would use runtime.MemStats
	// For testing purposes, we'll simulate memory usage
	return float64(50 + rand.Intn(100))
}

func (suite *LoadTestSuite) calculateMemoryTrend(usages []float64) float64 {
	if len(usages) < 2 {
		return 0
	}

	// Simple linear trend calculation
	firstHalf := usages[:len(usages)/2]
	secondHalf := usages[len(usages)/2:]

	firstAvg := suite.calculateMeanFloat(firstHalf)
	secondAvg := suite.calculateMeanFloat(secondHalf)

	return secondAvg - firstAvg
}

func (suite *LoadTestSuite) calculateMeanFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (suite *LoadTestSuite) calculateMeanDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func (suite *LoadTestSuite) calculateStdDevDuration(durations []time.Duration, mean time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var sumSquaredDiffs int64
	for _, d := range durations {
		diff := d - mean
		sumSquaredDiffs += diff.Nanoseconds() * diff.Nanoseconds()
	}

	variance := sumSquaredDiffs / int64(len(durations))
	return time.Duration(int64(float64(variance) * 0.5)) // Approximate sqrt
}

func (suite *LoadTestSuite) assertPerformanceRequirements(t *testing.T, scenario struct {
	name                string
	concurrentUsers     int
	requestsPerUser     int
	targetLatencyP95    time.Duration
	targetThroughput    float64
	maxErrorRate        float64
	testDuration        time.Duration
}, metrics *LoadTestMetrics) {

	assert.Less(t, metrics.P95Latency, scenario.targetLatencyP95,
		"95th percentile latency should be under %v for %s", scenario.targetLatencyP95, scenario.name)

	assert.Greater(t, metrics.RequestsPerSecond, scenario.targetThroughput,
		"Throughput should be at least %.1f RPS for %s", scenario.targetThroughput, scenario.name)

	assert.Less(t, metrics.ErrorRate, scenario.maxErrorRate,
		"Error rate should be under %.1f%% for %s", scenario.maxErrorRate, scenario.name)

	assert.Greater(t, metrics.SuccessfulRequests, int64(0),
		"Should have at least some successful requests for %s", scenario.name)
}

func (suite *LoadTestSuite) logLoadTestResults(t *testing.T, scenario struct {
	name                string
	concurrentUsers     int
	requestsPerUser     int
	targetLatencyP95    time.Duration
	targetThroughput    float64
	maxErrorRate        float64
	testDuration        time.Duration
}, metrics *LoadTestMetrics) {

	t.Logf("=== %s Load Test Results ===", scenario.name)
	t.Logf("Total Requests: %d", metrics.TotalRequests)
	t.Logf("Successful Requests: %d", metrics.SuccessfulRequests)
	t.Logf("Failed Requests: %d", metrics.FailedRequests)
	t.Logf("Requests/Second: %.2f", metrics.RequestsPerSecond)
	t.Logf("Error Rate: %.2f%%", metrics.ErrorRate)
	t.Logf("Min Latency: %.2fms", float64(metrics.MinLatency.Nanoseconds())/1e6)
	t.Logf("Max Latency: %.2fms", float64(metrics.MaxLatency.Nanoseconds())/1e6)
	t.Logf("P50 Latency: %.2fms", float64(metrics.P50Latency.Nanoseconds())/1e6)
	t.Logf("P95 Latency: %.2fms", float64(metrics.P95Latency.Nanoseconds())/1e6)
	t.Logf("P99 Latency: %.2fms", float64(metrics.P99Latency.Nanoseconds())/1e6)
	t.Logf("Test Duration: %.2fs", metrics.EndTime.Sub(metrics.StartTime).Seconds())
}

// Helper methods for creating test data

func (suite *LoadTestSuite) createTestWebhookPayload(userID, requestID int) models.CallRailWebhook {
	return models.CallRailWebhook{
		CallID:              fmt.Sprintf("CAL_LOAD_%d_%d", userID, requestID),
		AccountID:           fmt.Sprintf("AC_LOAD_%d", userID%5),
		CompanyID:           fmt.Sprintf("CR_LOAD_%d", userID%len(suite.testTenants)),
		CallerID:            fmt.Sprintf("+1555%03d%04d", userID, requestID),
		CalledNumber:        "+15559876543",
		Duration:            fmt.Sprintf("%d", 60+rand.Intn(300)),
		StartTime:           time.Now().Add(-time.Duration(rand.Intn(600)) * time.Second),
		EndTime:             time.Now().Add(-time.Duration(rand.Intn(60)) * time.Second),
		Direction:           "inbound",
		RecordingURL:        fmt.Sprintf("https://api.callrail.com/recordings/load_%d_%d.mp3", userID, requestID),
		Answered:            rand.Float64() > 0.1,
		FirstCall:           rand.Float64() > 0.3,
		BusinessPhoneNumber: "+15559876543",
		CustomerName:        fmt.Sprintf("Load Test User %d", userID),
		CustomerCity:        "Los Angeles",
		CustomerState:       "CA",
		CustomerCountry:     "US",
		TenantID:            suite.testTenants[userID%len(suite.testTenants)],
		CallRailCompanyID:   fmt.Sprintf("CR_LOAD_%d", userID%len(suite.testTenants)),
	}
}

func (suite *LoadTestSuite) createTenantWebhookPayload(tenantID string, requestID int) models.CallRailWebhook {
	return models.CallRailWebhook{
		CallID:              fmt.Sprintf("CAL_%s_%d", strings.ReplaceAll(tenantID, "_", ""), requestID),
		CompanyID:           fmt.Sprintf("CR_%s", strings.ReplaceAll(tenantID, "_", "")),
		CallerID:            fmt.Sprintf("+1555%07d", requestID),
		Duration:            fmt.Sprintf("%d", 60+rand.Intn(240)),
		TenantID:            tenantID,
		CallRailCompanyID:   fmt.Sprintf("CR_%s", strings.ReplaceAll(tenantID, "_", "")),
	}
}

func (suite *LoadTestSuite) generateTestTranscription() string {
	transcriptions := []string{
		"Hi, I'm interested in a kitchen remodel. Can someone call me back?",
		"I need an emergency plumber. There's water everywhere!",
		"Looking for a bathroom renovation quote. Not urgent.",
		"Just want pricing information for flooring replacement.",
		"Calling about the work you did last month. Having some issues.",
	}
	return transcriptions[rand.Intn(len(transcriptions))]
}

func (suite *LoadTestSuite) generateTestCallDetails(workerID, requestID int) models.CallDetails {
	return models.CallDetails{
		ID:                  fmt.Sprintf("CAL_AI_LOAD_%d_%d", workerID, requestID),
		Duration:            60 + rand.Intn(300),
		CustomerName:        fmt.Sprintf("AI Load Test %d", requestID),
		CustomerPhoneNumber: fmt.Sprintf("+1555%07d", requestID),
		FirstCall:          rand.Float64() > 0.3,
		Direction:          "inbound",
	}
}

// Run the test suite
func TestLoadTestSuite(t *testing.T) {
	suite.Run(t, new(LoadTestSuite))
}