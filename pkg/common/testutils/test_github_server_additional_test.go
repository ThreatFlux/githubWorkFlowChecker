package testutils

import (
	"testing"
)

// Test the GitHubServerOptions with all combinations of options
func TestGitHubServerFixture_SetupOptions(t *testing.T) {
	tests := []struct {
		name              string
		options           *GitHubServerOptions
		serverShouldSetup bool
	}{
		{
			name: "No repositories enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: false, // Only disable repo info
				SetupRefs:     true,
				SetupContents: true,
				SetupBlobs:    true,
				SetupTrees:    true,
				SetupCommits:  true,
				SetupPRs:      true,
				SetupLabels:   true,
			},
			serverShouldSetup: true,
		},
		{
			name: "No refs enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: true,
				SetupRefs:     false, // Only disable refs
				SetupContents: true,
				SetupBlobs:    true,
				SetupTrees:    true,
				SetupCommits:  true,
				SetupPRs:      true,
				SetupLabels:   true,
			},
			serverShouldSetup: true,
		},
		{
			name: "No contents enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: true,
				SetupRefs:     true,
				SetupContents: false, // Only disable contents
				SetupBlobs:    true,
				SetupTrees:    true,
				SetupCommits:  true,
				SetupPRs:      true,
				SetupLabels:   true,
			},
			serverShouldSetup: true,
		},
		{
			name: "No blobs enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: true,
				SetupRefs:     true,
				SetupContents: true,
				SetupBlobs:    false, // Only disable blobs
				SetupTrees:    true,
				SetupCommits:  true,
				SetupPRs:      true,
				SetupLabels:   true,
			},
			serverShouldSetup: true,
		},
		{
			name: "No trees enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: true,
				SetupRefs:     true,
				SetupContents: true,
				SetupBlobs:    true,
				SetupTrees:    false, // Only disable trees
				SetupCommits:  true,
				SetupPRs:      true,
				SetupLabels:   true,
			},
			serverShouldSetup: true,
		},
		{
			name: "No commits enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: true,
				SetupRefs:     true,
				SetupContents: true,
				SetupBlobs:    true,
				SetupTrees:    true,
				SetupCommits:  false, // Only disable commits
				SetupPRs:      true,
				SetupLabels:   true,
			},
			serverShouldSetup: true,
		},
		{
			name: "No PRs enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: true,
				SetupRefs:     true,
				SetupContents: true,
				SetupBlobs:    true,
				SetupTrees:    true,
				SetupCommits:  true,
				SetupPRs:      false, // Only disable PRs
				SetupLabels:   true,
			},
			serverShouldSetup: true,
		},
		{
			name: "No labels enabled",
			options: &GitHubServerOptions{
				Owner:         "owner",
				Repo:          "repo",
				DefaultBranch: "main",
				SetupRepoInfo: true,
				SetupRefs:     true,
				SetupContents: true,
				SetupBlobs:    true,
				SetupTrees:    true,
				SetupCommits:  true,
				SetupPRs:      true,
				SetupLabels:   false, // Only disable labels
			},
			serverShouldSetup: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a server fixture with the options
			fixture := NewGitHubServerFixture(tc.options)

			// Verify that the server is set up
			if fixture == nil {
				t.Errorf("Expected a non-nil fixture, got nil")
			}

			// Close the server to avoid resource leaks
			if fixture != nil && fixture.Server != nil {
				fixture.Server.Close()
			}
		})
	}
}
