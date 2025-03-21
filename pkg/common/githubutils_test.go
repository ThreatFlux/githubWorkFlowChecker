package common

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v58/github"
)

func TestNewGitHubClient(t *testing.T) {
	// Test with empty options
	client := NewGitHubClient(GitHubClientOptions{})
	if client == nil {
		t.Errorf("NewGitHubClient returned nil with empty options")
	}

	// Test with token
	tokenClient := NewGitHubClient(GitHubClientOptions{
		Token: "test-token",
	})
	if tokenClient == nil {
		t.Errorf("NewGitHubClient returned nil with token")
	}

	// Test with base URL
	baseURLClient := NewGitHubClient(GitHubClientOptions{
		BaseURL: "https://github.example.com/api/v3/",
	})
	if baseURLClient == nil {
		t.Errorf("NewGitHubClient returned nil with base URL")
	}

	// Test with invalid base URL (should not panic)
	invalidURLClient := NewGitHubClient(GitHubClientOptions{
		BaseURL: "://invalid-url",
	})
	if invalidURLClient == nil {
		t.Errorf("NewGitHubClient returned nil with invalid base URL")
	}

	// Test with all options
	fullClient := NewGitHubClient(GitHubClientOptions{
		Token:      "test-token",
		BaseURL:    "https://github.example.com/api/v3/",
		Timeout:    10 * time.Second,
		RetryCount: 5,
		RetryDelay: 2 * time.Second,
	})
	if fullClient == nil {
		t.Errorf("NewGitHubClient returned nil with all options")
	}
}

func TestNewGitHubClientWithToken(t *testing.T) {
	client := NewGitHubClientWithToken("test-token")
	if client == nil {
		t.Errorf("NewGitHubClientWithToken returned nil")
	}
}

func TestDefaultGitHubClientOptions(t *testing.T) {
	options := DefaultGitHubClientOptions()

	// Check default values
	if options.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", options.Timeout)
	}
	if options.RetryCount != 3 {
		t.Errorf("Expected default retry count to be 3, got %d", options.RetryCount)
	}
	if options.RetryDelay != 1*time.Second {
		t.Errorf("Expected default retry delay to be 1s, got %v", options.RetryDelay)
	}
	if options.Token != "" {
		t.Errorf("Expected default token to be empty, got %s", options.Token)
	}
	if options.BaseURL != "" {
		t.Errorf("Expected default base URL to be empty, got %s", options.BaseURL)
	}
}

func TestRateLimitHandler(t *testing.T) {
	client := github.NewClient(nil)
	handler := NewRateLimitHandler(client, 3, 100*time.Millisecond)

	// Test with nil response and error
	if handler.HandleRateLimit(nil, nil) {
		t.Errorf("Expected HandleRateLimit to return false with nil response and error")
	}

	// Test with non-rate-limit error
	resp := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	if handler.HandleRateLimit(resp, &github.ErrorResponse{}) {
		t.Errorf("Expected HandleRateLimit to return false with non-rate-limit error")
	}

	// Test with rate limit error but max retries reached
	zeroRetryHandler := NewRateLimitHandler(client, 0, 100*time.Millisecond)
	rateLimitResp := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusForbidden,
		},
		Rate: github.Rate{
			Remaining: 0,
			Limit:     5000,
			Reset:     github.Timestamp{Time: time.Now().Add(1 * time.Hour)},
		},
	}
	if zeroRetryHandler.HandleRateLimit(rateLimitResp, &github.RateLimitError{}) {
		t.Errorf("Expected HandleRateLimit to return false with max retries reached")
	}

	// Test GetRateLimitInfo with no previous response
	handler.lastResponse = nil
	info := handler.GetRateLimitInfo()
	if info != "No rate limit information available" {
		t.Errorf("Expected GetRateLimitInfo to return 'No rate limit information available', got %s", info)
	}

	// Test GetRateLimitInfo with response
	// Create a response with a valid rate limit
	validRateResp := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
		Rate: github.Rate{
			Remaining: 4000,
			Limit:     5000,
			Reset:     github.Timestamp{Time: time.Now().Add(1 * time.Hour)},
		},
	}
	handler.lastResponse = validRateResp
	info = handler.GetRateLimitInfo()
	if !strings.Contains(info, "Rate limit: 4000/5000") {
		t.Errorf("Expected GetRateLimitInfo to contain 'Rate limit: 4000/5000', got %s", info)
	}
}

func TestExecuteWithRetry(t *testing.T) {
	client := github.NewClient(nil)
	ctx := context.Background()

	// Test with successful execution
	callCount := 0
	err := ExecuteWithRetry(ctx, client, 3, 100*time.Millisecond, func() (*github.Response, error) {
		callCount++
		return &github.Response{
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}

	// Test with rate limit error and retry
	callCount = 0
	err = ExecuteWithRetry(ctx, client, 2, 100*time.Millisecond, func() (*github.Response, error) {
		callCount++
		if callCount == 1 {
			// First call hits rate limit
			return &github.Response{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
				Rate: github.Rate{
					Remaining: 0,
					Limit:     5000,
					Reset:     github.Timestamp{Time: time.Now().Add(50 * time.Millisecond)},
				},
			}, &github.RateLimitError{}
		}
		// Second call succeeds
		return &github.Response{
			Response: &http.Response{
				StatusCode: http.StatusOK,
			},
		}, nil
	})

	if err != nil {
		t.Errorf("Expected no error after retry, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("Expected function to be called twice, got %d", callCount)
	}

	// Test with context cancellation
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately
	err = ExecuteWithRetry(cancelCtx, client, 3, 100*time.Millisecond, func() (*github.Response, error) {
		return nil, context.Canceled
	})

	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	// Test with exhausted rate limit (no retries left)
	err = ExecuteWithRetry(ctx, client, 0, 100*time.Millisecond, func() (*github.Response, error) {
		return &github.Response{
				Response: &http.Response{
					StatusCode: http.StatusForbidden,
				},
				Rate: github.Rate{
					Remaining: 0,
					Limit:     5000,
					Reset:     github.Timestamp{Time: time.Now().Add(1 * time.Hour)},
				},
			}, &github.RateLimitError{
				Message: "API rate limit exceeded",
			}
	})

	if err == nil || !strings.Contains(err.Error(), "GitHub API rate limit exceeded") {
		t.Errorf("Expected rate limit error, got %v", err)
	}

	// Test with network failure
	err = ExecuteWithRetry(ctx, client, 0, 100*time.Millisecond, func() (*github.Response, error) {
		return &github.Response{
			Response: &http.Response{
				StatusCode: http.StatusInternalServerError,
			},
		}, errors.New("network error")
	})

	if err == nil || !strings.Contains(err.Error(), "network failure") {
		t.Errorf("Expected network failure error, got %v", err)
	}

	// Test with authentication error
	err = ExecuteWithRetry(ctx, client, 0, 100*time.Millisecond, func() (*github.Response, error) {
		return &github.Response{
			Response: &http.Response{
				StatusCode: http.StatusUnauthorized,
			},
		}, errors.New("authentication error")
	})

	if err == nil || !strings.Contains(err.Error(), "authentication error") {
		t.Errorf("Expected authentication error, got %v", err)
	}

	// Test with server error and no rate limit data
	err = ExecuteWithRetry(ctx, client, 0, 100*time.Millisecond, func() (*github.Response, error) {
		return nil, errors.New("server error")
	})

	if err == nil || !strings.Contains(err.Error(), "server error") {
		t.Errorf("Expected server error, got %v", err)
	}
}

func TestGitHubHelperFunctions(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a simple 200 OK response
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1}`)) // Ignoring error in test
	}))
	defer server.Close()

	// Create a client that uses the test server
	client := github.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")

	ctx := context.Background()

	// Test GetLatestRelease
	_, err := GetLatestRelease(ctx, client, "owner", "repo")
	// We expect an error here because the test server doesn't return a valid release,
	// but we're just testing that the function calls the API correctly
	// and handles the response without panicking
	t.Logf("GetLatestRelease error (expected): %v", err)

	// Test GetRef
	_, err = GetRef(ctx, client, "owner", "repo", "refs/heads/main")
	// Same as above
	t.Logf("GetRef error (expected): %v", err)

	// Test CreateRef
	ref := &github.Reference{
		Ref: github.String("refs/heads/test-branch"),
		Object: &github.GitObject{
			SHA: github.String("abcdef1234567890"),
		},
	}
	err = CreateRef(ctx, client, "owner", "repo", ref)
	// Same as above
	t.Logf("CreateRef error (expected): %v", err)

	// Test CreatePullRequest
	_, err = CreatePullRequest(ctx, client, "owner", "repo", &github.NewPullRequest{})
	// Same as above
	t.Logf("CreatePullRequest error (expected): %v", err)

	// Test AddLabelsToIssue
	err = AddLabelsToIssue(ctx, client, "owner", "repo", 1, []string{"label"})
	// Same as above
	t.Logf("AddLabelsToIssue error (expected): %v", err)
}
