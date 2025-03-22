package updater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScannerTestHelper(t *testing.T) {
	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Verify the helper was created properly
	assert.NotNil(t, helper.Env)
	assert.NotNil(t, helper.Scanner)
	assert.Equal(t, helper.Env.WorkDir, helper.Scanner.baseDir)
}

func TestCreateWorkflowFile(t *testing.T) {
	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Create a workflow file
	filename := "test-workflow.yml"
	content := "name: Test Workflow"
	permissions := os.FileMode(0600)

	filePath := helper.CreateWorkflowFile(filename, content, permissions)

	// Verify the file was created
	assert.FileExists(t, filePath)

	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(fileContent))

	// Check file permissions
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, permissions, info.Mode().Perm())
}

func TestCreateStandardWorkflow(t *testing.T) {
	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Create a standard workflow file
	filename := "standard-workflow.yml"
	actionRef := "actions/checkout@v3"

	filePath := helper.CreateStandardWorkflow(filename, actionRef)

	// Verify the file was created
	assert.FileExists(t, filePath)

	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(fileContent), "name: Test Workflow")
	assert.Contains(t, string(fileContent), "uses: actions/checkout@v3")

	// Check file permissions
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestCreateMultiActionWorkflow(t *testing.T) {
	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Create a workflow file with multiple actions
	filename := "multi-action-workflow.yml"
	actionRefs := []string{
		"actions/checkout@v3",
		"actions/setup-node@v2",
		"actions/cache@v2",
	}

	filePath := helper.CreateMultiActionWorkflow(filename, actionRefs)

	// Verify the file was created
	assert.FileExists(t, filePath)

	// Read the file content
	fileContent, err := os.ReadFile(filePath)
	require.NoError(t, err)

	// Check for each action reference
	for _, ref := range actionRefs {
		assert.Contains(t, string(fileContent), "- uses: "+ref)
	}

	// Check file permissions
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestSetupParseTestCase(t *testing.T) {
	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Create a standard workflow with a known action reference
	workflowFile := helper.CreateStandardWorkflow("test-workflow.yml", "actions/checkout@v2")

	// Use SetupParseTestCase with a string input (direct action reference)
	tc1 := testutils.TestCase{
		Name:  "Direct string input",
		Input: workflowFile, // Use the file path
	}

	refs1, err := helper.SetupParseTestCase(tc1)
	assert.NoError(t, err)
	assert.NotNil(t, refs1)
	assert.Len(t, refs1, 1)
	assert.Equal(t, "actions/checkout", refs1[0].Owner+"/"+refs1[0].Name)
	assert.Equal(t, "v2", refs1[0].Version)

	// Test with map input containing content
	tc2 := testutils.TestCase{
		Name: "Map with content",
		Input: map[string]interface{}{
			"content": `name: Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v2`,
			"filename": "workflow-map.yml",
		},
	}

	refs2, err := helper.SetupParseTestCase(tc2)
	assert.NoError(t, err)
	assert.NotNil(t, refs2)
	assert.Len(t, refs2, 2)
	assert.Equal(t, "actions/checkout", refs2[0].Owner+"/"+refs2[0].Name)
	assert.Equal(t, "v3", refs2[0].Version)
	assert.Equal(t, "actions/setup-node", refs2[1].Owner+"/"+refs2[1].Name)
	assert.Equal(t, "v2", refs2[1].Version)

	// Test with other input types (unsupported)
	tc3 := testutils.TestCase{
		Name:  "Invalid input type",
		Input: 123, // Not a supported input type
	}

	refs3, err := helper.SetupParseTestCase(tc3)
	assert.Nil(t, refs3)
	assert.Nil(t, err) // Should return nil, nil for unsupported types
}

func TestRunParseActionReferenceTest(t *testing.T) {
	// Create a scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Test the RunParseActionReferenceTest function with a valid action reference
	validRef := "actions/checkout@v3"
	validPath := "test/path/workflow.yml"
	validComments := []string{"# This is a comment"}

	// Create a test function to check the result
	validCheck := func(action *ActionReference, err error) {
		assert.NoError(t, err)
		assert.NotNil(t, action)
		assert.Equal(t, "actions", action.Owner)
		assert.Equal(t, "checkout", action.Name)
		assert.Equal(t, "v3", action.Version)
		assert.Equal(t, validPath, action.Path)
		assert.Equal(t, validComments, action.Comments)
	}

	// Run the test
	helper.RunParseActionReferenceTest(t, validRef, validPath, validComments, validCheck)

	// Test with an invalid reference
	invalidRef := "invalid-reference"
	invalidCheck := func(action *ActionReference, err error) {
		assert.Error(t, err)
		assert.Nil(t, action)
	}

	// Run the test with invalid reference
	helper.RunParseActionReferenceTest(t, invalidRef, validPath, validComments, invalidCheck)

	// Test with a reference with a commit hash
	commitRef := "actions/checkout@abcdef1234567890abcdef1234567890abcdef12"
	commitCheck := func(action *ActionReference, err error) {
		assert.NoError(t, err)
		assert.NotNil(t, action)
		assert.Equal(t, "actions", action.Owner)
		assert.Equal(t, "checkout", action.Name)
		assert.Equal(t, "abcdef1234567890abcdef1234567890abcdef12", action.Version)
		// Check if it's a commit hash by looking at the CommitHash field
		assert.NotEmpty(t, action.CommitHash)
	}

	// Run the test with commit reference
	helper.RunParseActionReferenceTest(t, commitRef, validPath, validComments, commitCheck)
}

func TestSetupInvalidYamlTest(t *testing.T) {
	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Setup an invalid YAML file
	filename := "invalid.yml"
	content := `invalid: yaml: content
  indentation: error
  - list
  not: formatted: properly
`
	filePath := helper.SetupInvalidYamlTest(filename, content, 0600)

	// Verify the file was created
	assert.FileExists(t, filePath)

	// Try to parse it with the scanner
	_, err := helper.Scanner.ParseActionReferences(filePath)
	assert.Error(t, err, "Parsing invalid YAML should fail")
}

func TestSetupEmptyWorkflows(t *testing.T) {
	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Setup empty workflows directory
	workflowsDir := helper.SetupEmptyWorkflows()

	// Verify directory was created
	assert.DirExists(t, workflowsDir)

	// Scan for workflows
	files, err := helper.Scanner.ScanWorkflows(workflowsDir)
	assert.NoError(t, err)
	assert.Empty(t, files, "Empty workflows directory should have no workflow files")
}

func TestCreateWorkflowWithPermissionError(t *testing.T) {
	// Skip on Windows as permissions work differently
	if os.PathSeparator == '\\' && os.PathListSeparator == ';' {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a new scanner test helper
	helper := NewScannerTestHelper(t)
	defer helper.Cleanup()

	// Create a workflow file with permission issues
	filename := "restricted.yml"
	filePath := helper.CreateWorkflowWithPermissionError(filename)

	// Verify the file was created
	assert.FileExists(t, filePath)

	// Check file permissions
	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0400), info.Mode().Perm(), "File should be read-only")
}

func TestSetupScanTestCase(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scan-test-*")
	require.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			require.NoError(t, err)
		}
	}(tempDir)

	// Create workflow files directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0750)
	require.NoError(t, err)

	// Create a test workflow file
	workflowPath := filepath.Join(workflowsDir, "test.yml")
	workflowContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
`
	err = os.WriteFile(workflowPath, []byte(workflowContent), 0600)
	require.NoError(t, err)

	// Create a scanner and scan for workflows
	scanner := NewScanner(tempDir)
	files, err := scanner.ScanWorkflows(workflowsDir)

	// Verify scan results
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files[0], "test.yml")

	// Test scanning an empty directory
	emptyDir, err := os.MkdirTemp("", "empty-dir-*")
	require.NoError(t, err)
	defer func(path string) {
		err := os.RemoveAll(path)
		require.NoError(t, err)
	}(emptyDir)

	// Create empty workflows directory
	emptyWorkflowsDir := filepath.Join(emptyDir, ".github", "workflows")
	err = os.MkdirAll(emptyWorkflowsDir, 0750)
	require.NoError(t, err)

	// Scan empty directory
	emptyScanner := NewScanner(emptyDir)
	emptyFiles, err := emptyScanner.ScanWorkflows(emptyWorkflowsDir)

	// Verify no files found
	assert.NoError(t, err)
	assert.Empty(t, emptyFiles, "No workflow files should be found")
}
