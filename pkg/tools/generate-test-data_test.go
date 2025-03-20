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

func TestWorkflowGeneration(t *testing.T) {
	tests := []struct {
		name          string
		workflowCount int
		wantErr       bool
		errCheck      func(string) bool
	}{
		{
			name:          "generate single workflow",
			workflowCount: 1,
			wantErr:       false,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Generated 1 workflow")
			},
		},
		{
			name:          "generate multiple workflows",
			workflowCount: 5,
			wantErr:       false,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Generated 5 workflow")
			},
		},
		{
			name:          "generate large number of workflows",
			workflowCount: 100,
			wantErr:       false,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Generated 100 workflow")
			},
		},
		{
			name:          "generate maximum workflows",
			workflowCount: 1000,
			wantErr:       false,
			errCheck: func(output string) bool {
				return strings.Contains(output, "Generated 1000 workflow")
			},
		},
		{
			name:          "exceed maximum workflows",
			workflowCount: 1001,
			wantErr:       true,
			errCheck: func(output string) bool {
				return strings.Contains(output, "exceeds maximum limit")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp("", "generate-test-*")
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			defer func(path string) {
				err := os.RemoveAll(path)
				if err != nil {
					t.Fatalf("Failed to remove temp dir: %v", err)
				}
			}(testDir)

			// Create workflow directory structure
			workflowDir := filepath.Join(testDir, ".github", "workflows")
			if err := os.MkdirAll(workflowDir, 0777); err != nil {
				t.Fatalf("Failed to create workflow directory: %v", err)
			}

			os.Args = []string{"cmd", testDir, fmt.Sprintf("%d", tt.workflowCount)}

			exitCode, output := runWithExit(main)
			t.Log("Output:", output)

			if tt.wantErr {
				if exitCode == 0 {
					t.Error("Expected program to exit with error")
				}
				if !tt.errCheck(output) {
					t.Errorf("Expected error message about maximum limit, got: %q", output)
				}
				return
			}

			if exitCode != 0 {
				t.Errorf("Expected program to succeed, got exit code %d with output: %q", exitCode, output)
				return
			}

			if !tt.errCheck(output) {
				t.Errorf("Expected success message, got: %q", output)
				return
			}

			// Verify results
			files, err := os.ReadDir(workflowDir)
			if err != nil {
				t.Fatalf("Failed to read workflow directory: %v", err)
			}

			count := len(files)
			if count != tt.workflowCount {
				t.Errorf("Expected %d workflow files, got %d", tt.workflowCount, count)
			}

			// Verify file contents
			for _, file := range files {
				filePath := filepath.Join(workflowDir, file.Name())
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Errorf("Failed to read workflow file %s: %v", file.Name(), err)
					continue
				}

				var workflow map[string]interface{}
				if err := yaml.Unmarshal(content, &workflow); err != nil {
					t.Errorf("Invalid YAML in file %s: %v", file.Name(), err)
					continue
				}

				verifyWorkflowStructure(t, workflow, file.Name())
			}
		})
	}
}

func TestInvalidArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "no arguments",
			args:    []string{"cmd"},
			wantErr: "Usage: go run generate-test-data.go <output-dir> <workflow-count>",
		},
		{
			name:    "missing workflow count",
			args:    []string{"cmd", "output-dir"},
			wantErr: "Usage: go run generate-test-data.go <output-dir> <workflow-count>",
		},
		{
			name:    "invalid workflow count",
			args:    []string{"cmd", "output-dir", "invalid"},
			wantErr: "Error parsing count",
		},
		{
			name:    "negative workflow count",
			args:    []string{"cmd", "output-dir", "-1"},
			wantErr: "Workflow count must be positive",
		},
		{
			name:    "zero workflow count",
			args:    []string{"cmd", "output-dir", "0"},
			wantErr: "Workflow count must be positive",
		},
		{
			name:    "non-numeric workflow count",
			args:    []string{"cmd", "output-dir", "abc"},
			wantErr: "Error parsing count",
		},
		{
			name:    "float workflow count",
			args:    []string{"cmd", "output-dir", "1.5"},
			wantErr: "Error parsing count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			exitCode, output := runWithExit(main)
			if exitCode == 0 {
				t.Error("Expected program to exit with error")
			}

			if !strings.Contains(output, tt.wantErr) {
				t.Errorf("Expected error message containing %q, got %q", tt.wantErr, output)
			}
		})
	}
}

func TestTemplateValidity(t *testing.T) {
	// Test template execution with various data
	tests := []struct {
		name        string
		data        WorkflowData
		template    string
		wantErr     bool
		wantContent string
	}{
		{
			name: "minimal workflow",
			data: WorkflowData{
				Number:  1,
				Actions: []Action{{Owner: "actions", Name: "checkout", Version: "v4"}},
			},
			template: workflowTemplate,
			wantErr:  false,
		},
		{
			name: "maximum actions",
			data: WorkflowData{
				Number:  2,
				Actions: make([]Action, 5),
			},
			template: workflowTemplate,
			wantErr:  false,
		},
		{
			name: "template execution error",
			data: WorkflowData{
				Number: 1,
				Actions: []Action{{
					Owner:   "test",
					Name:    "test",
					Version: "v1",
				}},
			},
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("workflow").Option("missingkey=error").Parse(tt.template)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}

			var output strings.Builder
			err = tmpl.Execute(&output, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected template execution error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected template error: %v", err)
				} else {
					result := output.String()
					if !strings.Contains(result, fmt.Sprintf("name: Workflow %d", tt.data.Number)) {
						t.Errorf("Template output missing workflow number")
					}
				}
			}
		})
	}
}

func TestActionSelection(t *testing.T) {
	// Test action selection boundaries
	t.Run("action count range", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			actionCount := 2 + (i % 3) // 2-4 additional actions + checkout = 3-5 total
			actions := make([]Action, 0, actionCount+1)
			usedActions := make(map[string]bool)

			// Always add checkout as first action
			actions = append(actions, commonActions[0])
			usedActions[fmt.Sprintf("%s/%s@%s", commonActions[0].Owner, commonActions[0].Name, commonActions[0].Version)] = true

			// Add remaining actions, skipping checkout and avoiding duplicates
			for j := 0; len(actions) < actionCount+1; j++ {
				actionIndex := 1 + ((i + j) % (len(commonActions) - 1))
				action := commonActions[actionIndex]
				key := fmt.Sprintf("%s/%s@%s", action.Owner, action.Name, action.Version)
				if !usedActions[key] {
					actions = append(actions, action)
					usedActions[key] = true
				}
			}

			total := len(actions)
			if total < 3 || total > 5 {
				t.Errorf("Iteration %d: action count %d is outside expected range [3,5]", i, total)
			}

			// Verify checkout is always first
			if actions[0] != commonActions[0] {
				t.Errorf("Iteration %d: first action is not checkout", i)
			}

			// Verify no duplicate actions
			seen := make(map[string]bool)
			for _, action := range actions {
				key := fmt.Sprintf("%s/%s@%s", action.Owner, action.Name, action.Version)
				if seen[key] {
					t.Errorf("Iteration %d: duplicate action %s", i, key)
				}
				seen[key] = true
			}
		}
	})
}

func verifyWorkflowStructure(t *testing.T, workflow map[string]interface{}, filename string) {
	// Verify required fields
	requiredFields := []string{"name", "on", "jobs"}
	for _, field := range requiredFields {
		if workflow[field] == nil {
			t.Errorf("Workflow %s missing '%s' field", filename, field)
		}
	}

	// Verify action references
	jobs := workflow["jobs"].(map[string]interface{})
	test := jobs["test"].(map[string]interface{})
	steps := test["steps"].([]interface{})

	// Check if number of actions is in expected range (3-5)
	actionCount := len(steps)
	if actionCount < 3 || actionCount > 5 {
		t.Errorf("Workflow %s has %d actions, expected 3-5", filename, actionCount)
	}

	// Verify action format
	for _, step := range steps {
		stepMap := step.(map[string]interface{})
		uses := stepMap["uses"].(string)
		if !strings.Contains(uses, "@") {
			t.Errorf("Invalid action reference format in %s: %s", filename, uses)
		}
	}
}
