package e2e

import (
	"os"
	"path/filepath"
	"testing"
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
					t.Fatalf("Failed to corrupt git config: %v", err)
				}

				// Attempt git operation with corrupted config
				cmd := env.createCommand("git", "status")
				cmd.Dir = repoPath
				if err := cmd.Run(); err == nil {
					t.Error("Expected error with corrupted git config, got nil")
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
					t.Fatalf("Failed to create branch: %v", err)
				}

				// Make some changes
				workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
				if err := os.WriteFile(workflowFile, []byte("invalid: content"), 0644); err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}

				// Try to push without committing
				cmd = env.createCommand("git", "push", "origin", branchName)
				cmd.Dir = repoPath
				if err := cmd.Run(); err == nil {
					t.Error("Expected error when pushing uncommitted changes, got nil")
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
					t.Error("Expected error when committing without staged changes, got nil")
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

				// Try to create branch with invalid name
				cmd := env.createCommand("git", "checkout", "-b", "invalid/branch/name")
				cmd.Dir = repoPath
				if err := cmd.Run(); err == nil {
					t.Error("Expected error when creating branch with invalid name, got nil")
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
					t.Fatalf("Failed to create branch: %v", err)
				}

				// Make conflicting changes
				workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
				if err := os.WriteFile(workflowFile, []byte("conflict: content"), 0644); err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}

				// Stage and commit changes
				cmd = env.createCommand("git", "add", ".")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to stage changes: %v", err)
				}

				cmd = env.createCommand("git", "commit", "-m", "test commit")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to commit changes: %v", err)
				}

				// Switch back to main and make conflicting changes
				cmd = env.createCommand("git", "checkout", "main")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to switch branch: %v", err)
				}

				if err := os.WriteFile(workflowFile, []byte("different: content"), 0644); err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}

				cmd = env.createCommand("git", "add", ".")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to stage changes: %v", err)
				}

				cmd = env.createCommand("git", "commit", "-m", "conflicting commit")
				cmd.Dir = repoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to commit changes: %v", err)
				}

				// Try to merge branches with conflicts
				cmd = env.createCommand("git", "merge", branchName)
				cmd.Dir = repoPath
				if err := cmd.Run(); err == nil {
					t.Error("Expected error when merging conflicting branches, got nil")
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
