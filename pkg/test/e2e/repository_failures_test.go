package e2e

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
	"github.com/google/go-github/v58/github"
	"golang.org/x/oauth2"
)

func TestRepositoryFailures(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "InvalidTokenFailure",
			testFunc: func(t *testing.T) {
				// Setup test environment with invalid token
				token := "invalid_token"
				ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
				defer cancel()

				ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
				tc := oauth2.NewClient(ctx, ts)
				client := github.NewClient(tc)

				// Attempt to create repository
				_, _, err := client.Repositories.Create(ctx, testRepoOwner, &github.Repository{
					Name:        github.String("test-repo-invalid"),
					Description: github.String("Test repository for failure scenarios"),
					AutoInit:    github.Bool(true),
					Private:     github.Bool(true),
				})

				if err == nil {
					t.Error("Expected error with invalid token, got nil")
				}
			},
		},
		{
			name: "RepositoryCreationFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Attempt to create repository with invalid name
				_, _, err := env.githubClient.Repositories.Create(env.ctx, testRepoOwner, &github.Repository{
					Name:        github.String("invalid/repo/name"),
					Description: github.String("Test repository with invalid name"),
					AutoInit:    github.Bool(true),
					Private:     github.Bool(true),
				})

				if err == nil {
					t.Error("Expected error with invalid repository name, got nil")
				}
			},
		},
		{
			name: "GitCloneFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Try to clone non-existent repository
				nonExistentRepo := "non-existent-repo-" + time.Now().Format("20060102150405")
				cloneURL := "https://github.com/" + testRepoOwner + "/" + nonExistentRepo + ".git"
				repoPath := filepath.Join(env.workDir, nonExistentRepo)

				if err := os.MkdirAll(repoPath, 0750); err != nil {
					t.Fatalf("Failed to create repo directory: %v", err)
				}

				err := env.cloneWithError(cloneURL, repoPath)
				if err == nil {
					t.Error("Expected error when cloning non-existent repository, got nil")
				}
			},
		},
		{
			name: "WorkflowDirectoryCreationFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Create repository
				repoPath := env.cloneTestRepo()

				// Make workflows directory read-only with no write permissions
				workflowDir := filepath.Join(repoPath, ".github", "workflows")
				if err := os.MkdirAll(workflowDir, 0750); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}
				if err := os.Chmod(workflowDir, 0555); err != nil {
					t.Fatalf("Failed to make directory read-only: %v", err)
				}

				// Attempt to create workflow file in read-only directory
				workflowFile := filepath.Join(workflowDir, "test-failure.yml")
				err := os.WriteFile(workflowFile, []byte("invalid: workflow: content"), 0600)
				if err == nil {
					t.Error("Expected error when writing to read-only directory, got nil")
				} else if !os.IsPermission(err) {
					t.Errorf("Expected permission denied error, got: %v", err)
				}

				// Restore permissions for cleanup
				if err := os.Chmod(workflowDir, 0750); err != nil {
					t.Fatalf("Failed to restore directory permissions: %v", err)
				}
			},
		},
		{
			name: "InvalidWorkflowContent",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Create repository
				repoPath := env.cloneTestRepo()

				// Create workflow directory
				workflowDir := filepath.Join(repoPath, ".github", "workflows")
				if err := os.MkdirAll(workflowDir, 0750); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create invalid workflow file
				workflowFile := filepath.Join(workflowDir, "invalid.yml")
				invalidContent := `
invalid:
  - yaml:
    content:
      - missing: colon
        broken syntax
`
				if err := os.WriteFile(workflowFile, []byte(invalidContent), 0600); err != nil {
					t.Fatalf("Failed to write invalid workflow file: %v", err)
				}

				// Attempt to parse invalid workflow
				scanner := env.createScanner()
				_, err := scanner.ParseActionReferences(workflowFile)
				if err == nil {
					t.Error("Expected error when parsing invalid workflow, got nil")
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

// Helper method to clone repository with error handling
func (e *testEnv) cloneWithError(cloneURL, repoPath string) error {
	cmd := e.createCommand("git", "clone", cloneURL, repoPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(output))
	}
	return nil
}

// Helper method to create scanner
func (e *testEnv) createScanner() *updater.Scanner {
	return updater.NewScanner(e.workDir)
}

// Helper method to create command with context
func (e *testEnv) createCommand(name string, args ...string) *exec.Cmd {
	return exec.CommandContext(e.ctx, name, args...)
}
