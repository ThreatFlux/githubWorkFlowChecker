package updater

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestValidatePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	tests := []struct {
		name    string
		baseDir string
		path    string
		wantErr bool
		errMsg  string // Add expected error message
	}{
		{
			name:    "valid path within base dir",
			baseDir: tempDir,
			path:    filepath.Join(tempDir, "test.yml"),
			wantErr: false,
		},
		{
			name:    "path outside base dir",
			baseDir: tempDir,
			path:    "/etc/passwd",
			wantErr: true,
			errMsg:  "path is outside of allowed directory",
		},
		{
			name:    "empty base dir",
			baseDir: "",
			path:    "test.yml",
			wantErr: true,
			errMsg:  "base directory not set",
		},
		{
			name:    "relative path traversal attempt",
			baseDir: tempDir,
			path:    filepath.Join(tempDir, "../../../etc/passwd"),
			wantErr: true,
			errMsg:  "path is outside of allowed directory",
		},
		{
			name:    "nil or empty path",
			baseDir: tempDir,
			path:    "",
			wantErr: true,
			errMsg:  "path is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewUpdateManager(tt.baseDir)
			err := manager.validatePath(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePath() error message = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestCreateUpdate_WithComments(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	manager := NewUpdateManager(tempDir)
	ctx := context.Background()

	tests := []struct {
		name          string
		action        ActionReference
		latestVersion string
		commitHash    string
		wantComments  []string
	}{
		{
			name: "preserve existing comments",
			action: ActionReference{
				Owner:      "actions",
				Name:       "checkout",
				Version:    "v2",
				CommitHash: "",
				Line:       5,
				Comments:   []string{"# Important comment", "# Do not remove"},
			},
			latestVersion: "v3",
			commitHash:    "abc123",
			wantComments:  []string{"# Important comment", "# Do not remove"},
		},
		{
			name: "update with commit hash",
			action: ActionReference{
				Owner:      "actions",
				Name:       "setup-node",
				Version:    "v2",
				CommitHash: "def456",
				Line:       6,
				Comments:   []string{"# Version comment", "# Security note"},
			},
			latestVersion: "v3",
			commitHash:    "xyz789",
			wantComments:  []string{"# Version comment", "# Security note"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, err := manager.CreateUpdate(ctx, "test.yml", tt.action, tt.latestVersion, tt.commitHash)
			if err != nil {
				t.Errorf("CreateUpdate() unexpected error = %v", err)
				return
			}
			if update == nil {
				t.Error("CreateUpdate() returned nil update")
				return
			}

			// Verify comments are preserved
			for _, wantComment := range tt.wantComments {
				found := false
				for _, comment := range update.Comments {
					if comment == wantComment {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Comment %q not preserved in update", wantComment)
				}
			}
		})
	}
}

func TestApplyUpdates_ConcurrentFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	// Create multiple test files
	files := []string{"workflow1.yml", "workflow2.yml", "workflow3.yml"}
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2`

	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Create updates for multiple files
	var updates []*Update
	for _, file := range files {
		updates = append(updates, &Update{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion: "v2",
			NewVersion: "v3",
			NewHash:    "abc123",
			FilePath:   filepath.Join(tempDir, file),
			LineNumber: 7,
		})
	}

	manager := NewUpdateManager(tempDir)
	if err := manager.ApplyUpdates(context.Background(), updates); err != nil {
		t.Fatalf("ApplyUpdates() error = %v", err)
	}

	// Verify updates were applied to all files
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(tempDir, file))
		if err != nil {
			t.Errorf("Failed to read file %s: %v", file, err)
			continue
		}

		if !strings.Contains(string(content), "actions/checkout@abc123") {
			t.Errorf("Update not applied correctly to file %s", file)
		}
	}
}

func TestApplyUpdates_FileLocking(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workflow-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	// Create test file
	workflowFile := filepath.Join(tempDir, "workflow.yml")
	content := `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-node@v2`

	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewUpdateManager(tempDir)

	// Create multiple goroutines trying to update the same file
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			update := &Update{
				Action: ActionReference{
					Owner:   "actions",
					Name:    "checkout",
					Version: "v2",
				},
				OldVersion: "v2",
				NewVersion: fmt.Sprintf("v3.%d", i),
				NewHash:    fmt.Sprintf("abc%d", i),
				FilePath:   workflowFile,
				LineNumber: 7,
			}
			if err := manager.ApplyUpdates(context.Background(), []*Update{update}); err != nil {
				errChan <- err
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		t.Errorf("Concurrent update error: %v", err)
	}

	// Verify file was updated
	updatedContent, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// The file should have been updated exactly once
	count := strings.Count(string(updatedContent), "actions/checkout@abc")
	if count != 1 {
		t.Errorf("Expected one update, got %d updates", count)
	}
}

func TestPreserveComments(t *testing.T) {
	tests := []struct {
		name       string
		action     ActionReference
		wantLength int
	}{
		{
			name: "no comments",
			action: ActionReference{
				Comments: nil,
			},
			wantLength: 0,
		},
		{
			name: "with version comment",
			action: ActionReference{
				Comments: []string{"# Some comment", "# Original version: v1"},
			},
			wantLength: 1,
		},
		{
			name: "multiple comments",
			action: ActionReference{
				Comments: []string{"# Comment 1", "# Comment 2", "# Comment 3"},
			},
			wantLength: 3,
		},
	}

	manager := NewUpdateManager("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preserved := manager.PreserveComments(tt.action)
			if len(preserved) != tt.wantLength {
				t.Errorf("PreserveComments() got %d comments, want %d", len(preserved), tt.wantLength)
			}
		})
	}
}
func TestNewUpdateManager(t *testing.T) {
	tests := []struct {
		name          string
		baseDir       string
		wantBaseDir   string
		wantEmptyBase bool
	}{
		{
			name:          "empty base directory",
			baseDir:       "",
			wantEmptyBase: true,
		},
		{
			name:        "normal base directory",
			baseDir:     "/tmp/test",
			wantBaseDir: filepath.Clean("/tmp/test"),
		},
		{
			name:        "relative path",
			baseDir:     "./test",
			wantBaseDir: filepath.Clean("./test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewUpdateManager(tt.baseDir)

			if tt.wantEmptyBase {
				if manager.baseDir != "" {
					t.Errorf("NewUpdateManager() baseDir = %q, want empty string", manager.baseDir)
				}
			} else {
				if manager.baseDir != tt.wantBaseDir {
					t.Errorf("NewUpdateManager() baseDir = %q, want %q", manager.baseDir, tt.wantBaseDir)
				}
			}
		})
	}
}
func TestSortUpdatesByLine(t *testing.T) {
	tests := []struct {
		name     string
		updates  []*Update
		want     []int  // Expected line numbers after sorting
		content  string // Initial file content
		wantFile string // Expected file content after updates
	}{
		{
			name:    "empty updates",
			updates: []*Update{},
			want:    []int{},
		},
		{
			name: "single update",
			updates: []*Update{
				{LineNumber: 5},
			},
			want: []int{5},
		},
		{
			name: "already sorted",
			updates: []*Update{
				{LineNumber: 10},
				{LineNumber: 8},
				{LineNumber: 5},
			},
			want: []int{10, 8, 5},
		},
		{
			name: "reverse sorted",
			updates: []*Update{
				{LineNumber: 5},
				{LineNumber: 8},
				{LineNumber: 10},
			},
			want: []int{10, 8, 5},
		},
		{
			name: "random order",
			updates: []*Update{
				{LineNumber: 8},
				{LineNumber: 5},
				{LineNumber: 15},
				{LineNumber: 10},
			},
			want: []int{15, 10, 8, 5},
		},
		{
			name: "demonstrate importance of order",
			content: `name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v1  # Line 7
      - uses: actions/setup-node@v1  # Line 8
      - uses: actions/cache@v1  # Line 9`,
			updates: []*Update{
				{
					LineNumber: 7,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "checkout",
						Version: "v1",
					},
					NewVersion: "v2",
					NewHash:    "abc123",
				},
				{
					LineNumber: 9,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "cache",
						Version: "v1",
					},
					NewVersion: "v2",
					NewHash:    "def456",
				},
				{
					LineNumber: 8,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "setup-node",
						Version: "v1",
					},
					NewVersion: "v2",
					NewHash:    "xyz789",
				},
			},
			want: []int{9, 8, 7},
			wantFile: `name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@abc123  # v2
      - uses: actions/setup-node@xyz789  # v2
      - uses: actions/cache@def456  # v2`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test sorting
			sortUpdatesByLine(tt.updates)

			got := make([]int, len(tt.updates))
			for i, u := range tt.updates {
				got[i] = u.LineNumber
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortUpdatesByLine() got line numbers = %v, want %v", got, tt.want)
			}

			// Test actual file updates if content provided
			if tt.content != "" {
				// Create temp file with test content
				tempDir, err := os.MkdirTemp("", "sort-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				defer func(path string) {
					err := os.RemoveAll(path)
					if err != nil {
						log.Printf("Failed to remove temp dir: %v", err)
					}
				}(tempDir)

				testFile := filepath.Join(tempDir, "test.yml")
				if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}

				// Apply updates
				manager := NewUpdateManager(tempDir)
				for _, update := range tt.updates {
					update.FilePath = testFile
				}
				if err := manager.ApplyUpdates(context.Background(), tt.updates); err != nil {
					t.Fatalf("ApplyUpdates() error = %v", err)
				}

				// Read and verify result
				got, err := os.ReadFile(testFile)
				if err != nil {
					t.Fatalf("Failed to read result file: %v", err)
				}

				if string(got) != tt.wantFile {
					t.Errorf("File content after updates:\ngot:\n%s\nwant:\n%s", string(got), tt.wantFile)
				}
			}
		})
	}
}

func TestApplyUpdates_PreservesIndentation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "indent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	content := `name: Test Workflow
on: push
jobs:
    test:
        runs-on: ubuntu-latest
        steps:
            - uses: actions/checkout@v1  # Line 7
                - uses: actions/setup-node@v1  # Line 8 (extra indent)
          - uses: actions/cache@v1  # Line 9 (different indent)`

	testFile := filepath.Join(tempDir, "test.yml")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	updates := []*Update{
		{
			LineNumber: 7,
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v1",
			},
			NewVersion: "v2",
			NewHash:    "abc123",
			FilePath:   testFile,
		},
		{
			LineNumber: 8,
			Action: ActionReference{
				Owner:   "actions",
				Name:    "setup-node",
				Version: "v1",
			},
			NewVersion: "v2",
			NewHash:    "xyz789",
			FilePath:   testFile,
		},
		{
			LineNumber: 9,
			Action: ActionReference{
				Owner:   "actions",
				Name:    "cache",
				Version: "v1",
			},
			NewVersion: "v2",
			NewHash:    "def456",
			FilePath:   testFile,
		},
	}

	manager := NewUpdateManager(tempDir)
	if err := manager.ApplyUpdates(context.Background(), updates); err != nil {
		t.Fatalf("ApplyUpdates() error = %v", err)
	}

	// Read and verify result
	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read result file: %v", err)
	}

	want := `name: Test Workflow
on: push
jobs:
    test:
        runs-on: ubuntu-latest
        steps:
            - uses: actions/checkout@abc123  # v2
                - uses: actions/setup-node@xyz789  # v2
          - uses: actions/cache@def456  # v2`

	if string(got) != want {
		t.Errorf("File content after updates:\ngot:\n%s\nwant:\n%s", string(got), want)
	}
}
func TestApplyUpdates_VersionReferences(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "version-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Printf("Failed to remove temp dir: %v", err)
		}
	}(tempDir)

	testCases := []struct {
		name     string
		content  string
		updates  []*Update
		expected string
	}{
		{
			name: "version to hash update",
			content: `steps:
              - uses: actions/checkout@v1
              - uses: actions/setup-node@v1`,
			updates: []*Update{
				{
					LineNumber: 2,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "checkout",
						Version: "v1",
					},
					NewVersion: "v2",
					NewHash:    "abc123",
				},
			},
			expected: `steps:
              - uses: actions/checkout@abc123  # v2
              - uses: actions/setup-node@v1`,
		},
		{
			name: "hash to hash update",
			content: `steps:
              - uses: actions/checkout@def456
              - uses: actions/setup-node@v1`,
			updates: []*Update{
				{
					LineNumber: 2,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "checkout",
						Version: "def456",
					},
					NewVersion: "v2",
					NewHash:    "abc123",
				},
			},
			expected: `steps:
              - uses: actions/checkout@abc123  # v2
              - uses: actions/setup-node@v1`,
		},
		{
			name: "multiple updates mixing versions and hashes",
			content: `steps:
              - uses: actions/checkout@v1  # old version
              - uses: actions/setup-node@def456  # hash reference
              - uses: actions/cache@v1`,
			updates: []*Update{
				{
					LineNumber: 2,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "checkout",
						Version: "v1",
					},
					NewVersion: "v2",
					NewHash:    "abc123",
				},
				{
					LineNumber: 3,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "setup-node",
						Version: "def456",
					},
					NewVersion: "v2",
					NewHash:    "xyz789",
				},
				{
					LineNumber: 4,
					Action: ActionReference{
						Owner:   "actions",
						Name:    "cache",
						Version: "v1",
					},
					NewVersion: "v2",
					NewHash:    "uvw456",
				},
			},
			expected: `steps:
              - uses: actions/checkout@abc123  # v2
              - uses: actions/setup-node@xyz789  # v2
              - uses: actions/cache@uvw456  # v2`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "test.yml")
			if err := os.WriteFile(testFile, []byte(tc.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Set file path for all updates
			for _, update := range tc.updates {
				update.FilePath = testFile
			}

			manager := NewUpdateManager(tempDir)
			if err := manager.ApplyUpdates(context.Background(), tc.updates); err != nil {
				t.Fatalf("ApplyUpdates() error = %v", err)
			}

			got, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read result file: %v", err)
			}

			if string(got) != tc.expected {
				t.Errorf("File content after updates:\ngot:\n%s\nwant:\n%s", string(got), tc.expected)
			}
		})
	}
}
