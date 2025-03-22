package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMakeDirectoryReadOnlyWithMock tests directory permissions with a mock
func TestMakeDirectoryReadOnlyWithMock(t *testing.T) {
	// Skip on Windows where permissions work differently
	if os.PathSeparator == '\\' {
		t.Skip("Skipping on Windows")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test directory structure
	mainDir := filepath.Join(env.WorkDir, "permission-test")
	err := os.MkdirAll(mainDir, 0700)
	assert.NoError(t, err)

	// Create a subfolder
	subDir := filepath.Join(mainDir, "sub")
	err = os.MkdirAll(subDir, 0700)
	assert.NoError(t, err)

	// Create a file in the subfolder
	testFile := filepath.Join(subDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0600)
	assert.NoError(t, err)

	// Make the directory read-only
	env.makeDirectoryReadOnly(mainDir)

	// Check permissions
	info, err := os.Stat(mainDir)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0500), info.Mode().Perm())

	// Try to create a file in the read-only directory (should fail)
	newFile := filepath.Join(mainDir, "new.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0600)
	assert.Error(t, err)

	// Restore permissions
	env.restoreDirectoryPermissions(mainDir)

	// Check permissions were restored
	info, err = os.Stat(mainDir)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	// Now creating a file should work
	err = os.WriteFile(newFile, []byte("new content"), 0600)
	assert.NoError(t, err)
	assert.FileExists(t, newFile)
}

// TestMakeFileReadOnlyWithMock tests file permissions with a mock
func TestMakeFileReadOnlyWithMock(t *testing.T) {
	// Skip on Windows where permissions work differently
	if os.PathSeparator == '\\' {
		t.Skip("Skipping on Windows")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test file
	testFile := filepath.Join(env.WorkDir, "readonly-test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	assert.NoError(t, err)

	// Make the file read-only
	env.makeFileReadOnly(testFile)

	// Check permissions
	info, err := os.Stat(testFile)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0400), info.Mode().Perm())

	// Try to write to the read-only file (should fail)
	err = os.WriteFile(testFile, []byte("new content"), 0600)
	assert.Error(t, err)

	// Restore permissions
	env.restoreFilePermissions(testFile)

	// Check permissions were restored
	info, err = os.Stat(testFile)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Now writing should work
	err = os.WriteFile(testFile, []byte("new content"), 0600)
	assert.NoError(t, err)

	// Read file to verify content
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}

// TestMakeWorkflowChangeMock tests making workflow changes
func TestMakeWorkflowChangeMock(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a repository with necessary directory structure
	repoPath := filepath.Join(env.WorkDir, "workflow-test-repo")
	workflowDir := filepath.Join(repoPath, ".github", "workflows")
	err := os.MkdirAll(workflowDir, 0700)
	assert.NoError(t, err)

	// Set up a test workflow file
	initialContent := "name: Initial Workflow"
	newContent := "name: Updated Workflow"

	// Make a workflow change (create the file)
	env.makeWorkflowChange(repoPath, initialContent)

	// Check if the file was created
	workflowFile := filepath.Join(workflowDir, "test.yml")
	assert.FileExists(t, workflowFile)

	// Check content
	content, err := os.ReadFile(workflowFile)
	assert.NoError(t, err)
	assert.Equal(t, initialContent, string(content))

	// Update the workflow file
	env.makeWorkflowChange(repoPath, newContent)

	// Check if content was updated
	content, err = os.ReadFile(workflowFile)
	assert.NoError(t, err)
	assert.Equal(t, newContent, string(content))
}
