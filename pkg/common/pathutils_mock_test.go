package common

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidatePathErrorCases tests error cases that are difficult to test with real files
func TestValidatePathErrorCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with a path that contains a null byte
	nullBytePath := "test\x00.txt"
	err = ValidatePath(tempDir, nullBytePath, DefaultPathValidationOptions())
	if err == nil {
		t.Errorf("Expected error for path with null byte, but got nil")
	}

	// Test with a base directory that contains a null byte
	nullByteBaseDir := "test\x00"
	err = ValidatePath(nullByteBaseDir, testFile, DefaultPathValidationOptions())
	if err == nil {
		t.Errorf("Expected error for base directory with null byte, but got nil")
	}

	// Test with a very long path
	veryLongPath := filepath.Join(tempDir, string(make([]byte, 1000)))
	err = ValidatePath(tempDir, veryLongPath, DefaultPathValidationOptions())
	if err == nil {
		t.Errorf("Expected error for very long path, but got nil")
	}

	// Test with a custom maximum path length
	err = ValidatePath(tempDir, testFile, PathValidationOptions{
		MaxPathLength: 10, // Very short max length
	})
	if err == nil {
		t.Errorf("Expected error for path exceeding custom maximum length, but got nil")
	}

	// Test with a path that is outside the base directory
	outsidePath := filepath.Join(os.TempDir(), "outside.txt")
	err = ValidatePath(tempDir, outsidePath, DefaultPathValidationOptions())
	if err == nil {
		t.Errorf("Expected error for path outside base directory, but got nil")
	}

	// Test with a path that contains traversal
	traversalPath := filepath.Join(tempDir, "..", "outside.txt")
	err = ValidatePath(tempDir, traversalPath, DefaultPathValidationOptions())
	if err == nil {
		t.Errorf("Expected error for path with traversal, but got nil")
	}

	// Test with a non-existent path and AllowNonExistent=false
	nonExistentPath := filepath.Join(tempDir, "nonexistent.txt")
	err = ValidatePath(tempDir, nonExistentPath, PathValidationOptions{
		AllowNonExistent: false,
	})
	if err == nil {
		t.Errorf("Expected error for non-existent path with AllowNonExistent=false, but got nil")
	}

	// Test with a directory and RequireRegularFile=true
	err = ValidatePath(tempDir, tempDir, PathValidationOptions{
		RequireRegularFile: true,
	})
	if err == nil {
		t.Errorf("Expected error for directory with RequireRegularFile=true, but got nil")
	}
}

// TestJoinAndValidatePathErrorCases tests error cases for JoinAndValidatePath
func TestJoinAndValidatePathErrorCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-join-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with empty elements
	_, err = JoinAndValidatePath(tempDir)
	if err == nil {
		t.Errorf("Expected error for empty elements, but got nil")
	}

	// Test with elements that result in a path outside the base directory
	_, err = JoinAndValidatePath(tempDir, "..", "outside.txt")
	if err == nil {
		t.Errorf("Expected error for path outside base directory, but got nil")
	}

	// Test with elements that contain null bytes
	_, err = JoinAndValidatePath(tempDir, "test\x00.txt")
	if err == nil {
		t.Errorf("Expected error for path with null byte, but got nil")
	}

	// Test with a very long path
	_, err = JoinAndValidatePath(tempDir, string(make([]byte, 1000)))
	if err == nil {
		t.Errorf("Expected error for very long path, but got nil")
	}

	// Test with an absolute path outside the base directory
	_, err = JoinAndValidatePath(tempDir, os.TempDir(), "outside.txt")
	if err == nil {
		t.Errorf("Expected error for absolute path outside base directory, but got nil")
	}
}

// TestSafeAbsErrorCases tests error cases for SafeAbs
func TestSafeAbsErrorCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "pathutils-abs-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with a path that contains a null byte
	_, err = SafeAbs(tempDir, "test\x00.txt")
	if err == nil {
		t.Errorf("Expected error for path with null byte, but got nil")
	}

	// Test with a path that is outside the base directory
	_, err = SafeAbs(tempDir, filepath.Join(os.TempDir(), "outside.txt"))
	if err == nil {
		t.Errorf("Expected error for path outside base directory, but got nil")
	}

	// Test with a path that contains traversal
	_, err = SafeAbs(tempDir, filepath.Join(tempDir, "..", "outside.txt"))
	if err == nil {
		t.Errorf("Expected error for path with traversal, but got nil")
	}

	// Test with an empty path
	_, err = SafeAbs(tempDir, "")
	if err == nil {
		t.Errorf("Expected error for empty path, but got nil")
	}
}
