package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockVersionChecker is a mock implementation of updater.VersionChecker
type MockVersionChecker struct {
	LatestVersion   string
	LatestHash      string
	UpdateAvailable bool
}

// GetLatestVersion implements updater.VersionChecker
func (m *MockVersionChecker) GetLatestVersion(_ context.Context, _ updater.ActionReference) (string, string, error) {
	return m.LatestVersion, m.LatestHash, nil
}

// IsUpdateAvailable implements updater.VersionChecker
func (m *MockVersionChecker) IsUpdateAvailable(_ context.Context, _ updater.ActionReference) (bool, string, string, error) {
	return m.UpdateAvailable, m.LatestVersion, m.LatestHash, nil
}

// GetCommitHash implements updater.VersionChecker
func (m *MockVersionChecker) GetCommitHash(_ context.Context, _ updater.ActionReference, _ string) (string, error) {
	return m.LatestHash, nil
}

// MockPRCreator is a mock implementation of updater.PRCreator
type MockPRCreator struct {
	Updates []*updater.Update
	Called  bool
}

// CreatePR implements updater.PRCreator
func (m *MockPRCreator) CreatePR(_ context.Context, updates []*updater.Update) error {
	m.Updates = updates
	m.Called = true
	return nil
}

func TestNewCommandTestHelper(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Assertions
	assert.NotEmpty(t, helper.TempDir)
	assert.DirExists(t, helper.TempDir)
	assert.Equal(t, t, helper.T)
	assert.NotNil(t, helper.OrigVersionFactory)
	assert.NotNil(t, helper.OrigPRFactory)
	assert.NotNil(t, helper.OrigAbsFunc)
	assert.NotEmpty(t, helper.OrigArgs)
	assert.NotEmpty(t, helper.OrigWorkdir)
}

func TestCleanup(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)

	// Store the tempdir for later checks
	tempDir := helper.TempDir
	assert.DirExists(t, tempDir)

	// Mock the factories and functions
	mockVersionChecker := &MockVersionChecker{}
	helper.SetupMockVersionChecker(mockVersionChecker)

	mockPRCreator := &MockPRCreator{}
	helper.SetupMockPRCreator(mockPRCreator)

	// Setup mock abs func
	originalPath := "/original/path"
	helper.SetupMockAbsFunc(func(path string) (string, error) {
		return originalPath, nil
	})

	// Setup custom args
	testArgs := []string{"test", "-flag", "value"}
	os.Args = testArgs

	// Now cleanup
	helper.Cleanup()

	// We can't directly compare function pointers, so we'll verify by checking implementation
	// Check if the original args were restored
	assert.Equal(t, helper.OrigArgs, os.Args)

	// The temp directory should be removed
	_, err := os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err), "Temp directory should be removed")
}

func TestSetupWorkflowsDir(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Create the workflows directory
	workflowsDir := helper.SetupWorkflowsDir()

	// Assert that the directory was created correctly
	expectedPath := filepath.Join(helper.TempDir, ".github", "workflows")
	assert.Equal(t, expectedPath, workflowsDir)
	assert.DirExists(t, workflowsDir)
}

func TestCreateWorkflowFile(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Test file content
	filename := "test-workflow.yml"
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
`

	// Create the workflow file
	filePath := helper.CreateWorkflowFile(filename, content)

	// Assert that the file was created correctly
	expectedPath := filepath.Join(helper.TempDir, ".github", "workflows", filename)
	assert.Equal(t, expectedPath, filePath)
	assert.FileExists(t, filePath)

	// Read the file content and verify
	fileContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(fileContent))

	// Check file permissions
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSetupCommandLine(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Setup command line with test args
	testArgs := []string{
		"ghactions-updater",
		"-repo", "/test/repo",
		"-owner", "testowner",
		"-repo-name", "testrepo",
		"-token", "testtoken",
	}
	helper.SetupCommandLine(testArgs)

	// Verify that the flags were parsed correctly
	assert.Equal(t, "/test/repo", *repoPath)
	assert.Equal(t, "testowner", *owner)
	assert.Equal(t, "testrepo", *repo)
	assert.Equal(t, "testtoken", *token)
	assert.Equal(t, ".github/workflows", *workflowsPath)
	assert.False(t, *dryRun)
	assert.False(t, *stage)
	assert.False(t, *version)

	// Test with dry-run option
	testArgs = []string{
		"ghactions-updater",
		"-dry-run",
	}
	helper.SetupCommandLine(testArgs)
	assert.True(t, *dryRun)

	// Test with stage option
	testArgs = []string{
		"ghactions-updater",
		"-stage",
	}
	helper.SetupCommandLine(testArgs)
	assert.True(t, *stage)
}

func TestSetupMockVersionChecker(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Create a mock version checker
	mockVersionChecker := &MockVersionChecker{
		LatestVersion:   "v2.0.0",
		LatestHash:      "abcdef1234567890",
		UpdateAvailable: true,
	}

	// Set it up
	helper.SetupMockVersionChecker(mockVersionChecker)

	// Verify that the factory was set up correctly
	checker := versionCheckerFactory("token")
	assert.Equal(t, mockVersionChecker, checker)
}

func TestSetupMockPRCreator(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Create a mock PR creator
	mockPRCreator := &MockPRCreator{}

	// Set it up
	helper.SetupMockPRCreator(mockPRCreator)

	// Verify that the factory was set up correctly
	creator := prCreatorFactory("token", "owner", "repo")
	assert.Equal(t, mockPRCreator, creator)
}

func TestSetupMockAbsFunc(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Create a mock Abs function
	testPath := "/test/abs/path"
	mockAbsFunc := func(path string) (string, error) {
		return testPath, nil
	}

	// Set it up
	helper.SetupMockAbsFunc(mockAbsFunc)

	// Verify that the function was set up correctly
	result, err := absFunc("any/path")
	assert.NoError(t, err)
	assert.Equal(t, testPath, result)
}

func TestSwitchToTempDir(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Get the original working directory
	origDir, err := os.Getwd()
	require.NoError(t, err)

	// Switch to the temp dir
	helper.SwitchToTempDir()

	// Get the current working directory
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	// On macOS, /var is a symlink to /private/var, so the paths might differ
	// Check if the current directory contains the temp dir path
	assert.Contains(t, currentDir, filepath.Base(helper.TempDir))
	assert.NotEqual(t, origDir, currentDir)

	// We can't effectively test the error path of SwitchToTempDir without
	// causing the test to terminate with t.Fatalf, but we've verified the happy path
	// which covers 50% of the function
}

// TestIntegration tests multiple helper functions together
func TestIntegration(t *testing.T) {
	// Create a helper
	helper := NewCommandTestHelper(t)
	defer helper.Cleanup()

	// Create a workflow file
	workflowContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v1
`
	filePath := helper.CreateWorkflowFile("test.yml", workflowContent)
	assert.FileExists(t, filePath)

	// Setup mock version checker
	mockVersionChecker := &MockVersionChecker{
		LatestVersion:   "v2.0.0",
		LatestHash:      "abcdef1234567890",
		UpdateAvailable: true,
	}
	helper.SetupMockVersionChecker(mockVersionChecker)

	// Setup mock PR creator
	mockPRCreator := &MockPRCreator{}
	helper.SetupMockPRCreator(mockPRCreator)

	// Setup command line
	helper.SetupCommandLine([]string{
		"ghactions-updater",
		"-repo", helper.TempDir,
		"-owner", "testowner",
		"-repo-name", "testrepo",
	})

	// Setup mock abs function to return the temp dir path
	helper.SetupMockAbsFunc(func(path string) (string, error) {
		return helper.TempDir, nil
	})

	// Switch to the temp dir
	helper.SwitchToTempDir()

	// Verify the current working directory
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	// On macOS, /var is a symlink to /private/var, so the paths might differ
	// Check if the current directory contains the temp dir path
	assert.Contains(t, currentDir, filepath.Base(helper.TempDir))
}
