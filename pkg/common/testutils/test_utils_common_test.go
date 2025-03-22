package testutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardWorkflowContent(t *testing.T) {
	// Test with different action references
	testCases := []struct {
		actionRef string
		expected  string
	}{
		{
			actionRef: "actions/checkout@v3",
			expected:  "      - uses: actions/checkout@v3",
		},
		{
			actionRef: "actions/setup-node@v2",
			expected:  "      - uses: actions/setup-node@v2",
		},
		{
			actionRef: "owner/action@hash",
			expected:  "      - uses: owner/action@hash",
		},
	}

	for _, tc := range testCases {
		content := StandardWorkflowContent(tc.actionRef)
		assert.Contains(t, content, tc.expected)
		assert.Contains(t, content, "name: Test Workflow")
		assert.Contains(t, content, "on: [push]")
		assert.Contains(t, content, "runs-on: ubuntu-latest")
	}
}

func TestExpectSubstring(t *testing.T) {
	// Use a custom testing.T to capture failure
	mockT := &testing.T{}

	// Test with substring present
	ExpectSubstring(mockT, "Hello World", "World", "Should contain World")
	assert.False(t, mockT.Failed())

	// Test with substring not present
	ExpectSubstring(mockT, "Hello World", "Universe", "Should contain Universe")
	assert.True(t, mockT.Failed())
}

func TestExpectNotSubstring(t *testing.T) {
	// Use a custom testing.T to capture failure
	mockT := &testing.T{}

	// Test with substring not present
	ExpectNotSubstring(mockT, "Hello World", "Universe", "Should not contain Universe")
	assert.False(t, mockT.Failed())

	// Test with substring present
	ExpectNotSubstring(mockT, "Hello World", "World", "Should not contain World")
	assert.True(t, mockT.Failed())
}

func TestAssertFileExists(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test-*")
	require.NoError(t, err)
	tempFile.Close()

	defer os.Remove(tempFile.Name())

	// Use a custom testing.T to capture failure
	mockT := &testing.T{}

	// Test with existing file
	AssertFileExists(mockT, tempFile.Name())
	assert.False(t, mockT.Failed())

	// Test with non-existent file
	AssertFileExists(mockT, tempFile.Name()+".nonexistent")
	assert.True(t, mockT.Failed())
}

func TestAssertFileNotExists(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test-*")
	require.NoError(t, err)
	tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Use a custom testing.T to capture failure
	mockT := &testing.T{}

	// Test with non-existent file
	AssertFileNotExists(mockT, tempFile.Name()+".nonexistent")
	assert.False(t, mockT.Failed())

	// Test with existing file
	AssertFileNotExists(mockT, tempFile.Name())
	assert.True(t, mockT.Failed())
}

func TestAssertFileContains(t *testing.T) {
	// Create a temporary file with content
	tempFile, err := os.CreateTemp("", "test-*")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	content := "test content for file contains"
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	tempFile.Close()

	// Use a custom testing.T to capture failure
	mockT := &testing.T{}

	// Test with content present
	AssertFileContains(mockT, tempFile.Name(), "test content")
	assert.False(t, mockT.Failed())

	// Reset mock
	mockT = &testing.T{}

	// Test with content not present
	AssertFileContains(mockT, tempFile.Name(), "missing content")
	assert.True(t, mockT.Failed())

	// Test with invalid file path
	mockT = &testing.T{}
	AssertFileContains(mockT, "../../../etc/passwd", "should not read this")
	assert.True(t, mockT.Failed())
}

func TestValidateFilePath(t *testing.T) {
	// Test valid paths
	validPaths := []string{
		"file.txt",
		"dir/file.txt",
		"/absolute/path/file.txt",
		"./relative/path/file.txt",
	}

	for _, path := range validPaths {
		assert.True(t, ValidateFilePath(path), "Path should be valid: %s", path)
	}

	// Test invalid paths with directory traversal
	invalidPaths := []string{
		"../file.txt",
		"../../file.txt",
		"dir/../../../etc/passwd",
		"./dir/../../file.txt",
	}

	for _, path := range invalidPaths {
		assert.False(t, ValidateFilePath(path), "Path should be invalid: %s", path)
	}
}

func TestAssertFileNotContains(t *testing.T) {
	// Create a temporary file with content
	tempFile, err := os.CreateTemp("", "test-*")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	content := "test content for file not contains"
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	tempFile.Close()

	// Use a custom testing.T to capture failure
	mockT := &testing.T{}

	// Test with content not present
	AssertFileNotContains(mockT, tempFile.Name(), "missing content")
	assert.False(t, mockT.Failed())

	// Reset mock
	mockT = &testing.T{}

	// Test with content present
	AssertFileNotContains(mockT, tempFile.Name(), "test content")
	assert.True(t, mockT.Failed())

	// Test with invalid file path
	mockT = &testing.T{}
	AssertFileNotContains(mockT, "../../../etc/passwd", "should not read this")
	assert.True(t, mockT.Failed())
}

func TestWaitForCondition(t *testing.T) {
	// Test condition that is immediately true
	result := WaitForCondition(func() bool {
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)
	assert.True(t, result, "Should succeed immediately")

	// Test condition that becomes true after a delay
	startTime := time.Now()
	delay := 50 * time.Millisecond
	result = WaitForCondition(func() bool {
		return time.Since(startTime) >= delay
	}, 200*time.Millisecond, 10*time.Millisecond)
	assert.True(t, result, "Should succeed after delay")

	// Test condition that times out
	result = WaitForCondition(func() bool {
		return false
	}, 50*time.Millisecond, 10*time.Millisecond)
	assert.False(t, result, "Should timeout")
}

func TestContainsComparable(t *testing.T) {
	// Test with integers
	intSlice := []int{1, 2, 3, 4, 5}
	assert.True(t, ContainsComparable(intSlice, 3))
	assert.False(t, ContainsComparable(intSlice, 6))

	// Test with strings
	strSlice := []string{"one", "two", "three"}
	assert.True(t, ContainsComparable(strSlice, "two"))
	assert.False(t, ContainsComparable(strSlice, "four"))

	// Test with empty slice
	var emptySlice []int
	assert.False(t, ContainsComparable(emptySlice, 1))
}

func TestCreateTestRepository(t *testing.T) {
	// Create a base test environment
	baseEnv := NewBaseTestEnvironment(t, "test-repo-*")
	defer baseEnv.Cleanup()

	// Create a test repository
	repoPath := CreateTestRepository(baseEnv, "test-repo")

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

	// Test with unsupported environment type
	assert.Panics(t, func() {
		CreateTestRepository("unsupported", "test-repo")
	})
}
