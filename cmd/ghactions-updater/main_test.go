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
		name            string
		workflowContent string
		versionChecker  *mockVersionChecker
		prCreator       *mockPRCreator
		wantErr         bool
	}{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for test files
			tempDir, err := os.MkdirTemp("", "workflow-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create .github/workflows directory
			workflowsDir := filepath.Join(tempDir, ".github", "workflows")
			if err := os.MkdirAll(workflowsDir, 0755); err != nil {
				t.Fatalf("Failed to create workflows dir: %v", err)
			}

			// Create a test workflow file if content is provided
			if tt.workflowContent != "" {
				if err := os.WriteFile(filepath.Join(workflowsDir, "test.yml"), []byte(tt.workflowContent), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			}

			// Create and change to a temporary working directory
			workingDir, err := os.MkdirTemp("", "test-working-dir")
			if err != nil {
				t.Fatalf("Failed to create working directory: %v", err)
			}
			defer os.RemoveAll(workingDir)

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

			os.Args = []string{"cmd", "-owner=test-owner", "-repo-name=test-repo", "-token=test-token"}
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

func TestMain(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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
	defer os.RemoveAll(workingDir)

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
	defer os.RemoveAll(tempDir)

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
	defer os.RemoveAll(workingDir)

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
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			repoPath = flag.String("repo", ".", "Path to the repository")
			owner = flag.String("owner", "", "Repository owner")
			repo = flag.String("repo-name", "", "Repository name")
			token = flag.String("token", "", "GitHub token")

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
