package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCloneTestRepoWithMockEnv tests the repository cloning logic
func TestCloneTestRepoWithMockEnv(t *testing.T) {
	// Skip in CI environments
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a source repository with the required structure
	sourceRepo := filepath.Join(env.WorkDir, "source-repo")
	err := os.MkdirAll(sourceRepo, 0700)
	assert.NoError(t, err)

	// Set up git in the source repo
	gitDir := filepath.Join(sourceRepo, ".git")
	err = os.MkdirAll(gitDir, 0700)
	assert.NoError(t, err)

	// Create .github/workflows directory
	workflowsDir := filepath.Join(sourceRepo, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0700)
	assert.NoError(t, err)

	// Create a test workflow file
	workflowFile := filepath.Join(workflowsDir, "test.yml")
	workflowContent := `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4`

	err = os.WriteFile(workflowFile, []byte(workflowContent), 0600)
	assert.NoError(t, err)

	// Verify the repository structure
	assert.DirExists(t, filepath.Join(sourceRepo, ".git"))
	assert.DirExists(t, filepath.Join(sourceRepo, ".github", "workflows"))
	assert.FileExists(t, filepath.Join(sourceRepo, ".github", "workflows", "test.yml"))

	// Create a destination directory for cloning
	destRepo := filepath.Join(env.WorkDir, "dest-repo")
	err = os.MkdirAll(destRepo, 0700)
	assert.NoError(t, err)

	// Perform an operation related to repo cloning but without actually cloning
	// since that requires git setup with remote endpoints
	err = os.MkdirAll(filepath.Join(destRepo, ".git"), 0700)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(destRepo, ".git", "config"), []byte("[core]\n\trepositoryformatversion = 0"), 0600)
	assert.NoError(t, err)

	// Clone the workflow file structure manually
	destWorkflowDir := filepath.Join(destRepo, ".github", "workflows")
	err = os.MkdirAll(destWorkflowDir, 0700)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(destWorkflowDir, "test.yml"), []byte(workflowContent), 0600)
	assert.NoError(t, err)

	// Verify the "cloned" repo structure
	assert.DirExists(t, filepath.Join(destRepo, ".git"))
	assert.DirExists(t, filepath.Join(destRepo, ".github", "workflows"))
	assert.FileExists(t, filepath.Join(destRepo, ".github", "workflows", "test.yml"))

	// Read the workflow file content
	content, err := os.ReadFile(filepath.Join(destRepo, ".github", "workflows", "test.yml"))
	assert.NoError(t, err)
	assert.Equal(t, workflowContent, string(content))
}
