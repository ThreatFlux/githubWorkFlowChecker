package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
)

type mockVersionChecker struct {
	latestVersion string
	latestHash    string
	err           error
}

func (m *mockVersionChecker) GetLatestVersion(ctx context.Context, action updater.ActionReference) (string, string, error) {
	return m.latestVersion, m.latestHash, m.err
}

func (m *mockVersionChecker) IsUpdateAvailable(ctx context.Context, action updater.ActionReference) (bool, string, string, error) {
	if m.err != nil {
		return false, "", "", m.err
	}
	return true, m.latestVersion, m.latestHash, nil
}

func (m *mockVersionChecker) GetCommitHash(ctx context.Context, action updater.ActionReference, version string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.latestHash, nil
}

type mockPRCreator struct {
	err error
}

func (m *mockPRCreator) CreatePR(ctx context.Context, updates []*updater.Update) error {
	return m.err
}

func TestRun(t *testing.T) {
	tests := []struct {
		name                string
		workflowContent     string
		additionalWorkflows map[string]string
		versionChecker      *mockVersionChecker
		prCreator           *mockPRCreator
		flags               map[string]string
		wantErr             bool
	}{
		{
			name: "version check failure - GetLatestVersion error",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "",
				latestHash:    "",
				err:           fmt.Errorf("failed to get latest version: rate limit exceeded"),
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false, // Should not error as it just logs and continues
		},
		{
			name: "version check failure - IsUpdateAvailable error",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123",
				err:           fmt.Errorf("failed to check update: network timeout"),
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false, // Should log error and continue with next action
		},
		{
			name: "version check failure - GetCommitHash error",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-python@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "",
				err:           fmt.Errorf("failed to get commit hash: invalid version"),
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false, // Should log error and continue
		},
		{
			name: "multiple actions with mixed version check results",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2
      - uses: actions/setup-python@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false,
		},
		{
			name: "successful update",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false,
		},
		{
			name: "version checker error",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "",
				latestHash:    "",
				err:           fmt.Errorf("API error"),
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false, // Should not error as it just logs and continues
		},
		{
			name: "pr creator error",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: fmt.Errorf("PR creation failed"),
			},
			wantErr: true,
		},
		{
			name:            "no workflow files",
			workflowContent: "",
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false,
		},
		{
			name: "multiple workflow files with mixed updates",
			workflowContent: `name: Test Workflow 1
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			additionalWorkflows: map[string]string{
				"test2.yml": `name: Test Workflow 2
on: [pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-node@v2`,
				"test3.yml": `name: Test Workflow 3
on: [workflow_dispatch]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-python@v2`,
			},
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false,
		},
		{
			name: "multiple workflow files with errors",
			workflowContent: `name: Test Workflow 1
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			additionalWorkflows: map[string]string{
				"invalid.yml": `invalid yaml content`,
				"test2.yml": `name: Test Workflow 2
on: [pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-node@v2`,
			},
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false, // Should continue processing valid files
		},
		{
			name: "multiple workflow files with version check errors",
			workflowContent: `name: Test Workflow 1
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			additionalWorkflows: map[string]string{
				"test2.yml": `name: Test Workflow 2
on: [pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-node@v2`,
			},
			versionChecker: &mockVersionChecker{
				latestVersion: "",
				latestHash:    "",
				err:           fmt.Errorf("API rate limit exceeded"),
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantErr: false, // Should log errors and continue
		},
		{
			name: "custom workflows path",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			flags: map[string]string{
				"workflows-path": "custom/workflows",
			},
			wantErr: false, // Should not error, but won't find any workflows in the custom path
			additionalWorkflows: map[string]string{
				"custom_workflow.yml": `name: Custom Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			},
		},
		{
			name: "dry run mode",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			flags: map[string]string{
				"dry-run": "true",
			},
			wantErr: false, // Should not error, just print changes
		},
		{
			name: "stage mode",
			workflowContent: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			flags: map[string]string{
				"stage": "true",
			},
			wantErr: false, // Should not error, apply changes locally
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test files
			tempDir, err := os.MkdirTemp("", "workflow-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func(path string) {
				err := os.RemoveAll(path)
				if err != nil {
					t.Fatalf("Failed to remove temp dir: %v", err)
				}
			}(tempDir)

			// Create .github/workflows directory
			workflowsDir := filepath.Join(tempDir, ".github", "workflows")
			if err := os.MkdirAll(workflowsDir, 0755); err != nil {
				t.Fatalf("Failed to create workflows dir: %v", err)
			}

			// Create test workflow files
			if tt.workflowContent != "" {
				if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), []byte(tt.workflowContent), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			}

			// Create custom workflows directory if needed
			if tt.flags != nil && tt.flags["workflows-path"] != "" {
				customDir := filepath.Join(tempDir, tt.flags["workflows-path"])
				if err := os.MkdirAll(customDir, 0755); err != nil {
					t.Fatalf("Failed to create custom workflows dir: %v", err)
				}

				// Create a test file in the custom directory
				if err := os.WriteFile(filepath.Join(customDir, "custom.yml"), []byte(tt.workflowContent), 0644); err != nil {
					t.Fatalf("Failed to create custom workflow file: %v", err)
				}
			}

			// Create additional workflow files if provided
			if tt.additionalWorkflows != nil {
				for filename, content := range tt.additionalWorkflows {
					if err := os.WriteFile(filepath.Join(workflowsDir, filename), []byte(content), 0644); err != nil {
						t.Fatalf("Failed to create additional workflow file %s: %v", filename, err)
					}
				}
			}

			// Create and change to a temporary working directory
			workingDir, err := os.MkdirTemp("", "test-working-dir")
			if err != nil {
				t.Fatalf("Failed to create working directory: %v", err)
			}
			defer func(path string) {
				err := os.RemoveAll(path)
				if err != nil {
					t.Fatalf("Failed to remove working directory: %v", err)
				}
			}(workingDir)

			if err := os.Chdir(workingDir); err != nil {
				t.Fatalf("Failed to change to working directory: %v", err)
			}

			// Save original working directory to restore later
			oldWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current working directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(oldWd); err != nil {
					t.Errorf("Failed to restore working directory: %v", err)
				}
			}()

			// Change to temp directory for test
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Reset and set up test flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			repoPath = flag.String("repo", ".", "Path to the repository")
			owner = flag.String("owner", "", "Repository owner")
			repo = flag.String("repo-name", "", "Repository name")
			token = flag.String("token", "", "GitHub token")
			workflowsPath = flag.String("workflows-path", ".github/workflows", "Path to workflow files")
			dryRun = flag.Bool("dry-run", false, "Show changes without applying them")
			stage = flag.Bool("stage", false, "Apply changes locally without creating a PR")

			// Build command line args
			args := []string{"cmd", "-owner=test-owner", "-repo-name=test-repo", "-token=test-token"}
			for k, v := range tt.flags {
				args = append(args, fmt.Sprintf("-%s=%s", k, v))
			}
			os.Args = args
			if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
				t.Fatalf("Failed to parse command line flags: %v", err)
			}

			// Save and restore factories
			oldVersionFactory := versionCheckerFactory
			oldPRFactory := prCreatorFactory
			defer func() {
				versionCheckerFactory = oldVersionFactory
				prCreatorFactory = oldPRFactory
			}()

			// Set up mock version checker
			versionCheckerFactory = func(token string) updater.VersionChecker {
				return tt.versionChecker
			}

			// Set up mock PR creator
			prCreatorFactory = func(token, owner, repo string) updater.PRCreator {
				return tt.prCreator
			}

			// Run the function with mocks
			err = run()
			if tt.wantErr {
				if err == nil {
					t.Error("run() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("run() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRunWithAbsError(t *testing.T) {
	// Save original Abs function and restore after test
	defer restoreAbs()

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	// Create .github/workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows dir: %v", err)
	}

	// Create a test workflow file
	workflowContent := []byte(`name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`)

	if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), workflowContent, 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Set repoPath to the temporary directory
	*repoPath = tempDir

	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	repoPath = flag.String("repo", ".", "Path to the repository")
	owner = flag.String("owner", "test-owner", "Repository owner")
	repo = flag.String("repo-name", "test-repo", "Repository name")
	token = flag.String("token", "test-token", "GitHub token")
	workflowsPath = flag.String("workflows-path", ".github/workflows", "Path to workflow files")
	dryRun = flag.Bool("dry-run", false, "Show changes without applying them")
	stage = flag.Bool("stage", false, "Apply changes locally without creating a PR")

	if err := flag.CommandLine.Parse([]string{}); err != nil {
		t.Fatalf("Failed to parse command line flags: %v", err)
	}

	// Mock version checker and PR creator
	oldVersionFactory := versionCheckerFactory
	oldPRFactory := prCreatorFactory
	defer func() {
		versionCheckerFactory = oldVersionFactory
		prCreatorFactory = oldPRFactory
	}()

	versionCheckerFactory = func(token string) updater.VersionChecker {
		return &mockVersionChecker{
			latestVersion: "v3",
			latestHash:    "abc123def456",
			err:           nil,
		}
	}

	prCreatorFactory = func(token, owner, repo string) updater.PRCreator {
		return &mockPRCreator{
			err: nil,
		}
	}

	// Test filepath.Abs error
	mockAbsWithError()
	err = run()
	if err == nil {
		t.Error("run() with Abs error: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "mock Abs error") {
		t.Errorf("run() with Abs error: expected 'mock Abs error', got %v", err)
	}
}

func TestMain(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	// Create .github/workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows dir: %v", err)
	}

	// Create a test workflow file
	workflowContent := []byte(`name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`)

	if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), workflowContent, 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Create and change to a temporary working directory
	workingDir, err := os.MkdirTemp("", "test-working-dir")
	if err != nil {
		t.Fatalf("Failed to create working directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove working directory: %v", err)
		}
	}(workingDir)

	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("Failed to change to working directory: %v", err)
	}

	// Save original working directory and args
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Change to temp directory for test
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Save and restore factories
	oldVersionFactory := versionCheckerFactory
	oldPRFactory := prCreatorFactory
	defer func() {
		versionCheckerFactory = oldVersionFactory
		prCreatorFactory = oldPRFactory
	}()

	tests := []struct {
		name           string
		args           []string
		envVars        map[string]string
		versionChecker *mockVersionChecker
		prCreator      *mockPRCreator
		wantPanic      bool
	}{
		{
			name: "successful run",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantPanic: false,
		},
		{
			name: "missing required flag",
			args: []string{
				"cmd",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				latestHash:    "abc123def456",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: nil,
			},
			wantPanic: true,
		},
		{
			name: "run error",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			versionChecker: &mockVersionChecker{
				latestVersion: "v3",
				err:           nil,
			},
			prCreator: &mockPRCreator{
				err: fmt.Errorf("PR creation failed"),
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags and command line
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			repoPath = flag.String("repo", ".", "Path to the repository")
			owner = flag.String("owner", "", "Repository owner")
			repo = flag.String("repo-name", "", "Repository name")
			token = flag.String("token", "", "GitHub token")
			version = flag.Bool("version", false, "Print version information")
			workflowsPath = flag.String("workflows-path", ".github/workflows", "Path to workflow files")
			dryRun = flag.Bool("dry-run", false, "Show changes without applying them")
			stage = flag.Bool("stage", false, "Apply changes locally without creating a PR")

			// Set up environment
			os.Args = tt.args
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Set up mocks
			versionCheckerFactory = func(token string) updater.VersionChecker {
				return tt.versionChecker
			}
			prCreatorFactory = func(token, owner, repo string) updater.PRCreator {
				return tt.prCreator
			}

			// Parse flags
			if err := flag.CommandLine.Parse(tt.args[1:]); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Capture log.Fatal calls
			var logBuf strings.Builder
			log.SetOutput(&logBuf)
			defer log.SetOutput(os.Stderr)

			// Capture log.Fatal calls
			fatalCalled := false
			oldFatalln := fatalln
			defer func() { fatalln = oldFatalln }()
			fatalln = func(v ...interface{}) {
				fatalCalled = true
				panic(fmt.Sprint(v...)) // Use panic to stop execution like log.Fatal would
			}

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil || !fatalCalled {
						t.Error("Expected log.Fatal to be called")
					}
				}()
			} else {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Unexpected panic: %v", r)
					}
				}()
			}

			main()

			if tt.wantPanic && !fatalCalled {
				t.Error("Expected log.Fatal to be called")
			}
		})
	}
}

func TestMainFlags(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	// Create .github/workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows dir: %v", err)
	}

	// Create a test workflow file
	workflowContent := []byte(`name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`)

	if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), workflowContent, 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Create and change to a temporary working directory
	workingDir, err := os.MkdirTemp("", "test-working-dir")
	if err != nil {
		t.Fatalf("Failed to create working directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove working directory: %v", err)
		}
	}(workingDir)

	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("Failed to change to working directory: %v", err)
	}

	// Save original working directory to restore later
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	// Change to temp directory for test
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid flags",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			args: []string{
				"cmd",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-token=test-token",
			},
			wantErr: true,
		},
		{
			name: "token from env",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
			},
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-token",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			os.Args = tt.args
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				if err != nil {
					return
				}
				defer func(key string) {
					err := os.Unsetenv(key)
					if err != nil {
						t.Fatalf("Failed to unset environment variable %s: %v", key, err)
					}
				}(k)
			}

			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			repoPath = flag.String("repo", ".", "Path to the repository")
			owner = flag.String("owner", "", "Repository owner")
			repo = flag.String("repo-name", "", "Repository name")
			token = flag.String("token", "", "GitHub token")
			workflowsPath = flag.String("workflows-path", ".github/workflows", "Path to workflow files")
			dryRun = flag.Bool("dry-run", false, "Show changes without applying them")
			stage = flag.Bool("stage", false, "Apply changes locally without creating a PR")

			// Parse flags
			if err := flag.CommandLine.Parse(tt.args[1:]); err != nil {
				t.Fatalf("Failed to parse command line flags: %v", err)
			}
			err := validateFlags()
			if tt.wantErr {
				if err == nil {
					t.Error("validateFlags() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validateFlags() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCountUniqueFiles tests the countUniqueFiles function
func TestCountUniqueFiles(t *testing.T) {
	tests := []struct {
		name     string
		updates  []*updater.Update
		expected int
	}{
		{
			name:     "empty updates",
			updates:  []*updater.Update{},
			expected: 0,
		},
		{
			name: "single file",
			updates: []*updater.Update{
				{
					FilePath: "file1.yml",
				},
			},
			expected: 1,
		},
		{
			name: "multiple files with duplicates",
			updates: []*updater.Update{
				{
					FilePath: "file1.yml",
				},
				{
					FilePath: "file2.yml",
				},
				{
					FilePath: "file1.yml", // Duplicate
				},
				{
					FilePath: "file3.yml",
				},
			},
			expected: 3,
		},
		{
			name: "all duplicates",
			updates: []*updater.Update{
				{
					FilePath: "file1.yml",
				},
				{
					FilePath: "file1.yml",
				},
				{
					FilePath: "file1.yml",
				},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countUniqueFiles(tt.updates)
			if result != tt.expected {
				t.Errorf("countUniqueFiles() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestVersionChecker
func TestVersionChecker(t *testing.T) {
	checker := versionCheckerFactory(*token)
	if checker == nil {
		t.Error("versionCheckerFactory() returned nil")
	}
	absPath, err := absFunc(*repoPath)
	if err != nil {
		fmt.Printf("Expected fail to get absolute path: %v", err)
	}

	// Create update manager with repository root as base directory
	manager := updater.NewUpdateManager(absPath)
	if manager == nil {
		t.Error("NewUpdateManager() returned nil")
	}
	// Create PR creator using factory
	creator := prCreatorFactory(*token, *owner, *repo)
	if creator == nil {
		t.Error("prCreatorFactory() returned nil")
	}
}

// TestValidateFlags
func TestValidateFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid flags",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			wantErr: false,
		},
		{
			name: "missing owner",
			args: []string{
				"cmd",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			wantErr: true,
		},
		{
			name: "missing repo",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-token=test-token",
			},
			wantErr: true,
		},
		{
			name: "token from env",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
			},
			envVars: map[string]string{
				"GITHUB_TOKEN": "test-token",
			},
			wantErr: false,
		},
		{
			name: "version flag",
			args: []string{
				"cmd",
				"-version=true",
			},
			wantErr: true,
		},
		{
			name: "missing token",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=''", // Empty token
			},
			envVars: map[string]string{
				"GITHUB_TOKEN": "",
			},
		},
		{
			name: "custom workflows path",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
				"-workflows-path=custom/workflows",
			},
			wantErr: false,
		},
		{
			name: "workflows path from env",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
			},
			envVars: map[string]string{
				"WORKFLOWS_PATH": "custom/workflows",
			},
			wantErr: false,
		},
		{
			name: "dry run mode",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
				"-dry-run",
			},
			wantErr: false,
		},
		{
			name: "stage mode",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
				"-stage",
			},
			wantErr: false,
		},
		{
			name: "dry run and stage both set",
			args: []string{
				"cmd",
				"-owner=test-owner",
				"-repo-name=test-repo",
				"-token=test-token",
				"-dry-run",
				"-stage",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			os.Args = tt.args
			for k, v := range tt.envVars {
				err := os.Setenv(k, v)
				if err != nil {
					return
				}
				defer func(key string) {
					err := os.Unsetenv(key)
					if err != nil {
						t.Fatalf("Failed to unset environment variable %s: %v", key, err)
					}
				}(k)
			}

			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			repoPath = flag.String("repo", ".", "Path to the repository")
			owner = flag.String("owner", "", "Repository owner")
			repo = flag.String("repo-name", "", "Repository name")
			token = flag.String("token", "", "GitHub token")
			version = flag.Bool("version", false, "Print version information")
			workflowsPath = flag.String("workflows-path", ".github/workflows", "Path to workflow files")
			dryRun = flag.Bool("dry-run", false, "Show changes without applying them")
			stage = flag.Bool("stage", false, "Apply changes locally without creating a PR")

			// Parse flags
			if err := flag.CommandLine.Parse(tt.args[1:]); err != nil {
				t.Fatalf("Failed to parse command line flags: %v", err)
			}
			err := validateFlags()
			if tt.wantErr {
				if err == nil {
					t.Error("validateFlags() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validateFlags() unexpected error: %v", err)
				}
			}
		})
	}
}
