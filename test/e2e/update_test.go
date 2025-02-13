package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
)

func TestEndToEndUpdate(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Clone the test repository
	repoPath := env.cloneTestRepo()

	tests := []struct {
		name     string
		testFunc func(t *testing.T, repoPath string, env *testEnv)
	}{
		{
			name: "ScanAndDetectUpdates",
			testFunc: func(t *testing.T, repoPath string, env *testEnv) {
				// Create scanner
				scanner := updater.NewScanner()

				// Get workflows directory path
				workflowsDir := filepath.Join(repoPath, ".github", "workflows")
				if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
					t.Fatal("Workflows directory not found")
				}

				// Scan for workflow files
				files, err := scanner.ScanWorkflows(workflowsDir)
				if err != nil {
					t.Fatalf("Failed to scan workflow files: %v", err)
				}

				if len(files) == 0 {
					t.Fatal("No workflow files found")
				}

				// Create version checker with GitHub token
				checker := updater.NewDefaultVersionChecker(os.Getenv("GITHUB_TOKEN"))

				// Create update manager
				manager := updater.NewUpdateManager()

				// Process each workflow file
				var allUpdates []*updater.Update
				for _, file := range files {
					actions, err := scanner.ParseActionReferences(file)
					if err != nil {
						t.Errorf("Failed to parse workflow file %s: %v", file, err)
						continue
					}

					// Check each action for updates
					for _, action := range actions {
						hasUpdate, latestVersion, latestHash, err := checker.IsUpdateAvailable(env.ctx, action)
						if err != nil {
							t.Errorf("Failed to check updates for %s/%s: %v", action.Owner, action.Name, err)
							continue
						}

						if hasUpdate {
							update, err := manager.CreateUpdate(env.ctx, file, action, latestVersion, latestHash)
							if err != nil {
								t.Errorf("Failed to create update for %s/%s: %v", action.Owner, action.Name, err)
								continue
							}
							if update != nil {
								allUpdates = append(allUpdates, update)
							}
						}
					}
				}

				// Log and apply updates
				t.Logf("Found %d potential updates", len(allUpdates))
				for _, update := range allUpdates {
					t.Logf("Action: %s/%s, Current: %s (%s), Latest: %s (%s)",
						update.Action.Owner,
						update.Action.Name,
						update.OldVersion,
						update.OldHash,
						update.NewVersion,
						update.NewHash)
				}

				if len(allUpdates) > 0 {
					// Create PR creator
					creator := updater.NewPRCreator(os.Getenv("GITHUB_TOKEN"), "ThreatFlux", "test-repo")

					// Apply updates and create PR
					if err := manager.ApplyUpdates(env.ctx, allUpdates); err != nil {
						t.Fatalf("Failed to apply updates: %v", err)
					}

					if err := creator.CreatePR(env.ctx, allUpdates); err != nil {
						t.Fatalf("Failed to create PR: %v", err)
					}

					t.Logf("Successfully created PR with %d updates", len(allUpdates))

					// Verify the updates locally
					for _, update := range allUpdates {
						content, err := os.ReadFile(update.FilePath)
						if err != nil {
							t.Errorf("Failed to read updated file %s: %v", update.FilePath, err)
							continue
						}

						// Check for version history comments
						expectedComment := fmt.Sprintf("# Using older hash from %s", update.OldVersion)
						if !strings.Contains(string(content), expectedComment) {
							t.Errorf("Version history comment not found in %s. Expected: %s", update.FilePath, expectedComment)
						}

						// Check for hash reference with version comment
						expectedRef := fmt.Sprintf("@%s  # %s", update.NewHash, update.NewVersion)
						if !strings.Contains(string(content), expectedRef) {
							t.Errorf("Hash reference not found in %s. Expected: %s", update.FilePath, expectedRef)
						}

						t.Logf("Verified update for %s/%s: %s -> %s (%s)",
							update.Action.Owner, update.Action.Name,
							update.OldVersion, update.NewVersion, update.NewHash)
					}
				}
			},
		},
		{
			name: "CreateUpdatePR",
			testFunc: func(t *testing.T, repoPath string, env *testEnv) {
				// Create scanner and scan workflow files
				scanner := updater.NewScanner()
				workflowFiles, err := scanner.ScanWorkflows(filepath.Join(repoPath, ".github/workflows"))
				if err != nil {
					t.Fatalf("Failed to scan workflow files: %v", err)
				}

				if len(workflowFiles) == 0 {
					t.Fatal("No workflow files found")
				}

				// Create a test workflow file
				workflowContent := `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # Original version: 11bd71901bbe5b1630ceea73d27597364c9af683
      - uses: actions/setup-go@93397bea11091df50f3d7e59dc26a7711a8bcfbe  # Original version: v4`

				workflowDir := filepath.Join(repoPath, ".github", "workflows")
				if err := os.MkdirAll(workflowDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				workflowFile := filepath.Join(workflowDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
					t.Fatalf("Failed to create workflow file: %v", err)
				}

				// Get the relative path and use it for logging
				relativeFile := strings.TrimPrefix(workflowFile, repoPath)
				relativeFile = strings.TrimPrefix(relativeFile, "/")
				t.Logf("Processing workflow file: %s", relativeFile)

				// Parse the workflow file
				actions, err := scanner.ParseActionReferences(workflowFile)
				if err != nil {
					t.Fatalf("Failed to parse workflow file: %v", err)
				}

				// Process all actions in the workflow
				var updates []*updater.Update
				checker := updater.NewDefaultVersionChecker(os.Getenv("GITHUB_TOKEN"))
				manager := updater.NewUpdateManager()

				for _, action := range actions {
					// Get latest version and hash
					latestVersion, _, err := checker.GetLatestVersion(env.ctx, action)
					if err != nil {
						t.Fatalf("Failed to get latest version for %s/%s: %v", action.Owner, action.Name, err)
					}

					latestHash, err := checker.GetCommitHash(env.ctx, action, latestVersion)
					if err != nil {
						t.Fatalf("Failed to get commit hash for %s/%s: %v", action.Owner, action.Name, err)
					}

					// Create update
					update, err := manager.CreateUpdate(env.ctx, workflowFile, action, latestVersion, latestHash)
					if err != nil {
						t.Fatalf("Failed to create update for %s/%s: %v", action.Owner, action.Name, err)
					}
					if update != nil {
						updates = append(updates, update)
					}
				}

				// Create PR creator and create PR
				creator := updater.NewPRCreator(os.Getenv("GITHUB_TOKEN"), "ThreatFlux", "test-repo")

				// Apply updates and create PR
				if err := manager.ApplyUpdates(env.ctx, updates); err != nil {
					t.Fatalf("Failed to apply updates: %v", err)
				}

				if err := creator.CreatePR(env.ctx, updates); err != nil {
					t.Fatalf("Failed to create PR: %v", err)
				}

				t.Logf("Successfully created PR with %d updates", len(updates))

				// Read the updated workflow file
				content, err := os.ReadFile(workflowFile)
				if err != nil {
					t.Fatalf("Failed to read updated workflow file: %v", err)
				}

				// Read and verify the updated content
				updatedContent := string(content)

				// Verify each update
				for _, update := range updates {
					// Check for version history comments
					expectedOldVersionComment := fmt.Sprintf("# Using older hash from %s", update.OriginalVersion)
					if !strings.Contains(updatedContent, expectedOldVersionComment) {
						t.Errorf("Version history comment not found for %s/%s. Expected: %s",
							update.Action.Owner, update.Action.Name, expectedOldVersionComment)
					}

					// Check for hash reference with version comment
					expectedRef := fmt.Sprintf("%s/%s@%s  # %s",
						update.Action.Owner, update.Action.Name, update.NewHash, update.NewVersion)
					if !strings.Contains(updatedContent, expectedRef) {
						t.Errorf("Hash reference not found for %s/%s. Expected: %s",
							update.Action.Owner, update.Action.Name, expectedRef)
					}

					// Make sure no version references are used
					unexpectedRef := fmt.Sprintf("%s/%s@v", update.Action.Owner, update.Action.Name)
					if strings.Contains(updatedContent, unexpectedRef) {
						t.Errorf("Found version reference instead of hash for %s/%s",
							update.Action.Owner, update.Action.Name)
					}

					t.Logf("Successfully verified changes for %s/%s: %s -> %s (%s)",
						update.Action.Owner, update.Action.Name,
						update.OriginalVersion, update.NewVersion, update.NewHash)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t, repoPath, env)
		})
	}
}
