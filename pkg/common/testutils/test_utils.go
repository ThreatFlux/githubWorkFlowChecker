package testutils

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

// TestEnvironment provides utilities for setting up and tearing down test environments
type TestEnvironment struct {
	*BaseTestEnvironment
}

// SetupTestEnvironment creates a new test environment
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	base := NewBaseTestEnvironment(t, "workflow-test-*")
	return &TestEnvironment{
		BaseTestEnvironment: base,
	}
}

// CreateTestRepo creates a test Git repository with default workflow files
func (env *TestEnvironment) CreateTestRepo() string {
	return CreateTestRepository(env.BaseTestEnvironment, "test-repo")
}

// CloneTestRepo clones the test repository into a new directory
func (env *TestEnvironment) CloneTestRepo() string {
	sourceRepo := env.CreateTestRepo()
	destRepo := filepath.Join(env.WorkDir, "cloned-repo")

	// Create clone directory
	if err := os.MkdirAll(destRepo, 0750); err != nil {
		env.T.Fatalf(common.ErrFailedToCreateRepoDir, err)
	}

	// Clone the repository
	cmd := env.CreateCommand("git", "clone", sourceRepo, destRepo)
	if err := cmd.Run(); err != nil {
		env.T.Fatalf(common.ErrFailedToCloneRepo, err)
	}

	return destRepo
}

// MockPath represents a mockable path utility for testing
type MockPath struct {
	Paths          map[string]string
	NonExistentMsg string
	AccessErrorMsg string
}

// NewMockPath creates a new MockPath with default options
func NewMockPath() *MockPath {
	return &MockPath{
		Paths:          make(map[string]string),
		NonExistentMsg: "path does not exist",
		AccessErrorMsg: "permission denied",
	}
}

// AddPath adds a path to the mock
func (m *MockPath) AddPath(path, resolvedPath string) {
	m.Paths[path] = resolvedPath
}

// Resolve resolves a path using the mock
func (m *MockPath) Resolve(path string) (string, error) {
	if resolved, ok := m.Paths[path]; ok {
		return resolved, nil
	}
	return "", errors.New(m.NonExistentMsg)
}

// Access checks if a path is accessible using the mock
func (m *MockPath) Access(path string) error {
	if _, ok := m.Paths[path]; ok {
		return nil
	}
	return errors.New(m.AccessErrorMsg)
}

// SetNonExistentMsg sets the error message for non-existent paths
func (m *MockPath) SetNonExistentMsg(msg string) {
	m.NonExistentMsg = msg
}

// SetAccessErrorMsg sets the error message for access errors
func (m *MockPath) SetAccessErrorMsg(msg string) {
	m.AccessErrorMsg = msg
}

// MockFileContent is a mock for file content operations
type MockFileContent struct {
	Files       map[string]string
	AccessError error
	WriteError  error
}

// NewMockFileContent creates a new MockFileContent
func NewMockFileContent() *MockFileContent {
	return &MockFileContent{
		Files: make(map[string]string),
	}
}

// AddFile adds a file to the mock
func (m *MockFileContent) AddFile(path, content string) {
	m.Files[path] = content
}

// ReadFile reads a file from the mock
func (m *MockFileContent) ReadFile(path string) (string, error) {
	if m.AccessError != nil {
		return "", m.AccessError
	}

	if content, ok := m.Files[path]; ok {
		return content, nil
	}

	return "", fmt.Errorf("file not found: %s", path)
}

// WriteFile writes to a file in the mock
func (m *MockFileContent) WriteFile(path, content string) error {
	if m.WriteError != nil {
		return m.WriteError
	}

	m.Files[path] = content
	return nil
}

// SetAccessError sets the error for file access operations
func (m *MockFileContent) SetAccessError(err error) {
	m.AccessError = err
}

// SetWriteError sets the error for file write operations
func (m *MockFileContent) SetWriteError(err error) {
	m.WriteError = err
}

// MockVersionControl provides mocks for version control operations
type MockVersionControl struct {
	Commands      map[string]string
	CommandErrors map[string]error
	DefaultError  error
}

// NewMockVersionControl creates a new MockVersionControl
func NewMockVersionControl() *MockVersionControl {
	return &MockVersionControl{
		Commands:      make(map[string]string),
		CommandErrors: make(map[string]error),
	}
}

// AddCommand adds a command output to the mock
func (m *MockVersionControl) AddCommand(command, output string) {
	m.Commands[command] = output
}

// AddCommandError adds an error for a specific command
func (m *MockVersionControl) AddCommandError(command string, err error) {
	m.CommandErrors[command] = err
}

// ExecuteCommand executes a command in the mock
func (m *MockVersionControl) ExecuteCommand(command string) (string, error) {
	if err, ok := m.CommandErrors[command]; ok && err != nil {
		return "", err
	}

	if output, ok := m.Commands[command]; ok {
		return output, nil
	}

	if m.DefaultError != nil {
		return "", m.DefaultError
	}

	return "", fmt.Errorf("unknown command: %s", command)
}

// SetDefaultError sets the default error for commands
func (m *MockVersionControl) SetDefaultError(err error) {
	m.DefaultError = err
}

// CaptureOutput captures stdout and stderr during a function execution
func CaptureOutput(fn func()) (stdout, stderr string, err error) {
	// Create temporary files for stdout and stderr
	stdoutFile, err := os.CreateTemp("", "stdout-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create stdout file: %w", err)
	}
	defer func(name string) {
		err := os.Remove(name)
		require.NoError(nil, err)
	}(stdoutFile.Name())
	defer func(stdoutFile *os.File) {
		err := stdoutFile.Close()
		require.NoError(nil, err)
	}(stdoutFile)

	stderrFile, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create stderr file: %w", err)
	}
	defer func(name string) {
		err := os.Remove(name)
		require.NoError(nil, err)
	}(stderrFile.Name())
	defer func(stderrFile *os.File) {
		err := stderrFile.Close()
		require.NoError(nil, err)
	}(stderrFile)

	// Save original stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Replace stdout and stderr with our temporary files
	os.Stdout = stdoutFile
	os.Stderr = stderrFile

	// Restore stdout and stderr when we're done
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Run the function
	fn()

	// Read captured output
	if _, err := stdoutFile.Seek(0, 0); err != nil {
		return "", "", fmt.Errorf("failed to seek stdout file: %w", err)
	}

	if _, err := stderrFile.Seek(0, 0); err != nil {
		return "", "", fmt.Errorf("failed to seek stderr file: %w", err)
	}

	stdoutBytes, err := os.ReadFile(stdoutFile.Name())
	if err != nil {
		return "", "", fmt.Errorf("failed to read stdout file: %w", err)
	}

	stderrBytes, err := os.ReadFile(stderrFile.Name())
	if err != nil {
		return "", "", fmt.Errorf("failed to read stderr file: %w", err)
	}

	return string(stdoutBytes), string(stderrBytes), nil
}

// MustWriteFile writes content to a file or fails the test
func MustWriteFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	err := os.WriteFile(path, content, perm)
	if err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

// MustMkdirAll creates a directory or fails the test
func MustMkdirAll(t *testing.T, path string, perm os.FileMode) {
	err := os.MkdirAll(path, perm)
	if err != nil {
		t.Fatalf("Failed to create directory %s: %v", path, err)
	}
}

// ToJSON converts an object to a JSON string
func ToJSON(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

// JoinStringSlice joins a slice of strings into a single string
func JoinStringSlice(slice []string, sep string) string {
	return strings.Join(slice, sep)
}

// HTTPWriter is an interface for writing to HTTP responses
type HTTPWriter interface {
	Write([]byte) (int, error)
	Header() HTTPHeader
}

// HTTPHeader is a type for HTTP headers
type HTTPHeader interface {
	Set(string, string)
}

// WriteHTTPJSON writes a JSON string to an HTTP response writer
func WriteHTTPJSON(w HTTPWriter, json string) error {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(json))
	return err
}
