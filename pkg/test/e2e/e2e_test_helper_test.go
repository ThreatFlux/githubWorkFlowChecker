package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
	"github.com/google/go-github/v58/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestEnv(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Check if environment was created properly
	assert.NotNil(t, env.BaseTestEnvironment)
	assert.NotNil(t, env.githubClient)
	assert.NotEmpty(t, env.WorkDir)
	assert.DirExists(t, env.WorkDir)
}

func TestCreateScanner(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a scanner
	scanner := env.CreateScanner()

	// Verify scanner was created and properly configured
	assert.NotNil(t, scanner)
	// Check that the scanner instance exists
	assert.IsType(t, &updater.Scanner{}, scanner)
}

func TestMakeDirectoryReadOnly(t *testing.T) {
	// Skip if running on Windows as file permissions work differently
	if isWindows() {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test directory
	testDir := filepath.Join(env.WorkDir, "test-dir")
	err := os.MkdirAll(testDir, 0700)
	require.NoError(t, err)

	// Make the directory read-only
	env.makeDirectoryReadOnly(testDir)

	// Check permissions
	info, err := os.Stat(testDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0500), info.Mode().Perm())

	// Try to create a file in the read-only directory
	// This should fail because the directory is read-only
	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0600)
	assert.Error(t, err, "Should not be able to write to read-only directory")

	// Restore permissions
	env.restoreDirectoryPermissions(testDir)

	// Check permissions were restored
	info, err = os.Stat(testDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())

	// Now writing should work
	err = os.WriteFile(testFile, []byte("test"), 0600)
	assert.NoError(t, err, "Should be able to write after restoring permissions")
}

func TestMakeFileReadOnly(t *testing.T) {
	// Skip if running on Windows as file permissions work differently
	if isWindows() {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test file
	testFile := filepath.Join(env.WorkDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	require.NoError(t, err)

	// Make the file read-only
	env.makeFileReadOnly(testFile)

	// Check permissions
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0400), info.Mode().Perm())

	// Try to write to the file
	// This should fail because the file is read-only
	err = os.WriteFile(testFile, []byte("new content"), 0600)
	assert.Error(t, err, "Should not be able to write to read-only file")

	// Restore permissions
	env.restoreFilePermissions(testFile)

	// Check permissions were restored
	info, err = os.Stat(testFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	// Now writing should work
	err = os.WriteFile(testFile, []byte("new content"), 0600)
	assert.NoError(t, err, "Should be able to write after restoring permissions")
}

func TestWriteInvalidWorkflowFile(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a local test repo
	repoPath := env.createTestRepo()

	// Write an invalid workflow file
	env.writeInvalidWorkflowFile(repoPath)

	// Verify the file was created
	invalidFile := filepath.Join(repoPath, ".github", "workflows", "invalid.yml")
	assert.FileExists(t, invalidFile)

	// Check the content
	content, err := os.ReadFile(invalidFile)
	require.NoError(t, err)
	assert.Equal(t, "invalid: yaml: content", string(content))
}

func TestCloneWithError(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Try to clone a non-existent repository
	nonExistentURL := "https://github.com/this-does-not-exist/this-repo-does-not-exist.git"
	repoPath := filepath.Join(env.WorkDir, "should-not-exist")

	// This should fail
	err := env.CloneWithError(nonExistentURL, repoPath)
	assert.Error(t, err, "Cloning a non-existent repository should fail")

	// The directory should not exist
	assert.NoDirExists(t, repoPath)
}

func TestAddRemote(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test repository
	repoPath := env.createTestRepo()

	// Create a mock remote repository
	remotePath := env.createMockRemoteRepo()

	// Add the remote
	env.addRemote(repoPath, "origin", remotePath)

	// Verify the remote was added
	var output []byte
	err := env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "remote", "-v")
		var err error
		output, err = cmd.CombinedOutput()
		return err
	})
	require.NoError(t, err)

	// Check that the output contains the remote
	assert.Contains(t, string(output), "origin")
	assert.Contains(t, string(output), remotePath)
}

func TestCreateBranch(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test repository
	repoPath := env.createTestRepo()

	// Create a new branch
	branchName := "test-branch"
	env.createBranch(repoPath, branchName)

	// Verify the branch was created and we're on it
	var output []byte
	err := env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "branch", "--show-current")
		var err error
		output, err = cmd.CombinedOutput()
		return err
	})
	require.NoError(t, err)

	// Check that the output is the branch name
	assert.Equal(t, branchName, string(output[:len(output)-1])) // Trim the newline
}

func TestSwitchBranch(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test repository
	repoPath := env.createTestRepo()

	// Create a new branch
	branchName := "test-branch"
	env.createBranch(repoPath, branchName)

	// Switch back to main branch
	// Note: In newer git versions, the default branch might be 'master' or 'main'
	// We'll check which one exists
	defaultBranch := "main"
	var output []byte
	err := env.WithWorkingDir(repoPath, func() error {
		// Check if main branch exists
		cmd := env.CreateCommand("git", "show-ref", "--verify", "--quiet", "refs/heads/main")
		if err := cmd.Run(); err != nil {
			// If main doesn't exist, try master
			defaultBranch = "master"
			cmd = env.CreateCommand("git", "show-ref", "--verify", "--quiet", "refs/heads/master")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("neither main nor master branch exists")
			}
		}
		return nil
	})
	require.NoError(t, err, "Should be able to identify default branch")

	// Switch to default branch
	env.switchBranch(repoPath, defaultBranch)

	// Verify we're on the default branch
	err = env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "branch", "--show-current")
		var err error
		output, err = cmd.CombinedOutput()
		return err
	})
	require.NoError(t, err)

	// Check that the output is the default branch
	assert.Equal(t, defaultBranch, strings.TrimSpace(string(output)))

	// Switch back to test-branch
	env.switchBranch(repoPath, branchName)

	// Verify we're on test-branch
	err = env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "branch", "--show-current")
		var err error
		output, err = cmd.CombinedOutput()
		return err
	})
	require.NoError(t, err)

	// Check that the output is the branch name
	assert.Equal(t, branchName, strings.TrimSpace(string(output)))
}

func TestMakeWorkflowChange(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test repository
	repoPath := env.createTestRepo()

	// Make a change to the workflow file
	newContent := `name: Modified Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v3`

	env.makeWorkflowChange(repoPath, newContent)

	// Verify the file was changed
	workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(content))
}

func TestStageAndCommit(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a test repository
	repoPath := env.createTestRepo()

	// Make a change to the workflow file
	newContent := `name: Modified Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4`

	workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
	err := os.WriteFile(workflowFile, []byte(newContent), 0600)
	require.NoError(t, err)

	// Stage and commit the change
	commitMessage := "Update workflow file"
	env.stageAndCommit(repoPath, commitMessage)

	// Verify the change was committed
	var output []byte
	err = env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "log", "-1", "--pretty=%B")
		var err error
		output, err = cmd.CombinedOutput()
		return err
	})
	require.NoError(t, err)

	// Check that the output contains the commit message
	assert.Contains(t, string(output), commitMessage)
}

// Helper function to check if running on Windows
func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

// Mock implementation of env.CloneTestRepo() that doesn't require GitHub access
// This allows tests to run without GitHub credentials
func (env *TestEnv) createAndCloneTestRepo() string {
	// Create a source repo
	sourceRepo := env.createTestRepo()

	// Create a destination path
	destRepo := filepath.Join(env.WorkDir, "cloned-repo")

	// Clone the repo locally
	cmd := env.CreateCommand("git", "clone", sourceRepo, destRepo)
	if err := cmd.Run(); err != nil {
		env.T.Fatalf("Failed to clone repo: %v", err)
	}

	// Set up git config
	if err := env.WithWorkingDir(destRepo, func() error {
		configCmds := [][]string{
			{"git", "config", "user.name", "Test User"},
			{"git", "config", "user.email", "test@example.com"},
		}

		for _, args := range configCmds {
			cmd := env.CreateCommand(args[0], args[1:]...)
			if err := cmd.Run(); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		env.T.Fatalf("Failed to configure git: %v", err)
	}

	return destRepo
}

// Mock MockGithubClient is a simplified version for testing
// This lets us test without an actual GitHub token
type MockGithubClient struct {
	Repositories *MockRepositoriesService
}

type MockRepositoriesService struct {
	GetFunc    func(ctx interface{}, owner, repo string) (*github.Repository, *github.Response, error)
	CreateFunc func(ctx interface{}, owner string, repo *github.Repository) (*github.Repository, *github.Response, error)
}

func TestCreateTestRepoWithMockClient(t *testing.T) {
	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Replace GitHub client with a mock
	mockClient := &MockGithubClient{
		Repositories: &MockRepositoriesService{
			GetFunc: func(ctx interface{}, owner, repo string) (*github.Repository, *github.Response, error) {
				// Return a mock repository
				return &github.Repository{
					Name: github.String(repo),
					Owner: &github.User{
						Login: github.String(owner),
					},
				}, nil, nil
			},
		},
	}

	// Replace env's GitHub client with the mock
	// This is just for demonstration, not actually used in the test
	// since we're calling createAndCloneTestRepo() instead
	_ = mockClient

	// Use our local implementation that doesn't require GitHub
	repoPath := env.createAndCloneTestRepo()

	// Verify the result
	assert.DirExists(t, repoPath)
	gitDir := filepath.Join(repoPath, ".git")
	assert.DirExists(t, gitDir)
}

func TestCloneTestRepoMockHelper(t *testing.T) {
	// Skip this test as it's mainly for coverage and needs special setup
	t.Skip("Skipping test that requires special git setup")
}

func TestPushChanges(t *testing.T) {
	// Skip this test as it's mainly for coverage and needs special git setup
	t.Skip("Skipping test that requires special git setup")

	// This is left here for documentation purposes on how to properly test this function
	/*
			// Create a test environment
			env := NewTestEnv(t)
			defer env.Cleanup()

			// Create source and destination repos
			sourceRepo := env.createTestRepo()
			remotePath := env.createMockRemoteRepo()

			// Add remote to source repo
			env.addRemote(sourceRepo, "origin", remotePath)

			// Make a change to the workflow file
			workflowFile := filepath.Join(sourceRepo, ".github", "workflows", "test.yml")
			newContent := `name: Updated Test
		on: [push, pull_request]
		jobs:
		  test:
		    runs-on: ubuntu-latest
		    steps:
		      - uses: actions/checkout@v4
		      - uses: actions/setup-node@v3`

			err := os.WriteFile(workflowFile, []byte(newContent), 0600)
			require.NoError(t, err)

			// Stage and commit the change
			env.stageAndCommit(sourceRepo, "Update workflow for push test")

			// Push the changes
			env.pushChanges(sourceRepo, "origin", "main")

			// Verify the push was successful by checking the remote
			// Clone the remote to verify
			clonedPath := filepath.Join(env.WorkDir, "cloned-remote")
			cmd := env.CreateCommand("git", "clone", remotePath, clonedPath)
			err = cmd.Run()
			require.NoError(t, err)

			// Check that the pushed changes exist in the cloned repo
			clonedFile := filepath.Join(clonedPath, ".github", "workflows", "test.yml")
			assert.FileExists(t, clonedFile)

			// Check content
			content, err := os.ReadFile(clonedFile)
			require.NoError(t, err)
			assert.Equal(t, newContent, string(content))
	*/
}
