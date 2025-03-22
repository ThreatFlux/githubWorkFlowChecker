package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
)

// We'll move these mock structs into specific test files to avoid conflicts
// This file will only contain helper functions for tests

// CommandTestHelper is a helper for command tests
type CommandTestHelper struct {
	TempDir            string
	T                  *testing.T
	OrigVersionFactory func(token string) updater.VersionChecker
	OrigPRFactory      func(token, owner, repo string) updater.PRCreator
	OrigAbsFunc        func(path string) (string, error)
	OrigArgs           []string
	OrigWorkdir        string
}

// NewCommandTestHelper creates a new command test helper
func NewCommandTestHelper(t *testing.T) *CommandTestHelper {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "command-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Save original factories and functions
	origVersionFactory := versionCheckerFactory
	origPRFactory := prCreatorFactory
	origAbsFunc := absFunc
	origArgs := os.Args

	// Save original working directory
	origWorkdir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	return &CommandTestHelper{
		TempDir:            tempDir,
		T:                  t,
		OrigVersionFactory: origVersionFactory,
		OrigPRFactory:      origPRFactory,
		OrigAbsFunc:        origAbsFunc,
		OrigArgs:           origArgs,
		OrigWorkdir:        origWorkdir,
	}
}

// Cleanup removes temporary resources and restores original state
func (h *CommandTestHelper) Cleanup() {
	// Restore original state
	versionCheckerFactory = h.OrigVersionFactory
	prCreatorFactory = h.OrigPRFactory
	absFunc = h.OrigAbsFunc
	os.Args = h.OrigArgs

	// Restore original working directory
	if err := os.Chdir(h.OrigWorkdir); err != nil {
		h.T.Errorf("Failed to restore working directory: %v", err)
	}

	// Remove temporary directory
	if h.TempDir != "" {
		if err := os.RemoveAll(h.TempDir); err != nil {
			h.T.Fatalf("Failed to remove temp directory: %v", err)
		}
	}
}

// SetupWorkflowsDir creates the .github/workflows directory structure
func (h *CommandTestHelper) SetupWorkflowsDir() string {
	workflowsDir := filepath.Join(h.TempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0750); err != nil {
		h.T.Fatalf("Failed to create workflows dir: %v", err)
	}
	return workflowsDir
}

// CreateWorkflowFile creates a workflow file with the given content
func (h *CommandTestHelper) CreateWorkflowFile(filename, content string) string {
	workflowsDir := h.SetupWorkflowsDir()
	filePath := filepath.Join(workflowsDir, filename)

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		h.T.Fatalf("Failed to create workflow file: %v", err)
	}

	return filePath
}

// SetupCommandLine sets up the command line flags and arguments
func (h *CommandTestHelper) SetupCommandLine(args []string) {
	// Reset the flag set
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Define flags
	repoPath = flag.String("repo", ".", "Path to the repository")
	owner = flag.String("owner", "", "Repository owner")
	repo = flag.String("repo-name", "", "Repository name")
	token = flag.String("token", "", "GitHub token")
	workflowsPath = flag.String("workflows-path", ".github/workflows", "Path to workflow files")
	dryRun = flag.Bool("dry-run", false, "Show changes without applying them")
	stage = flag.Bool("stage", false, "Apply changes locally without creating a PR")
	version = flag.Bool("version", false, "Print version information")

	// Set command line arguments
	os.Args = args

	// Parse flags
	if err := flag.CommandLine.Parse(args[1:]); err != nil {
		h.T.Fatalf("Failed to parse command line flags: %v", err)
	}
}

// SetupMockVersionChecker sets up a mock version checker factory
func (h *CommandTestHelper) SetupMockVersionChecker(checker updater.VersionChecker) {
	versionCheckerFactory = func(token string) updater.VersionChecker {
		return checker
	}
}

// SetupMockPRCreator sets up a mock PR creator factory
func (h *CommandTestHelper) SetupMockPRCreator(creator updater.PRCreator) {
	prCreatorFactory = func(token, owner, repo string) updater.PRCreator {
		return creator
	}
}

// SetupMockAbsFunc sets up a mock Abs function
func (h *CommandTestHelper) SetupMockAbsFunc(fn func(path string) (string, error)) {
	absFunc = fn
}

// SwitchToTempDir changes the working directory to the temporary directory
func (h *CommandTestHelper) SwitchToTempDir() {
	if err := os.Chdir(h.TempDir); err != nil {
		h.T.Fatalf("Failed to change to temp directory: %v", err)
	}
}

// These functions have been moved to test-specific implementations

// These test variables are now defined in run_errors_test.go

// This function has been moved to run_errors_test.go
