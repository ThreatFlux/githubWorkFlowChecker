package main

import (
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteFileError tests errors when writing files
func TestWriteFileError(t *testing.T) {
	// Set test mode flag to avoid Stdout.Sync errors
	inTestMode = true

	// Create a temporary directory
	testDir, err := os.MkdirTemp("", "test-file-write-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf(common.ErrFailedToRemoveTempDir, err)
		}
	}(testDir)

	// Create a read-only directory that will cause a permission error
	readOnlyDir := filepath.Join(testDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Make the directory read-only
	if err := os.Chmod(readOnlyDir, 0500); err != nil {
		t.Fatalf("Failed to set directory permissions: %v", err)
	}

	// Reset permissions on exit
	defer func(name string, mode os.FileMode) {
		err := os.Chmod(name, mode)
		if err != nil {
			t.Fatalf(common.ErrFailedToChangeFilePermissions, name)
		}
	}(readOnlyDir, 0755)

	// Run main with the read-only directory
	os.Args = []string{"cmd", readOnlyDir, "1"}

	exitCode, output := runWithExit(func() {
		main()
	})

	// Should fail due to permission error
	if exitCode == 0 {
		t.Errorf("Expected non-zero exit code for permission error, got 0")
	}

	// Check for error message
	if !strings.Contains(output, "error creating directories") &&
		!strings.Contains(output, "permission denied") {
		t.Errorf("Expected error message about permissions, got: %s", output)
	}
}
