package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

func TestGitOperations(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "GitConfigFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Corrupt git config
				gitConfigPath := filepath.Join(repoPath, ".git", "config")
				if err := os.WriteFile(gitConfigPath, []byte("invalid git config"), 0644); err != nil {
					t.Fatalf(common.ErrFailedToCorruptGitConfig, err)
				}

				// Attempt git operation with corrupted config
				cmd := env.createCommand("git", "status")
				cmd.Dir = repoPath
				if err := cmd.Run(); err == nil {
					t.Error(common.ErrExpectedGitConfigError)
				}
			},
		},
		{
			name: "GitPushFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Create a new branch
				branchName := "test-push-failure"
				cmd := env.createCommand("git", "checkout", "-b", branchName)
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToCreateBranch, err)
				}

				// Make some changes
				workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
				if err := os.WriteFile(workflowFile, []byte("invalid: content"), 0644); err != nil {
					t.Fatalf(common.ErrFailedToWriteFile, err)
				}

				// Stage and commit the changes
				cmd = env.createCommand("git", "add", workflowFile)
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToStageChanges, err)
				}

				cmd = env.createCommand("git", "commit", "-m", "test commit")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToCommitChanges, err)
				}

				// Add a non-existent remote
				cmd = env.createCommand("git", "remote", "add", "invalid", "https://invalid-remote/repo.git")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToAddRemote, err)
				}

				// Try to push to non-existent remote
				cmd = env.createCommand("git", "push", "invalid", branchName)
				cmd.Dir = repoPath
				if output, err := cmd.CombinedOutput(); err == nil {
					t.Error(common.ErrExpectedPushError)
				} else {
					// Verify error message
					errStr := string(output)
					if !strings.Contains(errStr, "Could not resolve host") && !strings.Contains(errStr, "failed to push") {
						t.Errorf(common.ErrUnexpectedErrorMessage, errStr)
					}
				}
			},
		},
		{
			name: "GitCommitFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Try to commit without staging changes
				cmd := env.createCommand("git", "commit", "-m", "test commit")
				cmd.Dir = repoPath
				if err := cmd.Run(); err == nil {
					t.Error(common.ErrExpectedCommitError)
				}
			},
		},
		{
			name: "GitBranchFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Try to create branch with invalid name containing spaces and special characters
				cmd := env.createCommand("git", "checkout", "-b", "invalid branch/name*&^%")
				cmd.Dir = repoPath
				if output, err := cmd.CombinedOutput(); err == nil {
					t.Error(common.ErrExpectedBranchError)
				} else {
					// Verify error message
					errStr := string(output)
					if !strings.Contains(errStr, "fatal: '") {
						t.Errorf(common.ErrUnexpectedErrorMessage, errStr)
					}
				}
			},
		},
		{
			name: "GitMergeFailure",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Create and switch to a new branch
				branchName := "test-merge-failure"
				cmd := env.createCommand("git", "checkout", "-b", branchName)
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToCreateBranch, err)
				}

				// Make conflicting changes
				workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
				if err := os.WriteFile(workflowFile, []byte("conflict: content"), 0644); err != nil {
					t.Fatalf(common.ErrFailedToWriteFile, err)
				}

				// Stage and commit changes
				cmd = env.createCommand("git", "add", ".")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToStageChanges, err)
				}

				cmd = env.createCommand("git", "commit", "-m", "test commit")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToCommitChanges, err)
				}

				// Switch back to main and make conflicting changes
				cmd = env.createCommand("git", "checkout", "main")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToSwitchBranch, err)
				}

				if err := os.WriteFile(workflowFile, []byte("different: content"), 0644); err != nil {
					t.Fatalf(common.ErrFailedToWriteFile, err)
				}

				cmd = env.createCommand("git", "add", ".")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToStageChanges, err)
				}

				cmd = env.createCommand("git", "commit", "-m", "conflicting commit")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf(common.ErrFailedToCommitChanges, err)
				}

				// Try to merge branches with conflicts
				cmd = env.createCommand("git", "merge", branchName)
				cmd.Dir = repoPath
				if err := cmd.Run(); err == nil {
					t.Error(common.ErrExpectedMergeError)
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
