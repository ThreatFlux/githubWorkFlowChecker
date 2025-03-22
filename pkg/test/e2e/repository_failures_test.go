package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-github/v58/github"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestRepositoryFailures(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "RepositoryCreationFailure",
			testFunc: func(t *testing.T) {
				// Create a GitHub client with invalid token for testing
				token := "invalid_token"
				testRepoOwner := "invalid_owner"
				testRepo := "invalid_repo"
				testTimeout := 5 * time.Second

				ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
				ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
				defer cancel()

				tc := oauth2.NewClient(ctx, ts)
				client := github.NewClient(tc)

				// Attempt to create repository - should fail with auth error
				_, _, err := client.Repositories.Create(ctx, testRepoOwner, &github.Repository{
					Name:    github.String(testRepo),
					Private: github.Bool(true),
				})

				assert.Error(t, err, "Expected error when creating repository with invalid token")
			},
		},
		{
			name: "InvalidRepositoryURL",
			testFunc: func(t *testing.T) {
				env := NewTestEnv(t)
				defer env.Cleanup()

				testRepoOwner := "non_existent_user"
				testRepo := "non_existent_repo"

				// Try to create a repository with invalid credentials
				// This should fail due to authentication
				_, _, err := env.githubClient.Repositories.Create(env.Context(), testRepoOwner, &github.Repository{
					Name:    github.String(testRepo),
					Private: github.Bool(true),
				})

				if err == nil {
					t.Error("Expected error when creating repository with insufficient permissions")
				}

				// Clone URL for non-existent repository
				cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", testRepoOwner, testRepo)
				repoPath := filepath.Join(env.WorkDir, "invalid-repo")

				// Attempt to clone - should fail
				err = env.CloneWithError(cloneURL, repoPath)
				if err == nil {
					t.Error("Expected error when cloning non-existent repository")
				}
			},
		},
		{
			name: "CloneRepositoryToInaccessibleDirectory",
			testFunc: func(t *testing.T) {
				// Skip on Windows since permissions work differently
				if os.Getenv("RUNNER_OS") == "Windows" {
					t.Skip("Skipping test on Windows")
				}

				env := NewTestEnv(t)
				defer env.Cleanup()

				// Create a read-only directory
				readOnlyDir := filepath.Join(env.WorkDir, "readonly")
				if err := os.MkdirAll(readOnlyDir, 0500); err != nil {
					t.Fatalf("Failed to create read-only directory: %v", err)
				}

				// Try to clone inside the read-only directory
				repoURL := "https://github.com/octocat/Hello-World.git"
				repoPath := filepath.Join(readOnlyDir, "repo")

				// This should fail due to permissions
				err := env.CloneWithError(repoURL, repoPath)
				if err == nil {
					t.Error("Expected error when cloning to read-only directory")
				}
			},
		},
		{
			name: "CloneInvalidURL",
			testFunc: func(t *testing.T) {
				env := NewTestEnv(t)
				defer env.Cleanup()

				// Invalid URLs
				invalidURLs := []string{
					"https://githubcom/invalid/repo.git",   // Malformed URL
					"git@github.com:invalid/user/repo.git", // Invalid SSH format
					"file:///non/existent/path",            // Invalid file protocol
				}

				for _, url := range invalidURLs {
					repoPath := filepath.Join(env.WorkDir, "invalid-repo")

					// Ensure the directory doesn't exist
					_ = os.RemoveAll(repoPath)

					// Attempt to clone - should fail
					err := env.CloneWithError(url, repoPath)
					if err == nil {
						t.Errorf("Expected error when cloning invalid URL: %s", url)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}
