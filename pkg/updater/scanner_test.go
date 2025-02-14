package updater

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			permissions: 0644,
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
			permissions: 0644,
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
			permissions: 0644,
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
			permissions: 0644,
		},
		{
			name:        "empty yaml document",
			content:     "",
			wantErrMsg:  "empty YAML document",
			permissions: 0644,
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
			permissions: 0644,
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

	scanner := NewScanner()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "workflow-test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Create test file
			testFile := filepath.Join(tempDir, "workflow.yml")
			err = os.WriteFile(testFile, []byte(tt.content), tt.permissions)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Parse action references
			_, err = scanner.ParseActionReferences(testFile)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErrMsg, err.Error())
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
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				return os.Chmod(dir, 0000)
			},
			wantErrMsg: "permission denied",
		},
		{
			name: "invalid workflow file",
			setup: func(dir string) error {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				// Create a file with invalid permissions for reading
				filePath := filepath.Join(dir, "workflow.yml")
				if err := os.WriteFile(filePath, []byte("invalid: yaml: content"), 0644); err != nil {
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
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			workflowsDir := filepath.Join(tempDir, ".github", "workflows")

			// Set up test case
			if err := tt.setup(workflowsDir); err != nil {
				t.Fatalf("Failed to set up test: %v", err)
			}

			scanner := NewScanner()
			_, err = scanner.ScanWorkflows(workflowsDir)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErrMsg, err.Error())
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
		{
			name:       "too many parts",
			ref:        "actions/checkout/extra@v2",
			path:       "workflow.yml",
			comments:   nil,
			wantErrMsg: "invalid action name format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseActionReference(tt.ref, tt.path, tt.comments)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErrMsg, err.Error())
			}
		})
	}
}
