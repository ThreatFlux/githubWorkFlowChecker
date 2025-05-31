package common

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v72/github"
)

func TestNewRateLimitHandler(t *testing.T) {
	client := &github.Client{}
	maxRetries := 5
	baseDelay := 2 * time.Second

	handler := NewRateLimitHandler(client, maxRetries, baseDelay)

	if handler.client != client {
		t.Errorf("Expected client to be set correctly")
	}
	if handler.maxRetries != maxRetries {
		t.Errorf("Expected maxRetries = %d, got %d", maxRetries, handler.maxRetries)
	}
	if handler.baseDelay != baseDelay {
		t.Errorf("Expected baseDelay = %v, got %v", baseDelay, handler.baseDelay)
	}
	if handler.maxDelay != 60*time.Second {
		t.Errorf("Expected default maxDelay = %v, got %v", 60*time.Second, handler.maxDelay)
	}
	if handler.attempt != 0 {
		t.Errorf("Expected initial attempt = 0, got %d", handler.attempt)
	}
}

func TestNewRateLimitHandlerWithOptions(t *testing.T) {
	client := &github.Client{}
	maxRetries := 3
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	handler := NewRateLimitHandlerWithOptions(client, maxRetries, baseDelay, maxDelay)

	if handler.client != client {
		t.Errorf("Expected client to be set correctly")
	}
	if handler.maxRetries != maxRetries {
		t.Errorf("Expected maxRetries = %d, got %d", maxRetries, handler.maxRetries)
	}
	if handler.baseDelay != baseDelay {
		t.Errorf("Expected baseDelay = %v, got %v", baseDelay, handler.baseDelay)
	}
	if handler.maxDelay != maxDelay {
		t.Errorf("Expected maxDelay = %v, got %v", maxDelay, handler.maxDelay)
	}
}

func TestRateLimitHandler_HandleRateLimit_NonRateLimitError(t *testing.T) {
	handler := NewRateLimitHandler(&github.Client{}, 3, time.Millisecond)

	tests := []struct {
		name      string
		resp      *github.Response
		err       error
		wantRetry bool
	}{
		{
			name:      "no error, no response",
			resp:      nil,
			err:       nil,
			wantRetry: false,
		},
		{
			name: "no error with response",
			resp: &github.Response{
				Response: &http.Response{StatusCode: http.StatusOK},
			},
			err:       nil,
			wantRetry: false,
		},
		{
			name: "non-forbidden error",
			resp: &github.Response{
				Response: &http.Response{StatusCode: http.StatusBadRequest},
			},
			err:       &github.ErrorResponse{Message: "Bad request"},
			wantRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset handler state
			handler.maxRetries = 3
			handler.attempt = 0

			got := handler.HandleRateLimit(tt.resp, tt.err)
			if got != tt.wantRetry {
				t.Errorf("HandleRateLimit() = %v, want %v", got, tt.wantRetry)
			}
		})
	}
}

func TestRateLimitHandler_HandleRateLimit_MaxRetriesExceeded(t *testing.T) {
	handler := NewRateLimitHandler(&github.Client{}, 0, time.Millisecond)

	resp := &github.Response{
		Response: &http.Response{StatusCode: http.StatusForbidden},
		Rate: github.Rate{
			Remaining: 0,
			Limit:     5000,
			Reset:     github.Timestamp{Time: time.Now().Add(time.Hour)},
		},
	}

	got := handler.HandleRateLimit(resp, &github.ErrorResponse{Message: "Rate limit exceeded"})
	if got != false {
		t.Errorf("HandleRateLimit() = %v, want false when maxRetries exceeded", got)
	}
}

func TestRateLimitHandler_HandleRateLimit_RateLimitReset(t *testing.T) {
	handler := NewRateLimitHandler(&github.Client{}, 3, time.Millisecond)

	// Create a response with rate limit reset in near future
	resetTime := time.Now().Add(50 * time.Millisecond)
	resp := &github.Response{
		Response: &http.Response{StatusCode: http.StatusForbidden},
		Rate: github.Rate{
			Remaining: 0,
			Limit:     5000,
			Reset:     github.Timestamp{Time: resetTime},
		},
	}

	start := time.Now()
	got := handler.HandleRateLimit(resp, &github.ErrorResponse{Message: "Rate limit exceeded"})
	elapsed := time.Since(start)

	if got != true {
		t.Errorf("HandleRateLimit() = %v, want true for rate limit with reset", got)
	}

	// Should have waited approximately until reset time
	expectedWait := 50*time.Millisecond + 100*time.Millisecond // reset time + buffer
	if elapsed < 40*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Expected wait time around %v, got %v", expectedWait, elapsed)
	}

	// Check that attempt counter and maxRetries were updated
	if handler.attempt != 1 {
		t.Errorf("Expected attempt = 1, got %d", handler.attempt)
	}
	if handler.maxRetries != 2 {
		t.Errorf("Expected maxRetries = 2, got %d", handler.maxRetries)
	}
}

func TestRateLimitHandler_HandleRateLimit_ExponentialBackoff(t *testing.T) {
	handler := NewRateLimitHandler(&github.Client{}, 5, 10*time.Millisecond)

	// Create a rate limit response without reset (or with far future reset)
	resp := &github.Response{
		Response: &http.Response{StatusCode: http.StatusForbidden},
		Rate: github.Rate{
			Remaining: 100, // Not exhausted, but still rate limited
			Limit:     5000,
			Reset:     github.Timestamp{Time: time.Now().Add(time.Hour)},
		},
	}

	// Test first few attempts to verify exponential backoff
	attempts := []struct {
		expectedMinWait time.Duration
		expectedMaxWait time.Duration
	}{
		{5 * time.Millisecond, 30 * time.Millisecond},   // ~20ms * 0.75-1.25, with extra margin
		{10 * time.Millisecond, 60 * time.Millisecond},  // ~40ms * 0.75-1.25, with extra margin
		{40 * time.Millisecond, 120 * time.Millisecond}, // ~80ms * 0.75-1.25, with extra margin
	}

	for i, expected := range attempts {
		start := time.Now()
		got := handler.HandleRateLimit(resp, &github.ErrorResponse{Message: "Rate limit exceeded"})
		elapsed := time.Since(start)

		if got != true {
			t.Errorf("Attempt %d: HandleRateLimit() = %v, want true", i+1, got)
		}

		if elapsed < expected.expectedMinWait || elapsed > expected.expectedMaxWait {
			t.Errorf("Attempt %d: wait time %v not in expected range %v-%v",
				i+1, elapsed, expected.expectedMinWait, expected.expectedMaxWait)
		}

		if handler.attempt != i+1 {
			t.Errorf("Attempt %d: expected attempt counter = %d, got %d", i+1, i+1, handler.attempt)
		}
	}
}

func TestRateLimitHandler_HandleRateLimit_FarFutureReset(t *testing.T) {
	handler := NewRateLimitHandler(&github.Client{}, 3, 10*time.Millisecond)

	// Create a response with rate limit reset far in the future
	resetTime := time.Now().Add(time.Hour) // 1 hour in future
	resp := &github.Response{
		Response: &http.Response{StatusCode: http.StatusForbidden},
		Rate: github.Rate{
			Remaining: 0,
			Limit:     5000,
			Reset:     github.Timestamp{Time: resetTime},
		},
	}

	start := time.Now()
	got := handler.HandleRateLimit(resp, &github.ErrorResponse{Message: "Rate limit exceeded"})
	elapsed := time.Since(start)

	if got != true {
		t.Errorf("HandleRateLimit() = %v, want true", got)
	}

	// Should use exponential backoff instead of waiting for reset
	// First attempt should be around 20ms (10ms * 2^1) with jitter
	if elapsed < 7*time.Millisecond || elapsed > 25*time.Millisecond {
		t.Errorf("Expected exponential backoff timing, got %v", elapsed)
	}
}

func TestRateLimitHandler_GetRateLimitInfo(t *testing.T) {
	handler := NewRateLimitHandler(&github.Client{}, 3, time.Second)

	// Test with no response
	info := handler.GetRateLimitInfo()
	if info != ErrNoRateLimitInfo {
		t.Errorf("Expected ErrNoRateLimitInfo, got %s", info)
	}

	// Test with response
	resetTime := time.Now().Add(30 * time.Second)
	resp := &github.Response{
		Rate: github.Rate{
			Remaining: 100,
			Limit:     5000,
			Reset:     github.Timestamp{Time: resetTime},
		},
	}
	handler.lastResponse = resp

	info = handler.GetRateLimitInfo()
	// Should contain rate limit information
	if len(info) == 0 {
		t.Errorf("Expected rate limit info, got empty string")
	}
	// Should contain the numbers from our test data
	if !strings.Contains(info, "100") || !strings.Contains(info, "5000") {
		t.Errorf("Rate limit info should contain rate numbers: %s", info)
	}
}
