package updater

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

// TestScanWorkflows tests the ScanWorkflows function to improve its coverage from 20%
func TestScanWorkflows(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "scanner-scan-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf(common.ErrFailedToRemoveTempDir, err)
		}
	}(tempDir)

	// Create .github/workflows directory
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	testWorkflow := filepath.Join(workflowsDir, "test.yml")
	if err := os.WriteFile(testWorkflow, []byte("name: Test"), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Create another workflow file with .yaml extension
	testWorkflow2 := filepath.Join(workflowsDir, "test2.yaml")
	if err := os.WriteFile(testWorkflow2, []byte("name: Test2"), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Create a non-workflow file
	nonWorkflow := filepath.Join(workflowsDir, "test.txt")
	if err := os.WriteFile(nonWorkflow, []byte("Not a workflow"), 0644); err != nil {
		t.Fatalf("Failed to create non-workflow file: %v", err)
	}

	// Create a directory to test scanning directories
	nestedDir := filepath.Join(workflowsDir, "nested")
	if err := os.Mkdir(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// Create a workflow file in the nested directory
	nestedWorkflow := filepath.Join(nestedDir, "nested.yml")
	if err := os.WriteFile(nestedWorkflow, []byte("name: Nested"), 0644); err != nil {
		t.Fatalf("Failed to create nested workflow file: %v", err)
	}

	// We'll skip creating unreadable files as the behavior is system-dependent
	// and might cause test failures on some platforms

	// Create scanner
	scanner := NewScanner(tempDir)

	// Test scanning workflows directory
	t.Run("scan valid workflows directory", func(t *testing.T) {
		workflows, err := scanner.ScanWorkflows(workflowsDir)
		if err != nil {
			t.Errorf("ScanWorkflows() error = %v", err)
			return
		}

		// Should find 3 workflow files (.yml and .yaml)
		expectedCount := 3
		if len(workflows) != expectedCount {
			t.Errorf("ScanWorkflows() found %d workflows, want %d", len(workflows), expectedCount)
		}

		// Check that all expected files are found
		foundYml := false
		foundYaml := false
		foundNested := false
		for _, workflow := range workflows {
			switch filepath.Base(workflow) {
			case "test.yml":
				foundYml = true
			case "test2.yaml":
				foundYaml = true
			case "nested.yml":
				foundNested = true
			}
		}

		if !foundYml {
			t.Errorf("ScanWorkflows() did not find test.yml")
		}
		if !foundYaml {
			t.Errorf("ScanWorkflows() did not find test2.yaml")
		}
		if !foundNested {
			t.Errorf("ScanWorkflows() did not find nested/nested.yml")
		}
	})

	// Test scanning non-existent directory
	t.Run("scan non-existent directory", func(t *testing.T) {
		nonexistentDir := filepath.Join(tempDir, "nonexistent")
		_, err := scanner.ScanWorkflows(nonexistentDir)
		if err == nil {
			t.Errorf("ScanWorkflows() error = nil, want error for non-existent directory")
		}
	})

	// Test scanning with invalid path
	t.Run("scan with invalid path", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "../../../../../etc/passwd")
		_, err := scanner.ScanWorkflows(invalidPath)
		if err == nil {
			t.Errorf("ScanWorkflows() error = nil, want error for invalid path")
		}
	})

	// Test with a non-workflow directory
	t.Run("scan non-workflow directory", func(t *testing.T) {
		// Create a directory that doesn't have .yml or .yaml files
		emptyDir := filepath.Join(tempDir, "empty-dir")
		if err := os.Mkdir(emptyDir, 0755); err != nil {
			t.Fatalf("Failed to create empty directory: %v", err)
		}

		// Should be successful but return empty list
		workflows, err := scanner.ScanWorkflows(emptyDir)
		if err != nil {
			t.Errorf("ScanWorkflows() on empty dir error = %v, expected success", err)
			return
		}

		if len(workflows) != 0 {
			t.Errorf("ScanWorkflows() on empty dir found %d workflows, want 0", len(workflows))
		}
	})
}

// TestParseActionReferenceFunction tests parseActionReference function to improve coverage from 57.1%
func TestParseActionReferenceFunction(t *testing.T) {
	testCases := []struct {
		name     string
		ref      string
		path     string
		comments []string
		wantErr  bool
	}{
		{
			name:     "valid reference with version",
			ref:      "actions/checkout@v2",
			path:     "workflow.yml",
			comments: []string{},
			wantErr:  false,
		},
		{
			name:     "valid reference with commit hash",
			ref:      "actions/checkout@abc123def456",
			path:     "workflow.yml",
			comments: []string{"# Original version: v2"},
			wantErr:  false,
		},
		{
			name:     "valid multi-part reference",
			ref:      "github/codeql-action/init@v2",
			path:     "workflow.yml",
			comments: []string{},
			wantErr:  false,
		},
		{
			name:     "invalid reference format - no @",
			ref:      "actions/checkout",
			path:     "workflow.yml",
			comments: []string{},
			wantErr:  true,
		},
		{
			name:     "invalid reference format - empty version",
			ref:      "actions/checkout@",
			path:     "workflow.yml",
			comments: []string{},
			wantErr:  true,
		},
		{
			name:     "invalid name format - no slash",
			ref:      "actions@v2",
			path:     "workflow.yml",
			comments: []string{},
			wantErr:  true,
		},
		{
			name:     "invalid name format - empty owner",
			ref:      "checkout@v2", // No slash, should be rejected as not having owner/name format
			path:     "workflow.yml",
			comments: []string{},
			wantErr:  true,
		},
		{
			name:     "no name part",
			ref:      "@v2",
			path:     "workflow.yml",
			comments: []string{},
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseActionReference(tc.ref, tc.path, tc.comments)

			if tc.wantErr {
				if err == nil {
					t.Errorf("parseActionReference(%q) expected error, got nil", tc.ref)
				}
				return
			}

			if err != nil {
				t.Errorf("parseActionReference(%q) unexpected error: %v", tc.ref, err)
				return
			}

			// Verify the returned action reference
			parts := filepath.SplitList(tc.ref)
			if len(parts) > 0 {
				refParts := filepath.SplitList(parts[0])
				if len(refParts) >= 2 {
					ownerAndName := refParts[0]
					nameParts := filepath.SplitList(ownerAndName)

					if result.Owner != nameParts[0] {
						t.Errorf("Expected owner %q, got %q", nameParts[0], result.Owner)
					}

					// Handle multi-part names like github/codeql-action/init
					if strings.Count(tc.ref, "/") > 1 {
						expectedName := strings.Split(tc.ref, "@")[0]
						expectedName = strings.SplitN(expectedName, "/", 2)[1]
						if result.Name != expectedName {
							t.Errorf("Expected name %q, got %q", expectedName, result.Name)
						}
					}
				}
			}

			// Check if comments are properly preserved
			if len(tc.comments) > 0 && len(result.Comments) != len(tc.comments) {
				t.Errorf("Expected %d comments, got %d", len(tc.comments), len(result.Comments))
			}

			// Check for commit hash handling
			if strings.Contains(tc.ref, "@") {
				version := strings.Split(tc.ref, "@")[1]
				if len(version) == 40 && common.IsHexString(version) {
					if result.CommitHash != version {
						t.Errorf("Expected CommitHash %q, got %q", version, result.CommitHash)
					}

					// If there's a comment with "Original version", that should be the version
					for _, comment := range tc.comments {
						if strings.Contains(comment, "Original version:") {
							versionPart := strings.SplitN(comment, ":", 2)[1]
							expectedVersion := strings.TrimSpace(versionPart)
							if result.Version != expectedVersion {
								t.Errorf("Expected Version from comment %q, got %q",
									expectedVersion, result.Version)
							}
							break
						}
					}
				}
			}
		})
	}
}
