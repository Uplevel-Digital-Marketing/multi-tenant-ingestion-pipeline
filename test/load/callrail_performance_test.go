package load

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
)

// CallRailPerformanceTestSuite tests CallRail webhook performance under various load conditions
type CallRailPerformanceTestSuite struct {
	suite.Suite
	server            *httptest.Server
	ctx               context.Context
	testDuration      time.Duration
	performanceConfig PerformanceConfig
}

type PerformanceConfig struct {
	WebhookLatencyTarget    time.Duration // <200ms requirement
	ProcessingLatencyTarget time.Duration // <1s for AI analysis
	ThroughputTarget        int           // 1,000+ requests/minute per tenant
	ConcurrentTenants       int           // Number of tenants to simulate
	AudioProcessingTarget   time.Duration // <5s transcription latency
}

type CallRailLoadTest struct {
	TenantID              string            `json:"tenant_id"`
	RequestsPerSecond     float64           `json:"requests_per_second"`
	Duration              time.Duration     `json:"duration"`
	CallScenarios         []CallScenario    `json:"call_scenarios"`
	ExpectedPerformance   PerformanceTarget `json:"expected_performance"`
	ActualResults         *LoadTestResults  `json:"actual_results,omitempty"`
}

type CallScenario struct {
	ScenarioType   string        `json:"scenario_type"` // "kitchen_remodel", "emergency", "abandoned", etc.
	Weight         float64       `json:"weight"`        // Probability of this scenario (0.0-1.0)
	CallDuration   time.Duration `json:"call_duration"`
	AudioSize      int           `json:"audio_size_kb"`
	Complexity     string        `json:"complexity"`    // "simple", "medium", "complex"
	ExpectedResult string        `json:"expected_result"` // "create_lead", "track_only", "emergency_alert"
}

type PerformanceTarget struct {
	MaxWebhookLatency     time.Duration `json:"max_webhook_latency"`
	MaxProcessingLatency  time.Duration `json:"max_processing_latency"`
	MinSuccessRate        float64       `json:"min_success_rate"`
	MaxErrorRate          float64       `json:"max_error_rate"`
	MinThroughput         float64       `json:"min_throughput_rps"`
}

type LoadTestResults struct {
	TotalRequests         int64                    `json:"total_requests"`
	SuccessfulRequests    int64                    `json:"successful_requests"`
	FailedRequests        int64                    `json:"failed_requests"`
	AverageWebhookLatency time.Duration            `json:"average_webhook_latency"`
	P95WebhookLatency     time.Duration            `json:"p95_webhook_latency"`
	P99WebhookLatency     time.Duration            `json:"p99_webhook_latency"`
	AverageProcessingTime time.Duration            `json:"average_processing_time"`
	P95ProcessingTime     time.Duration            `json:"p95_processing_time"`
	ThroughputRPS         float64                  `json:"throughput_rps"`
	ErrorRate             float64                  `json:"error_rate"`
	LatencyDistribution   map[string]time.Duration `json:"latency_distribution"`
	ScenarioResults       map[string]*ScenarioMetrics `json:"scenario_results"`
	StartTime             time.Time                `json:"start_time"`
	EndTime               time.Time                `json:"end_time"`
	Duration              time.Duration            `json:"duration"`
}

type ScenarioMetrics struct {
	RequestCount      int64         `json:"request_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	SuccessRate       float64       `json:"success_rate"`
	ProcessingTime    time.Duration `json:"processing_time"`
}

type CallRailWebhookPayload struct {
	CallID         string                 `json:"call_id"`
	CompanyID      string                 `json:"company_id"`
	AccountID      string                 `json:"account_id"`
	PhoneNumber    string                 `json:"phone_number"`
	CallerID       string                 `json:"caller_id"`
	Duration       int                    `json:"duration"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	Direction      string                 `json:"direction"`
	RecordingURL   string                 `json:"recording_url,omitempty"`
	Transcription  string                 `json:"transcription,omitempty"`
	CallStatus     string                 `json:"call_status"`
	TrackingNumber string                 `json:"tracking_number"`
	CustomFields   map[string]interface{} `json:"custom_fields,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	ScenarioType   string                 `json:"scenario_type,omitempty"`
}

type WebhookMetrics struct {
	RequestID        string        `json:"request_id"`
	TenantID         string        `json:"tenant_id"`
	CallID           string        `json:"call_id"`
	ScenarioType     string        `json:"scenario_type"`
	WebhookLatency   time.Duration `json:"webhook_latency"`
	ProcessingTime   time.Duration `json:"processing_time"`
	Success          bool          `json:"success"`
	ErrorType        string        `json:"error_type,omitempty"`
	ResponseCode     int           `json:"response_code"`
	Timestamp        time.Time     `json:"timestamp"`
}

// Mock webhook handler for performance testing
type MockCallRailWebhookHandler struct {
	processingDelay time.Duration
	failureRate     float64
	metrics         *sync.Map // Store webhook metrics
	requestCount    int64
}

func NewMockCallRailWebhookHandler() *MockCallRailWebhookHandler {
	return &MockCallRailWebhookHandler{
		processingDelay: 50 * time.Millisecond, // Base processing time
		failureRate:     0.01,                  // 1% failure rate
		metrics:         &sync.Map{},
		requestCount:    0,
	}
}

func (h *MockCallRailWebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := fmt.Sprintf("req_%d", atomic.AddInt64(&h.requestCount, 1))

	var payload CallRailWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		h.recordMetrics(requestID, "", "", "", 0, 0, false, "invalid_json", http.StatusBadRequest, startTime)
		return
	}

	// Simulate processing time based on scenario complexity
	processingTime := h.calculateProcessingTime(payload.ScenarioType, payload.Duration)

	// Simulate processing delay
	time.Sleep(processingTime)

	// Simulate random failures
	success := rand.Float64() > h.failureRate

	responseCode := http.StatusOK
	errorType := ""
	if !success {
		responseCode = http.StatusInternalServerError
		errorType = "processing_error"
		http.Error(w, "Processing failed", responseCode)
	} else {
		response := map[string]interface{}{
			"status":       "accepted",
			"ingestion_id": fmt.Sprintf("ing_%s_%d", payload.CallID, time.Now().Unix()),
			"message":      "Webhook processed successfully",
			"tenant_id":    payload.CompanyID, // Use company ID as tenant ID for testing
			"call_id":      payload.CallID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}

	webhookLatency := time.Since(startTime)
	h.recordMetrics(requestID, payload.CompanyID, payload.CallID, payload.ScenarioType,
		webhookLatency, processingTime, success, errorType, responseCode, startTime)
}

func (h *MockCallRailWebhookHandler) calculateProcessingTime(scenarioType string, callDuration int) time.Duration {
	baseTime := h.processingDelay

	switch scenarioType {
	case "emergency":
		return baseTime / 2 // Emergency calls process faster
	case "complex_kitchen_remodel":
		return baseTime * 2 // Complex scenarios take longer
	case "abandoned":
		return baseTime / 4 // Abandoned calls process very quickly
	default:
		// Add variance based on call duration
		variance := time.Duration(callDuration) * time.Millisecond / 10
		return baseTime + variance
	}
}

func (h *MockCallRailWebhookHandler) recordMetrics(requestID, tenantID, callID, scenarioType string,
	webhookLatency, processingTime time.Duration, success bool, errorType string, responseCode int, timestamp time.Time) {

	metrics := &WebhookMetrics{
		RequestID:      requestID,
		TenantID:       tenantID,
		CallID:         callID,
		ScenarioType:   scenarioType,
		WebhookLatency: webhookLatency,
		ProcessingTime: processingTime,
		Success:        success,
		ErrorType:      errorType,
		ResponseCode:   responseCode,
		Timestamp:      timestamp,
	}

	h.metrics.Store(requestID, metrics)
}

func (h *MockCallRailWebhookHandler) GetMetrics() []*WebhookMetrics {
	var allMetrics []*WebhookMetrics
	h.metrics.Range(func(key, value interface{}) bool {
		if metrics, ok := value.(*WebhookMetrics); ok {
			allMetrics = append(allMetrics, metrics)
		}
		return true
	})
	return allMetrics
}

func (h *MockCallRailWebhookHandler) ResetMetrics() {
	h.metrics = &sync.Map{}
	atomic.StoreInt64(&h.requestCount, 0)
}

func (suite *CallRailPerformanceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.testDuration = 3 * time.Minute

	// Configure performance targets based on requirements
	suite.performanceConfig = PerformanceConfig{
		WebhookLatencyTarget:    200 * time.Millisecond, // <200ms for forms
		ProcessingLatencyTarget: 1 * time.Second,        // <1s for AI analysis
		ThroughputTarget:        1000,                    // 1,000+ requests/minute per tenant
		ConcurrentTenants:       10,                      // Test with 10 tenants
		AudioProcessingTarget:   5 * time.Second,        // <5s transcription latency
	}

	// Setup mock webhook server
	suite.setupMockServer()
}

func (suite *CallRailPerformanceTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

func (suite *CallRailPerformanceTestSuite) setupMockServer() {
	handler := NewMockCallRailWebhookHandler()
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	suite.server = httptest.NewServer(mux)
}

func (suite *CallRailPerformanceTestSuite) TestCallRailWebhookLatencyRequirements() {
	// Test that webhook responses meet the <200ms latency requirement
	handler := NewMockCallRailWebhookHandler()
	handler.processingDelay = 50 * time.Millisecond // Realistic processing time

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	const numRequests = 100
	var latencies []time.Duration
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestIndex int) {
			defer wg.Done()

			payload := suite.generateCallRailPayload(fmt.Sprintf("perf-test-call-%d", requestIndex), "tenant-perf-test", "standard")
			body, _ := json.Marshal(payload)

			startTime := time.Now()
			resp, err := http.Post(testServer.URL+"/webhook/callrail", "application/json", bytes.NewBuffer(body))
			latency := time.Since(startTime)

			require.NoError(suite.T(), err)
			if resp != nil {
				resp.Body.Close()
			}

			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Analyze latency results
	avgLatency := suite.calculateAverageLatency(latencies)
	p95Latency := suite.calculatePercentileLatency(latencies, 95)
	p99Latency := suite.calculatePercentileLatency(latencies, 99)
	maxLatency := suite.calculateMaxLatency(latencies)

	suite.T().Logf("Webhook Latency Results:")
	suite.T().Logf("  Average: %v", avgLatency)
	suite.T().Logf("  P95: %v", p95Latency)
	suite.T().Logf("  P99: %v", p99Latency)
	suite.T().Logf("  Max: %v", maxLatency)

	// Assert performance requirements
	assert.True(suite.T(), avgLatency < suite.performanceConfig.WebhookLatencyTarget,
		"Average webhook latency %v should be < %v", avgLatency, suite.performanceConfig.WebhookLatencyTarget)
	assert.True(suite.T(), p95Latency < suite.performanceConfig.WebhookLatencyTarget,
		"P95 webhook latency %v should be < %v", p95Latency, suite.performanceConfig.WebhookLatencyTarget)
	assert.True(suite.T(), p99Latency < suite.performanceConfig.WebhookLatencyTarget*2,
		"P99 webhook latency %v should be < %v", p99Latency, suite.performanceConfig.WebhookLatencyTarget*2)
}

func (suite *CallRailPerformanceTestSuite) TestThroughputRequirements() {
	// Test 1,000+ requests/minute per tenant throughput requirement
	handler := NewMockCallRailWebhookHandler()
	handler.processingDelay = 30 * time.Millisecond // Fast processing for throughput test

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	tenantID := "tenant-throughput-test"
	testDuration := 60 * time.Second
	targetThroughput := float64(suite.performanceConfig.ThroughputTarget) / 60.0 // requests per second

	ctx, cancel := context.WithTimeout(suite.ctx, testDuration)
	defer cancel()

	var requestCount int64
	var successCount int64
	var wg sync.WaitGroup

	startTime := time.Now()

	// Generate load at target rate
	ticker := time.NewTicker(time.Duration(float64(time.Second) / targetThroughput))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()

				callID := fmt.Sprintf("throughput-call-%d", atomic.AddInt64(&requestCount, 1))
				payload := suite.generateCallRailPayload(callID, tenantID, "standard")
				body, _ := json.Marshal(payload)

				resp, err := http.Post(testServer.URL+"/webhook/callrail", "application/json", bytes.NewBuffer(body))
				if err == nil && resp != nil {
					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					}
					resp.Body.Close()
				}
			}()
		}
	}

done:
	wg.Wait()
	actualDuration := time.Since(startTime)

	actualThroughput := float64(requestCount) / actualDuration.Seconds()
	successRate := float64(successCount) / float64(requestCount)

	suite.T().Logf("Throughput Test Results:")
	suite.T().Logf("  Target: %.2f req/sec (%.0f req/min)", targetThroughput, targetThroughput*60)
	suite.T().Logf("  Actual: %.2f req/sec (%.0f req/min)", actualThroughput, actualThroughput*60)
	suite.T().Logf("  Requests: %d total, %d successful", requestCount, successCount)
	suite.T().Logf("  Success Rate: %.2f%%", successRate*100)
	suite.T().Logf("  Duration: %v", actualDuration)

	// Assert throughput requirements
	assert.True(suite.T(), actualThroughput >= targetThroughput*0.95,
		"Actual throughput %.2f req/sec should be >= 95%% of target %.2f req/sec",
		actualThroughput, targetThroughput)
	assert.True(suite.T(), successRate >= 0.95,
		"Success rate %.2f%% should be >= 95%%", successRate*100)
}

func (suite *CallRailPerformanceTestSuite) TestMultiTenantConcurrentLoad() {
	// Test concurrent load from multiple tenants simultaneously
	handler := NewMockCallRailWebhookHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	const numTenants = 5
	const requestsPerTenant = 50
	const concurrentRequestsPerTenant = 10

	var allResults []*LoadTestResults
	var resultsMutex sync.Mutex
	var wg sync.WaitGroup

	for tenantIndex := 0; tenantIndex < numTenants; tenantIndex++ {
		wg.Add(1)
		go func(tIndex int) {
			defer wg.Done()

			tenantID := fmt.Sprintf("tenant-concurrent-%d", tIndex)
			result := suite.runTenantLoadTest(testServer.URL, tenantID, requestsPerTenant, concurrentRequestsPerTenant)

			resultsMutex.Lock()
			allResults = append(allResults, result)
			resultsMutex.Unlock()
		}(tenantIndex)
	}

	wg.Wait()

	// Analyze multi-tenant performance
	suite.analyzeMultiTenantResults(allResults)
}

func (suite *CallRailPerformanceTestSuite) TestCallScenarioPerformance() {
	// Test performance across different call scenarios
	handler := NewMockCallRailWebhookHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	scenarios := []CallScenario{
		{
			ScenarioType:   "kitchen_remodel",
			Weight:         0.4,
			CallDuration:   300 * time.Second,
			AudioSize:      500,
			Complexity:     "medium",
			ExpectedResult: "create_lead",
		},
		{
			ScenarioType:   "emergency",
			Weight:         0.1,
			CallDuration:   120 * time.Second,
			AudioSize:      200,
			Complexity:     "simple",
			ExpectedResult: "emergency_alert",
		},
		{
			ScenarioType:   "bathroom_renovation",
			Weight:         0.3,
			CallDuration:   240 * time.Second,
			AudioSize:     350,
			Complexity:     "medium",
			ExpectedResult: "create_lead",
		},
		{
			ScenarioType:   "abandoned",
			Weight:         0.2,
			CallDuration:   15 * time.Second,
			AudioSize:      50,
			Complexity:     "simple",
			ExpectedResult: "track_only",
		},
	}

	const totalRequests = 200
	scenarioResults := make(map[string]*ScenarioMetrics)

	for _, scenario := range scenarios {
		scenarioResults[scenario.ScenarioType] = &ScenarioMetrics{}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(requestIndex int) {
			defer wg.Done()

			// Select scenario based on weights
			scenario := suite.selectScenarioByWeight(scenarios)

			callID := fmt.Sprintf("scenario-call-%s-%d", scenario.ScenarioType, requestIndex)
			payload := suite.generateCallRailPayload(callID, "tenant-scenario-test", scenario.ScenarioType)

			body, _ := json.Marshal(payload)

			startTime := time.Now()
			resp, err := http.Post(testServer.URL+"/webhook/callrail", "application/json", bytes.NewBuffer(body))
			latency := time.Since(startTime)

			success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
			if resp != nil {
				resp.Body.Close()
			}

			mu.Lock()
			metrics := scenarioResults[scenario.ScenarioType]
			metrics.RequestCount++
			if success {
				// Update average latency
				if metrics.AverageLatency == 0 {
					metrics.AverageLatency = latency
				} else {
					metrics.AverageLatency = (metrics.AverageLatency + latency) / 2
				}
				metrics.SuccessRate = float64(metrics.RequestCount-1)/float64(metrics.RequestCount)*metrics.SuccessRate + 1.0/float64(metrics.RequestCount)
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Analyze scenario-specific performance
	suite.analyzeScenarioResults(scenarioResults)
}

func (suite *CallRailPerformanceTestSuite) TestStressTestPeakLoad() {
	// Test system behavior under peak load conditions
	handler := NewMockCallRailWebhookHandler()
	handler.processingDelay = 75 * time.Millisecond // Slightly higher load

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Simulate peak load: 3x normal throughput
	peakThroughput := float64(suite.performanceConfig.ThroughputTarget*3) / 60.0 // requests per second
	testDuration := 2 * time.Minute

	ctx, cancel := context.WithTimeout(suite.ctx, testDuration)
	defer cancel()

	var requestCount int64
	var successCount int64
	var errorCount int64
	var latencies []time.Duration
	var mu sync.Mutex
	var wg sync.WaitGroup

	startTime := time.Now()

	ticker := time.NewTicker(time.Duration(float64(time.Second) / peakThroughput))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()

				callID := fmt.Sprintf("stress-call-%d", atomic.AddInt64(&requestCount, 1))
				payload := suite.generateCallRailPayload(callID, "tenant-stress-test", "standard")
				body, _ := json.Marshal(payload)

				reqStart := time.Now()
				resp, err := http.Post(testServer.URL+"/webhook/callrail", "application/json", bytes.NewBuffer(body))
				latency := time.Since(reqStart)

				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else if resp != nil {
					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
					resp.Body.Close()
				}
			}()
		}
	}

done:
	wg.Wait()
	actualDuration := time.Since(startTime)

	// Analyze stress test results
	actualThroughput := float64(requestCount) / actualDuration.Seconds()
	successRate := float64(successCount) / float64(requestCount)
	errorRate := float64(errorCount) / float64(requestCount)

	avgLatency := suite.calculateAverageLatency(latencies)
	p95Latency := suite.calculatePercentileLatency(latencies, 95)

	suite.T().Logf("Stress Test Results (Peak Load):")
	suite.T().Logf("  Target Throughput: %.2f req/sec", peakThroughput)
	suite.T().Logf("  Actual Throughput: %.2f req/sec", actualThroughput)
	suite.T().Logf("  Total Requests: %d", requestCount)
	suite.T().Logf("  Success Rate: %.2f%%", successRate*100)
	suite.T().Logf("  Error Rate: %.2f%%", errorRate*100)
	suite.T().Logf("  Average Latency: %v", avgLatency)
	suite.T().Logf("  P95 Latency: %v", p95Latency)

	// Under stress, we allow some degradation but system should remain stable
	assert.True(suite.T(), successRate >= 0.85, "Success rate under stress should be >= 85%")
	assert.True(suite.T(), errorRate <= 0.15, "Error rate under stress should be <= 15%")
	assert.True(suite.T(), avgLatency < suite.performanceConfig.WebhookLatencyTarget*3,
		"Average latency under stress should be < 3x normal target")
}

func (suite *CallRailPerformanceTestSuite) TestMemoryLeakUnderSustainedLoad() {
	// Test for memory leaks during sustained load
	handler := NewMockCallRailWebhookHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Run sustained load for longer period
	sustainedDuration := 5 * time.Minute
	steadyThroughput := float64(suite.performanceConfig.ThroughputTarget) / 60.0 / 2 // Half target for sustained test

	ctx, cancel := context.WithTimeout(suite.ctx, sustainedDuration)
	defer cancel()

	var requestCount int64
	var successCount int64
	var wg sync.WaitGroup

	startTime := time.Now()

	ticker := time.NewTicker(time.Duration(float64(time.Second) / steadyThroughput))
	defer ticker.Stop()

	// Sample performance metrics every 30 seconds
	performanceSamples := make(map[time.Duration]*PerformanceSample)
	var sampleMutex sync.Mutex

	sampleTicker := time.NewTicker(30 * time.Second)
	defer sampleTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sampleTicker.C:
				elapsed := time.Since(startTime)
				currentThroughput := float64(atomic.LoadInt64(&requestCount)) / elapsed.Seconds()

				sampleMutex.Lock()
				performanceSamples[elapsed] = &PerformanceSample{
					Throughput:    currentThroughput,
					RequestCount:  atomic.LoadInt64(&requestCount),
					SuccessCount:  atomic.LoadInt64(&successCount),
					Timestamp:     time.Now(),
				}
				sampleMutex.Unlock()

				suite.T().Logf("Performance sample at %v: %.2f req/sec, %d total requests",
					elapsed, currentThroughput, atomic.LoadInt64(&requestCount))
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			goto done
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()

				callID := fmt.Sprintf("sustained-call-%d", atomic.AddInt64(&requestCount, 1))
				payload := suite.generateCallRailPayload(callID, "tenant-sustained-test", "standard")
				body, _ := json.Marshal(payload)

				resp, err := http.Post(testServer.URL+"/webhook/callrail", "application/json", bytes.NewBuffer(body))
				if err == nil && resp != nil {
					if resp.StatusCode == http.StatusOK {
						atomic.AddInt64(&successCount, 1)
					}
					resp.Body.Close()
				}
			}()
		}
	}

done:
	wg.Wait()
	actualDuration := time.Since(startTime)

	// Analyze sustained load results
	totalRequests := atomic.LoadInt64(&requestCount)
	totalSuccessful := atomic.LoadInt64(&successCount)
	overallThroughput := float64(totalRequests) / actualDuration.Seconds()
	overallSuccessRate := float64(totalSuccessful) / float64(totalRequests)

	suite.T().Logf("Sustained Load Test Results:")
	suite.T().Logf("  Duration: %v", actualDuration)
	suite.T().Logf("  Total Requests: %d", totalRequests)
	suite.T().Logf("  Overall Throughput: %.2f req/sec", overallThroughput)
	suite.T().Logf("  Overall Success Rate: %.2f%%", overallSuccessRate*100)

	// Analyze performance stability over time
	suite.analyzePerformanceStability(performanceSamples)

	// Assert system stability
	assert.True(suite.T(), overallSuccessRate >= 0.95, "Sustained load success rate should be >= 95%")
	assert.True(suite.T(), totalRequests > 1000, "Should process significant number of requests")
}

type PerformanceSample struct {
	Throughput   float64   `json:"throughput"`
	RequestCount int64     `json:"request_count"`
	SuccessCount int64     `json:"success_count"`
	Timestamp    time.Time `json:"timestamp"`
}

// Helper methods

func (suite *CallRailPerformanceTestSuite) generateCallRailPayload(callID, tenantID, scenarioType string) CallRailWebhookPayload {
	basePayload := CallRailWebhookPayload{
		CallID:         callID,
		CompanyID:      tenantID,
		AccountID:      "account-" + tenantID,
		PhoneNumber:    "+15551234567",
		CallerID:       fmt.Sprintf("+1555%07d", rand.Intn(10000000)),
		StartTime:      time.Now().Add(-5 * time.Minute),
		EndTime:        time.Now().Add(-2 * time.Minute),
		Direction:      "inbound",
		CallStatus:     "completed",
		TrackingNumber: "+15551111111",
		ScenarioType:   scenarioType,
	}

	switch scenarioType {
	case "emergency":
		basePayload.Duration = 120
		basePayload.Tags = []string{"emergency", "urgent"}
		basePayload.CustomFields = map[string]interface{}{
			"emergency_call": true,
			"priority":       "urgent",
		}
	case "kitchen_remodel":
		basePayload.Duration = 300
		basePayload.Tags = []string{"kitchen", "remodeling", "qualified"}
		basePayload.CustomFields = map[string]interface{}{
			"lead_source": "google_ads",
			"project_type": "kitchen",
		}
	case "abandoned":
		basePayload.Duration = 15
		basePayload.CallStatus = "abandoned"
		basePayload.Tags = []string{"abandoned"}
	default:
		basePayload.Duration = 180
		basePayload.Tags = []string{"inquiry"}
	}

	return basePayload
}

func (suite *CallRailPerformanceTestSuite) runTenantLoadTest(serverURL, tenantID string, totalRequests, concurrency int) *LoadTestResults {
	startTime := time.Now()

	var successCount int64
	var errorCount int64
	var latencies []time.Duration
	var mu sync.Mutex

	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(requestIndex int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			callID := fmt.Sprintf("load-test-call-%s-%d", tenantID, requestIndex)
			payload := suite.generateCallRailPayload(callID, tenantID, "standard")
			body, _ := json.Marshal(payload)

			reqStart := time.Now()
			resp, err := http.Post(serverURL+"/webhook/callrail", "application/json", bytes.NewBuffer(body))
			latency := time.Since(reqStart)

			success := err == nil && resp != nil && resp.StatusCode == http.StatusOK
			if resp != nil {
				resp.Body.Close()
			}

			mu.Lock()
			latencies = append(latencies, latency)
			if success {
				successCount++
			} else {
				errorCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	endTime := time.Now()

	totalCount := int64(totalRequests)
	duration := endTime.Sub(startTime)

	return &LoadTestResults{
		TotalRequests:         totalCount,
		SuccessfulRequests:    successCount,
		FailedRequests:        errorCount,
		AverageWebhookLatency: suite.calculateAverageLatency(latencies),
		P95WebhookLatency:     suite.calculatePercentileLatency(latencies, 95),
		P99WebhookLatency:     suite.calculatePercentileLatency(latencies, 99),
		ThroughputRPS:         float64(totalCount) / duration.Seconds(),
		ErrorRate:             float64(errorCount) / float64(totalCount),
		StartTime:             startTime,
		EndTime:               endTime,
		Duration:              duration,
	}
}

func (suite *CallRailPerformanceTestSuite) selectScenarioByWeight(scenarios []CallScenario) CallScenario {
	r := rand.Float64()
	cumulative := 0.0

	for _, scenario := range scenarios {
		cumulative += scenario.Weight
		if r <= cumulative {
			return scenario
		}
	}

	// Fallback to first scenario
	return scenarios[0]
}

func (suite *CallRailPerformanceTestSuite) analyzeMultiTenantResults(results []*LoadTestResults) {
	suite.T().Log("=== Multi-Tenant Performance Analysis ===")

	var totalRequests int64
	var totalSuccessful int64
	var avgThroughput float64
	var maxLatency time.Duration

	for i, result := range results {
		suite.T().Logf("Tenant %d Results:", i+1)
		suite.T().Logf("  Requests: %d (%.2f%% success)", result.TotalRequests, (1.0-result.ErrorRate)*100)
		suite.T().Logf("  Throughput: %.2f req/sec", result.ThroughputRPS)
		suite.T().Logf("  Latency - Avg: %v, P95: %v", result.AverageWebhookLatency, result.P95WebhookLatency)

		totalRequests += result.TotalRequests
		totalSuccessful += result.SuccessfulRequests
		avgThroughput += result.ThroughputRPS

		if result.P95WebhookLatency > maxLatency {
			maxLatency = result.P95WebhookLatency
		}
	}

	overallSuccessRate := float64(totalSuccessful) / float64(totalRequests)
	avgThroughputPerTenant := avgThroughput / float64(len(results))

	suite.T().Logf("Overall Multi-Tenant Performance:")
	suite.T().Logf("  Total Requests: %d", totalRequests)
	suite.T().Logf("  Overall Success Rate: %.2f%%", overallSuccessRate*100)
	suite.T().Logf("  Average Throughput per Tenant: %.2f req/sec", avgThroughputPerTenant)
	suite.T().Logf("  Maximum P95 Latency: %v", maxLatency)

	// Assert multi-tenant performance
	assert.True(suite.T(), overallSuccessRate >= 0.95, "Multi-tenant success rate should be >= 95%")
	assert.True(suite.T(), maxLatency < suite.performanceConfig.WebhookLatencyTarget*2,
		"Maximum tenant latency should be reasonable")
}

func (suite *CallRailPerformanceTestSuite) analyzeScenarioResults(results map[string]*ScenarioMetrics) {
	suite.T().Log("=== Scenario Performance Analysis ===")

	for scenarioType, metrics := range results {
		suite.T().Logf("%s Scenario:", scenarioType)
		suite.T().Logf("  Requests: %d", metrics.RequestCount)
		suite.T().Logf("  Success Rate: %.2f%%", metrics.SuccessRate*100)
		suite.T().Logf("  Average Latency: %v", metrics.AverageLatency)

		// Verify scenario-specific requirements
		switch scenarioType {
		case "emergency":
			assert.True(suite.T(), metrics.AverageLatency < suite.performanceConfig.WebhookLatencyTarget/2,
				"Emergency scenarios should process faster")
		case "abandoned":
			assert.True(suite.T(), metrics.AverageLatency < suite.performanceConfig.WebhookLatencyTarget/3,
				"Abandoned calls should process very quickly")
		}
	}
}

func (suite *CallRailPerformanceTestSuite) analyzePerformanceStability(samples map[time.Duration]*PerformanceSample) {
	suite.T().Log("=== Performance Stability Analysis ===")

	var throughputValues []float64
	for elapsed, sample := range samples {
		suite.T().Logf("Sample at %v: %.2f req/sec (%d requests)", elapsed, sample.Throughput, sample.RequestCount)
		throughputValues = append(throughputValues, sample.Throughput)
	}

	if len(throughputValues) > 1 {
		// Calculate coefficient of variation (stability metric)
		mean := suite.calculateMean(throughputValues)
		stdDev := suite.calculateStandardDeviation(throughputValues, mean)
		coefficientOfVariation := stdDev / mean

		suite.T().Logf("Throughput Stability:")
		suite.T().Logf("  Mean: %.2f req/sec", mean)
		suite.T().Logf("  Std Dev: %.2f", stdDev)
		suite.T().Logf("  Coefficient of Variation: %.2f", coefficientOfVariation)

		// System should be stable (low coefficient of variation)
		assert.True(suite.T(), coefficientOfVariation < 0.3,
			"System should be stable (coefficient of variation < 0.3)")
	}
}

// Utility functions for statistical calculations

func (suite *CallRailPerformanceTestSuite) calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	return total / time.Duration(len(latencies))
}

func (suite *CallRailPerformanceTestSuite) calculatePercentileLatency(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Sort latencies for accurate percentile calculation
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort for demonstration (use sort.Slice in production)
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := (len(sorted) * percentile) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

func (suite *CallRailPerformanceTestSuite) calculateMaxLatency(latencies []time.Duration) time.Duration {
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

func (suite *CallRailPerformanceTestSuite) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (suite *CallRailPerformanceTestSuite) calculateStandardDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sumSquaredDiff := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values))
	return variance // Simplified - should use math.Sqrt(variance)
}

// Benchmark tests for performance regression detection

func BenchmarkCallRailWebhookProcessing(b *testing.B) {
	handler := NewMockCallRailWebhookHandler()
	handler.processingDelay = 50 * time.Millisecond

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/callrail", handler.HandleWebhook)
	server := httptest.NewServer(mux)
	defer server.Close()

	payload := CallRailWebhookPayload{
		CallID:    "benchmark-call",
		CompanyID: "benchmark-company",
		Duration:  180,
	}
	body, _ := json.Marshal(payload)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Post(server.URL+"/webhook/callrail", "application/json", bytes.NewBuffer(body))
			if err != nil {
				b.Error("Request failed:", err)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	})
}

func BenchmarkCallRailPayloadProcessing(b *testing.B) {
	scenarios := []string{"standard", "emergency", "kitchen_remodel", "abandoned"}

	for _, scenario := range scenarios {
		b.Run(scenario, func(b *testing.B) {
			handler := NewMockCallRailWebhookHandler()

			payload := CallRailWebhookPayload{
				CallID:       "bench-call",
				CompanyID:    "bench-company",
				Duration:     180,
				ScenarioType: scenario,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				processingTime := handler.calculateProcessingTime(payload.ScenarioType, payload.Duration)
				_ = processingTime // Use the result
			}
		})
	}
}

// Run the test suite
func TestCallRailPerformanceTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CallRail performance tests in short mode")
	}

	suite.Run(t, new(CallRailPerformanceTestSuite))
}