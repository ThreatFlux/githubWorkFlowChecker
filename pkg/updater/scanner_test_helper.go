package updater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common/testutils"
)

// ScannerTestHelper is a helper for scanner tests
type ScannerTestHelper struct {
	Env     *testutils.BaseTestEnvironment
	Scanner *Scanner
}

// NewScannerTestHelper creates a new scanner test helper
func NewScannerTestHelper(t *testing.T) *ScannerTestHelper {
	env := testutils.NewBaseTestEnvironment(t, "scanner-test-*")
	scanner := NewScanner(env.WorkDir)

	return &ScannerTestHelper{
		Env:     env,
		Scanner: scanner,
	}
}

// Cleanup cleans up temporary resources
func (h *ScannerTestHelper) Cleanup() {
	h.Env.Cleanup()
}

// CreateWorkflowFile creates a workflow file with the given content and permissions
func (h *ScannerTestHelper) CreateWorkflowFile(filename string, content string, permissions os.FileMode) string {
	workflowsDir := h.Env.CreateWorkflowsDir()
	testFile := filepath.Join(workflowsDir, filename)

	if err := os.WriteFile(testFile, []byte(content), permissions); err != nil {
		h.Env.T.Fatalf(common.ErrFailedToCreateTestFile, err)
	}

	return testFile
}

// CreateStandardWorkflow creates a workflow file with standard workflow content
// that includes the specified action reference
func (h *ScannerTestHelper) CreateStandardWorkflow(filename string, actionRef string) string {
	content := testutils.StandardWorkflowContent(actionRef)
	return h.CreateWorkflowFile(filename, content, 0600)
}

// CreateMultiActionWorkflow creates a workflow file with multiple action references
func (h *ScannerTestHelper) CreateMultiActionWorkflow(filename string, actionRefs []string) string {
	baseContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:`

	for _, ref := range actionRefs {
		baseContent += "\n      - uses: " + ref
	}

	return h.CreateWorkflowFile(filename, baseContent, 0600)
}

// SetupParseTestCase sets up a test case for testing action parsing
func (h *ScannerTestHelper) SetupParseTestCase(tc testutils.TestCase) ([]ActionReference, error) {
	switch input := tc.Input.(type) {
	case string:
		// Simple case - parse a single reference string
		refs, err := h.Scanner.ParseActionReferences(input)
		return refs, err
	case map[string]interface{}:
		// Create a workflow file and parse it
		if content, ok := input["content"].(string); ok {
			filename := "workflow.yml"
			if name, ok := input["filename"].(string); ok {
				filename = name
			}

			permissions := os.FileMode(0600)
			if perm, ok := input["permissions"].(os.FileMode); ok {
				permissions = perm
			}

			filePath := h.CreateWorkflowFile(filename, content, permissions)
			refs, err := h.Scanner.ParseActionReferences(filePath)
			return refs, err
		}
	case struct {
		Content     string
		Filename    string
		Permissions os.FileMode
	}:
		// Create a workflow file with the given parameters
		filePath := h.CreateWorkflowFile(input.Filename, input.Content, input.Permissions)
		refs, err := h.Scanner.ParseActionReferences(filePath)
		return refs, err
	}

	return nil, nil
}

// RunParseActionReferenceTest parses an action reference and validates the result
func (h *ScannerTestHelper) RunParseActionReferenceTest(t *testing.T, ref string, path string, comments []string, checkResult func(*ActionReference, error)) {
	action, err := parseActionReference(ref, path, comments)
	checkResult(action, err)
}

// SetupInvalidYamlTest creates an invalid YAML file for testing
func (h *ScannerTestHelper) SetupInvalidYamlTest(filename, content string, permissions os.FileMode) string {
	return h.CreateWorkflowFile(filename, content, permissions)
}

// SetupEmptyWorkflows creates an empty workflows directory
func (h *ScannerTestHelper) SetupEmptyWorkflows() string {
	return h.Env.CreateWorkflowsDir()
}

// CreateWorkflowWithPermissionError creates a workflow file with restrictive permissions
func (h *ScannerTestHelper) CreateWorkflowWithPermissionError(filename string) string {
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`

	// Create the file with normal permissions
	filePath := h.CreateWorkflowFile(filename, content, 0600)

	// Make the file read-only for current user, no permissions for group/others
	// This should be restrictive enough to cause permission errors without breaking security checks
	if err := os.Chmod(filePath, 0400); err != nil {
		h.Env.T.Fatalf("Failed to set restrictive permissions: %v", err)
	}

	return filePath
}

// SetupScanTestCase sets up a test case for scanner testing
func (h *ScannerTestHelper) SetupScanTestCase(setupFn func(string) error) ([]string, error) {
	workflowsDir := h.Env.GetWorkflowsPath()

	// Call the setup function if provided
	if setupFn != nil {
		if err := setupFn(workflowsDir); err != nil {
			h.Env.T.Fatalf(common.ErrFailedToSetupTest, err)
		}
	}

	// Scan for workflows
	return h.Scanner.ScanWorkflows(workflowsDir)
}
