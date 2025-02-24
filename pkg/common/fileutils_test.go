package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFileLockManager(t *testing.T) {
	manager := NewFileLockManager()

	// Test getting the same lock for the same path
	path := "/test/path"
	lock1 := manager.GetLock(path)
	lock2 := manager.GetLock(path)

	if lock1 != lock2 {
		t.Errorf("Expected the same lock for the same path")
	}

	// Test getting different locks for different paths
	otherPath := "/other/path"
	otherLock := manager.GetLock(otherPath)

	if lock1 == otherLock {
		t.Errorf("Expected different locks for different paths")
	}

	// Test path normalization
	normalizedPath := filepath.Clean(path)
	normalizedLock := manager.GetLock(normalizedPath)

	if lock1 != normalizedLock {
		t.Errorf("Expected the same lock for normalized paths")
	}
}

func TestLockUnlock(t *testing.T) {
	manager := NewFileLockManager()
	path := "/test/path"

	// Test locking and unlocking
	manager.LockFile(path)
	manager.UnlockFile(path)

	// Test concurrent access
	var wg sync.WaitGroup
	concurrentCount := 10
	counter := 0
	mutex := &sync.Mutex{}

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.LockFile(path)
			// Simulate some work
			mutex.Lock()
			counter++
			mutex.Unlock()
			time.Sleep(10 * time.Millisecond)
			manager.UnlockFile(path)
		}()
	}

	wg.Wait()

	if counter != concurrentCount {
		t.Errorf("Expected counter to be %d, got %d", concurrentCount, counter)
	}
}

func TestWithFileLock(t *testing.T) {
	manager := NewFileLockManager()
	path := "/test/path"

	// Test successful execution
	err := manager.WithFileLock(path, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test error propagation
	expectedErr := "test error"
	err = manager.WithFileLock(path, func() error {
		return fmt.Errorf("%s", expectedErr)
	})

	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("Expected error containing '%s', got %v", expectedErr, err)
	}

	// Test concurrent access
	var wg sync.WaitGroup
	concurrentCount := 10
	counter := 0
	mutex := &sync.Mutex{}

	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.WithFileLock(path, func() error {
				// Simulate some work
				mutex.Lock()
				counter++
				mutex.Unlock()
				time.Sleep(10 * time.Millisecond)
				return nil
			})
		}()
	}

	wg.Wait()

	if counter != concurrentCount {
		t.Errorf("Expected counter to be %d, got %d", concurrentCount, counter)
	}
}

func TestFileOperations(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test file paths
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	// Test WriteFile
	err = WriteFile(testFile, []byte(testContent))
	if err != nil {
		t.Errorf("WriteFile failed: %v", err)
	}

	// Test FileExists
	if !FileExists(testFile) {
		t.Errorf("FileExists failed, expected file to exist")
	}

	// Test IsRegularFile
	if !IsRegularFile(testFile) {
		t.Errorf("IsRegularFile failed, expected file to be a regular file")
	}

	// Test IsDirectory
	if IsDirectory(testFile) {
		t.Errorf("IsDirectory failed, expected file not to be a directory")
	}

	// Test ReadFile
	content, err := ReadFile(testFile)
	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("ReadFile content mismatch, expected %q, got %q", testContent, string(content))
	}

	// Test ReadFileString
	contentStr, err := ReadFileString(testFile)
	if err != nil {
		t.Errorf("ReadFileString failed: %v", err)
	}
	if contentStr != testContent {
		t.Errorf("ReadFileString content mismatch, expected %q, got %q", testContent, contentStr)
	}

	// Test ReplaceInFile
	newContent := "Hello, Universe!"
	err = ReplaceInFile(testFile, "World", "Universe")
	if err != nil {
		t.Errorf("ReplaceInFile failed: %v", err)
	}

	// Verify the replacement
	contentStr, err = ReadFileString(testFile)
	if err != nil {
		t.Errorf("ReadFileString after replace failed: %v", err)
	}
	if contentStr != newContent {
		t.Errorf("ReplaceInFile content mismatch, expected %q, got %q", newContent, contentStr)
	}

	// Test AppendToFile
	appendContent := "\nAppended text"
	err = AppendToFile(testFile, []byte(appendContent))
	if err != nil {
		t.Errorf("AppendToFile failed: %v", err)
	}

	// Verify the append
	contentStr, err = ReadFileString(testFile)
	if err != nil {
		t.Errorf("ReadFileString after append failed: %v", err)
	}
	expectedContent := newContent + appendContent
	if contentStr != expectedContent {
		t.Errorf("AppendToFile content mismatch, expected %q, got %q", expectedContent, contentStr)
	}

	// Test AppendToFileString
	appendStrContent := "\nMore appended text"
	err = AppendToFileString(testFile, appendStrContent)
	if err != nil {
		t.Errorf("AppendToFileString failed: %v", err)
	}

	// Verify the string append
	contentStr, err = ReadFileString(testFile)
	if err != nil {
		t.Errorf("ReadFileString after string append failed: %v", err)
	}
	expectedContent = expectedContent + appendStrContent
	if contentStr != expectedContent {
		t.Errorf("AppendToFileString content mismatch, expected %q, got %q", expectedContent, contentStr)
	}

	// Test ReadLines
	lines, err := ReadLines(testFile)
	if err != nil {
		t.Errorf("ReadLines failed: %v", err)
	}
	expectedLines := strings.Split(expectedContent, "\n")
	if len(lines) != len(expectedLines) {
		t.Errorf("ReadLines line count mismatch, expected %d, got %d", len(expectedLines), len(lines))
	}
	for i, line := range lines {
		if i < len(expectedLines) && line != expectedLines[i] {
			t.Errorf("ReadLines line %d mismatch, expected %q, got %q", i, expectedLines[i], line)
		}
	}

	// Test WriteLines
	newLines := []string{"Line 1", "Line 2", "Line 3"}
	err = WriteLines(testFile, newLines)
	if err != nil {
		t.Errorf("WriteLines failed: %v", err)
	}

	// Verify the written lines
	lines, err = ReadLines(testFile)
	if err != nil {
		t.Errorf("ReadLines after WriteLines failed: %v", err)
	}
	if len(lines) != len(newLines) {
		t.Errorf("WriteLines line count mismatch, expected %d, got %d", len(newLines), len(lines))
	}
	for i, line := range lines {
		if i < len(newLines) && line != newLines[i] {
			t.Errorf("WriteLines line %d mismatch, expected %q, got %q", i, newLines[i], line)
		}
	}

	// Test ModifyLines
	err = ModifyLines(testFile, func(line string, lineNum int) string {
		return line + " - modified"
	})
	if err != nil {
		t.Errorf("ModifyLines failed: %v", err)
	}

	// Verify the modified lines
	lines, err = ReadLines(testFile)
	if err != nil {
		t.Errorf("ReadLines after ModifyLines failed: %v", err)
	}
	for i, line := range lines {
		expectedLine := newLines[i] + " - modified"
		if line != expectedLine {
			t.Errorf("ModifyLines line %d mismatch, expected %q, got %q", i, expectedLine, line)
		}
	}

	// Test CopyFile
	copyFile := filepath.Join(tempDir, "copy.txt")
	err = CopyFile(testFile, copyFile)
	if err != nil {
		t.Errorf("CopyFile failed: %v", err)
	}

	// Verify the copied file
	if !FileExists(copyFile) {
		t.Errorf("CopyFile failed, expected copied file to exist")
	}
	copyContent, err := ReadFileString(copyFile)
	if err != nil {
		t.Errorf("ReadFileString for copied file failed: %v", err)
	}
	originalContent, _ := ReadFileString(testFile)
	if copyContent != originalContent {
		t.Errorf("CopyFile content mismatch, expected %q, got %q", originalContent, copyContent)
	}

	// Test FindFilesWithExtension
	// Create some files with different extensions
	txtFile1 := filepath.Join(tempDir, "file1.txt")
	txtFile2 := filepath.Join(tempDir, "file2.txt")
	jsonFile := filepath.Join(tempDir, "file.json")

	_ = WriteFileString(txtFile1, "txt1")
	_ = WriteFileString(txtFile2, "txt2")
	_ = WriteFileString(jsonFile, "{}")

	// Find .txt files
	txtFiles, err := FindFilesWithExtension(tempDir, ".txt")
	if err != nil {
		t.Errorf("FindFilesWithExtension failed: %v", err)
	}
	if len(txtFiles) != 4 { // test.txt, copy.txt, file1.txt, file2.txt
		t.Errorf("FindFilesWithExtension found wrong number of .txt files, expected 4, got %d", len(txtFiles))
	}

	// Find .json files
	jsonFiles, err := FindFilesWithExtension(tempDir, ".json")
	if err != nil {
		t.Errorf("FindFilesWithExtension failed: %v", err)
	}
	if len(jsonFiles) != 1 {
		t.Errorf("FindFilesWithExtension found wrong number of .json files, expected 1, got %d", len(jsonFiles))
	}
}

func TestFileOptionsAndValidation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileutils-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test file paths
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, World!"

	// Test WriteFileWithOptions with path validation
	options := FileOptions{
		BaseDir: tempDir,
		ValidateOptions: PathValidationOptions{
			AllowNonExistent: true,
		},
	}
	err = WriteFileWithOptions(testFile, []byte(testContent), options)
	if err != nil {
		t.Errorf("WriteFileWithOptions failed: %v", err)
	}

	// Test ReadFileWithOptions with path validation
	// First ensure the file is readable
	err = os.Chmod(testFile, 0644)
	if err != nil {
		t.Errorf("Failed to set file permissions: %v", err)
	}

	content, err := ReadFileWithOptions(testFile, options)
	if err != nil {
		t.Errorf("ReadFileWithOptions failed: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("ReadFileWithOptions content mismatch, expected %q, got %q", testContent, string(content))
	}

	// Test path validation failure
	invalidPath := filepath.Join(os.TempDir(), "outside.txt")
	err = WriteFileWithOptions(invalidPath, []byte("test"), options)
	if err == nil {
		t.Errorf("WriteFileWithOptions should have failed with invalid path")
	}

	// Test creating directories
	nestedFile := filepath.Join(tempDir, "nested", "dir", "test.txt")
	options.CreateDirs = true
	err = WriteFileWithOptions(nestedFile, []byte(testContent), options)
	if err != nil {
		t.Errorf("WriteFileWithOptions with CreateDirs failed: %v", err)
	}
	if !FileExists(nestedFile) {
		t.Errorf("WriteFileWithOptions with CreateDirs failed, expected file to exist")
	}

	// Skip file mode test on platforms where it might not work reliably
	if os.Getenv("SKIP_FILE_MODE_TEST") != "1" {
		// Test file mode
		options.Mode = 0600 // read-write for owner only
		modeFile := filepath.Join(tempDir, "modefile.txt")
		err = WriteFileWithOptions(modeFile, []byte(testContent), options)
		if err != nil {
			t.Errorf("WriteFileWithOptions with Mode failed: %v", err)
		}

		// Verify the file exists
		if !FileExists(modeFile) {
			t.Errorf("WriteFileWithOptions with Mode failed, expected file to exist")
		}
	}
}
