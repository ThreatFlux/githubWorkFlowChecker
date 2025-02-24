package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"gopkg.in/yaml.v3"
)

func TestPathValidationEdgeCases(t *testing.T) {
	testDir, err := os.MkdirTemp("", "validate-path-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	tests := []struct {
		name     string
		setup    func(string) error
		path     string
		wantErr  bool
		errCheck func(string) bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "empty path not allowed")
			},
		},
		{
			name:    "path with null bytes",
			path:    string([]byte{0}),
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "path contains null bytes")
			},
		},
		{
			name:    "path with invalid characters",
			path:    filepath.Join(testDir, "invalid\x00char"),
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "path contains null bytes")
			},
		},
		{
			name:    "path with double dots",
			path:    filepath.Join(testDir, "..", "..", "outside"),
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "path traversal attempt detected")
			},
		},
		{
			name: "path with symlink outside base",
			setup: func(dir string) error {
				target := filepath.Join(dir, "..", "outside")
				symlink := filepath.Join(dir, "symlink")
				return os.Symlink(target, symlink)
			},
			path:    filepath.Join(testDir, "symlink"),
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "symlink points outside allowed directory")
			},
		},
		{
			name:    "path with spaces",
			path:    filepath.Join(testDir, "path with spaces"),
			wantErr: false,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Generated 1 workflow")
			},
		},
		{
			name:    "path with unicode characters",
			path:    filepath.Join(testDir, "测试目录"),
			wantErr: false,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Generated 1 workflow")
			},
		},
		{
			name:    "path with special characters",
			path:    filepath.Join(testDir, "!@#$%^&*()_+"),
			wantErr: false,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Generated 1 workflow")
			},
		},
		{
			name:    "very long path",
			path:    filepath.Join(testDir, strings.Repeat("a", 1000)),
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "path exceeds maximum length")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(testDir); err != nil {
					t.Fatalf("Failed to setup test: %v", err)
				}
			}

			// Create workflow directory with proper permissions for non-error cases
			if !tt.wantErr {
				workflowDir := filepath.Join(tt.path, ".github", "workflows")
				if err := os.MkdirAll(workflowDir, 0777); err != nil {
					t.Fatalf("Failed to create workflow directory: %v", err)
				}
			}

			os.Args = []string{"cmd", tt.path, "1"}
			exitCode, output := runWithExit(main)

			if tt.wantErr {
				if exitCode == 0 {
					t.Errorf("Expected program to exit with error, got output: %q", output)
				}
			} else {
				if exitCode != 0 {
					t.Errorf("Expected program to succeed, got exit code %d with output: %q", exitCode, output)
				}
				if !tt.errCheck(output) {
					t.Errorf("Unexpected error in output: %q", output)
				}
			}
		})
	}
}

func TestAdvancedTemplateExecution(t *testing.T) {
	testDir, err := os.MkdirTemp("", "template-test-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create workflow directory structure with proper permissions
	workflowDir := filepath.Join(testDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0777); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}

	tests := []struct {
		name     string
		template string
		data     WorkflowData
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name: "invalid template syntax",
			template: `name: Workflow {{.Number}
on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    {{range .Actions}}
      - uses: {{.Owner}/{{.Name}}@{{.Version}} # Missing closing brace
    {{end}}
`,
			data: WorkflowData{
				Number:  1,
				Actions: []Action{{Owner: "actions", Name: "checkout", Version: "v4"}},
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "template: workflow:")
			},
		},
		{
			name: "missing required field",
			template: `name: Workflow {{.Number}}
on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    {{range .Actions}}
      - uses: {{.MissingField}}/{{.Name}}@{{.Version}}
    {{end}}
`,
			data: WorkflowData{
				Number:  1,
				Actions: []Action{{Owner: "actions", Name: "checkout", Version: "v4"}},
			},
			wantErr: true,
			errCheck: func(err error) bool {
				return strings.Contains(err.Error(), "can't evaluate field MissingField")
			},
		},
		{
			name: "invalid action format",
			template: `name: Workflow {{.Number}}
on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    {{range .Actions}}
      - uses: {{.Owner}}{{.Name}}{{.Version}} # Missing separators
    {{end}}
`,
			data: WorkflowData{
				Number:  1,
				Actions: []Action{{Owner: "actions", Name: "checkout", Version: "v4"}},
			},
			wantErr: false,
			errCheck: func(err error) bool {
				return err == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("workflow").Option("missingkey=error").Parse(tt.template)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Failed to parse template: %v", err)
				}
				return
			}

			var output strings.Builder
			err = tmpl.Execute(&output, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected template execution error, got nil")
				} else if !tt.errCheck(err) {
					t.Errorf("Error message did not match expected pattern: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected template error: %v", err)
				}
			}
		})
	}
}

func TestAdvancedWorkflowGeneration(t *testing.T) {
	testDir, err := os.MkdirTemp("", "workflow-test-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	tests := []struct {
		name          string
		workflowCount int
		setup         func(string) error
		verify        func(*testing.T, string, []os.DirEntry)
	}{
		{
			name:          "maximum number of actions",
			workflowCount: 1,
			setup: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0777)
			},
			verify: func(t *testing.T, dir string, files []os.DirEntry) {
				if len(files) != 1 {
					t.Errorf("Expected 1 workflow file, got %d", len(files))
					return
				}

				content, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", files[0].Name()))
				if err != nil {
					t.Errorf("Failed to read workflow file: %v", err)
					return
				}

				var workflow map[string]interface{}
				if err := yaml.Unmarshal(content, &workflow); err != nil {
					t.Errorf("Invalid YAML: %v", err)
					return
				}

				jobs := workflow["jobs"].(map[string]interface{})
				test := jobs["test"].(map[string]interface{})
				steps := test["steps"].([]interface{})

				if len(steps) > 5 {
					t.Errorf("Expected maximum 5 actions, got %d", len(steps))
				}

				// Verify checkout is first action
				firstStep := steps[0].(map[string]interface{})
				uses := firstStep["uses"].(string)
				if !strings.Contains(uses, "actions/checkout@") {
					t.Errorf("First action should be checkout, got: %s", uses)
				}
			},
		},
		{
			name:          "action selection boundaries",
			workflowCount: 10,
			setup: func(dir string) error {
				return os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0777)
			},
			verify: func(t *testing.T, dir string, files []os.DirEntry) {
				if len(files) != 10 {
					t.Errorf("Expected 10 workflow files, got %d", len(files))
					return
				}

				for _, file := range files {
					content, err := os.ReadFile(filepath.Join(dir, ".github", "workflows", file.Name()))
					if err != nil {
						t.Errorf("Failed to read workflow file: %v", err)
						continue
					}

					var workflow map[string]interface{}
					if err := yaml.Unmarshal(content, &workflow); err != nil {
						t.Errorf("Invalid YAML: %v", err)
						continue
					}

					jobs := workflow["jobs"].(map[string]interface{})
					test := jobs["test"].(map[string]interface{})
					steps := test["steps"].([]interface{})

					// Check action count is between 3 and 5
					if len(steps) < 3 || len(steps) > 5 {
						t.Errorf("Action count %d is outside range [3,5]", len(steps))
					}

					// Verify all actions have valid format
					for _, step := range steps {
						stepMap := step.(map[string]interface{})
						uses := stepMap["uses"].(string)
						if !strings.Contains(uses, "/") || !strings.Contains(uses, "@") {
							t.Errorf("Invalid action format: %s", uses)
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new test directory for each test
			testDir, err := os.MkdirTemp("", "workflow-test-*")
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			os.Args = []string{"cmd", testDir, fmt.Sprintf("%d", tt.workflowCount)}
			exitCode, output := runWithExit(main)

			if exitCode != 0 {
				t.Errorf("Expected program to succeed, got exit code %d with output: %q", exitCode, output)
				return
			}

			files, err := os.ReadDir(filepath.Join(testDir, ".github", "workflows"))
			if err != nil {
				t.Fatalf("Failed to read workflow directory: %v", err)
			}

			tt.verify(t, testDir, files)
		})
	}
}

func TestFileOperationErrors(t *testing.T) {
	testDir, err := os.MkdirTemp("", "file-test-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	tests := []struct {
		name     string
		setup    func(string) error
		wantErr  bool
		errCheck func(string) bool
	}{
		{
			name: "parent directory not writable",
			setup: func(dir string) error {
				if err := os.MkdirAll(filepath.Join(dir, ".github"), 0750); err != nil {
					return err
				}
				return os.Chmod(filepath.Join(dir, ".github"), 0500)
			},
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Error creating directory") ||
					strings.Contains(output, "permission denied")
			},
		},
		{
			name: "workflow directory exists but not writable",
			setup: func(dir string) error {
				if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0750); err != nil {
					return err
				}
				return os.Chmod(filepath.Join(dir, ".github", "workflows"), 0500)
			},
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Error creating file") &&
					strings.Contains(output, "directory is not writable")
			},
		},
		{
			name: "workflow file exists and read-only",
			setup: func(dir string) error {
				workflowDir := filepath.Join(dir, ".github", "workflows")
				if err := os.MkdirAll(workflowDir, 0750); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(workflowDir, "workflow-1.yml"), []byte("test"), 0400)
			},
			wantErr: true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Error creating file") &&
					strings.Contains(output, "permission denied")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new test directory for each test
			testDir, err := os.MkdirTemp("", "generate-test-*")
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			// Setup the test case
			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			os.Args = []string{"cmd", testDir, "1"}
			exitCode, output := runWithExit(main)

			if tt.wantErr {
				if exitCode == 0 {
					t.Errorf("Expected program to exit with error, got output: %q", output)
				}
				if !tt.errCheck(output) {
					t.Errorf("Expected error message about permissions, got: %q", output)
				}
			} else {
				if exitCode != 0 {
					t.Errorf("Expected program to succeed, got exit code %d with output: %q", exitCode, output)
				}
			}

			// Reset permissions for cleanup
			// Ignore errors during cleanup as the test is already done
			_ = os.Chmod(filepath.Join(testDir, ".github", "workflows"), 0777)
			_ = os.Chmod(filepath.Join(testDir, ".github"), 0777)
			_ = os.Chmod(testDir, 0777)
		})
	}
}
