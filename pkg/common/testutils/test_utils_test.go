package testutils

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupTestEnvironment(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Verify the test environment is properly set up
	assert.NotEmpty(t, env.WorkDir)
	assert.DirExists(t, env.WorkDir)
	assert.Equal(t, t, env.T)
	assert.NotNil(t, env.BaseTestEnvironment)
}

func TestCreateTestRepo(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Create a test repository
	repoPath := env.CreateTestRepo()

	// Verify the repository was created
	assert.DirExists(t, repoPath)

	// Check for .git directory
	gitDir := filepath.Join(repoPath, ".git")
	assert.DirExists(t, gitDir)

	// Check for workflow file
	workflowFile := filepath.Join(repoPath, ".github", "workflows", "test.yml")
	assert.FileExists(t, workflowFile)

	// Verify workflow content
	content, err := os.ReadFile(workflowFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "actions/checkout@v2")
}

func TestCloneTestRepo(t *testing.T) {
	env := SetupTestEnvironment(t)
	defer env.Cleanup()

	// Clone test repository
	clonedRepoPath := env.CloneTestRepo()

	// Verify the cloned repository was created
	assert.DirExists(t, clonedRepoPath)

	// Check for .git directory in cloned repo
	gitDir := filepath.Join(clonedRepoPath, ".git")
	assert.DirExists(t, gitDir)

	// Check for workflow file in cloned repo
	workflowFile := filepath.Join(clonedRepoPath, ".github", "workflows", "test.yml")
	assert.FileExists(t, workflowFile)
}

func TestMockPath(t *testing.T) {
	mockPath := NewMockPath()

	// Test default state
	_, err := mockPath.Resolve("/test/path")
	assert.Error(t, err)
	assert.Equal(t, "path does not exist", err.Error())

	err = mockPath.Access("/test/path")
	assert.Error(t, err)
	assert.Equal(t, "permission denied", err.Error())

	// Add a path
	testPath := "/test/path"
	resolvedPath := "/resolved/test/path"
	mockPath.AddPath(testPath, resolvedPath)

	// Test successful resolution
	result, err := mockPath.Resolve(testPath)
	assert.NoError(t, err)
	assert.Equal(t, resolvedPath, result)

	// Test successful access
	err = mockPath.Access(testPath)
	assert.NoError(t, err)

	// Set custom error messages
	customNonExistentMsg := "custom non-existent error"
	customAccessErrorMsg := "custom access error"

	mockPath.SetNonExistentMsg(customNonExistentMsg)
	mockPath.SetAccessErrorMsg(customAccessErrorMsg)

	// Test with custom error messages
	_, err = mockPath.Resolve("/nonexistent/path")
	assert.Error(t, err)
	assert.Equal(t, customNonExistentMsg, err.Error())

	err = mockPath.Access("/nonexistent/path")
	assert.Error(t, err)
	assert.Equal(t, customAccessErrorMsg, err.Error())
}

func TestMockFileContent(t *testing.T) {
	mockFiles := NewMockFileContent()

	// Test initial state - file not found
	_, err := mockFiles.ReadFile("/test/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")

	// Add a file
	testPath := "/test/file.txt"
	fileContent := "test content"
	mockFiles.AddFile(testPath, fileContent)

	// Test successful read
	content, err := mockFiles.ReadFile(testPath)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, content)

	// Test write
	newPath := "/test/newfile.txt"
	newContent := "new content"
	err = mockFiles.WriteFile(newPath, newContent)
	assert.NoError(t, err)

	// Verify write
	content, err = mockFiles.ReadFile(newPath)
	assert.NoError(t, err)
	assert.Equal(t, newContent, content)

	// Test access error
	accessErr := errors.New("access denied")
	mockFiles.SetAccessError(accessErr)

	_, err = mockFiles.ReadFile(testPath)
	assert.Error(t, err)
	assert.Equal(t, accessErr, err)

	// Test write error
	writeErr := errors.New("write failed")
	mockFiles.SetWriteError(writeErr)

	err = mockFiles.WriteFile("/test/write.txt", "content")
	assert.Error(t, err)
	assert.Equal(t, writeErr, err)
}

func TestMockVersionControl(t *testing.T) {
	mockVC := NewMockVersionControl()

	// Test initial state - unknown command
	_, err := mockVC.ExecuteCommand("git status")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")

	// Add command output
	gitStatus := "On branch main\nnothing to commit, working tree clean"
	mockVC.AddCommand("git status", gitStatus)

	// Test command execution
	output, err := mockVC.ExecuteCommand("git status")
	assert.NoError(t, err)
	assert.Equal(t, gitStatus, output)

	// Add command error
	cmdErr := errors.New("command failed")
	mockVC.AddCommandError("git push", cmdErr)

	// Test command error
	_, err = mockVC.ExecuteCommand("git push")
	assert.Error(t, err)
	assert.Equal(t, cmdErr, err)

	// Set default error
	defaultErr := errors.New("default error")
	mockVC.SetDefaultError(defaultErr)

	// Test default error
	_, err = mockVC.ExecuteCommand("unknown command")
	assert.Error(t, err)
	assert.Equal(t, defaultErr, err)
}

func TestCaptureOutput(t *testing.T) {
	// Define a test function that writes to stdout and stderr
	testFn := func() {
		os.Stdout.WriteString("stdout message\n")
		os.Stderr.WriteString("stderr message\n")
	}

	// Capture output
	stdout, stderr, err := CaptureOutput(testFn)
	assert.NoError(t, err)
	assert.Contains(t, stdout, "stdout message")
	assert.Contains(t, stderr, "stderr message")

	// Test with empty output
	emptyFn := func() {
		// Do nothing
	}

	stdout, stderr, err = CaptureOutput(emptyFn)
	assert.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Empty(t, stderr)
}

func TestMustWriteFile(t *testing.T) {
	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test writing a file
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("test content")

	MustWriteFile(t, testFile, testContent, 0600)

	// Verify file was written
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)

	// Check file permissions
	info, err := os.Stat(testFile)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestMustMkdirAll(t *testing.T) {
	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test creating nested directories
	testDir := filepath.Join(tempDir, "level1", "level2", "level3")

	MustMkdirAll(t, testDir, 0700)

	// Verify directories were created
	assert.DirExists(t, testDir)

	// Check directory permissions
	info, err := os.Stat(testDir)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestToJSON(t *testing.T) {
	// Test with a simple struct
	type testStruct struct {
		Name string
		Age  int
	}

	testObj := testStruct{Name: "Test", Age: 30}

	// Convert to JSON
	jsonStr := ToJSON(testObj)

	// Very basic verification
	assert.Contains(t, jsonStr, "Test")
	assert.Contains(t, jsonStr, "30")
}

func TestJoinStringSlice(t *testing.T) {
	// Test with a slice of strings
	testSlice := []string{"one", "two", "three"}

	// Join with comma
	result := JoinStringSlice(testSlice, ",")
	assert.Equal(t, "one,two,three", result)

	// Join with space
	result = JoinStringSlice(testSlice, " ")
	assert.Equal(t, "one two three", result)

	// Join with empty string
	result = JoinStringSlice(testSlice, "")
	assert.Equal(t, "onetwothree", result)

	// Join empty slice
	result = JoinStringSlice([]string{}, ",")
	assert.Equal(t, "", result)
}

// Mock HTTP writer for testing WriteHTTPJSON
type mockHTTPWriter struct {
	written []byte
	header  mockHTTPHeader
}

func (m *mockHTTPWriter) Write(b []byte) (int, error) {
	m.written = b
	return len(b), nil
}

func (m *mockHTTPWriter) Header() HTTPHeader {
	return &m.header
}

type mockHTTPHeader struct {
	headers map[string]string
}

func (m *mockHTTPHeader) Set(key, value string) {
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	m.headers[key] = value
}

func TestWriteHTTPJSON(t *testing.T) {
	// Create mock HTTP writer
	writer := &mockHTTPWriter{
		header: mockHTTPHeader{},
	}

	// Test writing JSON
	testJSON := `{"name":"Test","age":30}`

	err := WriteHTTPJSON(writer, testJSON)
	assert.NoError(t, err)

	// Verify content type header was set
	assert.Equal(t, "application/json", writer.header.headers["Content-Type"])

	// Verify content was written
	assert.Equal(t, testJSON, string(writer.written))
}
