package e2e

import (
	"fmt"
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
			name: "PushToNonExistentRemoteShouldFail",
			testFunc: func(t *testing.T) {
				env := NewTestEnv(t)
				defer env.Cleanup()

				// Clone repository
				repoPath := env.CloneTestRepo()

				// Make a change to push
				workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
				workflowContent := `name: Updated Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
`
				err := os.WriteFile(workflowFile, []byte(workflowContent), 0600)
				if err != nil {
					t.Fatalf(common.ErrFailedToWriteFile, err)
				}

				// Stage and commit the change
				env.stageAndCommit(repoPath, "Update workflow")

				// Add a non-existent remote
				env.addRemote(repoPath, "invalid", "https://example.com/invalid.git")

				// Attempt to push to the non-existent remote
				if err := env.WithWorkingDir(repoPath, func() error {
					cmd := env.CreateCommand("git", "push", "invalid", "test-push-failure")
					return cmd.Run()
				}); err == nil {
					t.Error(common.ErrExpectedPushError)
				}
			},
		},
		{
			name: "MultipleBranchesWithDifferentContent",
			testFunc: func(t *testing.T) {
				env := NewTestEnv(t)
				defer env.Cleanup()

				// Clone repository
				repoPath := env.CloneTestRepo()

				// Create multiple branches with different content
				branches := []string{"feature-1", "feature-2", "bugfix-1"}
				workflowContents := make(map[string]string)

				for i, branch := range branches {
					// Create a new branch
					env.createBranch(repoPath, branch)

					// Create workflow file with unique content
					workflowContent := fmt.Sprintf(`name: %s Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v%d
`, branch, i+1)
					workflowContents[branch] = workflowContent

					// Write the workflow file
					workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
					err := os.WriteFile(workflowFile, []byte(workflowContent), 0600)
					if err != nil {
						t.Fatalf(common.ErrFailedToWriteFile, err)
					}

					// Stage and commit
					env.stageAndCommit(repoPath, fmt.Sprintf("Update workflow for %s", branch))

					// Switch back to main
					env.switchBranch(repoPath, "main")
				}

				// Switch between branches and verify content
				for _, branch := range branches {
					if err := env.WithWorkingDir(repoPath, func() error {
						cmd := env.CreateCommand("git", "checkout", branch)
						return cmd.Run()
					}); err != nil {
						t.Fatalf(common.ErrFailedToSwitchBranch, err)
					}

					// Read workflow file and verify content
					workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
					content, err := os.ReadFile(workflowFile)
					if err != nil {
						t.Fatalf(common.ErrFailedToReadWorkflowFile, err)
					}

					expectedContent := workflowContents[branch]
					if !strings.Contains(string(content), expectedContent) {
						t.Errorf(common.ErrWrongWorkflowContent, string(content), expectedContent)
					}
				}
			},
		},
		{
			name: "CheckoutBranchAndMerge",
			testFunc: func(t *testing.T) {
				env := NewTestEnv(t)
				defer env.Cleanup()

				// Clone repository
				repoPath := env.CloneTestRepo()

				// Create a feature branch
				env.createBranch(repoPath, "feature")

				// Make changes in feature branch
				workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
				featureContent := `name: Feature Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
`
				err := os.WriteFile(workflowFile, []byte(featureContent), 0600)
				if err != nil {
					t.Fatalf(common.ErrFailedToWriteFile, err)
				}

				env.stageAndCommit(repoPath, "Update workflow in feature branch")

				// Switch back to main
				if err := env.WithWorkingDir(repoPath, func() error {
					cmd := env.CreateCommand("git", "checkout", "main")
					return cmd.Run()
				}); err != nil {
					t.Fatalf(common.ErrFailedToSwitchBranch, err)
				}

				// Merge feature branch into main
				if err := env.WithWorkingDir(repoPath, func() error {
					cmd := env.CreateCommand("git", "merge", "feature")
					return cmd.Run()
				}); err != nil {
					t.Fatalf("Failed to merge feature branch: %v", err)
				}

				// Verify workflow file contains feature branch changes
				content, err := os.ReadFile(workflowFile)
				if err != nil {
					t.Fatalf(common.ErrFailedToReadWorkflowFile, err)
				}

				if !strings.Contains(string(content), "actions/setup-node@v3") {
					t.Error("Workflow file does not contain expected changes from feature branch")
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
