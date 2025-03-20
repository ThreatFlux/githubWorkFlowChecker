package updater

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

func TestParseActionReferencesInvalidSyntax(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErrMsg  string
		permissions os.FileMode
	}{
		{
			name: "invalid yaml syntax - missing colon",
			content: `name Test Workflow
on: [push]
jobs:
  test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			wantErrMsg:  "error parsing workflow YAML",
			permissions: 0600,
		},
		{
			name: "invalid yaml syntax - incorrect indentation",
			content: `name: Test Workflow
on: [push]
jobs:
test:
  runs-on: ubuntu-latest
   steps:
    - uses: actions/checkout@v2`,
			wantErrMsg:  "error parsing workflow YAML",
			permissions: 0600,
		},
		{
			name: "malformed action reference - missing @",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkoutv2`,
			wantErrMsg:  "invalid action reference format",
			permissions: 0600,
		},
		{
			name: "malformed action reference - missing owner",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: checkout@v2`,
			wantErrMsg:  "invalid action name format",
			permissions: 0600,
		},
		{
			name:        "empty yaml document",
			content:     "",
			wantErrMsg:  "empty YAML document",
			permissions: 0600,
		},
		{
			name: "invalid yaml syntax - unmatched quotes",
			content: `name: "Test Workflow
on: [push]
jobs:
  test:
    runs-on: 'ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			wantErrMsg:  "error parsing workflow YAML",
			permissions: 0600,
		},
		{
			name: "permission error",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
			wantErrMsg:  "permission denied",
			permissions: 0000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "workflow-test")
			if err != nil {
				t.Fatalf(common.ErrFailedToCreateTempDir, err)
			}
			defer func(path string) {
				err := os.RemoveAll(path)
				if err != nil {
					t.Fatalf(common.ErrFailedToRemoveTempDir, err)
				}
			}(tempDir)

			// Set secure permissions on temp directory
			if err := os.Chmod(tempDir, 0750); err != nil {
				t.Fatalf(common.ErrFailedToSetTempDirPermissions, err)
			}

			// Create scanner with temp directory as base
			scanner := NewScanner(tempDir)

			// Create test file
			testFile := filepath.Join(tempDir, "workflow.yml")
			err = os.WriteFile(testFile, []byte(tt.content), tt.permissions)
			if err != nil {
				t.Fatalf(common.ErrFailedToCreateTestFile, err)
			}

			// Parse action references
			_, err = scanner.ParseActionReferences(testFile)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf(common.ErrExpectedErrorContaining, tt.wantErrMsg, err.Error())
			}
		})
	}
}

func TestScanWorkflowsErrors(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(string) error
		wantErrMsg string
	}{
		{
			name: "non-existent directory",
			setup: func(dir string) error {
				return nil // Don't create the directory
			},
			wantErrMsg: "workflows directory not found",
		},
		{
			name: "permission denied",
			setup: func(dir string) error {
				if err := os.MkdirAll(dir, 0750); err != nil {
					return err
				}
				return os.Chmod(dir, 0000)
			},
			wantErrMsg: "permission denied",
		},
		{
			name: "invalid workflow file",
			setup: func(dir string) error {
				if err := os.MkdirAll(dir, 0750); err != nil {
					return err
				}
				// Create a file with invalid permissions for reading
				filePath := filepath.Join(dir, "workflow.yml")
				if err := os.WriteFile(filePath, []byte("invalid: yaml: content"), 0600); err != nil {
					return err
				}
				return os.Chmod(filePath, 0000)
			},
			wantErrMsg: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "workflow-test")
			if err != nil {
				t.Fatalf(common.ErrFailedToCreateTempDir, err)
			}
			defer func(path string) {
				err := os.RemoveAll(path)
				if err != nil {
					t.Fatalf(common.ErrFailedToRemoveTempDir, err)
				}
			}(tempDir)

			// Set secure permissions on temp directory
			if err := os.Chmod(tempDir, 0750); err != nil {
				t.Fatalf(common.ErrFailedToSetTempDirPermissions, err)
			}

			workflowsDir := filepath.Join(tempDir, ".github", "workflows")

			// Set up test case
			if err := tt.setup(workflowsDir); err != nil {
				t.Fatalf(common.ErrFailedToSetupTest, err)
			}

			// Create scanner with temp directory as base
			scanner := NewScanner(tempDir)
			_, err = scanner.ScanWorkflows(workflowsDir)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf(common.ErrExpectedErrorContaining, tt.wantErrMsg, err.Error())
			}
		})
	}
}

func TestParseActionReferenceErrors(t *testing.T) {
	tests := []struct {
		name       string
		ref        string
		path       string
		comments   []string
		wantErrMsg string
	}{
		{
			name:       "missing @ symbol",
			ref:        "actions/checkoutv2",
			path:       "workflow.yml",
			comments:   nil,
			wantErrMsg: "invalid action reference format",
		},
		{
			name:       "missing owner",
			ref:        "checkout@v2",
			path:       "workflow.yml",
			comments:   nil,
			wantErrMsg: "invalid action name format",
		},
		{
			name:       "empty reference",
			ref:        "",
			path:       "workflow.yml",
			comments:   nil,
			wantErrMsg: "invalid action reference format",
		},
		{
			name:       "missing version",
			ref:        "actions/checkout@",
			path:       "workflow.yml",
			comments:   nil,
			wantErrMsg: "invalid action reference format: actions/checkout@",
		},
		// Removed "too many parts" test case since we now support multi-part action names
		// like github/codeql-action/init
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseActionReference(tt.ref, tt.path, tt.comments)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf(common.ErrExpectedErrorContaining, tt.wantErrMsg, err.Error())
			}
		})
	}
}

func TestParseActionReferenceSuccess(t *testing.T) {
	tests := []struct {
		name           string
		ref            string
		path           string
		comments       []string
		expectedOwner  string
		expectedName   string
		expectedVer    string
		expectedCommit string
	}{
		{
			name:           "standard version reference",
			ref:            "actions/checkout@v2",
			path:           "workflow.yml",
			comments:       nil,
			expectedOwner:  "actions",
			expectedName:   "checkout",
			expectedVer:    "v2",
			expectedCommit: "",
		},
		{
			name:           "commit hash reference",
			ref:            "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675",
			path:           "workflow.yml",
			comments:       nil,
			expectedOwner:  "actions",
			expectedName:   "checkout",
			expectedVer:    "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			expectedCommit: "a81bbbf8298c0fa03ea29cdc473d45769f953675",
		},
		{
			name:           "commit hash with version comment",
			ref:            "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675",
			path:           "workflow.yml",
			comments:       []string{"# Comment 1", "# Original version: v2", "# Comment 2"},
			expectedOwner:  "actions",
			expectedName:   "checkout",
			expectedVer:    "v2",
			expectedCommit: "a81bbbf8298c0fa03ea29cdc473d45769f953675",
		},
		{
			name:           "version with comments",
			ref:            "actions/setup-node@v3",
			path:           "workflow.yml",
			comments:       []string{"# Using Node.js v16", "# Latest version"},
			expectedOwner:  "actions",
			expectedName:   "setup-node",
			expectedVer:    "v3",
			expectedCommit: "",
		},
		{
			name:           "short commit hash",
			ref:            "actions/checkout@a81bbbf",
			path:           "workflow.yml",
			comments:       nil,
			expectedOwner:  "actions",
			expectedName:   "checkout",
			expectedVer:    "a81bbbf",
			expectedCommit: "",
		},
		{
			name:           "version with patch",
			ref:            "actions/setup-python@v3.10.4",
			path:           "workflow.yml",
			comments:       nil,
			expectedOwner:  "actions",
			expectedName:   "setup-python",
			expectedVer:    "v3.10.4",
			expectedCommit: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := parseActionReference(tt.ref, tt.path, tt.comments)
			if err != nil {
				t.Errorf(common.ErrUnexpectedError, err)
				return
			}

			if action.Owner != tt.expectedOwner {
				t.Errorf(common.ErrExpectedResult, tt.expectedOwner, action.Owner)
			}
			if action.Name != tt.expectedName {
				t.Errorf(common.ErrExpectedResult, tt.expectedName, action.Name)
			}
			if action.Version != tt.expectedVer {
				t.Errorf(common.ErrExpectedResult, tt.expectedVer, action.Version)
			}
			if action.CommitHash != tt.expectedCommit {
				t.Errorf(common.ErrExpectedResult, tt.expectedCommit, action.CommitHash)
			}
			if action.Path != tt.path {
				t.Errorf(common.ErrExpectedResult, tt.path, action.Path)
			}
			if len(action.Comments) != len(tt.comments) {
				t.Errorf(common.ErrExpectedResult, len(tt.comments), len(action.Comments))
			}
		})
	}
}

func TestParseActionReferencesSuccess(t *testing.T) {
	// Create a valid workflow file with various action references
	workflowContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      # Standard version reference
      - uses: actions/checkout@v2
      
      # Original version: v3
      - uses: actions/setup-node@a81bbbf8298c0fa03ea29cdc473d45769f953675
      
      # Matrix expression
      - uses: ${{ matrix.action }}@${{ matrix.version }}
      
      # Run step (should be ignored)
      - run: |
          echo "This is a run step with actions/checkout@v2 in the text"
          
      # Nested action in a job
      - name: Nested job
        uses: actions/setup-python@v3.10.4
`

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf(common.ErrFailedToCreateTempDir, err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf(common.ErrFailedToRemoveTempDir, err)
		}
	}(tempDir)

	// Set secure permissions on temp directory
	if err := os.Chmod(tempDir, 0750); err != nil {
		t.Fatalf(common.ErrFailedToSetTempDirPermissions, err)
	}

	// Create scanner with temp directory as base
	scanner := NewScanner(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "workflow.yml")
	err = os.WriteFile(testFile, []byte(workflowContent), 0600)
	if err != nil {
		t.Fatalf(common.ErrFailedToCreateTestFile, err)
	}

	// Parse action references
	actions, err := scanner.ParseActionReferences(testFile)
	if err != nil {
		t.Fatalf(common.ErrUnexpectedError, err)
	}

	// Check the number of actions found
	expectedCount := 4 // 3 regular actions + 1 matrix action
	if len(actions) != expectedCount {
		t.Errorf(common.ErrExpectedActions, expectedCount, len(actions))
	}

	// Check specific actions
	for _, action := range actions {
		switch {
		case action.Owner == "actions" && action.Name == "checkout" && action.Version == "v2":
			// Standard version reference
			if action.CommitHash != "" {
				t.Errorf(common.ErrExpectedEmptyCommitHash, "checkout@v2", action.CommitHash)
			}
		case action.Owner == "actions" && action.Name == "setup-node":
			// Commit hash reference with original version comment
			if action.CommitHash != "a81bbbf8298c0fa03ea29cdc473d45769f953675" {
				t.Errorf(common.ErrExpectedCommitHash, "a81bbbf8298c0fa03ea29cdc473d45769f953675", action.CommitHash)
			}
			if action.Version != "v3" {
				t.Errorf(common.ErrExpectedVersionFromComment, "v3", action.Version)
			}
		case action.Owner == "matrix" && action.Name == "action" && action.Version == "dynamic":
			// Matrix expression
			// This is handled correctly
		case action.Owner == "actions" && action.Name == "setup-python" && action.Version == "v3.10.4":
			// Nested action
			if action.CommitHash != "" {
				t.Errorf(common.ErrExpectedEmptyCommitHash, "setup-python@v3.10.4", action.CommitHash)
			}
		default:
			t.Errorf(common.ErrUnexpectedActionFound, action.Owner, action.Name, action.Version)
		}
	}
}

func TestScanWorkflowsSuccess(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf(common.ErrFailedToCreateTempDir, err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf(common.ErrFailedToRemoveTempDir, err)
		}
	}(tempDir)

	// Set secure permissions on temp directory
	if err := os.Chmod(tempDir, 0750); err != nil {
		t.Fatalf(common.ErrFailedToSetTempDirPermissions, err)
	}

	// Create workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0750); err != nil {
		t.Fatalf(common.ErrFailedToCreateWorkflowsDir, err)
	}

	// Create test workflow files
	files := []struct {
		name    string
		content string
	}{
		{
			name: "workflow1.yml",
			content: `name: Workflow 1
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
		},
		{
			name: "workflow2.yaml",
			content: `name: Workflow 2
on: [pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3`,
		},
		{
			name:    "not-a-workflow.txt",
			content: `This is not a workflow file and should be ignored`,
		},
	}

	for _, file := range files {
		filePath := filepath.Join(workflowsDir, file.name)
		if err := os.WriteFile(filePath, []byte(file.content), 0600); err != nil {
			t.Fatalf(common.ErrFailedToCreateTestFileNamed, file.name, err)
		}
	}

	// Create scanner with temp directory as base
	scanner := NewScanner(tempDir)

	// Scan workflows
	workflows, err := scanner.ScanWorkflows(workflowsDir)
	if err != nil {
		t.Fatalf(common.ErrUnexpectedError, err)
	}

	// Check the number of workflows found
	expectedCount := 2 // Only .yml and .yaml files
	if len(workflows) != expectedCount {
		t.Errorf(common.ErrExpectedWorkflows, expectedCount, len(workflows))
	}

	// Check that the correct files were found
	foundWorkflow1 := false
	foundWorkflow2 := false
	for _, workflow := range workflows {
		switch filepath.Base(workflow) {
		case "workflow1.yml":
			foundWorkflow1 = true
		case "workflow2.yaml":
			foundWorkflow2 = true
		default:
			t.Errorf(common.ErrUnexpectedWorkflowFile, workflow)
		}
	}

	if !foundWorkflow1 {
		t.Errorf(common.ErrSpecificWorkflowNotFound, "workflow1.yml")
	}
	if !foundWorkflow2 {
		t.Errorf(common.ErrSpecificWorkflowNotFound, "workflow2.yaml")
	}
}
