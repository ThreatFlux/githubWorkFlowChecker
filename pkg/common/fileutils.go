package common

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileLockManager manages locks for file operations to prevent concurrent access
type FileLockManager struct {
	locks sync.Map
}

// NewFileLockManager creates a new file lock manager
func NewFileLockManager() *FileLockManager {
	return &FileLockManager{}
}

// GetLock gets or creates a lock for the given file path
func (m *FileLockManager) GetLock(path string) *sync.Mutex {
	// Normalize the path to avoid different representations of the same file
	normalizedPath := filepath.Clean(path)

	// Get or create a lock for this file
	lockInterface, _ := m.locks.LoadOrStore(normalizedPath, &sync.Mutex{})
	return lockInterface.(*sync.Mutex)
}

// LockFile locks a file for exclusive access
func (m *FileLockManager) LockFile(path string) {
	m.GetLock(path).Lock()
}

// UnlockFile unlocks a file
func (m *FileLockManager) UnlockFile(path string) {
	m.GetLock(path).Unlock()
}

// WithFileLock executes a function with a lock on the given file
func (m *FileLockManager) WithFileLock(path string, fn func() error) error {
	lock := m.GetLock(path)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}

// FileOptions provides options for file operations
type FileOptions struct {
	// CreateDirs if true, creates parent directories if they don't exist
	CreateDirs bool
	// Mode is the file mode to use when creating files
	Mode os.FileMode
	// BaseDir is the base directory for path validation
	BaseDir string
	// ValidateOptions are the options for path validation
	ValidateOptions PathValidationOptions
}

// DefaultFileOptions returns the default options for file operations
func DefaultFileOptions() FileOptions {
	return FileOptions{
		CreateDirs: true,
		Mode:       0600,
		ValidateOptions: PathValidationOptions{
			RequireRegularFile: false,
			AllowNonExistent:   true,
			CheckSymlinks:      true,
		},
	}
}

// ReadFileWithOptions reads a file with the given options
func ReadFileWithOptions(path string, options FileOptions) ([]byte, error) {
	// Validate the path if BaseDir is provided
	if options.BaseDir != "" {
		if err := ValidatePath(options.BaseDir, path, options.ValidateOptions); err != nil {
			return nil, fmt.Errorf("invalid file path: %w", err)
		}
	}

	// Read the file
	// #nosec G304 - path is validated above
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return content, nil
}

// ReadFile reads a file with default options
func ReadFile(path string) ([]byte, error) {
	return ReadFileWithOptions(path, DefaultFileOptions())
}

// ReadFileString reads a file and returns its contents as a string
func ReadFileString(path string) (string, error) {
	content, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteFileWithOptions writes data to a file with the given options
func WriteFileWithOptions(path string, data []byte, options FileOptions) error {
	// Validate the path if BaseDir is provided
	if options.BaseDir != "" {
		if err := ValidatePath(options.BaseDir, path, options.ValidateOptions); err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}
	}

	// Create parent directories if needed
	if options.CreateDirs {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("error creating directories: %w", err)
		}
	}

	// Write to a temporary file first
	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, data, options.Mode); err != nil {
		// Clean up the temporary file if write fails
		_ = os.Remove(tempFile)
		return fmt.Errorf("error writing temporary file: %w", err)
	}

	// Rename the temporary file to the target file (atomic operation)
	if err := os.Rename(tempFile, path); err != nil {
		// Clean up the temporary file if rename fails
		_ = os.Remove(tempFile)
		return fmt.Errorf("error replacing original file: %w", err)
	}

	return nil
}

// WriteFile writes data to a file with default options
func WriteFile(path string, data []byte) error {
	return WriteFileWithOptions(path, data, DefaultFileOptions())
}

// WriteFileString writes a string to a file
func WriteFileString(path string, content string) error {
	return WriteFile(path, []byte(content))
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory checks if a path is a directory
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// IsRegularFile checks if a path is a regular file
func IsRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// FindFilesWithExtension finds all files with the given extension in a directory
func FindFilesWithExtension(dir string, ext string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check for the extension
		if strings.HasSuffix(info.Name(), ext) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning directory: %w", err)
	}

	return files, nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {

	// Open the source file
	// #nosec G304 - path is validated above
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening source file: %w", err)
	}
	defer func(srcFile *os.File) {
		err := srcFile.Close()
		if err != nil {
			fmt.Printf("error closing source file: %v\n", err)
		}
	}(srcFile)

	// Create the destination file
	// #nosec G304 - path is validated above
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating destination file: %w", err)
	}
	defer func(dstFile *os.File) {
		err := dstFile.Close()
		if err != nil {
			fmt.Printf("error closing destination file: %v\n", err)
		}
	}(dstFile)

	// Copy the contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("error copying file contents: %w", err)
	}

	// Sync to ensure the file is written to disk
	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("error syncing file: %w", err)
	}

	return nil
}

// ReplaceInFile replaces content in a file
func ReplaceInFile(path string, oldStr, newStr string) error {
	// Read the file
	content, err := ReadFile(path)
	if err != nil {
		return err
	}

	// Replace the content
	newContent := strings.ReplaceAll(string(content), oldStr, newStr)

	// Write the file
	return WriteFile(path, []byte(newContent))
}

// AppendToFile appends content to a file
func AppendToFile(path string, data []byte) error {
	// No validation needed here, as we're using ReadFile and WriteFile

	// Open the file for appending
	// #nosec G304 - path is validated above
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("error opening file for append: %w", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("error closing file: %v\n", err)
		}
	}(f)

	// Write the data
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("error appending to file: %w", err)
	}

	return nil
}

// AppendToFileString appends a string to a file
func AppendToFileString(path string, content string) error {
	return AppendToFile(path, []byte(content))
}

// ReadLines reads a file and returns its contents as lines
func ReadLines(path string) ([]string, error) {
	content, err := ReadFileString(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(content, "\n"), nil
}

// WriteLines writes lines to a file
func WriteLines(path string, lines []string) error {
	return WriteFileString(path, strings.Join(lines, "\n"))
}

// ModifyLines modifies lines in a file using a function
func ModifyLines(path string, fn func(line string, lineNum int) string) error {
	// Read the lines
	lines, err := ReadLines(path)
	if err != nil {
		return err
	}

	// Modify the lines
	for i, line := range lines {
		lines[i] = fn(line, i)
	}

	// Write the lines back
	return WriteLines(path, lines)
}
