package updater

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v58/github"
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

// GetLatestVersion returns the latest version and its commit hash for a given action
func (c *DefaultVersionChecker) GetLatestVersion(ctx context.Context, action ActionReference) (string, string, error) {
	// First try to get the latest release
	var release *github.RepositoryRelease
	var resp *github.Response
	var err error

	if c.mockGetLatestRelease != nil {
		release, resp, err = c.mockGetLatestRelease(ctx, action.Owner, action.Name)
	} else {
		release, resp, err = c.client.Repositories.GetLatestRelease(ctx, action.Owner, action.Name)
	}

	// Get the latest tag and its commit hash
	var tagName string
	if err == nil && release != nil && release.TagName != nil {
		tagName = *release.TagName
	} else if resp != nil && resp.StatusCode == http.StatusNotFound || err != nil {
		// If no releases found or error occurred, try listing tags
		opts := &github.ListOptions{
			PerPage: 1,
		}
		tags, _, err := c.client.Repositories.ListTags(ctx, action.Owner, action.Name, opts)
		if err != nil {
			return "", "", fmt.Errorf("error getting tags: %w", err)
		}
		if len(tags) == 0 || tags[0].Name == nil {
			return "", "", fmt.Errorf("no version information found for %s/%s", action.Owner, action.Name)
		}
		tagName = *tags[0].Name
	} else {
		return "", "", fmt.Errorf("no version information found for %s/%s", action.Owner, action.Name)
	}

	// Get the commit hash for the tag
	commitHash, err := c.GetCommitHash(ctx, action, tagName)
	if err != nil {
		return "", "", err
	}

	return tagName, commitHash, nil
}

// IsUpdateAvailable checks if a newer version is available
func (c *DefaultVersionChecker) IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, string, error) {
	latestVersion, latestHash, err := c.GetLatestVersion(ctx, action)
	if err != nil {
		return false, "", "", err
	}

	// If current version is a commit SHA, compare directly
	if len(action.Version) == 40 && isHexString(action.Version) {
		return action.Version != latestHash, latestVersion, latestHash, nil
	}

	// If current version is a tag, check if it's older
	if action.CommitHash != "" {
		return action.CommitHash != latestHash, latestVersion, latestHash, nil
	}

	// If no commit hash is available, check version strings
	if IsNewer(latestVersion, action.Version) {
		return true, latestVersion, latestHash, nil
	}

	return false, latestVersion, latestHash, nil
}

// GetCommitHash returns the commit hash for a specific version of an action
func (c *DefaultVersionChecker) GetCommitHash(ctx context.Context, action ActionReference, version string) (string, error) {
	// Get the commit hash for the tag/version
	ref, _, err := c.client.Git.GetRef(ctx, action.Owner, action.Name, "tags/"+version)
	if err != nil {
		return "", fmt.Errorf("error getting ref for tag %s: %w", version, err)
	}

	if ref.Object == nil || ref.Object.SHA == nil {
		return "", fmt.Errorf("no commit hash found for tag %s", version)
	}

	// If the tag points to an annotated tag object, we need to get the commit it points to
	if ref.Object.Type != nil && *ref.Object.Type == "tag" {
		tag, _, err := c.client.Git.GetTag(ctx, action.Owner, action.Name, *ref.Object.SHA)
		if err != nil {
			return "", fmt.Errorf("error getting annotated tag %s: %w", version, err)
		}
		if tag.Object == nil || tag.Object.SHA == nil {
			return "", fmt.Errorf("no commit hash found in annotated tag %s", version)
		}
		return *tag.Object.SHA, nil
	}

	return *ref.Object.SHA, nil
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
