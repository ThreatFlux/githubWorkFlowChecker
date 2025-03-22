package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllPermissionFunctions tests all permission-related functions
func TestAllPermissionFunctions(t *testing.T) {
	// Skip on Windows where permissions work differently
	if os.PathSeparator == '\\' && os.PathListSeparator == ';' {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// ========= Testing makeDirectoryReadOnly and restoreDirectoryPermissions =========

	// Create a test directory
	testDir := filepath.Join(env.WorkDir, "permission-test-dir")
	err := os.MkdirAll(testDir, 0700)
	require.NoError(t, err)

	// Create a file in the directory to test access
	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0600)
	require.NoError(t, err)

	// Verify the file exists
	assert.FileExists(t, testFile)

	// Make the directory read-only
	env.makeDirectoryReadOnly(testDir)

	// Check permissions
	info, err := os.Stat(testDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0500), info.Mode().Perm())

	// Try to create another file (should fail)
	newFile := filepath.Join(testDir, "new.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0600)
	assert.Error(t, err, "Should not be able to create file in read-only directory")

	// Restore directory permissions
	env.restoreDirectoryPermissions(testDir)

	// Check permissions were restored
	info, err = os.Stat(testDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	// Now should be able to create a file
	err = os.WriteFile(newFile, []byte("new content"), 0600)
	assert.NoError(t, err, "Should be able to create file after restoring permissions")
	assert.FileExists(t, newFile)

	// ========= Testing makeFileReadOnly and restoreFilePermissions =========

	// Create a test file
	rwFile := filepath.Join(env.WorkDir, "rw-test.txt")
	err = os.WriteFile(rwFile, []byte("initial content"), 0600)
	require.NoError(t, err)

	// Make the file read-only
	env.makeFileReadOnly(rwFile)

	// Check permissions
	info, err = os.Stat(rwFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0400), info.Mode().Perm())

	// Try to write to the file (should fail)
	err = os.WriteFile(rwFile, []byte("modified content"), 0600)
	assert.Error(t, err, "Should not be able to write to read-only file")

	// Restore file permissions
	env.restoreFilePermissions(rwFile)

	// Check permissions were restored
	info, err = os.Stat(rwFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Now should be able to write to the file
	err = os.WriteFile(rwFile, []byte("modified content"), 0600)
	assert.NoError(t, err, "Should be able to write after restoring permissions")

	// Read the file to verify content was changed
	content, err := os.ReadFile(rwFile)
	require.NoError(t, err)
	assert.Equal(t, "modified content", string(content))
}
