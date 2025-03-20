package common

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v58/github"
	"golang.org/x/oauth2"
)

// GitHubClientOptions provides configuration options for GitHub client creation
type GitHubClientOptions struct {
	// Token is the GitHub API token
	Token string
	// BaseURL is the base URL for the GitHub API (optional, for GitHub Enterprise)
	BaseURL string
	// Timeout is the default timeout for API requests
	Timeout time.Duration
	// RetryCount is the number of times to retry failed requests
	RetryCount int
	// RetryDelay is the delay between retries
	RetryDelay time.Duration
}

// DefaultGitHubClientOptions returns the default options for GitHub client creation
func DefaultGitHubClientOptions() GitHubClientOptions {
	return GitHubClientOptions{
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 1 * time.Second,
	}
}

// NewGitHubClient creates a new GitHub client with the given options
func NewGitHubClient(options GitHubClientOptions) *github.Client {
	var httpClient *http.Client

	if options.Token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: options.Token},
		)
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	client := github.NewClient(httpClient)

	if options.BaseURL != "" {
		var err error
		client, err = client.WithEnterpriseURLs(options.BaseURL, options.BaseURL)
		if err != nil {
			// Fall back to the default client if enterprise URL is invalid
			// Use %v instead of %w for error printing
			fmt.Printf("invalid enterprise URL: %v\n", err)
			client = github.NewClient(httpClient)
		}
	}

	return client
}

// NewGitHubClientWithToken creates a new GitHub client with just a token
func NewGitHubClientWithToken(token string) *github.Client {
	options := DefaultGitHubClientOptions()
	options.Token = token
	return NewGitHubClient(options)
}

// RateLimitHandler provides rate limit handling for GitHub API requests
type RateLimitHandler struct {
	client       *github.Client
	maxRetries   int
	retryDelay   time.Duration
	lastResponse *github.Response
}

// NewRateLimitHandler creates a new rate limit handler for the given client
func NewRateLimitHandler(client *github.Client, maxRetries int, retryDelay time.Duration) *RateLimitHandler {
	return &RateLimitHandler{
		client:     client,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// HandleRateLimit handles rate limiting for GitHub API requests
// It returns true if the request should be retried, and false otherwise
func (h *RateLimitHandler) HandleRateLimit(resp *github.Response, err error) bool {
	h.lastResponse = resp

	// If there's no error or it's not a rate limit error, don't retry
	if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		return false
	}

	// Check if we've exceeded the maximum number of retries
	if h.maxRetries <= 0 {
		return false
	}

	// Check if we need to wait for rate limit reset
	if resp.Rate.Remaining == 0 {
		// Calculate how long to wait
		resetTime := resp.Rate.Reset.Time
		waitTime := time.Until(resetTime)

		if waitTime > 0 && waitTime < h.retryDelay*10 {
			// Sleep until the rate limit resets
			time.Sleep(waitTime + 100*time.Millisecond)
			h.maxRetries--
			return true
		}
	}

	// Otherwise, just wait the retry delay
	time.Sleep(h.retryDelay)
	h.maxRetries--
	return true
}

// GetRateLimitInfo returns information about the current rate limit
func (h *RateLimitHandler) GetRateLimitInfo() string {
	if h.lastResponse == nil {
		return ErrNoRateLimitInfo
	}

	rate := h.lastResponse.Rate
	resetTime := rate.Reset.Time
	waitTime := time.Until(resetTime)

	return fmt.Sprintf(ErrRateLimitFormat,
		rate.Remaining, rate.Limit, waitTime.Round(time.Second))
}

// ExecuteWithRetry executes a GitHub API request with retry logic for rate limiting
func ExecuteWithRetry(ctx context.Context, client *github.Client, maxRetries int, retryDelay time.Duration,
	fn func() (*github.Response, error)) error {

	handler := NewRateLimitHandler(client, maxRetries, retryDelay)

	for {
		resp, err := fn()

		if !handler.HandleRateLimit(resp, err) {
			// Check if it's a rate limit error
			if resp != nil && resp.StatusCode == http.StatusForbidden && resp.Rate.Remaining == 0 {
				return fmt.Errorf(ErrRateLimitExceeded, err)
			}

			// Check if it's a network error
			if err != nil && (resp == nil || resp.StatusCode >= 500) {
				return fmt.Errorf(ErrNetworkFailure, err)
			}

			// Check if it's an authentication error
			if resp != nil && resp.StatusCode == http.StatusUnauthorized {
				return fmt.Errorf(ErrAuthentication, err)
			}

			return err
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue with retry
		}
	}
}

// GetLatestRelease gets the latest release for a repository with retry logic
func GetLatestRelease(ctx context.Context, client *github.Client, owner, repo string) (*github.RepositoryRelease, error) {
	var release *github.RepositoryRelease

	err := ExecuteWithRetry(ctx, client, 3, time.Second, func() (*github.Response, error) {
		var resp *github.Response
		var err error
		release, resp, err = client.Repositories.GetLatestRelease(ctx, owner, repo)
		return resp, err
	})

	return release, err
}

// GetRef gets a reference (branch, tag) with retry logic
func GetRef(ctx context.Context, client *github.Client, owner, repo, ref string) (*github.Reference, error) {
	var reference *github.Reference

	err := ExecuteWithRetry(ctx, client, 3, time.Second, func() (*github.Response, error) {
		var resp *github.Response
		var err error
		reference, resp, err = client.Git.GetRef(ctx, owner, repo, ref)
		return resp, err
	})

	return reference, err
}

// CreateRef creates a reference (branch, tag) with retry logic
func CreateRef(ctx context.Context, client *github.Client, owner, repo string, ref *github.Reference) error {
	err := ExecuteWithRetry(ctx, client, 3, time.Second, func() (*github.Response, error) {
		_, resp, err := client.Git.CreateRef(ctx, owner, repo, ref)
		return resp, err
	})

	return err
}

// CreatePullRequest creates a pull request with retry logic
func CreatePullRequest(ctx context.Context, client *github.Client, owner, repo string, pull *github.NewPullRequest) (*github.PullRequest, error) {
	var pr *github.PullRequest

	err := ExecuteWithRetry(ctx, client, 3, time.Second, func() (*github.Response, error) {
		var resp *github.Response
		var err error
		pr, resp, err = client.PullRequests.Create(ctx, owner, repo, pull)
		return resp, err
	})

	return pr, err
}

// AddLabelsToIssue adds labels to an issue or pull request with retry logic
func AddLabelsToIssue(ctx context.Context, client *github.Client, owner, repo string, number int, labels []string) error {
	err := ExecuteWithRetry(ctx, client, 3, time.Second, func() (*github.Response, error) {
		_, resp, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, number, labels)
		return resp, err
	})

	return err
}
