package e2e

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
	"github.com/google/go-github/v72/github"
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
					Name:        github.Ptr("test-repo-invalid"),
					Description: github.Ptr("Test repository for failure scenarios"),
					AutoInit:    github.Ptr(true),
					Private:     github.Ptr(true),
				})

				if err == nil {
					t.Errorf(common.ErrExpectedError, "authentication error")
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
					Name:        github.Ptr("invalid/repo/name"),
					Description: github.Ptr("Test repository with invalid name"),
					AutoInit:    github.Ptr(true),
					Private:     github.Ptr(true),
				})

				if err == nil {
					t.Errorf(common.ErrExpectedError, "invalid repository name")
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
					t.Fatalf(common.ErrFailedToCreateRepoDir, err)
				}

				err := env.cloneWithError(cloneURL, repoPath)
				if err == nil {
					t.Errorf(common.ErrExpectedError, "repository not found")
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
					t.Fatalf(common.ErrFailedToCreateWorkflowsDir, err)
				}
				if err := os.Chmod(workflowDir, 0555); err != nil {
					t.Fatalf(common.ErrFailedToChangePermissions, err)
				}

				// Attempt to create workflow file in read-only directory
				workflowFile := filepath.Join(workflowDir, "test-failure.yml")
				err := os.WriteFile(workflowFile, []byte("invalid: workflow: content"), 0600)
				if err == nil {
					t.Errorf(common.ErrExpectedError, "permission denied")
				} else if !os.IsPermission(err) {
					t.Errorf(common.ErrUnexpectedError, err)
				}

				// Restore permissions for cleanup
				if err := os.Chmod(workflowDir, 0750); err != nil {
					t.Fatalf(common.ErrFailedToRestorePermissions, err)
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
					t.Fatalf(common.ErrFailedToCreateWorkflowsDir, err)
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
					t.Fatalf(common.ErrFailedToWriteWorkflowFile, err)
				}

				// Attempt to parse invalid workflow
				scanner := env.createScanner()
				_, err := scanner.ParseActionReferences(workflowFile)
				if err == nil {
					t.Errorf(common.ErrExpectedError, "invalid YAML")
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
