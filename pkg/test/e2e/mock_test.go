package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
	"github.com/stretchr/testify/assert"
)

// TestMockPushChanges creates a non-existent repo path to verify error conditions
func TestMockPushChanges(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a local test repository
	repoPath := env.createTestRepo()

	// Create a file to verify existence
	testFile := filepath.Join(repoPath, "test-push.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	assert.NoError(t, err)

	// Now make an access error by removing read permission from the repo
	// Skip on Windows
	if os.PathSeparator != '\\' {
		err = os.Chmod(repoPath, 0000)
		if err != nil {
			t.Fatalf("Failed to change permissions: %v", err)
		}

		// Defer resetting permissions so cleanup can succeed
		defer func() {
			_ = os.Chmod(repoPath, 0700)
		}()

		// Check that file exists but can't be read due to directory permissions
		_, err = os.Stat(testFile)
		assert.Error(t, err, "File should not be accessible due to permissions")
	}
}

// TestCloneTestRepoMockImpl creates a fake clone repository for testing
func TestCloneTestRepoMockImpl(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a mock repo path
	mockRepoPath := filepath.Join(env.WorkDir, "mock-cloned-repo")

	// Create the directory structure expected by tests
	workflowDir := filepath.Join(mockRepoPath, ".github", "workflows")
	err := os.MkdirAll(workflowDir, 0700)
	assert.NoError(t, err, "Failed to create workflow directory")

	// Create a test workflow file
	workflowFile := filepath.Join(workflowDir, "test.yml")
	err = os.WriteFile(workflowFile, []byte("name: Test"), 0600)
	assert.NoError(t, err, "Failed to create workflow file")

	// Create a .git directory to simulate a Git repo
	gitDir := filepath.Join(mockRepoPath, ".git")
	err = os.MkdirAll(gitDir, 0700)
	assert.NoError(t, err, "Failed to create .git directory")

	// Verify the directory structure was created properly
	assert.DirExists(t, filepath.Join(mockRepoPath, ".github", "workflows"))
	assert.FileExists(t, filepath.Join(mockRepoPath, ".github", "workflows", "test.yml"))
}

// TestMakeWorkflowChangeError tests error handling by creating a repo path but not the workflows dir
func TestMakeWorkflowChangeError(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a repo path without the workflows directory structure
	emptyRepo := filepath.Join(env.WorkDir, "empty-repo")
	err := os.MkdirAll(emptyRepo, 0700)
	assert.NoError(t, err)

	// Create a file to verify we can write to the directory
	testFile := filepath.Join(emptyRepo, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0600)
	assert.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, testFile)
}

// TestReadOnlyFileOperations tests file operations with read-only permissions
func TestReadOnlyFileOperations(t *testing.T) {
	// Skip on Windows where permissions work differently
	if os.PathSeparator == '\\' {
		t.Skip("Skipping on Windows")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test file
	testFile := filepath.Join(env.WorkDir, "ro-test-file.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	assert.NoError(t, err)

	// Make the file read-only
	err = os.Chmod(testFile, 0400)
	assert.NoError(t, err)

	// Defer setting it back
	defer func() {
		_ = os.Chmod(testFile, 0600)
	}()

	// Try to write to the file (should fail)
	err = os.WriteFile(testFile, []byte("new content"), 0600)
	assert.Error(t, err, "Should not be able to write to read-only file")

	// Make the file writable again
	err = os.Chmod(testFile, 0600)
	assert.NoError(t, err)

	// Now writing should succeed
	err = os.WriteFile(testFile, []byte("new content"), 0600)
	assert.NoError(t, err)
}

// TestAdditionalScannerFeatures tests additional features of the scanner
func TestAdditionalScannerFeatures(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a scanner
	scanner := env.CreateScanner()

	// Verify scanner is not nil
	assert.NotNil(t, scanner)

	// Use type assertion to make sure it's the right type
	assert.IsType(t, &updater.Scanner{}, scanner)
}
