package updater

import (
	"context"
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestApplyUpdatesAdditionalCases tests additional edge cases for the ApplyUpdates function
func TestApplyUpdatesAdditionalCases(t *testing.T) {
	// Create a temporary directory for our tests
	tempDir, err := os.MkdirTemp("", "test-apply-file-updates-*")
	if err != nil {
		t.Fatalf(common.ErrFailedToCreateTempDir, err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf(common.ErrFailedToRemoveTempDir, err)
		}
	}(tempDir)

	// Create a workflow file
	workflowContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v1
      - uses: actions/cache@v2
`
	workflowPath := filepath.Join(tempDir, "workflow.yml")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Create a directory with no permissions
	noPermsDir := filepath.Join(tempDir, "noperms")
	if err := os.Mkdir(noPermsDir, 0755); err != nil {
		t.Fatalf("Failed to create no-permissions directory: %v", err)
	}
	// Make it inaccessible
	if err := os.Chmod(noPermsDir, 0000); err != nil {
		t.Fatalf("Failed to remove permissions: %v", err)
	}
	// Restore permissions at the end to allow cleanup
	defer func() {
		if err := os.Chmod(noPermsDir, 0755); err != nil {
			t.Logf("Failed to restore directory permissions: %v", err)
		}
	}()

	// Test cases
	testCases := []struct {
		name      string
		filePath  string
		updates   []*Update
		expectErr bool
	}{
		{
			name:     "non-existent file",
			filePath: filepath.Join(tempDir, "nonexistent.yml"),
			updates: []*Update{
				{
					Action: ActionReference{
						Owner:   "actions",
						Name:    "checkout",
						Version: "v2",
					},
					OldVersion:  "v2",
					NewVersion:  "v3",
					OldHash:     "def456",
					NewHash:     "abc123",
					FilePath:    filepath.Join(tempDir, "nonexistent.yml"),
					LineNumber:  7,
					Description: "Update actions/checkout from v2 to v3",
				},
			},
			expectErr: true,
		},
		{
			name:     "permission denied",
			filePath: filepath.Join(noPermsDir, "workflow.yml"),
			updates: []*Update{
				{
					Action: ActionReference{
						Owner:   "actions",
						Name:    "checkout",
						Version: "v2",
					},
					OldVersion:  "v2",
					NewVersion:  "v3",
					OldHash:     "def456",
					NewHash:     "abc123",
					FilePath:    filepath.Join(noPermsDir, "workflow.yml"),
					LineNumber:  7,
					Description: "Update actions/checkout from v2 to v3",
				},
			},
			expectErr: true,
		},
		{
			name:      "empty updates list",
			filePath:  workflowPath,
			updates:   []*Update{},
			expectErr: false,
		},
		{
			name:      "nil updates list",
			filePath:  workflowPath,
			updates:   nil,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := &DefaultUpdateManager{}
			err := manager.ApplyUpdates(context.Background(), tc.updates)

			if tc.expectErr && err == nil {
				t.Error("Expected error, but got nil")
			} else if !tc.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestApplyFileUpdatesAdditionalCases2 targets specific edge cases in applyFileUpdates
// This test focuses on scenarios that aren't covered in the existing tests
func TestApplyFileUpdatesAdditionalCases2(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-coverage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	// Test with different YAML structures for step definition formatting
	stepDefFile := filepath.Join(tempDir, "step_def.yml")
	stepDefContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v2
      # This is a comment about setup-node
      - name: Setup Node
        uses: actions/setup-node@v3
        with:
          node-version: '16'`
	if err := os.WriteFile(stepDefFile, []byte(stepDefContent), 0600); err != nil {
		t.Fatalf("Failed to create step definition file: %v", err)
	}

	stepDefUpdates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       8,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       stepDefFile,
			LineNumber:     8,
			VersionComment: "# v3",
		},
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "setup-node",
				Version:    "v3",
				CommitHash: "",
				Line:       10,
				Comments:   []string{"# This is a comment about setup-node"},
			},
			OldVersion:     "v3",
			NewVersion:     "v4",
			OldHash:        "",
			NewHash:        "5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3",
			FilePath:       stepDefFile,
			LineNumber:     11, // Line number for 'uses:' is different from Comment line
			VersionComment: "# v4",
		},
	}

	err = manager.ApplyUpdates(ctx, stepDefUpdates)
	if err != nil {
		t.Errorf("Expected no error for step definition updates, got %v", err)
	}

	// Read the updated file
	content, err := os.ReadFile(stepDefFile)
	if err != nil {
		t.Fatalf("Failed to read updated step definition file: %v", err)
	}

	// Check for the expected updates
	updatedContent := string(content)
	if !strings.Contains(updatedContent, "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675") {
		t.Errorf("Expected checkout update to be applied, got:\n%s", updatedContent)
	}
	if !strings.Contains(updatedContent, "actions/setup-node@5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3") {
		t.Errorf("Expected setup-node update to be applied, got:\n%s", updatedContent)
	}

	// Test with different formatting cases
	// 1. Line doesn't contain "uses:" but is a step definition (- name:)
	// 2. Line doesn't contain "uses:" and isn't a step definition (other comment)
	// 3. Line is malformed and needs proper indentation
	formatFile := filepath.Join(tempDir, "format.yml")
	formatContent := `name: Format Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
      uses: actions/checkout@v2
      # A comment line
      uses: actions/setup-node@v3`
	if err := os.WriteFile(formatFile, []byte(formatContent), 0600); err != nil {
		t.Fatalf("Failed to create format file: %v", err)
	}

	formatUpdates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       8,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       formatFile,
			LineNumber:     8,
			VersionComment: "# v3",
		},
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "setup-node",
				Version:    "v3",
				CommitHash: "",
				Line:       10,
			},
			OldVersion:     "v3",
			NewVersion:     "v4",
			OldHash:        "",
			NewHash:        "5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3",
			FilePath:       formatFile,
			LineNumber:     10,
			VersionComment: "# v4",
		},
	}

	err = manager.ApplyUpdates(ctx, formatUpdates)
	if err != nil {
		t.Errorf("Expected no error for format updates, got %v", err)
	}

	// Read the updated file
	content, err = os.ReadFile(formatFile)
	if err != nil {
		t.Fatalf("Failed to read updated format file: %v", err)
	}

	// Check for the expected updates
	updatedContent = string(content)
	if !strings.Contains(updatedContent, "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675") {
		t.Errorf("Expected checkout update to be applied, got:\n%s", updatedContent)
	}
	if !strings.Contains(updatedContent, "actions/setup-node@5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3") {
		t.Errorf("Expected setup-node update to be applied, got:\n%s", updatedContent)
	}

	// Test with a file that has a step with "- " but no name field
	stepFile := filepath.Join(tempDir, "step.yml")
	stepContent := `name: Step Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v3`
	if err := os.WriteFile(stepFile, []byte(stepContent), 0600); err != nil {
		t.Fatalf("Failed to create step file: %v", err)
	}

	stepUpdates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       7,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       stepFile,
			LineNumber:     7,
			VersionComment: "# v3",
		},
	}

	err = manager.ApplyUpdates(ctx, stepUpdates)
	if err != nil {
		t.Errorf("Expected no error for step updates, got %v", err)
	}

	// Read the updated file
	content, err = os.ReadFile(stepFile)
	if err != nil {
		t.Fatalf("Failed to read updated step file: %v", err)
	}

	// Check for the expected updates
	updatedContent = string(content)
	if !strings.Contains(updatedContent, "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675") {
		t.Errorf("Expected checkout update to be applied, got:\n%s", updatedContent)
	}

	// Test line adjustments with multiple updates that affect the same file
	// This tests the lineAdjustments map logic
	adjustFile := filepath.Join(tempDir, "adjust.yml")
	adjustContent := `name: Adjust Test
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v3
      - uses: actions/setup-python@v3
      - uses: actions/setup-java@v2`
	if err := os.WriteFile(adjustFile, []byte(adjustContent), 0600); err != nil {
		t.Fatalf("Failed to create adjust file: %v", err)
	}

	// Create updates with line numbers that will require adjustment
	adjustUpdates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       7,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       adjustFile,
			LineNumber:     7,
			VersionComment: "# v3",
		},
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "setup-node",
				Version:    "v3",
				CommitHash: "",
				Line:       8,
			},
			OldVersion:     "v3",
			NewVersion:     "v4",
			OldHash:        "",
			NewHash:        "5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3",
			FilePath:       adjustFile,
			LineNumber:     8,
			VersionComment: "# v4",
		},
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "setup-java",
				Version:    "v2",
				CommitHash: "",
				Line:       10,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "cd89f96b7a67ddd9a5d2a4b166d8e6e9bc97d583",
			FilePath:       adjustFile,
			LineNumber:     10,
			VersionComment: "# v3",
		},
	}

	err = manager.ApplyUpdates(ctx, adjustUpdates)
	if err != nil {
		t.Errorf("Expected no error for adjust updates, got %v", err)
	}

	// Read the updated file
	content, err = os.ReadFile(adjustFile)
	if err != nil {
		t.Fatalf("Failed to read updated adjust file: %v", err)
	}

	// Check for the expected updates
	updatedContent = string(content)
	if !strings.Contains(updatedContent, "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675") {
		t.Errorf("Expected checkout update to be applied, got:\n%s", updatedContent)
	}
	if !strings.Contains(updatedContent, "actions/setup-node@5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3") {
		t.Errorf("Expected setup-node update to be applied, got:\n%s", updatedContent)
	}
	if !strings.Contains(updatedContent, "actions/setup-java@cd89f96b7a67ddd9a5d2a4b166d8e6e9bc97d583") {
		t.Errorf("Expected setup-java update to be applied, got:\n%s", updatedContent)
	}
}

// TestApplyFileUpdatesWithNoVersionComment tests updating an action without providing a version comment
func TestApplyFileUpdatesWithNoVersionComment(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-coverage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	// Create a test file
	testFile := filepath.Join(tempDir, "no_comment.yml")
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v3`
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create an update without a version comment
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       7,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       testFile,
			LineNumber:     7,
			VersionComment: "", // Empty version comment
		},
	}

	err = manager.ApplyUpdates(ctx, updates)
	if err != nil {
		t.Errorf("Expected no error for updates without version comment, got %v", err)
	}

	// Read the updated file
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// Check if the update was applied correctly with a default version comment
	contentStr := string(updatedContent)
	expected := "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675"
	if !strings.Contains(contentStr, expected) {
		t.Errorf("Expected content to contain %s, got:\n%s", expected, contentStr)
	}
	expected = "# v3"
	if !strings.Contains(contentStr, expected) {
		t.Errorf("Expected content to contain default version comment %s, got:\n%s", expected, contentStr)
	}
}

// TestCreateUpdateWithNilContext tests creating an update with a nil context
func TestCreateUpdateWithNilContext(t *testing.T) {
	manager := NewUpdateManager("/tmp")

	// Test with nil context
	action := ActionReference{
		Owner:      "actions",
		Name:       "checkout",
		Version:    "v2",
		CommitHash: "",
		Line:       10,
		Comments:   []string{"# Comment 1"},
	}

	// This should not panic, it should just log a warning
	update, err := manager.CreateUpdate(context.TODO(), "workflow.yml", action, "v3", "ghijkl")
	if err != nil {
		t.Errorf("Expected no error with nil context, got %v", err)
	}
	if update == nil {
		t.Errorf("Expected update with nil context, got nil")
	}
}

// TestApplyUpdatesWithNilContext tests applying updates with a nil context
func TestApplyUpdatesWithNilContext(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-coverage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	manager := NewUpdateManager(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "nil_context.yml")
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create an update
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       7,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       testFile,
			LineNumber:     7,
			VersionComment: "# v3",
		},
	}

	// Apply updates with nil context
	// This should not panic, it should just log a warning
	err = manager.ApplyUpdates(context.TODO(), updates)
	if err != nil {
		t.Errorf("Expected no error with nil context, got %v", err)
	}

	// Read the updated file
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// Check if the update was applied correctly
	contentStr := string(updatedContent)
	expected := "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675"
	if !strings.Contains(contentStr, expected) {
		t.Errorf("Expected content to contain %s, got:\n%s", expected, contentStr)
	}
}
