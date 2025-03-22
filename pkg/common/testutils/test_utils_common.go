package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

// StandardWorkflowContent returns a standard GitHub workflow content for testing
func StandardWorkflowContent(actionRef string) string {
	return fmt.Sprintf(`name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: %s`, actionRef)
}

// ExpectSubstring checks if a string contains a substring
func ExpectSubstring(t *testing.T, haystack, needle, message string) {
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s\nExpected substring: %q\nActual string: %q", message, needle, haystack)
	}
}

// ExpectNotSubstring checks if a string does not contain a substring
func ExpectNotSubstring(t *testing.T, haystack, needle, message string) {
	if strings.Contains(haystack, needle) {
		t.Errorf("%s\nUnexpected substring found: %q\nActual string: %q", message, needle, haystack)
	}
}

// AssertFileExists checks if a file exists
func AssertFileExists(t *testing.T, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file to exist: %s", path)
	}
}

// AssertFileNotExists checks if a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("Expected file not to exist: %s", path)
	}
}

// AssertFileContains checks if a file contains specific content
func AssertFileContains(t *testing.T, path, content string) {
	// Validate path to avoid directory traversal
	if !ValidateFilePath(path) {
		t.Errorf("Invalid file path: %s", path)
		return
	}

	// Create a safe path for validation
	safePath := filepath.Clean(path)

	bytes, err := os.ReadFile(safePath)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
		return
	}

	fileContent := string(bytes)
	if !strings.Contains(fileContent, content) {
		t.Errorf("Expected file to contain: %q\nActual content: %q", content, fileContent)
	}
}

// ValidateFilePath checks if a path is safe by ensuring it doesn't contain directory traversal attempts
func ValidateFilePath(path string) bool {
	// Make sure the path is canonical
	clean := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(clean, "..") {
		return false
	}

	// Additional validation could be added here
	return true
}

// AssertFileNotContains checks if a file does not contain specific content
func AssertFileNotContains(t *testing.T, path, content string) {
	// Validate path to avoid directory traversal
	if !ValidateFilePath(path) {
		t.Errorf("Invalid file path: %s", path)
		return
	}

	// Create a safe path for validation
	safePath := filepath.Clean(path)

	bytes, err := os.ReadFile(safePath)
	if err != nil {
		t.Errorf("Failed to read file: %v", err)
		return
	}

	fileContent := string(bytes)
	if strings.Contains(fileContent, content) {
		t.Errorf("Expected file not to contain: %q\nActual content: %q", content, fileContent)
	}
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return false // Timeout
		case <-ticker.C:
			if condition() {
				return true // Condition met
			}
		}
	}
}

// ContainsComparable checks if a slice contains a comparable value
func ContainsComparable[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// CreateTestRepository creates a generic test Git repository with workflow files
// This is a common implementation used across test environments
func CreateTestRepository(env interface{}, repoName string) string {
	// Dynamically determine the type of test environment
	var workDir string
	var t *testing.T
	var createCommand func(name string, args ...string) interface{}
	var withWorkingDir func(dir string, fn func() error) error

	// Extract needed methods from the test environment
	switch e := env.(type) {
	case *BaseTestEnvironment:
		workDir = e.WorkDir
		t = e.T
		createCommand = func(name string, args ...string) interface{} {
			return e.CreateCommand(name, args...)
		}
		withWorkingDir = e.WithWorkingDir
	default:
		// Get common access methods through reflection
		// This is a fallback mechanism that could be implemented
		// if needed for additional environment types
		panic("Unsupported test environment type")
	}

	// Create repository path
	repoPath := filepath.Join(workDir, repoName)

	// Create repository directory
	if err := os.MkdirAll(repoPath, 0750); err != nil {
		t.Fatalf(common.ErrFailedToCreateRepoDir, err)
	}

	// Create workflows directory
	workflowsDir := filepath.Join(repoPath, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0700); err != nil {
		t.Fatalf(common.ErrFailedToCreateWorkflowsDir, err)
	}

	// Create a test workflow file
	workflowContent := []byte(StandardWorkflowContent("actions/checkout@v2"))

	workflowFile := filepath.Join(workflowsDir, "test.yml")
	if err := os.WriteFile(workflowFile, workflowContent, 0600); err != nil {
		t.Fatalf(common.ErrFailedToWriteWorkflowFile, err)
	}

	// Initialize Git repository
	if err := withWorkingDir(repoPath, func() error {
		// Initialize Git repository
		cmd := createCommand("git", "init").(interface{ Run() error })
		if err := cmd.Run(); err != nil {
			return err
		}

		// Configure Git
		configCmds := [][]string{
			{"git", "config", "user.name", "Test User"},
			{"git", "config", "user.email", "test@example.com"},
			{"git", "config", "init.defaultBranch", "main"},
		}

		for _, args := range configCmds {
			cmd := createCommand(args[0], args[1:]...).(interface{ Run() error })
			if err := cmd.Run(); err != nil {
				return err
			}
		}

		// Add and commit files
		addCmd := createCommand("git", "add", ".").(interface{ Run() error })
		if err := addCmd.Run(); err != nil {
			return err
		}

		commitCmd := createCommand("git", "commit", "-m", "Initial commit").(interface{ Run() error })
		return commitCmd.Run()
	}); err != nil {
		t.Fatalf("Failed to initialize Git repository: %v", err)
	}

	return repoPath
}
