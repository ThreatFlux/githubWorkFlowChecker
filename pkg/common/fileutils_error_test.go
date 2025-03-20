package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileWithOptionsErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with invalid path validation
	options := FileOptions{
		BaseDir: tempDir,
		ValidateOptions: PathValidationOptions{
			AllowNonExistent: false,
		},
	}

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	_, err = ReadFileWithOptions(nonExistentFile, options)
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}

	// Test with path outside base directory
	outsideFile := filepath.Join(os.TempDir(), "outside.txt")
	_, err = ReadFileWithOptions(outsideFile, options)
	if err == nil {
		t.Errorf("Expected error for path outside base directory, got nil")
	}

	// Test with a directory instead of a file
	_, err = ReadFileWithOptions(tempDir, options)
	if err == nil {
		t.Errorf("Expected error for directory instead of file, got nil")
	}

	// Test with a file with no read permissions
	noReadFile := filepath.Join(tempDir, "noread.txt")
	if err := os.WriteFile(noReadFile, []byte("test"), 0200); err != nil {
		t.Fatalf("Failed to create no-read file: %v", err)
	}

	// Skip this test on Windows where file permissions work differently
	if os.Getenv("SKIP_PERMISSION_TEST") != "1" {
		_, err = ReadFileWithOptions(noReadFile, options)
		if err == nil {
			t.Errorf("Expected error for file with no read permissions, got nil")
		}
	}
}

func TestReadFileStringErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	_, err = ReadFileString(nonExistentFile)
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestWriteFileWithOptionsErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with invalid path validation
	options := FileOptions{
		BaseDir: tempDir,
		ValidateOptions: PathValidationOptions{
			AllowNonExistent: true,
		},
	}

	// Test with path outside base directory
	outsideFile := filepath.Join(os.TempDir(), "outside.txt")
	err = WriteFileWithOptions(outsideFile, []byte("test"), options)
	if err == nil {
		t.Errorf("Expected error for path outside base directory, got nil")
	}

	// Test with CreateDirs=false and non-existent directory
	options.CreateDirs = false
	nestedFile := filepath.Join(tempDir, "nested", "dir", "test.txt")
	err = WriteFileWithOptions(nestedFile, []byte("test"), options)
	if err == nil {
		t.Errorf("Expected error for non-existent directory with CreateDirs=false, got nil")
	}

	// Test with a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0500); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Skip this test on Windows where file permissions work differently
	if os.Getenv("SKIP_PERMISSION_TEST") != "1" {
		readOnlyFile := filepath.Join(readOnlyDir, "test.txt")
		err = WriteFileWithOptions(readOnlyFile, []byte("test"), options)
		if err == nil {
			t.Errorf("Expected error for write to read-only directory, got nil")
		}
	}
}

func TestCopyFileErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with non-existent source file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	destFile := filepath.Join(tempDir, "dest.txt")
	err = CopyFile(nonExistentFile, destFile)
	if err == nil {
		t.Errorf("Expected error for non-existent source file, got nil")
	}

	// Create a source file
	srcFile := filepath.Join(tempDir, "src.txt")
	if err := os.WriteFile(srcFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Test with a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0500); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Skip this test on Windows where file permissions work differently
	if os.Getenv("SKIP_PERMISSION_TEST") != "1" {
		readOnlyFile := filepath.Join(readOnlyDir, "test.txt")
		err = CopyFile(srcFile, readOnlyFile)
		if err == nil {
			t.Errorf("Expected error for copy to read-only directory, got nil")
		}
	}
}

func TestAppendToFileErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0500); err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	// Skip this test on Windows where file permissions work differently
	if os.Getenv("SKIP_PERMISSION_TEST") != "1" {
		readOnlyFile := filepath.Join(readOnlyDir, "test.txt")
		err = AppendToFile(readOnlyFile, []byte("test"))
		if err == nil {
			t.Errorf("Expected error for append to read-only directory, got nil")
		}
	}

	// Test with a directory instead of a file
	err = AppendToFile(tempDir, []byte("test"))
	if err == nil {
		t.Errorf("Expected error for append to directory, got nil")
	}
}

func TestReadLinesErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	_, err = ReadLines(nonExistentFile)
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestFindFilesWithExtensionErrors(t *testing.T) {
	// Test with non-existent directory
	nonExistentDir := "/nonexistent/dir"
	_, err := FindFilesWithExtension(nonExistentDir, ".txt")
	if err == nil {
		t.Errorf("Expected error for non-existent directory, got nil")
	}

	// Note: FindFilesWithExtension doesn't check if the input is a directory,
	// it just skips files that aren't directories during the walk.
	// This is expected behavior, so we don't test for an error when passing a file.
}

func TestModifyLinesErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	err = ModifyLines(nonExistentFile, func(line string, lineNum int) string {
		return line
	})
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestReplaceInFileErrors(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	err = ReplaceInFile(nonExistentFile, "old", "new")
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}

	// Note: ReplaceInFile uses ReadFile and WriteFile internally,
	// which handle file permissions. The test for read-only files
	// is already covered by the WriteFile tests.
}
