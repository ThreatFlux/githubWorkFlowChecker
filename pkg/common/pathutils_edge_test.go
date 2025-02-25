package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePathEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-edge-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

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

	// Create a file with no read permissions
	noReadFile := filepath.Join(tempDir, "noread.txt")
	if err := os.WriteFile(noReadFile, []byte("test"), 0200); err != nil {
		t.Fatalf("Failed to create no-read file: %v", err)
	}

	// Create a symlink if the OS supports it
	symlink := filepath.Join(tempDir, "symlink")
	symlinkTarget := filepath.Join(tempDir, "test.txt")
	_ = os.Symlink(symlinkTarget, symlink) // Ignore error on Windows

	// Create a symlink outside the base directory if the OS supports it
	outsideSymlink := filepath.Join(tempDir, "outside-symlink")
	outsideTarget := filepath.Join(os.TempDir(), "outside.txt")
	if err := os.WriteFile(outsideTarget, []byte("outside"), 0600); err != nil {
		t.Fatalf("Failed to create outside target file: %v", err)
	}
	outsideSymErr := os.Symlink(outsideTarget, outsideSymlink)
	defer os.Remove(outsideTarget)

	// Create a broken symlink
	brokenSymlink := filepath.Join(tempDir, "broken-symlink")
	brokenTarget := filepath.Join(tempDir, "nonexistent.txt")
	brokenSymErr := os.Symlink(brokenTarget, brokenSymlink)

	// Create a recursive symlink
	recursiveSymlink := filepath.Join(tempDir, "recursive-symlink")
	recursiveSymErr := os.Symlink(recursiveSymlink, recursiveSymlink)

	tests := []struct {
		name        string
		baseDir     string
		path        string
		options     PathValidationOptions
		expectError bool
		skipOnError error // Skip this test if this error occurred during setup
	}{
		{
			name:    "Path exceeding maximum length",
			baseDir: tempDir,
			path:    filepath.Join(tempDir, strings.Repeat("a", MaxPathLength)),
			options: PathValidationOptions{
				MaxPathLength: MaxPathLength,
			},
			expectError: true,
		},
		{
			name:    "Path with custom maximum length",
			baseDir: tempDir,
			path:    filepath.Join(tempDir, "test.txt"),
			options: PathValidationOptions{
				MaxPathLength: 10, // Very short max length
			},
			expectError: true,
		},
		{
			name:        "Path with traversal attempt",
			baseDir:     tempDir,
			path:        filepath.Join(tempDir, "..", "outside.txt"),
			options:     DefaultPathValidationOptions(),
			expectError: true,
		},
		{
			name:        "Path with encoded traversal attempt",
			baseDir:     tempDir,
			path:        filepath.Join(tempDir, "subdir", "..", "..", "outside.txt"),
			options:     DefaultPathValidationOptions(),
			expectError: true,
		},
		{
			name:    "Path with insufficient permissions",
			baseDir: tempDir,
			path:    noReadFile,
			options: PathValidationOptions{
				RequireRegularFile: true,
				AllowNonExistent:   false,
			},
			expectError: false, // File exists and is a regular file, permissions don't affect validation
		},
		{
			name:    "Symlink outside base directory",
			baseDir: tempDir,
			path:    outsideSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: false, // Set to false because we want to test the symlink itself
			},
			expectError: true, // Should fail because the symlink points outside the base directory
			skipOnError: outsideSymErr,
		},
		{
			name:    "Broken symlink",
			baseDir: tempDir,
			path:    brokenSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: true,
				AllowNonExistent:   false,
			},
			expectError: true,
			skipOnError: brokenSymErr,
		},
		{
			name:    "Recursive symlink",
			baseDir: tempDir,
			path:    recursiveSymlink,
			options: PathValidationOptions{
				CheckSymlinks:      true,
				RequireRegularFile: true,
			},
			expectError: true,
			skipOnError: recursiveSymErr,
		},
		{
			name:    "Zero MaxPathLength",
			baseDir: tempDir,
			path:    testFile,
			options: PathValidationOptions{
				MaxPathLength: 0, // Should use default
			},
			expectError: false,
		},
		{
			name:    "Negative MaxPathLength",
			baseDir: tempDir,
			path:    testFile,
			options: PathValidationOptions{
				MaxPathLength: -1, // Should use default
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOnError != nil {
				t.Skipf("Skipping test due to setup error: %v", tc.skipOnError)
			}

			err := ValidatePath(tc.baseDir, tc.path, tc.options)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidatePathWithRelativePaths(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-relative-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save current working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// Change to the temporary directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temporary directory: %v", err)
	}

	// Create a subdirectory
	subDir := "subdir"
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(subDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		baseDir     string
		path        string
		options     PathValidationOptions
		expectError bool
	}{
		{
			name:        "Relative path within base directory",
			baseDir:     ".",
			path:        testFile,
			options:     DefaultPathValidationOptions(),
			expectError: false,
		},
		{
			name:        "Relative base directory",
			baseDir:     subDir,
			path:        testFile,
			options:     DefaultPathValidationOptions(),
			expectError: false,
		},
		{
			name:        "Relative path with traversal",
			baseDir:     subDir,
			path:        "../outside.txt",
			options:     DefaultPathValidationOptions(),
			expectError: true,
		},
		{
			name:        "Relative path with dot",
			baseDir:     ".",
			path:        "./subdir/test.txt",
			options:     DefaultPathValidationOptions(),
			expectError: false,
		},
		{
			name:        "Relative path with double dot",
			baseDir:     subDir,
			path:        "../subdir/test.txt",
			options:     DefaultPathValidationOptions(),
			expectError: true, // This is actually invalid because it goes outside the base directory
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

func TestJoinAndValidatePathEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-join-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	tests := []struct {
		name        string
		baseDir     string
		elements    []string
		expectError bool
	}{
		{
			name:        "Absolute path within base directory",
			baseDir:     tempDir,
			elements:    []string{subDir, "file.txt"},
			expectError: false,
		},
		{
			name:        "Absolute path outside base directory",
			baseDir:     subDir,
			elements:    []string{tempDir, "file.txt"},
			expectError: true,
		},
		{
			name:        "Path with encoded traversal",
			baseDir:     tempDir,
			elements:    []string{"subdir", "..", "..", "outside.txt"},
			expectError: true,
		},
		{
			name:        "Path with multiple elements and traversal",
			baseDir:     tempDir,
			elements:    []string{"subdir", "nested", "..", "..", "..", "outside.txt"},
			expectError: true,
		},
		{
			name:        "Very long path",
			baseDir:     tempDir,
			elements:    []string{strings.Repeat("a/", 1000) + "file.txt"},
			expectError: true, // This should exceed the maximum path length
		},
		{
			name:        "Path with empty element",
			baseDir:     tempDir,
			elements:    []string{"subdir", "", "file.txt"},
			expectError: false, // filepath.Join handles empty elements
		},
		{
			name:        "Path with null byte",
			baseDir:     tempDir,
			elements:    []string{"subdir", string('\x00'), "file.txt"},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := JoinAndValidatePath(tc.baseDir, tc.elements...)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSafeAbsEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-abs-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save current working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// Change to the temporary directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temporary directory: %v", err)
	}

	// Create a subdirectory
	subDir := "subdir"
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	tests := []struct {
		name        string
		baseDir     string
		path        string
		expectError bool
	}{
		{
			name:        "Relative path within base directory",
			baseDir:     ".",
			path:        subDir,
			expectError: false,
		},
		{
			name:        "Relative path with traversal",
			baseDir:     subDir,
			path:        "../outside.txt",
			expectError: true,
		},
		{
			name:        "Path with null byte",
			baseDir:     ".",
			path:        subDir + string('\x00'),
			expectError: true,
		},
		{
			name:        "Empty path",
			baseDir:     ".",
			path:        "",
			expectError: true,
		},
		{
			name:        "Path with encoded traversal",
			baseDir:     ".",
			path:        filepath.Join(subDir, "..", "..", "outside.txt"),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SafeAbs(tc.baseDir, tc.path)
			if tc.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
