package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestWorkflowGeneration(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "workflow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	// Test cases
	tests := []struct {
		name          string
		workflowCount int
		wantErr       bool
	}{
		{
			name:          "generate single workflow",
			workflowCount: 1,
			wantErr:       false,
		},
		{
			name:          "generate multiple workflows",
			workflowCount: 5,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
			testDir := filepath.Join(tempDir, tt.name)
			os.Args = []string{"cmd", testDir, fmt.Sprintf("%d", tt.workflowCount)}

			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run main
			func() {
				defer func() {
				}()
				main()
			}()

			// Restore stdout
			err := w.Close()
			if err != nil {
				return
			}
			os.Stdout = oldStdout

			// Read output
			var output strings.Builder
			if _, err := io.Copy(&output, r); err != nil {
				t.Errorf("Failed to capture output: %v", err)
			}

			// Check results
			if tt.wantErr {
				t.Errorf("Expected error for count %d, got success", tt.workflowCount)
				return
			}

			// Verify directory structure
			workflowDir := filepath.Join(testDir, ".github", "workflows")
			if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
				t.Errorf("Workflow directory not created: %v", err)
			}

			// Count and verify generated files
			files, err := os.ReadDir(workflowDir)
			if err != nil {
				t.Fatalf("Failed to read workflow directory: %v", err)
			}

			if len(files) != tt.workflowCount {
				t.Errorf("Expected %d workflow files, got %d", tt.workflowCount, len(files))
			}

			// Verify file contents
			for _, file := range files {
				filePath := filepath.Join(workflowDir, file.Name())
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read workflow file %s: %v", file.Name(), err)
					continue
				}

				// Verify YAML structure
				var workflow map[string]interface{}
				if err := yaml.Unmarshal(content, &workflow); err != nil {
					t.Errorf("Invalid YAML in file %s: %v", file.Name(), err)
					continue
				}

				// Verify required fields
				if workflow["name"] == nil {
					t.Errorf("Workflow %s missing 'name' field", file.Name())
				}
				if workflow["on"] == nil {
					t.Errorf("Workflow %s missing 'on' field", file.Name())
				}
				if workflow["jobs"] == nil {
					t.Errorf("Workflow %s missing 'jobs' field", file.Name())
				}

				// Verify action references
				jobs := workflow["jobs"].(map[string]interface{})
				test := jobs["test"].(map[string]interface{})
				steps := test["steps"].([]interface{})

				// Check if number of actions is in expected range (3-5)
				actionCount := len(steps)
				if actionCount < 3 || actionCount > 5 {
					t.Errorf("Workflow %s has %d actions, expected 3-5", file.Name(), actionCount)
				}

				// Verify action format
				for _, step := range steps {
					stepMap := step.(map[string]interface{})
					uses := stepMap["uses"].(string)
					if !strings.Contains(uses, "@") {
						t.Errorf("Invalid action reference format in %s: %s", file.Name(), uses)
					}
				}
			}
		})
	}
}

func TestTemplateValidity(t *testing.T) {
	// Test template parsing
	_, err := template.New("workflow").Parse(workflowTemplate)
	if err != nil {
		t.Errorf("Invalid workflow template: %v", err)
	}
}

func TestActionSelection(t *testing.T) {
	// Verify unique combinations of actions
	seen := make(map[string]bool)
	for i := 1; i <= 10; i++ {
		actionCount := 3 + (i % 3)
		actions := make([]Action, actionCount)
		for j := 0; j < actionCount; j++ {
			actions[j] = commonActions[(i+j)%len(commonActions)]
		}

		// Create a string representation of the action combination
		key := ""
		for _, action := range actions {
			key += action.Owner + "/" + action.Name + "@" + action.Version + ","
		}

		if seen[key] {
			t.Errorf("Duplicate action combination found: %s", key)
		}
		seen[key] = true
	}
}

func TestPermissions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "workflow-perm-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf("Failed to remove temp directory: %v", err)
		}
	}(tempDir)

	os.Args = []string{"cmd", tempDir, "1"}
	main()

	// Check directory permissions
	workflowDir := filepath.Join(tempDir, ".github", "workflows")
	info, err := os.Stat(workflowDir)
	if err != nil {
		t.Fatalf("Failed to stat workflow directory: %v", err)
	}

	if info.Mode().Perm() != 0750 {
		t.Errorf("Expected directory permissions 0750, got %v", info.Mode().Perm())
	}

	// Check file permissions
	files, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatalf("Failed to read workflow directory: %v", err)
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			t.Errorf("Failed to get file info: %v", err)
			continue
		}

		if info.Mode().Perm() != 0400 {
			t.Errorf("Expected file permissions 0400, got %v for %s", info.Mode().Perm(), file.Name())
		}
	}
}
