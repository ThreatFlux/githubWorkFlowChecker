package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
)

func TestConcurrentOperations(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "ConcurrentWorkflowScanning",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Create multiple workflow files
				workflowDir := filepath.Join(repoPath, ".github", "workflows")
				for i := 0; i < 5; i++ {
					workflowContent := fmt.Sprintf(`
name: Workflow-%d
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
`, i)
					workflowFile := filepath.Join(workflowDir, fmt.Sprintf("test-%d.yml", i))
					if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
						t.Fatalf("Failed to create workflow file: %v", err)
					}
				}

				// Scan workflows concurrently
				var wg sync.WaitGroup
				results := make(chan error, 5)
				scanner := updater.NewScanner()

				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						files, err := scanner.ScanWorkflows(workflowDir)
						if err != nil {
							results <- err
							return
						}
						if len(files) == 0 {
							results <- fmt.Errorf("no workflow files found")
							return
						}
						results <- nil
					}()
				}

				// Wait for all scans to complete
				wg.Wait()
				close(results)

				// Check results
				for err := range results {
					if err != nil {
						t.Errorf("Concurrent workflow scanning failed: %v", err)
					}
				}
			},
		},
		{
			name: "ConcurrentPRCreation",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Create multiple workflow files with different actions
				workflowDir := filepath.Join(repoPath, ".github", "workflows")
				actions := []struct {
					name    string
					version string
					hash    string
				}{
					{"actions/checkout", "v2", "1234567"},
					{"actions/setup-node", "v2", "2345678"},
					{"actions/setup-python", "v2", "3456789"},
					{"actions/setup-java", "v2", "4567890"},
					{"actions/cache", "v2", "5678901"},
				}

				for i, action := range actions {
					workflowContent := fmt.Sprintf(`
name: Workflow-%d
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: %s@%s  # Original version: %s
`, i, action.name, action.hash, action.version)
					workflowFile := filepath.Join(workflowDir, fmt.Sprintf("test-%d.yml", i))
					if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
						t.Fatalf("Failed to create workflow file: %v", err)
					}
				}

				// Create updates concurrently
				var wg sync.WaitGroup
				results := make(chan error, len(actions))
				manager := updater.NewUpdateManager()
				checker := updater.NewDefaultVersionChecker(os.Getenv("GITHUB_TOKEN"))

				for i, action := range actions {
					wg.Add(1)
					go func(i int, action struct {
						name    string
						version string
						hash    string
					}) {
						defer wg.Done()

						// Parse action reference
						parts := strings.Split(action.name, "/")
						actionRef := updater.ActionReference{
							Owner:   parts[0],
							Name:    parts[1],
							Version: action.version,
						}

						// Get latest version
						latestVersion, _, err := checker.GetLatestVersion(env.ctx, actionRef)
						if err != nil {
							results <- fmt.Errorf("failed to get latest version: %v", err)
							return
						}

						// Get latest hash
						latestHash, err := checker.GetCommitHash(env.ctx, actionRef, latestVersion)
						if err != nil {
							results <- fmt.Errorf("failed to get commit hash: %v", err)
							return
						}

						// Create update
						workflowFile := filepath.Join(workflowDir, fmt.Sprintf("test-%d.yml", i))
						update, err := manager.CreateUpdate(env.ctx, workflowFile, actionRef, latestVersion, latestHash)
						if err != nil {
							results <- fmt.Errorf("failed to create update: %v", err)
							return
						}

						if update != nil {
							// Apply update
							if err := manager.ApplyUpdates(env.ctx, []*updater.Update{update}); err != nil {
								results <- fmt.Errorf("failed to apply update: %v", err)
								return
							}
						}

						results <- nil
					}(i, action)
				}

				// Wait for all updates to complete
				wg.Wait()
				close(results)

				// Check results
				for err := range results {
					if err != nil {
						t.Errorf("Concurrent PR creation failed: %v", err)
					}
				}
			},
		},
		{
			name: "ConcurrentRepositoryOperations",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Perform multiple operations concurrently
				var wg sync.WaitGroup
				results := make(chan error, 3)

				// Operation 1: File modifications
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; i < 5; i++ {
						file := filepath.Join(repoPath, fmt.Sprintf("file-%d.txt", i))
						if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
							results <- fmt.Errorf("failed to write file: %v", err)
							return
						}
						time.Sleep(10 * time.Millisecond) // Simulate work
					}
					results <- nil
				}()

				// Operation 2: Git operations
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := 0; i < 3; i++ {
						branchName := fmt.Sprintf("feature-%d", i)
						cmd := env.createCommand("git", "checkout", "-b", branchName)
						cmd.Dir = repoPath
						if err := cmd.Run(); err != nil {
							results <- fmt.Errorf("failed git operation: %v", err)
							return
						}
						time.Sleep(10 * time.Millisecond) // Simulate work
					}
					results <- nil
				}()

				// Operation 3: Workflow scanning
				wg.Add(1)
				go func() {
					defer wg.Done()
					scanner := updater.NewScanner()
					for i := 0; i < 3; i++ {
						if _, err := scanner.ScanWorkflows(filepath.Join(repoPath, ".github", "workflows")); err != nil {
							results <- fmt.Errorf("failed to scan workflows: %v", err)
							return
						}
						time.Sleep(10 * time.Millisecond) // Simulate work
					}
					results <- nil
				}()

				// Wait for all operations to complete
				wg.Wait()
				close(results)

				// Check results
				for err := range results {
					if err != nil {
						t.Errorf("Concurrent repository operation failed: %v", err)
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
