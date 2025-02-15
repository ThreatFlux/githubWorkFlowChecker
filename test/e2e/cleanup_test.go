package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// restorePermissions recursively restores write permissions for a directory and its contents
func restorePermissions(path string) error {
	// First restore permissions of the current path
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("failed to restore permissions for %s: %v", path, err)
	}

	// Read directory entries
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %v", path, err)
	}

	// Recursively restore permissions for subdirectories and files
	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			if err := restorePermissions(fullPath); err != nil {
				return err
			}
		} else {
			// For files, just restore normal permissions
			if err := os.Chmod(fullPath, 0644); err != nil {
				return fmt.Errorf("failed to restore permissions for %s: %v", fullPath, err)
			}
		}
	}

	return nil
}

func TestCleanupScenarios(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "CleanupWithLockedFiles",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Create a file and keep it open
				lockFile := filepath.Join(repoPath, "locked-file.txt")
				f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
				if err != nil {
					t.Fatalf("Failed to create lock file: %v", err)
				}

				// Write some content
				if _, err := f.WriteString("test content"); err != nil {
					t.Fatalf("Failed to write to lock file: %v", err)
				}

				// Keep file open in a goroutine
				done := make(chan bool)
				go func() {
					defer f.Close()
					<-done
				}()

				// Try cleanup with file still open
				cleanupErr := make(chan error, 1)
				go func() {
					cleanupErr <- os.RemoveAll(repoPath)
				}()

				// Wait a bit and then release the file
				time.Sleep(100 * time.Millisecond)
				done <- true

				// Check cleanup result
				if err := <-cleanupErr; err != nil {
					t.Errorf("Cleanup failed with locked file: %v", err)
				}
			},
		},
		{
			name: "CleanupDuringGitOperations",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Start multiple git operations
				var wg sync.WaitGroup
				for i := 0; i < 3; i++ {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						branchName := fmt.Sprintf("test-branch-%d", i)
						cmd := env.createCommand("git", "checkout", "-b", branchName)
						cmd.Dir = repoPath
						_ = cmd.Run() // Ignore errors as cleanup might occur during execution
					}(i)
				}

				// Start cleanup in parallel
				cleanupErr := make(chan error, 1)
				go func() {
					// Wait a bit to let git operations start
					time.Sleep(50 * time.Millisecond)
					cleanupErr <- os.RemoveAll(repoPath)
				}()

				// Wait for all operations to complete
				wg.Wait()

				// Check cleanup result
				if err := <-cleanupErr; err != nil {
					t.Errorf("Cleanup failed during git operations: %v", err)
				}
			},
		},
		{
			name: "CleanupWithPartialChanges",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone repository
				repoPath := env.cloneTestRepo()

				// Create multiple files and directories
				testFiles := []string{
					filepath.Join(repoPath, "file1.txt"),
					filepath.Join(repoPath, "dir1", "file2.txt"),
					filepath.Join(repoPath, "dir2", "subdir", "file3.txt"),
				}

				for _, file := range testFiles {
					dir := filepath.Dir(file)
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("Failed to create directory %s: %v", dir, err)
					}
					if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
						t.Fatalf("Failed to create file %s: %v", file, err)
					}
				}

				// Make some directories read-only
				readOnlyDir := filepath.Join(repoPath, "dir2")
				if err := os.Chmod(readOnlyDir, 0444); err != nil {
					t.Fatalf("Failed to make directory read-only: %v", err)
				}

				// Restore permissions before cleanup
				if err := restorePermissions(readOnlyDir); err != nil {
					t.Fatalf("Failed to restore permissions: %v", err)
				}

				// Cleanup should now work with restored permissions
				if err := os.RemoveAll(repoPath); err != nil {
					t.Errorf("Cleanup failed with partial changes: %v", err)
				}

				// Verify cleanup
				if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
					t.Error("Repository directory still exists after cleanup")
				}
			},
		},
		{
			name: "CleanupAfterPanic",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				repoPath := env.cloneTestRepo()

				// Simulate a panic during operations
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Log("Recovered from panic:", r)
						}
					}()

					// Create some files
					testFile := filepath.Join(repoPath, "panic-test.txt")
					if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}

					// Simulate a panic
					panic("simulated panic during operations")
				}()

				// Cleanup should still work after panic
				if err := os.RemoveAll(repoPath); err != nil {
					t.Errorf("Cleanup failed after panic: %v", err)
				}

				// Verify cleanup
				if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
					t.Error("Repository directory still exists after cleanup")
				}
			},
		},
		{
			name: "CleanupWithNestedRepositories",
			testFunc: func(t *testing.T) {
				env := setupTestEnv(t)
				defer env.cleanup()

				// Clone main repository
				mainRepoPath := env.cloneTestRepo()

				// Create a nested repository structure
				subRepoPath := filepath.Join(mainRepoPath, "subrepo")
				if err := os.MkdirAll(subRepoPath, 0755); err != nil {
					t.Fatalf("Failed to create sub-repository directory: %v", err)
				}

				// Initialize sub-repository
				cmd := env.createCommand("git", "init")
				cmd.Dir = subRepoPath
				if err := cmd.Run(); err != nil {
					t.Fatalf("Failed to initialize sub-repository: %v", err)
				}

				// Create some files in sub-repository
				subRepoFile := filepath.Join(subRepoPath, "test.txt")
				if err := os.WriteFile(subRepoFile, []byte("test content"), 0644); err != nil {
					t.Fatalf("Failed to create file in sub-repository: %v", err)
				}

				// Cleanup should handle nested repositories
				if err := os.RemoveAll(mainRepoPath); err != nil {
					t.Errorf("Cleanup failed with nested repositories: %v", err)
				}

				// Verify cleanup
				if _, err := os.Stat(mainRepoPath); !os.IsNotExist(err) {
					t.Error("Main repository directory still exists after cleanup")
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
