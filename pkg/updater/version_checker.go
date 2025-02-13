package updater

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// DefaultVersionChecker implements the VersionChecker interface using GitHub API
type DefaultVersionChecker struct {
	client *github.Client
	// For testing
	mockGetLatestRelease func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
}

// NewDefaultVersionChecker creates a new DefaultVersionChecker instance
func NewDefaultVersionChecker(token string) *DefaultVersionChecker {
	var client *github.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		client = github.NewClient(oauth2.NewClient(context.Background(), ts))
	} else {
		client = github.NewClient(nil)
	}
	return &DefaultVersionChecker{client: client}
}

// GetLatestVersion returns the latest version for a given action
func (c *DefaultVersionChecker) GetLatestVersion(ctx context.Context, action ActionReference) (string, error) {
	// First try to get the latest release
	var release *github.RepositoryRelease
	var resp *github.Response
	var err error

	if c.mockGetLatestRelease != nil {
		release, resp, err = c.mockGetLatestRelease(ctx, action.Owner, action.Name)
	} else {
		release, resp, err = c.client.Repositories.GetLatestRelease(ctx, action.Owner, action.Name)
	}
	if err == nil && release != nil && release.TagName != nil {
		return *release.TagName, nil
	}

	// If no releases found or error occurred, try listing tags
	if resp != nil && resp.StatusCode == http.StatusNotFound || err != nil {
		opts := &github.ListOptions{
			PerPage: 1,
		}
		tags, _, err := c.client.Repositories.ListTags(ctx, action.Owner, action.Name, opts)
		if err != nil {
			return "", fmt.Errorf("error getting tags: %w", err)
		}
		if len(tags) > 0 && tags[0].Name != nil {
			return *tags[0].Name, nil
		}
	}

	return "", fmt.Errorf("no version information found for %s/%s", action.Owner, action.Name)
}

// IsUpdateAvailable checks if a newer version is available
func (c *DefaultVersionChecker) IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, error) {
	latestVersion, err := c.GetLatestVersion(ctx, action)
	if err != nil {
		return false, "", err
	}

	// If current version is a commit SHA, always suggest update to latest tag
	if len(action.Version) == 40 && isHexString(action.Version) {
		return true, latestVersion, nil
	}

	// Compare versions
	if IsNewer(latestVersion, action.Version) {
		return true, latestVersion, nil
	}

	return false, "", nil
}

// isHexString checks if a string is a valid hexadecimal string (for commit SHAs)
func isHexString(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

// IsNewer compares two version strings and returns true if v1 is newer than v2
func IsNewer(v1, v2 string) bool {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split versions into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] > parts2[i] {
			return true
		}
		if parts1[i] < parts2[i] {
			return false
		}
	}

	// If all parts are equal, longer version is newer
	return len(parts1) > len(parts2)
}
