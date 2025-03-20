package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Use error constants from the shared error_util.go file

// PathValidationTestCase defines a test case structure for ValidatePath tests
type PathValidationTestCase struct {
	name        string
	baseDir     string
	path        string
	options     PathValidationOptions
	expectError bool
	skipOnError error // Skip this test if this error occurred during setup
}

// JoinPathTestCase defines a test case structure for JoinAndValidatePath tests
type JoinPathTestCase struct {
	name        string
	baseDir     string
	elements    []string
	expectError bool
}

// SafeAbsTestCase defines a test case structure for SafeAbs tests
type SafeAbsTestCase struct {
	name        string
	baseDir     string
	path        string
	expectError bool
}

// setupTestDirectory creates a temporary directory for testing and returns its path.
// It fails the test if directory creation fails.
func setupTestDirectory(t *testing.T, prefix string) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf(ErrFailedToCreateTempDir, err)
	}
	return tempDir
}

// cleanupTestDirectory removes a temporary directory.
// It fails the test if directory removal fails.
func cleanupTestDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf(ErrFailedToRemoveTempDir, err)
	}
}

// createSubDirectory creates a subdirectory in the specified path.
// It fails the test if directory creation fails.
func createSubDirectory(t *testing.T, parent, name string) string {
	t.Helper()
	subDir := filepath.Join(parent, name)
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf(ErrFailedToCreateSubdir, err)
	}
	return subDir
}

// createTestFile creates a file with the specified content and permissions.
// It fails the test if file creation fails.
func createTestFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatalf(ErrFailedToCreateTestFile, err)
	}
}

// saveWorkingDirectory saves the current working directory and returns
// a function that restores it when called.
func saveWorkingDirectory(t *testing.T) func() {
	t.Helper()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf(ErrFailedToGetWorkingDir, err)
	}
	return func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Warning: Failed to restore working directory: %v", err)
		}
	}
}

func TestValidatePathEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := setupTestDirectory(t, "pathutils-edge-test")
	defer cleanupTestDirectory(t, tempDir)

	// Create a subdirectory
	createSubDirectory(t, tempDir, "subdir")

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	createTestFile(t, testFile, []byte("test"), 0600)

	// Create a file with no read permissions
	noReadFile := filepath.Join(tempDir, "noread.txt")
	createTestFile(t, noReadFile, []byte("test"), 0200)

	// Create a symlink if the OS supports it
	symlink := filepath.Join(tempDir, "symlink")
	symlinkTarget := filepath.Join(tempDir, "test.txt")
	_ = os.Symlink(symlinkTarget, symlink) // Ignore error on Windows

	// Create a symlink outside the base directory if the OS supports it
	outsideSymlink := filepath.Join(tempDir, "outside-symlink")
	outsideTarget := filepath.Join(os.TempDir(), "outside.txt")
	createTestFile(t, outsideTarget, []byte("outside"), 0600)
	outsideSymErr := os.Symlink(outsideTarget, outsideSymlink)
	defer func() {
		if err := os.Remove(outsideTarget); err != nil {
			t.Fatalf(ErrFailedToRemoveSymlink, err)
		}
	}()

	// Create a broken symlink
	brokenSymlink := filepath.Join(tempDir, "broken-symlink")
	brokenTarget := filepath.Join(tempDir, "nonexistent.txt")
	brokenSymErr := os.Symlink(brokenTarget, brokenSymlink)

	// Create a recursive symlink
	recursiveSymlink := filepath.Join(tempDir, "recursive-symlink")
	recursiveSymErr := os.Symlink(recursiveSymlink, recursiveSymlink)

	tests := []PathValidationTestCase{
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

	runValidatePathTests(t, tests)
}

// runValidatePathTests executes the validation test cases and checks the results.
func runValidatePathTests(t *testing.T, tests []PathValidationTestCase) {
	t.Helper()
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
	tempDir := setupTestDirectory(t, "pathutils-relative-test")
	defer cleanupTestDirectory(t, tempDir)

	// Save current working directory and restore it later
	restoreWd := saveWorkingDirectory(t)
	defer restoreWd()

	// Change to the temporary directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf(ErrFailedToChangeTempDir, err)
	}

	// Create a subdirectory
	subDir := "subdir"
	createSubDirectory(t, ".", subDir)

	// Create a test file
	testFile := filepath.Join(subDir, "test.txt")
	createTestFile(t, testFile, []byte("test"), 0600)

	tests := []PathValidationTestCase{
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

	runValidatePathTests(t, tests)
}

func TestJoinAndValidatePathEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := setupTestDirectory(t, "pathutils-join-test")
	defer cleanupTestDirectory(t, tempDir)

	// Create a subdirectory
	createSubDirectory(t, tempDir, "subdir")

	tests := []JoinPathTestCase{
		{
			name:        "Absolute path within base directory",
			baseDir:     tempDir,
			elements:    []string{filepath.Join(tempDir, "subdir"), "file.txt"},
			expectError: false,
		},
		{
			name:        "Absolute path outside base directory",
			baseDir:     filepath.Join(tempDir, "subdir"),
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

	runJoinPathTests(t, tests)
}

// runJoinPathTests executes the test cases for the JoinAndValidatePath function.
func runJoinPathTests(t *testing.T, tests []JoinPathTestCase) {
	t.Helper()
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
	tempDir := setupTestDirectory(t, "pathutils-abs-test")
	defer cleanupTestDirectory(t, tempDir)

	// Save current working directory and restore it later
	restoreWd := saveWorkingDirectory(t)
	defer restoreWd()

	// Change to the temporary directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf(ErrFailedToChangeTempDir, err)
	}

	// Create a subdirectory
	subDir := "subdir"
	createSubDirectory(t, ".", subDir)

	tests := []SafeAbsTestCase{
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

	runSafeAbsTests(t, tests)
}

// runSafeAbsTests executes the test cases for the SafeAbs function.
func runSafeAbsTests(t *testing.T, tests []SafeAbsTestCase) {
	t.Helper()
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
