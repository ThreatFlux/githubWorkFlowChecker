package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTempDirHelper(t *testing.T) {
	// Create a temp dir helper
	helper := NewTempDirHelper(t, "temp-dir-test-*")

	// Verify the helper was created correctly
	assert.NotNil(t, helper)
	assert.NotEmpty(t, helper.WorkDir)
	assert.DirExists(t, helper.WorkDir)

	// Verify the cleanup function
	cleanup := helper.Cleanup
	assert.NotNil(t, cleanup)

	// Create a test file in the temp dir
	testFile := filepath.Join(helper.WorkDir, "test-file.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	require.NoError(t, err)
	assert.FileExists(t, testFile)

	// Store the workdir path for checking after cleanup
	workDir := helper.WorkDir

	// Run cleanup
	cleanup()

	// Verify the directory was removed
	_, err = os.Stat(workDir)
	assert.True(t, os.IsNotExist(err), "Temp directory should have been removed")
}
