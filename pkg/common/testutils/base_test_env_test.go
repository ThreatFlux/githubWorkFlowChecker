package testutils

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseTestEnvironment(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "base-test-env-*")
	defer env.Cleanup()

	// Verify the environment was created correctly
	assert.NotEmpty(t, env.WorkDir)
	assert.DirExists(t, env.WorkDir)
	assert.Equal(t, t, env.T)
	assert.NotNil(t, env.ctx)
	assert.NotNil(t, env.cancel)
	assert.Empty(t, env.Commands)
	assert.Empty(t, env.cleanupFuncs)
}

func TestAddCleanupFunc(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "cleanup-test-*")

	// Create a file to clean up
	tempFile, err := os.CreateTemp(env.WorkDir, "test-file-*")
	require.NoError(t, err)
	err = tempFile.Close()
	require.NoError(t, err)
	// Add a cleanup function
	cleanupCalled := false
	env.AddCleanupFunc(func() error {
		cleanupCalled = true
		return nil
	})

	// Verify the function was added
	assert.Len(t, env.cleanupFuncs, 1)

	// Call cleanup
	env.Cleanup()

	// Verify cleanup function was called
	assert.True(t, cleanupCalled)
}

func TestContext(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "context-test-*")
	defer env.Cleanup()

	// Get the context
	ctx := env.Context()

	// Verify it's the expected context
	assert.NotNil(t, ctx)
	assert.Equal(t, env.ctx, ctx)

	// Verify it has a deadline
	deadline, hasDeadline := ctx.Deadline()
	assert.True(t, hasDeadline)
	assert.True(t, deadline.After(time.Now()))
}

func TestCreateCommandContext(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "cmd-ctx-test-*")
	defer env.Cleanup()

	// Create a command with context
	ctx := context.Background()
	cmd := env.CreateCommandContext(ctx, "echo", "test")

	// Verify command was created and tracked
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Path, "echo")
	assert.Contains(t, cmd.Args[0], "echo")
	assert.Equal(t, "test", cmd.Args[1])
	assert.Contains(t, env.Commands, cmd)

	// Run the command
	output, err := cmd.Output()
	assert.NoError(t, err)
	assert.Equal(t, "test\n", string(output))
}

func TestCreateSubDir(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "subdir-test-*")
	defer env.Cleanup()

	// Create a subdirectory
	subdir := "test-subdir"
	path, err := env.CreateSubDir(subdir)

	// Verify subdirectory was created
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.DirExists(t, path)
	assert.Equal(t, filepath.Join(env.WorkDir, subdir), path)
}

func TestCreateFile(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "file-test-*")
	defer env.Cleanup()

	// Create a file
	filename := "test-file.txt"
	content := []byte("test content")
	permissions := os.FileMode(0600)

	path, err := env.CreateFile(filename, content, permissions)

	// Verify file was created
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.FileExists(t, path)
	assert.Equal(t, filepath.Join(env.WorkDir, filename), path)

	// Verify file content and permissions
	fileContent, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, content, fileContent)

	info, err := os.Stat(path)
	assert.NoError(t, err)
	assert.Equal(t, permissions, info.Mode().Perm())
}

func TestMustCreateSubDir(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "must-subdir-test-*")
	defer env.Cleanup()

	// Create a subdirectory
	subdir := "test-must-subdir"
	path := env.MustCreateSubDir(subdir)

	// Verify subdirectory was created
	assert.NotEmpty(t, path)
	assert.DirExists(t, path)
	assert.Equal(t, filepath.Join(env.WorkDir, subdir), path)
}

func TestMustCreateFile(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "must-file-test-*")
	defer env.Cleanup()

	// Create a file
	filename := "test-must-file.txt"
	content := []byte("test content")
	permissions := os.FileMode(0600)

	path := env.MustCreateFile(filename, content, permissions)

	// Verify file was created
	assert.NotEmpty(t, path)
	assert.FileExists(t, path)
	assert.Equal(t, filepath.Join(env.WorkDir, filename), path)

	// Verify file content and permissions
	fileContent, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, content, fileContent)

	info, err := os.Stat(path)
	assert.NoError(t, err)
	assert.Equal(t, permissions, info.Mode().Perm())
}

func TestCreateWorkflowsDir(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "workflows-dir-test-*")
	defer env.Cleanup()

	// Create workflows directory
	path := env.CreateWorkflowsDir()

	// Verify directory was created
	assert.NotEmpty(t, path)
	assert.DirExists(t, path)
	assert.Equal(t, filepath.Join(env.WorkDir, ".github", "workflows"), path)
}

func TestGetWorkflowsPath(t *testing.T) {
	// Create a new base test environment
	env := NewBaseTestEnvironment(t, "workflows-path-test-*")
	defer env.Cleanup()

	// Get workflows path
	path := env.GetWorkflowsPath()

	// Verify path is correct
	assert.NotEmpty(t, path)
	assert.Equal(t, filepath.Join(env.WorkDir, ".github", "workflows"), path)
}
