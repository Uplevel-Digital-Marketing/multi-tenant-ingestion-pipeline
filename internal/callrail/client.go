package callrail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/home-renovators/ingestion-pipeline/pkg/models"
)

const (
	BaseURL = "https://api.callrail.com/v3"
	TimeoutDuration = 30 * time.Second
)

// Client represents a CallRail API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new CallRail API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: TimeoutDuration,
		},
		baseURL: BaseURL,
	}
}

// GetCallDetails retrieves detailed call information from CallRail API
func (c *Client) GetCallDetails(ctx context.Context, accountID, callID, apiKey string) (*models.CallDetails, error) {
	url := fmt.Sprintf("%s/a/%s/calls/%s.json", c.baseURL, accountID, callID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token token=\"%s\"", apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var callDetails models.CallDetails
	if err := json.NewDecoder(resp.Body).Decode(&callDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &callDetails, nil
}

// GetCallRecording retrieves call recording information from CallRail API
func (c *Client) GetCallRecording(ctx context.Context, accountID, callID, apiKey string) (*models.RecordingDetails, error) {
	url := fmt.Sprintf("%s/a/%s/calls/%s/recording.json", c.baseURL, accountID, callID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token token=\"%s\"", apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var recordingDetails models.RecordingDetails
	if err := json.NewDecoder(resp.Body).Decode(&recordingDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &recordingDetails, nil
}

// DownloadRecording downloads the actual recording file from CallRail
func (c *Client) DownloadRecording(ctx context.Context, recordingURL, apiKey string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", recordingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token token=\"%s\"", apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return audioData, nil
}

// RateLimitAwareRequest implements rate limiting for CallRail API calls
type RateLimitAwareRequest struct {
	client      *Client
	rateLimiter chan struct{}
}

// NewRateLimitAwareClient creates a new rate-limited CallRail client
// CallRail allows 120 requests per minute
func NewRateLimitAwareClient() *RateLimitAwareRequest {
	rateLimiter := make(chan struct{}, 120) // 120 requests per minute

	// Fill the rate limiter
	for i := 0; i < 120; i++ {
		rateLimiter <- struct{}{}
	}

	// Refill the rate limiter every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Refill the rate limiter
				for len(rateLimiter) < 120 {
					select {
					case rateLimiter <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	return &RateLimitAwareRequest{
		client:      NewClient(),
		rateLimiter: rateLimiter,
	}
}

// GetCallDetailsWithRateLimit gets call details with rate limiting
func (r *RateLimitAwareRequest) GetCallDetailsWithRateLimit(ctx context.Context, accountID, callID, apiKey string) (*models.CallDetails, error) {
	// Wait for rate limit token
	select {
	case <-r.rateLimiter:
		// Got token, proceed with request
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return r.client.GetCallDetails(ctx, accountID, callID, apiKey)
}

// GetCallRecordingWithRateLimit gets call recording with rate limiting
func (r *RateLimitAwareRequest) GetCallRecordingWithRateLimit(ctx context.Context, accountID, callID, apiKey string) (*models.RecordingDetails, error) {
	// Wait for rate limit token
	select {
	case <-r.rateLimiter:
		// Got token, proceed with request
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return r.client.GetCallRecording(ctx, accountID, callID, apiKey)
}

// DownloadRecordingWithRateLimit downloads recording with rate limiting
func (r *RateLimitAwareRequest) DownloadRecordingWithRateLimit(ctx context.Context, recordingURL, apiKey string) ([]byte, error) {
	// Wait for rate limit token
	select {
	case <-r.rateLimiter:
		// Got token, proceed with request
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return r.client.DownloadRecording(ctx, recordingURL, apiKey)
}

// RetryableClient wraps the CallRail client with retry logic
type RetryableClient struct {
	client     *RateLimitAwareRequest
	maxRetries int
	backoff    time.Duration
}

// NewRetryableClient creates a new retryable CallRail client
func NewRetryableClient() *RetryableClient {
	return &RetryableClient{
		client:     NewRateLimitAwareClient(),
		maxRetries: 3,
		backoff:    time.Second,
	}
}

// GetCallDetailsWithRetry gets call details with retry logic
func (r *RetryableClient) GetCallDetailsWithRetry(ctx context.Context, accountID, callID, apiKey string) (*models.CallDetails, error) {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			waitTime := r.backoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		callDetails, err := r.client.GetCallDetailsWithRateLimit(ctx, accountID, callID, apiKey)
		if err == nil {
			return callDetails, nil
		}

		lastErr = err

		// Don't retry on authentication errors
		if isAuthError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", r.maxRetries+1, lastErr)
}

// GetCallRecordingWithRetry gets call recording with retry logic
func (r *RetryableClient) GetCallRecordingWithRetry(ctx context.Context, accountID, callID, apiKey string) (*models.RecordingDetails, error) {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			waitTime := r.backoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		recording, err := r.client.GetCallRecordingWithRateLimit(ctx, accountID, callID, apiKey)
		if err == nil {
			return recording, nil
		}

		lastErr = err

		// Don't retry on authentication errors
		if isAuthError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", r.maxRetries+1, lastErr)
}

// DownloadRecordingWithRetry downloads recording with retry logic
func (r *RetryableClient) DownloadRecordingWithRetry(ctx context.Context, recordingURL, apiKey string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			waitTime := r.backoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		audioData, err := r.client.DownloadRecordingWithRateLimit(ctx, recordingURL, apiKey)
		if err == nil {
			return audioData, nil
		}

		lastErr = err

		// Don't retry on authentication errors
		if isAuthError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", r.maxRetries+1, lastErr)
}

// isAuthError checks if the error is an authentication error (shouldn't be retried)
func isAuthError(err error) bool {
	// This is a simple check - in a real implementation,
	// you'd want to parse the HTTP status code from the error
	return false
}