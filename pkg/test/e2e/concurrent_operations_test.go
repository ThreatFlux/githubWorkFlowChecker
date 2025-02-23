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
					if err := os.WriteFile(workflowFile, []byte(workflowContent), 0600); err != nil {
						t.Fatalf("Failed to create workflow file: %v", err)
					}
				}

				// Scan workflows concurrently
				var wg sync.WaitGroup
				results := make(chan error, 5)
				scanner := updater.NewScanner(repoPath)

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
				if err := os.MkdirAll(workflowDir, 0750); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

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

				// Create workflow files in separate branches
				for i, action := range actions {
					// Create a new branch from main for this workflow
					branchName := fmt.Sprintf("workflow-%d", i)
					setupCmds := []struct {
						args []string
						msg  string
					}{
						{[]string{"checkout", "main"}, "switch to main"},
						{[]string{"checkout", "-b", branchName}, "create branch"},
						{[]string{"rm", "-rf", ".github/workflows"}, "clean workflows dir"},
					}

					for _, c := range setupCmds {
						cmd := env.createCommand("git", c.args...)
						cmd.Dir = repoPath
						if output, err := cmd.CombinedOutput(); err != nil {
							t.Fatalf("Failed to %s: %v\nOutput: %s", c.msg, err, string(output))
						}
					}

					// Create workflow content
					workflowContent := fmt.Sprintf(`name: Workflow-%d
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: %s@%s  # Original version: %s
      - name: Test
        run: echo "test"`, i, action.name, action.hash, action.version)

					// Ensure workflow directory exists
					if err := os.MkdirAll(workflowDir, 0750); err != nil {
						t.Fatalf("Failed to create workflow directory: %v", err)
					}

					// Write workflow file with Unix line endings and sync to disk
					workflowFile := filepath.Join(workflowDir, fmt.Sprintf("test-%d.yml", i))
					content := []byte(strings.ReplaceAll(workflowContent, "\r\n", "\n"))
					if err := os.WriteFile(workflowFile, content, 0600); err != nil {
						t.Fatalf("Failed to write workflow file: %v", err)
					}

					// Verify file was written correctly
					if _, err := os.Stat(workflowFile); err != nil {
						t.Fatalf("Failed to verify workflow file: %v", err)
					}

					// Configure git for this commit
					gitConfigs := []struct {
						name  string
						value string
					}{
						{"user.name", "GitHub Actions Bot"},
						{"user.email", "actions-bot@github.com"},
					}

					for _, config := range gitConfigs {
						cmd := env.createCommand("git", "config", config.name, config.value)
						cmd.Dir = repoPath
						if err := cmd.Run(); err != nil {
							t.Fatalf("Failed to configure git %s: %v", config.name, err)
						}
					}

					// Stage and commit each workflow file with debug output
					cmds := []struct {
						args []string
						msg  string
					}{
						{[]string{"add", "."}, "stage workflow file"},
						{[]string{"commit", "-m", fmt.Sprintf("Add workflow %d", i)}, "commit workflow file"},
					}

					for _, c := range cmds {
						cmd := env.createCommand("git", c.args...)
						cmd.Dir = repoPath
						output, err := cmd.CombinedOutput()
						t.Logf("Command output (%s):\n%s", c.msg, string(output))
						if err != nil {
							t.Fatalf("Failed to %s: %v\nOutput: %s", c.msg, err, string(output))
						}
					}

					// Push branch and switch back to main
					finalCmds := []struct {
						args []string
						msg  string
					}{
						{[]string{"push", "-f", "-u", "origin", branchName}, "push branch"},
						{[]string{"checkout", "main"}, "switch back to main"},
					}

					for _, c := range finalCmds {
						cmd := env.createCommand("git", c.args...)
						cmd.Dir = repoPath
						if output, err := cmd.CombinedOutput(); err != nil {
							t.Fatalf("Failed to %s: %v\nOutput: %s", c.msg, err, string(output))
						}
					}
				}

				// Create updates concurrently
				var wg sync.WaitGroup
				results := make(chan error, len(actions))
				manager := updater.NewUpdateManager(repoPath)
				checker := updater.NewDefaultVersionChecker(os.Getenv("GITHUB_TOKEN"))
				var gitMutex sync.Mutex // Mutex for git operations

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

						// Lock git operations
						gitMutex.Lock()
						defer gitMutex.Unlock()

						// Switch to the correct branch
						branchName := fmt.Sprintf("workflow-%d", i)
						cmd := env.createCommand("git", "checkout", branchName)
						cmd.Dir = repoPath
						if output, err := cmd.CombinedOutput(); err != nil {
							results <- fmt.Errorf("failed to switch to branch %s: %v\nOutput: %s", branchName, err, string(output))
							return
						}

						// Parse workflow file to get line number
						workflowFile := filepath.Join(workflowDir, fmt.Sprintf("test-%d.yml", i))
						scanner := updater.NewScanner(repoPath)
						refs, err := scanner.ParseActionReferences(workflowFile)
						if err != nil {
							results <- fmt.Errorf("failed to parse workflow file: %v", err)
							return
						}

						// Find the action reference with matching owner/name
						var foundRef *updater.ActionReference
						for _, ref := range refs {
							if ref.Owner == actionRef.Owner && ref.Name == actionRef.Name {
								foundRef = &ref
								break
							}
						}

						if foundRef == nil {
							results <- fmt.Errorf("failed to find action reference in workflow file")
							return
						}

						// Copy line number to our action reference
						actionRef.Line = foundRef.Line

						// Create update
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

							// Stage and commit the update
							cmds := []struct {
								args []string
								msg  string
							}{
								{[]string{"add", "."}, "stage updated workflow"},
								{[]string{"commit", "-m", fmt.Sprintf("Update workflow %d", i)}, "commit update"},
								{[]string{"push", "-f", "origin", branchName}, "push update"},
							}

							for _, c := range cmds {
								cmd := env.createCommand("git", c.args...)
								cmd.Dir = repoPath
								if output, err := cmd.CombinedOutput(); err != nil {
									results <- fmt.Errorf("failed to %s: %v\nOutput: %s", c.msg, err, string(output))
									return
								}
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
						if err := os.WriteFile(file, []byte("test content"), 0600); err != nil {
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
					scanner := updater.NewScanner(repoPath)
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
