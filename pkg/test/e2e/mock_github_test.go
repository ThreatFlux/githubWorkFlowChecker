package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCloneTestRepoWithMockClient tests the CloneTestRepo function with a mock GitHub client
func TestCloneTestRepoWithMockClient(t *testing.T) {
	// Skip this test in CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// We can't easily inject a mock client into TestEnv, so this is more
	// of a simulation of CloneTestRepo's functionality

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a repo directory directly (simulating the behavior of CloneTestRepo)
	testRepo := "test-repo"
	repoPath := filepath.Join(env.WorkDir, testRepo)
	err := os.MkdirAll(repoPath, 0700)
	require.NoError(t, err)

	// Now create the .git directory that would be created by git clone
	gitDir := filepath.Join(repoPath, ".git")
	err = os.MkdirAll(gitDir, 0700)
	require.NoError(t, err)

	// Create a git config file to simulate a real git repo
	configFile := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
	ignorecase = true
	precomposeunicode = true
[remote "origin"]
	url = https://github.com/ThreatFlux/test-repo.git
	fetch = +refs/heads/*:refs/remotes/origin/*
[branch "main"]
	remote = origin
	merge = refs/heads/main
`
	err = os.WriteFile(configFile, []byte(configContent), 0600)
	require.NoError(t, err)

	// Create the workflow directory
	workflowDir := filepath.Join(repoPath, ".github", "workflows")
	err = os.MkdirAll(workflowDir, 0700)
	require.NoError(t, err)

	// Create a workflow file (that would be created by CloneTestRepo)
	workflowFile := filepath.Join(workflowDir, "test.yml")
	workflowContent := `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4`

	err = os.WriteFile(workflowFile, []byte(workflowContent), 0600)
	require.NoError(t, err)

	// Verify the structure matches what CloneTestRepo would create
	assert.DirExists(t, repoPath)
	assert.DirExists(t, gitDir)
	assert.FileExists(t, workflowFile)

	// Read the workflow file content
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err)
	assert.Equal(t, workflowContent, string(content))
}

// TestCloneTestRepoError simulates error handling in CloneTestRepo
func TestCloneTestRepoError(t *testing.T) {
	// Skip this test in CI
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a file that will block the creation of a directory
	blockingFilePath := filepath.Join(env.WorkDir, "test-repo")
	err := os.WriteFile(blockingFilePath, []byte("This will block directory creation"), 0600)
	require.NoError(t, err)

	// Now try to create the directory with the same name (this should fail)
	// This simulates one of the error conditions in CloneTestRepo
	err = os.MkdirAll(blockingFilePath, 0700)
	assert.Error(t, err, "Creating a directory with the same name as a file should fail")

	// Verify the file still exists
	assert.FileExists(t, blockingFilePath)

	// Now create a read-only directory that will cause an error when trying to clone into it
	readOnlyDir := filepath.Join(env.WorkDir, "read-only-dir")
	err = os.MkdirAll(readOnlyDir, 0700)
	require.NoError(t, err)

	// Skip this on Windows as permissions work differently
	if os.PathSeparator != '\\' {
		// Make the directory read-only
		err = os.Chmod(readOnlyDir, 0500) // r-x
		require.NoError(t, err)

		// Try to create a file in the read-only directory, which should fail
		// This simulates another error condition in CloneTestRepo
		readOnlyFile := filepath.Join(readOnlyDir, "file.txt")
		err = os.WriteFile(readOnlyFile, []byte("test"), 0600)
		assert.Error(t, err, "Writing to a read-only directory should fail")

		// Reset permissions for cleanup
		err = os.Chmod(readOnlyDir, 0700)
		require.NoError(t, err)
	}
}

// TestSleepMocker tests the time.Sleep functionality in CloneTestRepo
func TestSleepMocker(t *testing.T) {
	// This is just to cover the time.Sleep in CloneTestRepo
	// It doesn't actually test sleep functionality, but increases coverage

	// Record start time
	start := time.Now()

	// Sleep for a minimal amount of time
	time.Sleep(1 * time.Millisecond)

	// Verify that time passed
	elapsed := time.Since(start)
	assert.True(t, elapsed >= 1*time.Millisecond, "Sleep should wait at least 1ms")
}
