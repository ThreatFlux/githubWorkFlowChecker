package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const workflowTemplate = `name: Workflow {{.Number}}

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
      - uses: {{.Owner}}/{{.Name}}@{{.Version}}
    {{end}}
`

type Action struct {
	Owner   string
	Name    string
	Version string
}

type WorkflowData struct {
	Number  int
	Actions []Action
}

var commonActions = []Action{
	{Owner: "actions", Name: "checkout", Version: "v4"},
	{Owner: "actions", Name: "setup-node", Version: "v4"},
	{Owner: "actions", Name: "setup-python", Version: "v5"},
	{Owner: "actions", Name: "setup-go", Version: "v5"},
	{Owner: "actions", Name: "cache", Version: "v3"},
	{Owner: "actions", Name: "upload-artifact", Version: "v4"},
	{Owner: "actions", Name: "download-artifact", Version: "v4"},
	{Owner: "docker", Name: "build-push-action", Version: "v5"},
	{Owner: "docker", Name: "login-action", Version: "v3"},
	{Owner: "docker", Name: "metadata-action", Version: "v5"},
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run generate-test-data.go <output-dir> <workflow-count>")
		os.Exit(1)
	}

	outputDir := os.Args[1]
	count := 0
	if _, err := fmt.Sscanf(os.Args[2], "%d", &count); err != nil {
		fmt.Printf("Error parsing count: %v\n", err)
		os.Exit(1)
	}

	if count <= 0 {
		fmt.Println("Workflow count must be positive")
		os.Exit(1)
	}

	// Create output directory structure
	dirs := []string{
		outputDir,
		filepath.Join(outputDir, ".github"),
		filepath.Join(outputDir, ".github", "workflows"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Parse template
	tmpl, err := template.New("workflow").Parse(workflowTemplate)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Generate workflow files
	workflowDir := filepath.Join(outputDir, ".github", "workflows")
	for i := 1; i <= count; i++ {
		// Select 3-5 actions for each workflow (including checkout)
		actionCount := 2 + (i % 3) // 2-4 additional actions + checkout = 3-5 total
		actions := make([]Action, actionCount+1)

		// Always add checkout as first action
		actions[0] = commonActions[0] // checkout action

		// Add remaining actions, skipping checkout
		for j := 0; j < actionCount; j++ {
			actionIndex := 1 + ((i + j) % (len(commonActions) - 1)) // Skip index 0 (checkout)
			actions[j+1] = commonActions[actionIndex]
		}

		data := WorkflowData{
			Number:  i,
			Actions: actions,
		}

		filename := filepath.Join(workflowDir, fmt.Sprintf("workflow-%d.yml", i))

		// Validate the file path is within the intended directory
		if !strings.HasPrefix(filename, workflowDir) {
			fmt.Printf("Error: Invalid file path %s\n", filename)
			continue
		}
		filepath.Clean(filename)
		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", filename, err)
			continue
		}

		if err := tmpl.Execute(file, data); err != nil {
			fmt.Printf("Error generating workflow %d: %v\n", i, err)
			if closeErr := file.Close(); closeErr != nil {
				fmt.Printf("Error closing file after template error: %v\n", closeErr)
			}
			continue
		}

		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %v\n", err)
			continue
		}
	}

	fmt.Printf("Generated %d workflow files in %s\n", count, workflowDir)
}
