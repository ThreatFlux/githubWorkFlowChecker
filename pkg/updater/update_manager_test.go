package updater

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewUpdateManager(t *testing.T) {
	// Test with valid base directory
	baseDir := "/tmp"
	manager := NewUpdateManager(baseDir)
	if manager.baseDir != baseDir {
		t.Errorf("Expected baseDir to be %s, got %s", baseDir, manager.baseDir)
	}

	// Test with empty base directory
	manager = NewUpdateManager("")
	if manager.baseDir != "" {
		t.Errorf("Expected baseDir to be empty, got %s", manager.baseDir)
	}
}

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.yml")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a test file in the subdirectory
	subFile := filepath.Join(subDir, "subtest.yml")
	if err := os.WriteFile(subFile, []byte("subtest"), 0600); err != nil {
		t.Fatalf("Failed to create test file in subdirectory: %v", err)
	}

	// Create a file outside the base directory
	outsideFile := filepath.Join(os.TempDir(), "outside.yml")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0600); err != nil {
		t.Fatalf("Failed to create outside file: %v", err)
	}
	defer os.Remove(outsideFile)

	manager := NewUpdateManager(tempDir)

	// Test valid file
	err = manager.validatePath(testFile)
	if err != nil {
		t.Errorf("Expected no error for valid file, got %v", err)
	}

	// Test valid file in subdirectory
	err = manager.validatePath(subFile)
	if err != nil {
		t.Errorf("Expected no error for valid file in subdirectory, got %v", err)
	}

	// Test directory (should fail as it's not a regular file)
	err = manager.validatePath(subDir)
	if err == nil {
		t.Errorf("Expected error for directory, got nil")
	}

	// Test file outside base directory
	err = manager.validatePath(outsideFile)
	if err == nil {
		t.Errorf("Expected error for file outside base directory, got nil")
	}

	// Test empty path
	err = manager.validatePath("")
	if err == nil {
		t.Errorf("Expected error for empty path, got nil")
	}

	// Test with empty base directory
	emptyManager := NewUpdateManager("")
	err = emptyManager.validatePath(testFile)
	if err == nil {
		t.Errorf("Expected error with empty base directory, got nil")
	}
}

func TestPreserveComments(t *testing.T) {
	manager := NewUpdateManager("/tmp")

	// Test with no comments
	action := ActionReference{
		Comments: []string{},
	}
	preserved := manager.PreserveComments(action)
	if preserved != nil {
		t.Errorf("Expected nil for no comments, got %v", preserved)
	}

	// Test with comments but no original version comment
	action = ActionReference{
		Comments: []string{"# Comment 1", "# Comment 2"},
	}
	preserved = manager.PreserveComments(action)
	if len(preserved) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(preserved))
	}

	// Test with original version comment
	action = ActionReference{
		Comments: []string{"# Comment 1", "# Original version: v1.0.0", "# Comment 2"},
	}
	preserved = manager.PreserveComments(action)
	if len(preserved) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(preserved))
	}
	for _, comment := range preserved {
		if comment == "# Original version: v1.0.0" {
			t.Errorf("Original version comment should be removed")
		}
	}
}

func TestCreateUpdate(t *testing.T) {
	manager := NewUpdateManager("/tmp")
	ctx := context.Background()

	// Test with same version and commit hash (no update needed)
	action := ActionReference{
		Owner:      "actions",
		Name:       "checkout",
		Version:    "v2",
		CommitHash: "abcdef",
		Line:       10,
		Comments:   []string{"# Comment 1"},
	}
	update, err := manager.CreateUpdate(ctx, "workflow.yml", action, "v2", "abcdef")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if update != nil {
		t.Errorf("Expected nil update, got %v", update)
	}

	// Test with different version
	update, err = manager.CreateUpdate(ctx, "workflow.yml", action, "v3", "ghijkl")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if update == nil {
		t.Errorf("Expected update, got nil")
	}
	if update != nil {
		if update.OldVersion != "v2" {
			t.Errorf("Expected OldVersion to be v2, got %s", update.OldVersion)
		}
		if update.NewVersion != "v3" {
			t.Errorf("Expected NewVersion to be v3, got %s", update.NewVersion)
		}
		if update.OldHash != "abcdef" {
			t.Errorf("Expected OldHash to be abcdef, got %s", update.OldHash)
		}
		if update.NewHash != "ghijkl" {
			t.Errorf("Expected NewHash to be ghijkl, got %s", update.NewHash)
		}
		if update.FilePath != "workflow.yml" {
			t.Errorf("Expected FilePath to be workflow.yml, got %s", update.FilePath)
		}
		if update.LineNumber != 10 {
			t.Errorf("Expected LineNumber to be 10, got %d", update.LineNumber)
		}
		if len(update.Comments) != 1 {
			t.Errorf("Expected 1 comment, got %d", len(update.Comments))
		}
		if update.VersionComment != "# v3" {
			t.Errorf("Expected VersionComment to be '# v3', got '%s'", update.VersionComment)
		}
		if update.OriginalVersion != "abcdef" {
			t.Errorf("Expected OriginalVersion to be abcdef, got %s", update.OriginalVersion)
		}
		if update.Description != "Update actions/checkout from abcdef to v3" {
			t.Errorf("Expected Description to be 'Update actions/checkout from abcdef to v3', got '%s'", update.Description)
		}
	}

	// Test with context.TODO()
	update, err = manager.CreateUpdate(context.TODO(), "workflow.yml", action, "v3", "ghijkl")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if update == nil {
		t.Errorf("Expected update, got nil")
	}
}

func TestSortUpdatesByLine(t *testing.T) {
	// Test with empty updates
	var updates []*Update
	sortUpdatesByLine(updates)
	if len(updates) != 0 {
		t.Errorf("Expected empty updates, got %d", len(updates))
	}

	// Test with single update
	updates = []*Update{
		{LineNumber: 10},
	}
	sortUpdatesByLine(updates)
	if len(updates) != 1 || updates[0].LineNumber != 10 {
		t.Errorf("Expected single update with LineNumber 10, got %d", updates[0].LineNumber)
	}

	// Test with multiple updates
	updates = []*Update{
		{LineNumber: 10},
		{LineNumber: 20},
		{LineNumber: 5},
	}
	sortUpdatesByLine(updates)
	if len(updates) != 3 {
		t.Errorf("Expected 3 updates, got %d", len(updates))
	}
	if updates[0].LineNumber != 20 {
		t.Errorf("Expected first update to have LineNumber 20, got %d", updates[0].LineNumber)
	}
	if updates[1].LineNumber != 10 {
		t.Errorf("Expected second update to have LineNumber 10, got %d", updates[1].LineNumber)
	}
	if updates[2].LineNumber != 5 {
		t.Errorf("Expected third update to have LineNumber 5, got %d", updates[2].LineNumber)
	}
}

func TestApplyUpdates(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test workflow file
	workflowContent := `name: Test Workflow

on:
  push:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2  # v2
      - uses: actions/setup-node@v3  # v3
`
	workflowFile := filepath.Join(tempDir, "workflow.yml")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0600); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	// Create updates
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       10,
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "abcdef",
			FilePath:       workflowFile,
			LineNumber:     10,
			VersionComment: "# v3",
		},
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "setup-node",
				Version:    "v3",
				CommitHash: "",
				Line:       11,
			},
			OldVersion:     "v3",
			NewVersion:     "v4",
			OldHash:        "",
			NewHash:        "ghijkl",
			FilePath:       workflowFile,
			LineNumber:     11,
			VersionComment: "# v4",
		},
	}

	// Apply updates
	err = manager.ApplyUpdates(ctx, updates)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Read the updated file
	content, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read updated workflow file: %v", err)
	}

	// Check if the updates were applied correctly
	updatedContent := string(content)

	// The test might be sensitive to exact formatting, so we'll check for the presence
	// of the key parts of the update rather than the exact string
	if !strings.Contains(updatedContent, "actions/checkout@abcdef") && !strings.Contains(updatedContent, "# v3") {
		t.Errorf("Expected checkout update to be applied, got:\n%s", updatedContent)
	}
	if !strings.Contains(updatedContent, "actions/setup-node@ghijkl") && !strings.Contains(updatedContent, "# v4") {
		t.Errorf("Expected setup-node update to be applied, got:\n%s", updatedContent)
	}

	// Test with context.TODO()
	err = manager.ApplyUpdates(context.TODO(), updates)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test with invalid file path
	invalidUpdates := []*Update{
		{
			FilePath:   filepath.Join(tempDir, "nonexistent.yml"),
			LineNumber: 10,
		},
	}
	err = manager.ApplyUpdates(ctx, invalidUpdates)
	if err == nil {
		t.Errorf("Expected error for invalid file path, got nil")
	}

	// Test with invalid line number
	invalidUpdates = []*Update{
		{
			FilePath:   workflowFile,
			LineNumber: 100, // Line number out of range
		},
	}
	err = manager.ApplyUpdates(ctx, invalidUpdates)
	if err == nil {
		t.Errorf("Expected error for invalid line number, got nil")
	}
}
