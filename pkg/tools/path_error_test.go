package main

import (
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"os"
	"testing"
)

// TestPathValidationInMain tests that the main function handles path validation errors
func TestPathValidationInMain(t *testing.T) {
	// Set test mode flag to avoid Stdout.Sync errors
	inTestMode = true

	// Create a temporary directory
	testDir, err := os.MkdirTemp("", "test-path-invalid-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf(common.ErrFailedToRemoveTempDir, err)
		}
	}(testDir)

	// Create an invalid path by using a path that's too long for the second argument
	longName := string(make([]byte, 300))

	// Set command line arguments with the long path
	os.Args = []string{"cmd", testDir + "/" + longName, "1"}

	// Run main with the invalid path
	exitCode, output := runWithExit(func() {
		main()
	})

	// Should fail due to path validation error
	if exitCode == 0 {
		t.Errorf("Expected non-zero exit code for invalid path, got 0")
	}

	// Check for error message
	if !contains(output, "invalid directory path") {
		t.Errorf("Expected error message about invalid directory path, got: %s", output)
	}
}
