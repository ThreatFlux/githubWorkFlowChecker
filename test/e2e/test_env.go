package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-github/v58/github"
	"golang.org/x/oauth2"
)

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
		t.Fatal("GITHUB_TOKEN environment variable is required")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)

	// Create GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Create temporary work directory
	workDir, err := os.MkdirTemp("", "ghactions-updater-e2e-*")
	if err != nil {
		cancel()
		t.Fatalf("Failed to create work directory: %v", err)
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

	// Create repo directory
	repoPath := filepath.Join(e.workDir, testRepo)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		e.t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Try to get the repository first
	repo, resp, err := e.githubClient.Repositories.Get(e.ctx, testRepoOwner, testRepo)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			// Repository doesn't exist, create it in the organization
			repo, _, err = e.githubClient.Repositories.Create(e.ctx, testRepoOwner, &github.Repository{
				Name:        github.String(testRepo),
				Description: github.String("Test repository for GitHub Actions workflow updater"),
				AutoInit:    github.Bool(true),
				Private:     github.Bool(true),
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

	// Clone the repository
	cmd := exec.CommandContext(e.ctx, "git", "clone", cloneURL, repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("Failed to clone repository: %v\nOutput: %s", err, output)
	}

	// Configure git
	cmds := []struct {
		name string
		args []string
	}{
		{"config user.name", []string{"config", "user.name", "GitHub Actions Bot"}},
		{"config user.email", []string{"config", "user.email", "actions-bot@github.com"}},
	}

	for _, c := range cmds {
		cmd := exec.CommandContext(e.ctx, "git", c.args...)
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			e.t.Fatalf("Failed to %s: %v\nOutput: %s", c.name, err, output)
		}
	}

	// Create test workflow file if it doesn't exist
	workflowDir := filepath.Join(repoPath, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "test.yml")

	if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
		// Create workflows directory
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			e.t.Fatalf("Failed to create workflows directory: %v", err)
		}

		// Create workflow file
		workflowContent := `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v3
      - uses: actions/setup-go@93397bea11091df50f3d7e59dc26a7711a8bcfbe  # v4`

		if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
			e.t.Fatalf("Failed to create workflow file: %v", err)
		}

		// Commit and push the workflow file
		cmds = []struct {
			name string
			args []string
		}{
			{"add", []string{"add", "."}},
			{"commit", []string{"commit", "-m", "Add test workflow", "--author=GitHub Actions Bot <actions-bot@github.com>"}},
			{"push", []string{"push", "origin", "main"}},
		}

		for _, c := range cmds {
			cmd := exec.CommandContext(e.ctx, "git", c.args...)
			cmd.Dir = repoPath
			if output, err := cmd.CombinedOutput(); err != nil {
				e.t.Fatalf("Failed to %s: %v\nOutput: %s", c.name, err, output)
			}
		}
	}

	return repoPath
}
