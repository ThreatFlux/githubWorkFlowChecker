package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common/testutils"
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
	"github.com/google/go-github/v58/github"
	"golang.org/x/oauth2"
)

// TestEnv represents a test environment for end-to-end tests
type TestEnv struct {
	*testutils.BaseTestEnvironment
	githubClient *github.Client
}

// NewTestEnv creates a new test environment
func NewTestEnv(t *testing.T) *TestEnv {
	base := testutils.NewBaseTestEnvironment(t, "e2e-test-*")

	// Create GitHub client if token is available
	var client *github.Client
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tc := oauth2.NewClient(base.Context(), ts)
		client = github.NewClient(tc)
	} else {
		// Create a client without authentication
		client = github.NewClient(nil)
	}

	return &TestEnv{
		BaseTestEnvironment: base,
		githubClient:        client,
	}
}

// CloneTestRepo creates and clones a test Git repository
func (env *TestEnv) CloneTestRepo() string {
	// Get GitHub token
	token := os.Getenv("GITHUB_TOKEN")

	// Define repository constants
	testRepoOwner := "ThreatFlux"
	testRepo := "test-repo"

	// Check if repository exists or create it
	_, resp, err := env.githubClient.Repositories.Get(env.Context(), testRepoOwner, testRepo)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			// Repository doesn't exist, create it
			repoObj, _, err := env.githubClient.Repositories.Create(env.Context(), testRepoOwner, &github.Repository{
				Name:        github.String(testRepo),
				Description: github.String("Test repository for GitHub Actions workflow updater"),
				AutoInit:    github.Bool(true),
				Private:     github.Bool(true),
			})
			if err != nil {
				env.T.Fatalf("Failed to create test repository: %v", err)
			}
			_ = repoObj // Used to prevent unused variable warning

			// Wait a moment for repository to be initialized
			time.Sleep(2 * time.Second)
		} else {
			env.T.Fatalf("Failed to get repository: %v", err)
		}
	}

	// Create clone URL with token
	var cloneURL string
	if token != "" {
		cloneURL = fmt.Sprintf("https://%s@github.com/%s/%s.git", token, testRepoOwner, testRepo)
	} else {
		cloneURL = fmt.Sprintf("https://github.com/%s/%s.git", testRepoOwner, testRepo)
	}

	// Create repo directory
	repoPath := filepath.Join(env.WorkDir, testRepo)
	if err := os.MkdirAll(repoPath, 0700); err != nil {
		env.T.Fatalf(common.ErrFailedToCreateRepoDir, err)
	}

	// Clone the repository
	cmd := env.CreateCommand("git", "clone", cloneURL, repoPath)
	if err := cmd.Run(); err != nil {
		env.T.Fatalf(common.ErrFailedToCloneRepo, err)
	}

	// Configure git
	if err := env.WithWorkingDir(repoPath, func() error {
		// Set git configuration
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
		env.T.Fatalf("Failed to configure Git: %v", err)
	}

	// Create test workflow file if it doesn't exist
	workflowDir := filepath.Join(repoPath, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "test.yml")

	if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
		// Create workflows directory
		if err := os.MkdirAll(workflowDir, 0700); err != nil {
			env.T.Fatalf(common.ErrFailedToCreateWorkflowsDir, err)
		}

		// Create workflow file
		workflowContent := `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4`

		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0600); err != nil {
			env.T.Fatalf(common.ErrFailedToWriteWorkflowFile, err)
		}

		// Commit and push the workflow file
		env.stageAndCommit(repoPath, "Add test workflow")
	}

	return repoPath
}

// createTestRepo creates a test Git repository
// nolint:unused // This is used in specific scenarios during manual testing
func (env *TestEnv) createTestRepo() string {
	return testutils.CreateTestRepository(env.BaseTestEnvironment, "test-repo")
}

// stageAndCommit stages and commits changes
func (env *TestEnv) stageAndCommit(repoPath, message string) {
	if err := env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "add", ".")
		if err := cmd.Run(); err != nil {
			return err
		}

		cmd = env.CreateCommand("git", "commit", "-m", message)
		return cmd.Run()
	}); err != nil {
		env.T.Fatalf(common.ErrFailedToCommitChanges, err)
	}
}

// createMockRemoteRepo creates a mock remote repository
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) createMockRemoteRepo() string {
	remotePath := filepath.Join(env.WorkDir, "remote-repo")

	// Create bare repository
	if err := os.MkdirAll(remotePath, 0750); err != nil {
		env.T.Fatalf(common.ErrFailedToCreateRepoDir, err)
	}

	cmd := env.CreateCommand("git", "init", "--bare", remotePath)
	if err := cmd.Run(); err != nil {
		env.T.Fatalf("Failed to initialize bare repository: %v", err)
	}

	return remotePath
}

// addRemote adds a remote to a repository
func (env *TestEnv) addRemote(repoPath, remoteName, remoteURL string) {
	if err := env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "remote", "add", remoteName, remoteURL)
		return cmd.Run()
	}); err != nil {
		env.T.Fatalf(common.ErrFailedToAddRemote, err)
	}
}

// makeWorkflowChange creates a change in a workflow file
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) makeWorkflowChange(repoPath, content string) {
	workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
	if err := os.WriteFile(workflowFile, []byte(content), 0600); err != nil {
		env.T.Fatalf(common.ErrFailedToWriteFile, err)
	}
}

// createBranch creates a new branch and switches to it
func (env *TestEnv) createBranch(repoPath, branchName string) {
	if err := env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "checkout", "-b", branchName)
		return cmd.Run()
	}); err != nil {
		env.T.Fatalf(common.ErrFailedToCreateBranch, err)
	}
}

// switchBranch switches to an existing branch
func (env *TestEnv) switchBranch(repoPath, branchName string) {
	if err := env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "checkout", branchName)
		return cmd.Run()
	}); err != nil {
		env.T.Fatalf(common.ErrFailedToSwitchBranch, err)
	}
}

// pushChanges pushes changes to a remote repository
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) pushChanges(repoPath, remoteName, branchName string) {
	if err := env.WithWorkingDir(repoPath, func() error {
		cmd := env.CreateCommand("git", "push", remoteName, branchName)
		return cmd.Run()
	}); err != nil {
		env.T.Fatalf(common.ErrFailedToPushChanges, err)
	}
}

// makeDirectoryReadOnly makes a directory read-only
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) makeDirectoryReadOnly(dirPath string) {
	// #nosec G302 - Test case needs specific permissions to trigger errors
	if err := os.Chmod(dirPath, 0500); err != nil {
		env.T.Fatalf(common.ErrFailedToChangePermissions, err)
	}
}

// restoreDirectoryPermissions restores directory permissions
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) restoreDirectoryPermissions(dirPath string) {
	// #nosec G302 - Test directories need group/other permissions for test commands to work
	if err := os.Chmod(dirPath, 0700); err != nil {
		env.T.Fatalf(common.ErrFailedToRestorePermissions, err)
	}
}

// makeFileReadOnly makes a file read-only
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) makeFileReadOnly(filePath string) {
	if err := os.Chmod(filePath, 0400); err != nil {
		env.T.Fatalf(common.ErrFailedToChangeFilePermissions, err)
	}
}

// restoreFilePermissions restores file permissions
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) restoreFilePermissions(filePath string) {
	if err := os.Chmod(filePath, 0600); err != nil {
		env.T.Fatalf(common.ErrFailedToChangeFilePermissions, err)
	}
}

// writeInvalidWorkflowFile writes an invalid workflow file
// nolint:unused // Used in specific testing scenarios
func (env *TestEnv) writeInvalidWorkflowFile(repoPath string) {
	workflowsDir := filepath.Join(repoPath, ".github", "workflows")
	invalidFile := filepath.Join(workflowsDir, "invalid.yml")
	if err := os.WriteFile(invalidFile, []byte("invalid: yaml: content"), 0600); err != nil {
		env.T.Fatalf(common.ErrFailedToWriteWorkflowFile, err)
	}
}

// CreateScanner creates a scanner for the test environment
func (env *TestEnv) CreateScanner() *updater.Scanner {
	return updater.NewScanner(env.WorkDir)
}

// CloneWithError attempts to clone a repository and returns any error
func (env *TestEnv) CloneWithError(cloneURL, repoPath string) error {
	cmd := env.CreateCommand("git", "clone", cloneURL, repoPath)
	cmd.Dir = env.WorkDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone: %v - %s", err, string(output))
	}
	return nil
}
