package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
)

// allowedGitCommands defines the allowed git commands and their arguments
var allowedGitCommands = map[string][]string{
	"clone":  {"clone"},
	"config": {"config", "user.name", "user.email"},
	"add":    {"add", "."},
	"commit": {"commit", "-m", "--author"},
	"push":   {"push", "origin", "main"},
}

// validateGitArgs checks if the git command and its arguments are allowed
func validateGitArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no git arguments provided")
	}

	cmd := args[0]
	allowedArgs, ok := allowedGitCommands[cmd]
	if !ok {
		return fmt.Errorf("git command not allowed: %s", cmd)
	}

	// Special handling for specific commands
	switch cmd {
	case "clone":
		if len(args) != 3 || !strings.HasPrefix(args[1], "https://") {
			return fmt.Errorf("invalid clone command format")
		}
		return nil
	case "config":
		if len(args) != 3 || !strings.HasPrefix(args[1], "user.") {
			return fmt.Errorf("invalid config command format")
		}
		return nil
	}

	// Validate other commands' arguments
	for _, arg := range args {
		valid := false
		for _, allowedArg := range allowedArgs {
			if strings.HasPrefix(arg, allowedArg) {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("git argument not allowed: %s", arg)
		}
	}

	return nil
}

const (
	testRepoOwner = "ThreatFlux"
	testRepo      = "test-repo"
	testTimeout   = 5 * time.Minute
)

type testEnv struct {
	t            *testing.T
	ctx          context.Context
	cancel       context.CancelFunc
	githubClient *github.Client
	workDir      string
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// Get GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Fatalf("GITHUB_TOKEN environment variable is required")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)

	// Create GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Create temporary work directory with secure permissions
	workDir, err := os.MkdirTemp("", "ghactions-updater-e2e-*")
	if err != nil {
		cancel()
		t.Fatalf("Failed to create work directory: %v", err)
	}
	// Set secure permissions for work directory
	//#nosec G302 - 0700 permissions are appropriate for test work directory
	if err := os.Chmod(workDir, 0700); err != nil {
		cancel()
		t.Fatalf("Failed to set work directory permissions: %v", err)
	}

	return &testEnv{
		t:            t,
		ctx:          ctx,
		cancel:       cancel,
		githubClient: client,
		workDir:      workDir,
	}
}

func (e *testEnv) cleanup() {
	e.cancel()
	if err := os.RemoveAll(e.workDir); err != nil {
		e.t.Errorf("Failed to cleanup work directory: %v", err)
	}
}

func (e *testEnv) cloneTestRepo() string {
	// Get GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		e.t.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// Create clone URL with token
	cloneURL := fmt.Sprintf("https://%s@github.com/%s/%s.git",
		token, testRepoOwner, testRepo)

	// Create repo directory with secure permissions
	repoPath := filepath.Join(e.workDir, testRepo)
	if err := os.MkdirAll(repoPath, 0700); err != nil {
		e.t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Try to get the repository first
	repo, resp, err := e.githubClient.Repositories.Get(e.ctx, testRepoOwner, testRepo)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			// Repository doesn't exist, create it in the organization
			repo, _, err = e.githubClient.Repositories.Create(e.ctx, testRepoOwner, &github.Repository{
				Name:        github.Ptr(testRepo),
				Description: github.Ptr("Test repository for GitHub Actions workflow updater"),
				AutoInit:    github.Ptr(true),
				Private:     github.Ptr(true),
			})
			if err != nil {
				e.t.Fatalf("Failed to create test repository: %v", err)
			}

			// Wait a moment for repository to be initialized
			time.Sleep(2 * time.Second)
		} else {
			e.t.Fatalf("Failed to get repository: %v", err)
		}
	}

	// Ensure we have push access
	if !repo.GetPermissions()["push"] {
		e.t.Fatal("GitHub token does not have push access to the repository")
	}

	// Validate and execute git clone
	args := []string{"clone", cloneURL, repoPath}
	if err := validateGitArgs(args); err != nil {
		e.t.Fatalf("Invalid git command: %v", err)
	}
	//#nosec G204 - git commands are validated through validateGitArgs
	cmd := exec.CommandContext(e.ctx, "git", args...)
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"GITHUB_TOKEN=" + os.Getenv("GITHUB_TOKEN"),
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
	}

	// Configure git with validation
	cmds := []struct {
		name string
		args []string
	}{
		{"config user.name", []string{"config", "user.name", "GitHub Actions Bot"}},
		{"config user.email", []string{"config", "user.email", "actions-bot@github.com"}},
	}

	for _, c := range cmds {
		if err := validateGitArgs(c.args); err != nil {
			e.t.Fatalf("Invalid git command: %v", err)
		}
		//#nosec G204 - git commands are validated through validateGitArgs
		cmd := exec.CommandContext(e.ctx, "git", c.args...)
		cmd.Dir = repoPath
		cmd.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + os.Getenv("HOME"),
			"GITHUB_TOKEN=" + os.Getenv("GITHUB_TOKEN"),
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			e.t.Fatalf("Failed to %s: %v\nOutput: %s", c.name, err, output)
		}
	}

	// Create test workflow file if it doesn't exist
	workflowDir := filepath.Join(repoPath, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "test.yml")

	if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
		// Create workflows directory with secure permissions
		if err := os.MkdirAll(workflowDir, 0700); err != nil {
			e.t.Fatalf("Failed to create workflows directory: %v", err)
		}

		// Create workflow file with secure permissions
		workflowContent := `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v3
      - uses: actions/setup-go@93397bea11091df50f3d7e59dc26a7711a8bcfbe  # v4`

		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0400); err != nil {
			e.t.Fatalf("Failed to create workflow file: %v", err)
		}

		// Commit and push the workflow file with validation
		cmds = []struct {
			name string
			args []string
		}{
			{"add", []string{"add", "."}},
			{"commit", []string{"commit", "-m", "Add test workflow", "--author=GitHub Actions Bot <actions-bot@github.com>"}},
			{"push", []string{"push", "origin", "main"}},
		}

		for _, c := range cmds {
			if err := validateGitArgs(c.args); err != nil {
				e.t.Fatalf("Invalid git command: %v", err)
			}
			//#nosec G204 - git commands are validated through validateGitArgs
			cmd := exec.CommandContext(e.ctx, "git", c.args...)
			cmd.Dir = repoPath
			cmd.Env = []string{
				"PATH=" + os.Getenv("PATH"),
				"HOME=" + os.Getenv("HOME"),
				"GITHUB_TOKEN=" + os.Getenv("GITHUB_TOKEN"),
			}
			if output, err := cmd.CombinedOutput(); err != nil {
				e.t.Fatalf("Failed to %s: %v\nOutput: %s", c.name, err, output)
			}
		}
	}

	return repoPath
}
