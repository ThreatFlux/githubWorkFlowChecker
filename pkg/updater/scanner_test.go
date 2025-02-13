package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanWorkflows(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .github/workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflows dir: %v", err)
	}

	// Create test workflow files
	testFiles := []struct {
		name    string
		content string
	}{
		{
			name: "workflow1.yml",
			content: `name: Test Workflow 1
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`,
		},
		{
			name: "workflow2.yaml",
			content: `name: Test Workflow 2
on: [pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-node@v3`,
		},
	}

	for _, tf := range testFiles {
		err := os.WriteFile(filepath.Join(workflowsDir, tf.name), []byte(tf.content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.name, err)
		}
	}

	// Create scanner and scan workflows
	scanner := NewScanner()
	files, err := scanner.ScanWorkflows(workflowsDir)
	if err != nil {
		t.Fatalf("ScanWorkflows() error = %v", err)
	}

	// Check number of files found
	if len(files) != len(testFiles) {
		t.Errorf("ScanWorkflows() found %d files, want %d", len(files), len(testFiles))
	}

	// Check file extensions
	for _, file := range files {
		ext := filepath.Ext(file)
		if ext != ".yml" && ext != ".yaml" {
			t.Errorf("ScanWorkflows() found file with invalid extension: %s", ext)
		}
	}
}

func TestParseActionReferences(t *testing.T) {
	// Create a temporary workflow file
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	workflowContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v3
      - uses: actions/cache@d1255ad9362389eac595a9ae406b8e8cb3331f16
      - run: npm test`

	workflowFile := filepath.Join(tempDir, "workflow.yml")
	err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse action references
	scanner := NewScanner()
	refs, err := scanner.ParseActionReferences(workflowFile)
	if err != nil {
		t.Fatalf("ParseActionReferences() error = %v", err)
	}

	// Check number of references found
	expectedRefs := 3
	if len(refs) != expectedRefs {
		t.Errorf("ParseActionReferences() found %d refs, want %d", len(refs), expectedRefs)
	}

	// Check specific references
	expectedActions := []struct {
		owner   string
		name    string
		version string
	}{
		{"actions", "checkout", "v2"},
		{"actions", "setup-node", "v3"},
		{"actions", "cache", "d1255ad9362389eac595a9ae406b8e8cb3331f16"},
	}

	for i, expected := range expectedActions {
		if refs[i].Owner != expected.owner {
			t.Errorf("Action[%d] owner = %s, want %s", i, refs[i].Owner, expected.owner)
		}
		if refs[i].Name != expected.name {
			t.Errorf("Action[%d] name = %s, want %s", i, refs[i].Name, expected.name)
		}
		if refs[i].Version != expected.version {
			t.Errorf("Action[%d] version = %s, want %s", i, refs[i].Version, expected.version)
		}
	}
}
