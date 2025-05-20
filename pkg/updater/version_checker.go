package updater

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"github.com/google/go-github/v58/github"
)

// DefaultVersionChecker implements the VersionChecker interface using GitHub API
type DefaultVersionChecker struct {
	client *github.Client
	// For testing
	mockGetLatestRelease func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error)
}

// NewDefaultVersionChecker creates a new DefaultVersionChecker instance
func NewDefaultVersionChecker(token string) *DefaultVersionChecker {
	client := common.NewGitHubClientWithToken(token)
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
			return "", "", fmt.Errorf(common.ErrGettingTags, err)
		}
		if len(tags) == 0 || tags[0].Name == nil {
			return "", "", fmt.Errorf(common.ErrNoVersionInfo, action.Owner, action.Name)
		}
		tagName = *tags[0].Name
	} else {
		return "", "", fmt.Errorf(common.ErrNoVersionInfo, action.Owner, action.Name)
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
	if len(action.Version) == 40 && common.IsHexString(action.Version) {
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
		return "", fmt.Errorf(common.ErrGettingRefForTag, version, err)
	}

	if ref.Object == nil || ref.Object.SHA == nil {
		return "", fmt.Errorf(common.ErrNoCommitHashForTag, version)
	}

	// If the tag points to an annotated tag object, we need to get the commit it points to
	if ref.Object.Type != nil && *ref.Object.Type == "tag" {
		tag, _, err := c.client.Git.GetTag(ctx, action.Owner, action.Name, *ref.Object.SHA)
		if err != nil {
			return "", fmt.Errorf(common.ErrGettingAnnotatedTag, version, err)
		}
		if tag.Object == nil || tag.Object.SHA == nil {
			return "", fmt.Errorf(common.ErrNoCommitHashInTag, version)
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

	// Determine the max length to iterate over
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	// Compare each part numerically when possible
	for i := 0; i < maxLen; i++ {
		p1 := ""
		p2 := ""
		if i < len(parts1) {
			p1 = parts1[i]
		}
		if i < len(parts2) {
			p2 = parts2[i]
		}

		n1 := numericPrefix(p1)
		n2 := numericPrefix(p2)

		if n1 > n2 {
			return true
		}
		if n1 < n2 {
			return false
		}

		s1 := p1[lenNumericPrefix(p1):]
		s2 := p2[lenNumericPrefix(p2):]
		if s1 > s2 {
			return true
		}
		if s1 < s2 {
			return false
		}
	}

	return false
}

// numericPrefix extracts the leading numeric portion of a version part.
func numericPrefix(part string) int {
	end := lenNumericPrefix(part)
	if end == 0 {
		return 0
	}
	n, _ := strconv.Atoi(part[:end])
	return n
}

// lenNumericPrefix returns the length of the leading numeric portion of a string.
func lenNumericPrefix(part string) int {
	for i, r := range part {
		if r < '0' || r > '9' {
			return i
		}
	}
	return len(part)
}
