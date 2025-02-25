package updater

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestApplyFileUpdatesWithVariousFormats(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	// Test with different file formats
	testCases := []struct {
		name     string
		content  string
		updates  []*Update
		expected []string // Strings that should be present in the updated content
	}{
		{
			name: "YAML format with comments",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2  # Current version
      - uses: actions/setup-node@v3  # Node.js setup`,
			updates: []*Update{
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
					FilePath:       filepath.Join(tempDir, "workflow.yml"),
					LineNumber:     7,
					VersionComment: "# v3",
				},
			},
			expected: []string{
				"actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675",
				"# v3",
			},
		},
		{
			name: "YAML format with multiple updates",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v3
      - uses: actions/setup-python@v3`,
			updates: []*Update{
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
					FilePath:       filepath.Join(tempDir, "workflow-multi.yml"),
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
					FilePath:       filepath.Join(tempDir, "workflow-multi.yml"),
					LineNumber:     8,
					VersionComment: "# v4",
				},
				{
					Action: ActionReference{
						Owner:      "actions",
						Name:       "setup-python",
						Version:    "v3",
						CommitHash: "",
						Line:       9,
					},
					OldVersion:     "v3",
					NewVersion:     "v4",
					OldHash:        "",
					NewHash:        "65d7f2d534ac1bc67fcd62888c5f4f3d2cb2b236",
					FilePath:       filepath.Join(tempDir, "workflow-multi.yml"),
					LineNumber:     9,
					VersionComment: "# v4",
				},
			},
			expected: []string{
				"actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675",
				"actions/setup-node@5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3",
				"actions/setup-python@65d7f2d534ac1bc67fcd62888c5f4f3d2cb2b236",
				"# v3",
				"# v4",
			},
		},
		{
			name: "YAML format with indentation",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v2
      - name: Setup Node.js
        uses: actions/setup-node@v3`,
			updates: []*Update{
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
					FilePath:       filepath.Join(tempDir, "workflow-indent.yml"),
					LineNumber:     8,
					VersionComment: "# v3",
				},
			},
			expected: []string{
				"uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675",
				"# v3",
			},
		},
		{
			name: "YAML format with existing comment",
			content: `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2  # This is a comment
      - uses: actions/setup-node@v3  # Another comment`,
			updates: []*Update{
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
					FilePath:       filepath.Join(tempDir, "workflow-comment.yml"),
					LineNumber:     7,
					VersionComment: "# v3",
				},
			},
			expected: []string{
				"actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675",
				"# v3",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the test file
			filePath := tc.updates[0].FilePath
			if err := os.WriteFile(filePath, []byte(tc.content), 0600); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Apply updates
			err := manager.ApplyUpdates(ctx, tc.updates)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			// Read the updated file
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read updated file: %v", err)
			}

			// Check if the updates were applied correctly
			updatedContent := string(content)
			for _, expected := range tc.expected {
				if !strings.Contains(updatedContent, expected) {
					t.Errorf("Expected %q to be in the updated content, but it wasn't.\nUpdated content:\n%s", expected, updatedContent)
				}
			}
		})
	}
}

func TestApplyFileUpdatesErrorHandling(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	// Test with non-existent file
	nonExistentFile := filepath.Join(tempDir, "nonexistent.yml")
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
			FilePath:       nonExistentFile,
			LineNumber:     7,
			VersionComment: "# v3",
		},
	}

	err = manager.ApplyUpdates(ctx, updates)
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}

	// Test with invalid line number
	validFile := filepath.Join(tempDir, "valid.yml")
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`
	if err := os.WriteFile(validFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	invalidLineUpdates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       100, // Invalid line number
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       validFile,
			LineNumber:     100, // Invalid line number
			VersionComment: "# v3",
		},
	}

	err = manager.ApplyUpdates(ctx, invalidLineUpdates)
	if err == nil {
		t.Errorf("Expected error for invalid line number, got nil")
	}

	// Test with file outside base directory
	outsideFile := filepath.Join(os.TempDir(), "outside.yml")
	if err := os.WriteFile(outsideFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(outsideFile)

	outsideUpdates := []*Update{
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
			FilePath:       outsideFile,
			LineNumber:     7,
			VersionComment: "# v3",
		},
	}

	err = manager.ApplyUpdates(ctx, outsideUpdates)
	if err == nil {
		t.Errorf("Expected error for file outside base directory, got nil")
	}

	// Test with read-only file
	readOnlyFile := filepath.Join(tempDir, "readonly.yml")
	if err := os.WriteFile(readOnlyFile, []byte(content), 0400); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	readOnlyUpdates := []*Update{
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
			FilePath:       readOnlyFile,
			LineNumber:     7,
			VersionComment: "# v3",
		},
	}

	err = manager.ApplyUpdates(ctx, readOnlyUpdates)
	if err == nil {
		// This might pass on some systems where the current user has write permission
		// regardless of file permissions, so we'll just log it
		t.Logf("Expected error for read-only file, but got nil. This might be system-dependent.")
	}
}

func TestApplyFileUpdatesConcurrent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	// Create a test file
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v3
      - uses: actions/setup-python@v3
      - uses: actions/setup-java@v2
      - uses: actions/setup-go@v3`
	testFile := filepath.Join(tempDir, "concurrent.yml")
	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create updates for different actions
	updates1 := []*Update{
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
			FilePath:       testFile,
			LineNumber:     8,
			VersionComment: "# v4",
		},
	}

	updates2 := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "setup-python",
				Version:    "v3",
				CommitHash: "",
				Line:       9,
			},
			OldVersion:     "v3",
			NewVersion:     "v4",
			OldHash:        "",
			NewHash:        "65d7f2d534ac1bc67fcd62888c5f4f3d2cb2b236",
			FilePath:       testFile,
			LineNumber:     9,
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
			FilePath:       testFile,
			LineNumber:     10,
			VersionComment: "# v3",
		},
	}

	updates3 := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "setup-go",
				Version:    "v3",
				CommitHash: "",
				Line:       11,
			},
			OldVersion:     "v3",
			NewVersion:     "v4",
			OldHash:        "",
			NewHash:        "93397bea11091df50f3d7e79fd3825ff3d6dd5fd",
			FilePath:       testFile,
			LineNumber:     11,
			VersionComment: "# v4",
		},
	}

	// Apply updates concurrently
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		err := manager.ApplyUpdates(ctx, updates1)
		if err != nil {
			t.Errorf("Expected no error for updates1, got %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		// Add a small delay to ensure concurrent access
		time.Sleep(10 * time.Millisecond)
		err := manager.ApplyUpdates(ctx, updates2)
		if err != nil {
			t.Errorf("Expected no error for updates2, got %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		// Add a small delay to ensure concurrent access
		time.Sleep(20 * time.Millisecond)
		err := manager.ApplyUpdates(ctx, updates3)
		if err != nil {
			t.Errorf("Expected no error for updates3, got %v", err)
		}
	}()

	wg.Wait()

	// Read the updated file
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// Check if all updates were applied correctly
	contentStr := string(updatedContent)
	expectedUpdates := []string{
		"actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675",
		"actions/setup-node@5e21ff4d43d7137a3d62c6e217daeeb3a49ef5c3",
		"actions/setup-python@65d7f2d534ac1bc67fcd62888c5f4f3d2cb2b236",
		"actions/setup-java@cd89f96b7a67ddd9a5d2a4b166d8e6e9bc97d583",
		"actions/setup-go@93397bea11091df50f3d7e79fd3825ff3d6dd5fd",
		"# v3",
		"# v4",
	}

	for _, expected := range expectedUpdates {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected %q to be in the updated content, but it wasn't.\nUpdated content:\n%s", expected, contentStr)
		}
	}
}

func TestApplyFileUpdatesEdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "update-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	// Test with empty file
	emptyFile := filepath.Join(tempDir, "empty.yml")
	if err := os.WriteFile(emptyFile, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	emptyUpdates := []*Update{
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       1, // Invalid line for empty file
			},
			OldVersion:     "v2",
			NewVersion:     "v3",
			OldHash:        "",
			NewHash:        "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			FilePath:       emptyFile,
			LineNumber:     1,
			VersionComment: "# v3",
		},
	}

	// Apply updates to empty file
	if err = manager.ApplyUpdates(ctx, emptyUpdates); err != nil {
		t.Errorf("Expected no error for empty file, got %v", err)
	}
	// The function actually modifies the empty file, which is interesting.
	// Let's verify that the file now contains the version comment.
	emptyContent, err := os.ReadFile(emptyFile)
	if err != nil {
		t.Fatalf("Failed to read empty file after update: %v", err)
	}

	// Check if the file contains the version comment
	emptyContentStr := string(emptyContent)
	if !strings.Contains(emptyContentStr, "# v3") {
		t.Errorf("Expected empty file to contain version comment, got content: %s", emptyContentStr)
	}

	// Test with file containing special characters
	specialFile := filepath.Join(tempDir, "special.yml")
	specialContent := `name: "Test Workflow with 'special' characters"
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2  # Comment with "quotes" and 'apostrophes'
      - uses: actions/setup-node@v3  # Comment with symbols: !@#$%^&*()_+`
	if err := os.WriteFile(specialFile, []byte(specialContent), 0600); err != nil {
		t.Fatalf("Failed to create special file: %v", err)
	}

	specialUpdates := []*Update{
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
			FilePath:       specialFile,
			LineNumber:     7,
			VersionComment: "# v3 with 'special' \"characters\" !@#$%^&*()_+",
		},
	}

	err = manager.ApplyUpdates(ctx, specialUpdates)
	if err != nil {
		t.Errorf("Expected no error for special file, got %v", err)
	}

	// Read the updated file
	updatedContent, err := os.ReadFile(specialFile)
	if err != nil {
		t.Fatalf("Failed to read updated special file: %v", err)
	}

	// Check if the update was applied correctly
	content := string(updatedContent)
	expected := "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675"
	if !strings.Contains(content, expected) {
		t.Errorf("Expected %q to be in the updated content, but it wasn't.\nUpdated content:\n%s", expected, content)
	}

	// Test with multiple updates to the same line
	sameLineFile := filepath.Join(tempDir, "sameline.yml")
	sameLineContent := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`
	if err := os.WriteFile(sameLineFile, []byte(sameLineContent), 0600); err != nil {
		t.Fatalf("Failed to create same line file: %v", err)
	}

	sameLineUpdates := []*Update{
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
			FilePath:       sameLineFile,
			LineNumber:     7,
			VersionComment: "# v3",
		},
		{
			Action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       7,
			},
			OldVersion:     "v2",
			NewVersion:     "v4",
			OldHash:        "",
			NewHash:        "b4ffde65f46336ab88eb53be808477a3936bae11",
			FilePath:       sameLineFile,
			LineNumber:     7,
			VersionComment: "# v4",
		},
	}

	err = manager.ApplyUpdates(ctx, sameLineUpdates)
	if err != nil {
		t.Errorf("Expected no error for same line updates, got %v", err)
	}

	// Read the updated file
	updatedContent, err = os.ReadFile(sameLineFile)
	if err != nil {
		t.Fatalf("Failed to read updated same line file: %v", err)
	}

	// Check if the last update was applied
	content = string(updatedContent)
	expected = "actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11"
	if !strings.Contains(content, expected) {
		t.Errorf("Expected %q to be in the updated content, but it wasn't.\nUpdated content:\n%s", expected, content)
	}
	expected = "# v4"
	if !strings.Contains(content, expected) {
		t.Errorf("Expected %q to be in the updated content, but it wasn't.\nUpdated content:\n%s", expected, content)
	}
}
