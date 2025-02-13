package updater

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateUpdate(t *testing.T) {
	manager := NewUpdateManager()
	ctx := context.Background()

	tests := []struct {
		name          string
		action        ActionReference
		latestVersion string
		commitHash    string
		wantUpdate    bool
		wantErr       bool
	}{
		{
			name: "existing action with version",
			action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
				Line:    5,
			},
			latestVersion: "v3",
			commitHash:    "abc123def456",
			wantUpdate:    true,
			wantErr:       false,
		},
		{
			name: "existing action with hash",
			action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "abc123",
				Line:       5,
				Comments:   []string{"Using checkout action"},
			},
			latestVersion: "v3",
			commitHash:    "def456",
			wantUpdate:    true,
			wantErr:       false,
		},
		{
			name: "no update needed",
			action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v3",
				CommitHash: "abc123",
				Line:       5,
			},
			latestVersion: "v3",
			commitHash:    "abc123",
			wantUpdate:    false,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, err := manager.CreateUpdate(ctx, "test.yml", tt.action, tt.latestVersion, tt.commitHash)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantUpdate {
				if update == nil {
					t.Error("CreateUpdate() = nil, want update")
				} else {
					// Verify update fields
					if update.NewVersion != tt.latestVersion {
						t.Errorf("update.NewVersion = %v, want %v", update.NewVersion, tt.latestVersion)
					}
					if update.NewHash != tt.commitHash {
						t.Errorf("update.NewHash = %v, want %v", update.NewHash, tt.commitHash)
					}
					if len(tt.action.Comments) > 0 && len(update.Comments) == 0 {
						t.Error("Comments were not preserved")
					}
					if !strings.Contains(strings.Join(update.Comments, " "), "Original version:") {
						t.Error("Original version comment not added")
					}
				}
			} else if update != nil {
				t.Error("CreateUpdate() = update, want nil")
			}
		})
	}
}

func TestApplyUpdates(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test workflow file
	workflowContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2`

	workflowFile := filepath.Join(tempDir, "workflow.yml")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create updates
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			NewHash:     "def456",
			FilePath:    workflowFile,
			LineNumber:  7,
			Description: "Update actions/checkout from v2 to v3",
		},
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "setup-node",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			NewHash:     "uvw456",
			FilePath:    workflowFile,
			LineNumber:  8,
			Description: "Update actions/setup-node from v2 to v3",
		},
	}

	// Apply updates
	manager := NewUpdateManager()
	if err := manager.ApplyUpdates(context.Background(), updates); err != nil {
		t.Fatalf("ApplyUpdates() error = %v", err)
	}

	// Read updated file
	content, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// Check if updates were applied
	updatedContent := string(content)
	if !strings.Contains(updatedContent, "actions/checkout@def456") {
		t.Error("Update for checkout action hash was not applied")
	}
	if !strings.Contains(updatedContent, "actions/setup-node@uvw456") {
		t.Error("Update for setup-node action hash was not applied")
	}
	if !strings.Contains(updatedContent, "# Current version: v3") {
		t.Error("Version comments were not added")
	}
}

func TestSortUpdatesByLine(t *testing.T) {
	updates := []*Update{
		{LineNumber: 1},
		{LineNumber: 5},
		{LineNumber: 3},
		{LineNumber: 2},
		{LineNumber: 4},
	}

	sortUpdatesByLine(updates)

	for i := 0; i < len(updates)-1; i++ {
		if updates[i].LineNumber < updates[i+1].LineNumber {
			t.Errorf("Updates not sorted correctly: %d should be after %d",
				updates[i].LineNumber, updates[i+1].LineNumber)
		}
	}
}

func TestApplyFileUpdates_InvalidLineNumber(t *testing.T) {
	manager := NewUpdateManager()
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion: "v2",
			NewVersion: "v3",
			FilePath:   "test.yml",
			LineNumber: 999, // Invalid line number
		},
	}

	err := manager.applyFileUpdates("test.yml", updates)
	if err == nil {
		t.Error("applyFileUpdates() with invalid line number should return error")
	}
}

func TestApplyFileUpdates_NonexistentFile(t *testing.T) {
	manager := NewUpdateManager()
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion: "v2",
			NewVersion: "v3",
			FilePath:   "nonexistent.yml",
			LineNumber: 1,
		},
	}

	err := manager.applyFileUpdates("nonexistent.yml", updates)
	if err == nil {
		t.Error("applyFileUpdates() with nonexistent file should return error")
	}
}
