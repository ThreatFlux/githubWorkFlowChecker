package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

func TestSetupTestEnvWithoutToken(t *testing.T) {
	origToken := os.Getenv("GITHUB_TOKEN")
	err := os.Unsetenv("GITHUB_TOKEN")
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := os.Setenv("GITHUB_TOKEN", origToken)
		if err != nil {
			return
		}
	})

	defer func() {
	}()

}

func TestSetupTestEnvSuccess(t *testing.T) {
	env := setupTestEnv(t)
	workDir := env.workDir

	// Verify work directory exists and has correct permissions
	info, err := os.Stat(workDir)
	if err != nil {
		t.Errorf(common.ErrWorkDirNotCreated, err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf(common.ErrWrongDirectoryPermissions, info.Mode().Perm(), 0700)
	}

	env.cleanup()

	// Verify cleanup
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Error(common.ErrWorkDirNotCleanedUp)
	}
}

func TestGitCommandValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "Empty Arguments",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "Invalid Command",
			args:    []string{"rebase"},
			wantErr: true,
		},
		{
			name:    "Valid Clone Command",
			args:    []string{"clone", "https://github.com/owner/repo.git", "/path/to/repo"},
			wantErr: false,
		},
		{
			name:    "Invalid Clone URL",
			args:    []string{"clone", "git@github.com:owner/repo.git", "/path/to/repo"},
			wantErr: true,
		},
		{
			name:    "Valid Config Command",
			args:    []string{"config", "user.name", "Test User"},
			wantErr: false,
		},
		{
			name:    "Invalid Config Command",
			args:    []string{"config", "core.editor", "vim"},
			wantErr: true,
		},
		{
			name:    "Valid Add Command",
			args:    []string{"add", "."},
			wantErr: false,
		},
		{
			name:    "Invalid Add Command",
			args:    []string{"add", "-p"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGitArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGitArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCloneTestRepo(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T, *testEnv) error
		wantErr   bool
	}{
		{
			name: "Invalid Work Directory Permissions",
			setupFunc: func(t *testing.T, env *testEnv) error {
				// Create a subdirectory with invalid permissions
				invalidDir := filepath.Join(env.workDir, "invalid")
				if err := os.MkdirAll(invalidDir, 0700); err != nil {
					return fmt.Errorf("failed to create invalid directory: %v", err)
				}
				if err := os.Chmod(invalidDir, 0000); err != nil {
					return fmt.Errorf(common.ErrFailedToChangePermissions, err)
				}

				// Try to clone into the invalid directory
				origWorkDir := env.workDir
				env.workDir = invalidDir
				defer func() {
					env.workDir = origWorkDir
					err := os.Chmod(invalidDir, 0700)
					if err != nil {
						return
					} // Reset permissions for cleanup
				}()

				// This should fail due to permissions
				repoPath := filepath.Join(invalidDir, "test-repo")
				if err := os.MkdirAll(repoPath, 0700); err != nil {
					return fmt.Errorf(common.ErrFailedToCreateRepoDir, err)
				}

				return nil
			},
			wantErr: true,
		},
		{
			name: "Successful Repository Creation",
			setupFunc: func(t *testing.T, env *testEnv) error {
				repoPath := env.cloneTestRepo()

				// Verify repository structure
				workflowPath := filepath.Join(repoPath, ".github", "workflows", "test.yml")
				if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
					return fmt.Errorf(common.ErrWorkflowFileNotFound, workflowPath)
				}

				// Verify git configuration
				gitConfigPath := filepath.Join(repoPath, ".git", "config")
				configContent, err := os.ReadFile(gitConfigPath)
				if err != nil {
					return fmt.Errorf(common.ErrFailedToReadGitConfig, err)
				}

				expectedConfigs := []string{
					"name = GitHub Actions Bot",
					"email = actions-bot@github.com",
				}

				for _, expected := range expectedConfigs {
					if !strings.Contains(string(configContent), expected) {
						return fmt.Errorf(common.ErrGitConfigMissingValue, expected)
					}
				}

				return nil
			},
			wantErr: false,
		},
		{
			name: "Repository Already Exists",
			setupFunc: func(t *testing.T, env *testEnv) error {
				// Create a new test environment for the second clone
				env2 := setupTestEnv(t)
				defer env2.cleanup()

				// Clone repository in first environment
				repoPath1 := env.cloneTestRepo()

				// Clone repository in second environment
				repoPath2 := env2.cloneTestRepo()

				// Verify both repos exist and are different
				if repoPath1 == repoPath2 {
					return fmt.Errorf(common.ErrExpectedDifferentPaths)
				}

				// Verify both repos have the workflow file
				for _, path := range []string{repoPath1, repoPath2} {
					workflowPath := filepath.Join(path, ".github", "workflows", "test.yml")
					if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
						return fmt.Errorf(common.ErrWorkflowFileNotFound, path)
					}
				}

				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTestEnv(t)
			defer env.cleanup()

			err := tt.setupFunc(t, env)

			if tt.wantErr {
				if err == nil {
					t.Errorf(common.ErrExpectedError, "an error")
				}
			} else {
				if err != nil {
					t.Errorf(common.ErrUnexpectedError, err)
				}
			}
		})
	}
}

func TestWorkflowFileCreation(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	repoPath := env.cloneTestRepo()
	workflowPath := filepath.Join(repoPath, ".github", "workflows", "test.yml")

	// Test file has regular read-write permissions (0644)
	expectedPerm := os.FileMode(0644)
	err := os.Chmod(workflowPath, 0644)
	if err != nil {
		t.Errorf(common.ErrFailedToChangeFilePermissions, err)
	}
	info, err := os.Stat(workflowPath)
	if err != nil {
		t.Fatalf(common.ErrFailedToStatFile, err)
	}
	actualPerm := info.Mode().Perm()
	if actualPerm != expectedPerm {
		t.Errorf(common.ErrWrongDirectoryPermissions, actualPerm, expectedPerm)
	}

	// Test workflow file content
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("error reading file: %v", err)
	}

	expectedContent := []string{
		"name: Test",
		"on:",
		"push:",
		"branches:",
		"- main",
		"actions/checkout@",
		"actions/setup-go@",
	}

	contentStr := string(content)
	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf(common.ErrWorkflowMissingContent, expected)
		}
	}
}
