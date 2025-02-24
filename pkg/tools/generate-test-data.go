package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

// osExit is used to make the exit function testable
var osExit = os.Exit

const (
	workflowTemplate = `name: Workflow {{.Number}}

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
	maxWorkflowCount = 1000 // Maximum number of workflows allowed
	maxPathLength    = 255  // Maximum path length
)

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

// validatePath checks if the given path is safe to use
func validatePath(base, path string) error {
	// Check for empty path first, before any cleaning
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("empty path not allowed")
	}

	// Check for null bytes in both base and path
	if strings.ContainsRune(base, 0) || strings.ContainsRune(path, 0) {
		return fmt.Errorf("path contains null bytes")
	}

	if len(path) > maxPathLength {
		return fmt.Errorf("path exceeds maximum length of %d characters", maxPathLength)
	}

	// Clean and resolve both paths
	cleanBase := filepath.Clean(base)
	cleanPath := filepath.Clean(path)

	// Convert to absolute paths if needed
	absBase := cleanBase
	if !filepath.IsAbs(cleanBase) {
		var err error
		absBase, err = filepath.Abs(cleanBase)
		if err != nil {
			return fmt.Errorf("failed to resolve base path: %v", err)
		}
	}

	absPath := cleanPath
	if !filepath.IsAbs(cleanPath) {
		var err error
		absPath, err = filepath.Abs(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %v", err)
		}
	}

	// Check if path is within base directory
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return fmt.Errorf("failed to determine relative path: %v", err)
	}

	if strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "/") {
		return fmt.Errorf("path traversal attempt detected")
	}

	// Check for symlinks that point outside the base directory
	// Only if the path exists and is a symlink
	if info, err := os.Lstat(absPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		evalPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return fmt.Errorf("failed to evaluate symlink: %v", err)
		}
		if err := validatePath(base, evalPath); err != nil {
			return fmt.Errorf("symlink points outside allowed directory: %v", err)
		}
	}

	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run generate-test-data.go <output-dir> <workflow-count>")
		os.Stdout.Sync()
		osExit(1)
		return
	}

	// Parse workflow count first to avoid path validation errors
	count, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("Error parsing count:", err)
		os.Stdout.Sync()
		osExit(1)
		return
	}

	if count <= 0 {
		fmt.Println("Workflow count must be positive")
		os.Stdout.Sync()
		osExit(1)
		return
	}

	if count > maxWorkflowCount {
		fmt.Printf("Workflow count exceeds maximum limit of %d\n", maxWorkflowCount)
		os.Stdout.Sync()
		osExit(1)
		return
	}

	outputDir := os.Args[1] // Don't clean the path before validation

	// Get temporary directory as base for path validation
	tempDir := os.TempDir()

	// First validate the output directory path
	if err := validatePath(tempDir, outputDir); err != nil {
		fmt.Printf("Invalid workflow directory path: %v\n", err)
		os.Stdout.Sync()
		osExit(1)
		return
	}

	// Clean the path after validation
	outputDir = filepath.Clean(outputDir)

	// Create output directory structure
	dirs := []string{
		outputDir,
		filepath.Join(outputDir, ".github"),
		filepath.Join(outputDir, ".github", "workflows"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Stdout.Sync()
			osExit(1)
			return
		}
	}

	// Parse template
	tmpl, err := template.New("workflow").Option("missingkey=error").Parse(workflowTemplate)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Stdout.Sync()
		osExit(1)
		return
	}

	// Generate workflow files
	workflowDir := filepath.Join(outputDir, ".github", "workflows")
	successCount := 0

	// Keep track of used actions to avoid duplicates
	usedActions := make(map[string]bool)

	for i := 1; i <= count; i++ {
		// Select 3-5 actions for each workflow (including checkout)
		actionCount := 2 + (i % 3) // 2-4 additional actions + checkout = 3-5 total
		actions := make([]Action, 0, actionCount+1)

		// Always add checkout as first action
		actions = append(actions, commonActions[0]) // checkout action
		usedActions[fmt.Sprintf("%s/%s@%s", commonActions[0].Owner, commonActions[0].Name, commonActions[0].Version)] = true

		// Add remaining actions, skipping checkout and avoiding duplicates
		for j := 0; len(actions) < actionCount+1; j++ {
			actionIndex := 1 + ((i + j) % (len(commonActions) - 1)) // Skip index 0 (checkout)
			action := commonActions[actionIndex]
			key := fmt.Sprintf("%s/%s@%s", action.Owner, action.Name, action.Version)
			if !usedActions[key] {
				actions = append(actions, action)
				usedActions[key] = true
			}
		}

		data := WorkflowData{
			Number:  i,
			Actions: actions,
		}

		filename := filepath.Join(workflowDir, fmt.Sprintf("workflow-%d.yml", i))

		// Validate the file path
		if err := validatePath(outputDir, filename); err != nil {
			fmt.Printf("Invalid file path %s: %v\n", filename, err)
			os.Stdout.Sync()
			osExit(1)
			return
		}

		// Check if directory is writable
		if info, err := os.Stat(workflowDir); err == nil {
			if info.Mode().Perm()&0200 == 0 {
				fmt.Printf("Error creating file %s: directory is not writable\n", filename)
				os.Stdout.Sync()
				osExit(1)
				return
			}
		}

		file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0400)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", filename, err)
			os.Stdout.Sync()
			osExit(1)
			return
		}

		if err := tmpl.Execute(file, data); err != nil {
			fmt.Printf("Error generating workflow %d: %v\n", i, err)
			if closeErr := file.Close(); closeErr != nil {
				fmt.Printf("Error closing file after template error: %v\n", closeErr)
			}
			os.Stdout.Sync()
			osExit(1)
			return
		}

		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %v\n", err)
			os.Stdout.Sync()
			osExit(1)
			return
		}

		successCount++
		// Clear used actions for the next workflow
		usedActions = make(map[string]bool)
	}

	fmt.Printf("Generated %d workflow files in %s\n", successCount, workflowDir)
	os.Stdout.Sync()
}
