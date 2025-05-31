package common

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
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
	// RetryDelay is the base delay for exponential backoff retries
	RetryDelay time.Duration
	// MaxRetryDelay is the maximum delay between retries
	MaxRetryDelay time.Duration
}

// DefaultGitHubClientOptions returns the default options for GitHub client creation
func DefaultGitHubClientOptions() GitHubClientOptions {
	return GitHubClientOptions{
		Timeout:       30 * time.Second,
		RetryCount:    3,
		RetryDelay:    1 * time.Second,
		MaxRetryDelay: 60 * time.Second,
	}
}

// CalculateBackoff returns the next backoff duration with jitter for exponential backoff
func CalculateBackoff(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}

	// Calculate exponential backoff: baseDelay * 2^attempt
	// Use bit shifting for efficient power of 2 calculation, but cap at 30 to prevent overflow
	shiftAmount := uint(attempt)
	if shiftAmount > 30 {
		shiftAmount = 30
	}
	backoff := baseDelay * time.Duration(1<<shiftAmount)

	// Cap at maximum delay
	if backoff > maxDelay {
		backoff = maxDelay
	}

	// Add jitter (±25% randomization) to prevent thundering herd
	jitterRange := float64(backoff) * 0.25
	jitter := (rand.Float64()*2 - 1) * jitterRange

	finalDelay := time.Duration(float64(backoff) + jitter)
	if finalDelay < 0 {
		finalDelay = baseDelay
	}

	return finalDelay
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
	baseDelay    time.Duration
	maxDelay     time.Duration
	lastResponse *github.Response
	attempt      int
}

// NewRateLimitHandler creates a new rate limit handler for the given client
func NewRateLimitHandler(client *github.Client, maxRetries int, baseDelay time.Duration) *RateLimitHandler {
	return &RateLimitHandler{
		client:     client,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   60 * time.Second, // Default max delay
	}
}

// NewRateLimitHandlerWithOptions creates a new rate limit handler with full configuration
func NewRateLimitHandlerWithOptions(client *github.Client, maxRetries int, baseDelay, maxDelay time.Duration) *RateLimitHandler {
	return &RateLimitHandler{
		client:     client,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
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

	var waitTime time.Duration

	// Check if we need to wait for rate limit reset
	if resp.Rate.Remaining == 0 {
		// Calculate how long to wait until rate limit resets
		resetTime := resp.Rate.Reset.Time
		resetWaitTime := time.Until(resetTime)

		// If the reset time is reasonable (less than 10x max delay), wait for it
		if resetWaitTime > 0 && resetWaitTime < h.maxDelay*10 {
			waitTime = resetWaitTime + 100*time.Millisecond
		} else {
			// Otherwise use exponential backoff
			waitTime = CalculateBackoff(h.attempt, h.baseDelay, h.maxDelay)
		}
	} else {
		// Use exponential backoff for other rate limit scenarios
		waitTime = CalculateBackoff(h.attempt, h.baseDelay, h.maxDelay)
	}

	fmt.Printf("Rate limited. Retrying in %v (attempt %d/%d)\n",
		waitTime, h.attempt+1, h.maxRetries+h.attempt+1)

	time.Sleep(waitTime)
	h.attempt++
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

// executeGitHubAPIWithResult is a generic function to execute GitHub API calls with a result and retry logic
func executeGitHubAPIWithResult[T any](
	ctx context.Context,
	client *github.Client,
	apiFn func() (T, *github.Response, error),
) (T, error) {
	var result T

	options := DefaultGitHubClientOptions()
	err := ExecuteWithRetry(ctx, client, options.RetryCount, options.RetryDelay, func() (*github.Response, error) {
		var resp *github.Response
		var err error
		result, resp, err = apiFn()
		return resp, err
	})

	return result, err
}

// executeGitHubAPIWithNoResult is a generic function to execute GitHub API calls without a result and with retry logic
func executeGitHubAPIWithNoResult(
	ctx context.Context,
	client *github.Client,
	apiFn func() (*github.Response, error),
) error {
	options := DefaultGitHubClientOptions()
	return ExecuteWithRetry(ctx, client, options.RetryCount, options.RetryDelay, apiFn)
}

// GetLatestRelease gets the latest release for a repository with retry logic
func GetLatestRelease(ctx context.Context, client *github.Client, owner, repo string) (*github.RepositoryRelease, error) {
	return executeGitHubAPIWithResult(ctx, client, func() (*github.RepositoryRelease, *github.Response, error) {
		return client.Repositories.GetLatestRelease(ctx, owner, repo)
	})
}

// GetRef gets a reference (branch, tag) with retry logic
func GetRef(ctx context.Context, client *github.Client, owner, repo, ref string) (*github.Reference, error) {
	return executeGitHubAPIWithResult(ctx, client, func() (*github.Reference, *github.Response, error) {
		return client.Git.GetRef(ctx, owner, repo, ref)
	})
}

// CreateRef creates a reference (branch, tag) with retry logic
func CreateRef(ctx context.Context, client *github.Client, owner, repo string, ref *github.Reference) error {
	return executeGitHubAPIWithNoResult(ctx, client, func() (*github.Response, error) {
		_, resp, err := client.Git.CreateRef(ctx, owner, repo, ref)
		return resp, err
	})
}

// CreatePullRequest creates a pull request with retry logic
func CreatePullRequest(ctx context.Context, client *github.Client, owner, repo string, pull *github.NewPullRequest) (*github.PullRequest, error) {
	return executeGitHubAPIWithResult(ctx, client, func() (*github.PullRequest, *github.Response, error) {
		return client.PullRequests.Create(ctx, owner, repo, pull)
	})
}

// AddLabelsToIssue adds labels to an issue or pull request with retry logic
func AddLabelsToIssue(ctx context.Context, client *github.Client, owner, repo string, number int, labels []string) error {
	return executeGitHubAPIWithNoResult(ctx, client, func() (*github.Response, error) {
		_, resp, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, number, labels)
		return resp, err
	})
}

// ValidateTokenScopes validates that the GitHub token has the required scopes
// It checks if the token is valid and has the necessary permissions to perform
// operations like reading repositories, modifying workflows, and creating pull requests.
func ValidateTokenScopes(ctx context.Context, client *github.Client) error {
	// Check if we can access the API by getting the authenticated user
	user, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf(ErrInvalidGitHubToken, err)
		}
		return fmt.Errorf(ErrFailedToValidateToken, err)
	}

	// For unauthenticated clients, we can't check scopes
	if user.Login == nil {
		// This is an unauthenticated client, which is allowed but has limitations
		return nil
	}

	// Check OAuth scopes from response headers
	// The X-OAuth-Scopes header contains the scopes granted to the token
	scopesHeader := resp.Header.Get("X-OAuth-Scopes")
	if scopesHeader == "" {
		// Some token types (like GitHub App installation tokens) don't have scopes
		// We'll allow these as they have their own permission model
		return nil
	}

	// Define required scopes
	// We need either 'repo' (full repo access) or 'public_repo' (public repo access only)
	// and 'workflow' scope for modifying workflow files
	requiredScopes := []string{"workflow"}
	hasRepoScope := false

	// Check if we have repo or public_repo scope
	if strings.Contains(scopesHeader, "repo") || strings.Contains(scopesHeader, "public_repo") {
		hasRepoScope = true
	}

	if !hasRepoScope {
		return fmt.Errorf(ErrTokenMissingScope, "repo or public_repo")
	}

	// Check for other required scopes
	for _, required := range requiredScopes {
		if !strings.Contains(scopesHeader, required) {
			return fmt.Errorf(ErrTokenMissingScope, required)
		}
	}

	return nil
}
