package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSymlinkValidation(t *testing.T) {
	// Skip on platforms that don't support symlinks
	if !supportsSymlinks() {
		t.Skip("Skipping symlink tests on platform that doesn't support symlinks")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-symlink-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a file in the subdirectory
	subDirFile := filepath.Join(subDir, "subfile.txt")
	if err := os.WriteFile(subDirFile, []byte("subdir test"), 0600); err != nil {
		t.Fatalf("Failed to create file in subdirectory: %v", err)
	}

	// Create a directory outside the base directory
	outsideDir, err := os.MkdirTemp("", "pathutils-outside")
	if err != nil {
		t.Fatalf("Failed to create outside directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove outside directory: %v", err)
		}
	}(outsideDir)

	// Create a file in the outside directory
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside test"), 0600); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}

	// Create various symlinks for testing
	symlinks := createTestSymlinks(t, tempDir, subDir, testFile, outsideFile)

	// Test cases
	tests := []struct {
		name        string
		baseDir     string
		path        string
		options     PathValidationOptions
		expectError bool
	}{
		{
			name:    "Valid symlink within base directory",
			baseDir: tempDir,
			path:    symlinks.validSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: false,
		},
		{
			name:    "Valid symlink with RequireRegularFile=true",
			baseDir: tempDir,
			path:    symlinks.validSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: true,
			},
			expectError: false, // Should pass because it points to a regular file
		},
		{
			name:    "Symlink to directory with RequireRegularFile=true",
			baseDir: tempDir,
			path:    symlinks.dirSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: true,
			},
			expectError: true, // Should fail because it points to a directory, not a regular file
		},
		{
			name:    "Symlink to directory with RequireRegularFile=false",
			baseDir: tempDir,
			path:    symlinks.dirSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: false, // Should pass because RequireRegularFile is false
		},
		{
			name:    "Symlink outside base directory",
			baseDir: tempDir,
			path:    symlinks.outsideSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: true, // Should fail because it points outside the base directory
		},
		{
			name:    "Symlink outside base directory with CheckSymlinks=false",
			baseDir: tempDir,
			path:    symlinks.outsideSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      false,
				RequireRegularFile: false,
			},
			expectError: false, // Should pass because we're not checking symlinks
		},
		{
			name:    "Broken symlink with AllowNonExistent=false",
			baseDir: tempDir,
			path:    symlinks.brokenSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: true,
				AllowNonExistent:   false,
			},
			expectError: true, // Should fail because target doesn't exist and AllowNonExistent is false
		},
		{
			name:    "Broken symlink with AllowNonExistent=true",
			baseDir: tempDir,
			path:    symlinks.brokenSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
				AllowNonExistent:   true,
			},
			expectError: true, // Should still fail because EvalSymlinks will fail
		},
		{
			name:    "Recursive symlink",
			baseDir: tempDir,
			path:    symlinks.recursiveSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: true, // Should fail because of infinite recursion
		},
		{
			name:    "Nested symlink within base directory",
			baseDir: tempDir,
			path:    symlinks.nestedSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: false, // Should pass because it ultimately points within the base directory
		},
		{
			name:    "Nested symlink outside base directory",
			baseDir: tempDir,
			path:    symlinks.nestedOutsideSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: true, // Should fail because it ultimately points outside the base directory
		},
		{
			name:    "Relative symlink within base directory",
			baseDir: tempDir,
			path:    symlinks.relativeSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: false, // Should pass because it points within the base directory
		},
		{
			name:    "Relative symlink with traversal",
			baseDir: tempDir,
			path:    symlinks.relativeTraversalSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false,
			},
			expectError: true, // Should fail because it uses .. to go outside the base directory
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePath(tc.baseDir, tc.path, tc.options)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// Test structure to hold various symlink paths
type testSymlinks struct {
	validSymlink             string
	dirSymlink               string
	outsideSymlink           string
	brokenSymlink            string
	recursiveSymlink         string
	nestedSymlink            string
	nestedOutsideSymlink     string
	relativeSymlink          string
	relativeTraversalSymlink string
}

// Create various symlinks for testing
func createTestSymlinks(t *testing.T, tempDir, subDir, testFile, outsideFile string) testSymlinks {
	var symlinks testSymlinks

	// Valid symlink within base directory
	symlinks.validSymlink = filepath.Join(tempDir, "valid-symlink")
	if err := os.Symlink(testFile, symlinks.validSymlink); err != nil {
		t.Fatalf("Failed to create valid symlink: %v", err)
	}

	// Symlink to a directory
	symlinks.dirSymlink = filepath.Join(tempDir, "dir-symlink")
	if err := os.Symlink(subDir, symlinks.dirSymlink); err != nil {
		t.Fatalf("Failed to create directory symlink: %v", err)
	}

	// Symlink outside base directory
	symlinks.outsideSymlink = filepath.Join(tempDir, "outside-symlink")
	if err := os.Symlink(outsideFile, symlinks.outsideSymlink); err != nil {
		t.Fatalf("Failed to create outside symlink: %v", err)
	}

	// Broken symlink
	symlinks.brokenSymlink = filepath.Join(tempDir, "broken-symlink")
	if err := os.Symlink(filepath.Join(tempDir, "nonexistent.txt"), symlinks.brokenSymlink); err != nil {
		t.Fatalf("Failed to create broken symlink: %v", err)
	}

	// Recursive symlink
	symlinks.recursiveSymlink = filepath.Join(tempDir, "recursive-symlink")
	if err := os.Symlink(symlinks.recursiveSymlink, symlinks.recursiveSymlink); err != nil {
		t.Fatalf("Failed to create recursive symlink: %v", err)
	}

	// Nested symlink within base directory
	intermediateSymlink := filepath.Join(tempDir, "intermediate-symlink")
	if err := os.Symlink(testFile, intermediateSymlink); err != nil {
		t.Fatalf("Failed to create intermediate symlink: %v", err)
	}
	symlinks.nestedSymlink = filepath.Join(tempDir, "nested-symlink")
	if err := os.Symlink(intermediateSymlink, symlinks.nestedSymlink); err != nil {
		t.Fatalf("Failed to create nested symlink: %v", err)
	}

	// Nested symlink outside base directory
	intermediateOutsideSymlink := filepath.Join(tempDir, "intermediate-outside-symlink")
	if err := os.Symlink(outsideFile, intermediateOutsideSymlink); err != nil {
		t.Fatalf("Failed to create intermediate outside symlink: %v", err)
	}
	symlinks.nestedOutsideSymlink = filepath.Join(tempDir, "nested-outside-symlink")
	if err := os.Symlink(intermediateOutsideSymlink, symlinks.nestedOutsideSymlink); err != nil {
		t.Fatalf("Failed to create nested outside symlink: %v", err)
	}

	// Relative symlink within base directory
	symlinks.relativeSymlink = filepath.Join(tempDir, "relative-symlink")
	if err := os.Symlink("test.txt", symlinks.relativeSymlink); err != nil {
		t.Fatalf("Failed to create relative symlink: %v", err)
	}

	// Relative symlink with traversal
	symlinks.relativeTraversalSymlink = filepath.Join(tempDir, "relative-traversal-symlink")
	if err := os.Symlink("../outside.txt", symlinks.relativeTraversalSymlink); err != nil {
		t.Fatalf("Failed to create relative traversal symlink: %v", err)
	}

	return symlinks
}

// Check if the platform supports symlinks
func supportsSymlinks() bool {
	tempDir, err := os.MkdirTemp("", "symlink-test")
	if err != nil {
		return false
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			return
		}
	}(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return false
	}

	symlink := filepath.Join(tempDir, "symlink")
	err = os.Symlink(testFile, symlink)
	return err == nil
}
