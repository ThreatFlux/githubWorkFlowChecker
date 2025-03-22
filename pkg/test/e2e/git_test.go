package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateMockRemoteRepo tests the createMockRemoteRepo function
func TestCreateMockRemoteRepo(t *testing.T) {
	// Skip in CI environments
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a mock remote repository
	remotePath := env.createMockRemoteRepo()

	// Verify the remote repository was created
	assert.DirExists(t, remotePath)

	// Check if it's a bare repository by looking for typical bare repo files
	gitConfigPath := filepath.Join(remotePath, "config")
	assert.FileExists(t, gitConfigPath)

	headsDir := filepath.Join(remotePath, "refs", "heads")
	assert.DirExists(t, headsDir)
}

// TestSimpleGitOperations tests basic git operations without relying on actual remote operations
func TestSimpleGitOperations(t *testing.T) {
	// Skip in CI environments
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}

	// Create a test environment
	env := NewTestEnv(t)
	defer env.Cleanup()

	// Create a local repository
	repoPath := env.createTestRepo()
	assert.DirExists(t, repoPath)

	// Create a file and commit it
	testFile := filepath.Join(repoPath, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0600)
	require.NoError(t, err)

	// Add and commit
	env.stageAndCommit(repoPath, "Add test file")

	// Create a branch for our work
	branchName := "feature-branch"
	env.createBranch(repoPath, branchName)

	// Verify we're on the branch we just created
	err = env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "branch", "--show-current")
		output, err := cmd.Output()
		if err != nil {
			return err
		}

		// Convert bytes to string and trim trailing newline
		branch := string(output)
		if len(branch) > 0 && branch[len(branch)-1] == '\n' {
			branch = branch[:len(branch)-1]
		}

		if branch != branchName {
			t.Errorf("Expected to be on branch %s but was on %s", branchName, branch)
		}
		return nil
	})
	assert.NoError(t, err)

	// Create a bare repository to serve as a remote
	remotePath := env.createMockRemoteRepo()
	assert.DirExists(t, remotePath)

	// Add the remote to the local repository
	remoteName := "test-remote"
	env.addRemote(repoPath, remoteName, remotePath)

	// Verify the remote was added
	err = env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "remote")
		output, err := cmd.Output()
		if err != nil {
			return err
		}

		remotes := string(output)
		if len(remotes) > 0 && remotes[len(remotes)-1] == '\n' {
			remotes = remotes[:len(remotes)-1]
		}

		if remotes != remoteName {
			t.Errorf("Expected remote to be %s but got %s", remoteName, remotes)
		}
		return nil
	})
	assert.NoError(t, err)

	// We'll skip the push operation since it's flaky in tests
	// The coverage for pushChanges is provided by TestPushChangesWithSafeExecution in push_test.go
}

// Implementation of pushChanges that returns errors instead of calling t.Fatalf
func pushChangesWithoutFatalf(env *TestEnv, repoPath, remoteName, branchName string) error {
	// This is the exact same implementation as in pushChanges, but returns errors
	// instead of calling t.Fatalf
	return env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "push", remoteName, branchName)
		return cmd.Run()
	})
}
