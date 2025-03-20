package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
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
	// Check for path length
	if len(path) > maxPathLength {
		return fmt.Errorf(common.ErrPathExceedsMaxLength, maxPathLength)
	}

	// Use common path validation utility
	options := common.PathValidationOptions{
		AllowNonExistent: true,
		CheckSymlinks:    true,
	}
	return common.ValidatePath(base, path, options)
}

// SysOutCall handling for os.Stdout.Sync() call
// This is a no-op in test environments to avoid "bad file descriptor" errors
var inTestMode = false

func SysOutCall() {
	if !inTestMode {
		_ = os.Stdout.Sync()
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run generate-test-data.go <output-dir> <workflow-count>")
		SysOutCall()
		osExit(1)
		return
	}

	// Parse workflow count first to avoid path validation errors
	count, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Printf(common.ErrInvalidTestParameters+"\n", "workflow count: "+err.Error())
		SysOutCall()
		osExit(1)
		return
	}

	if count <= 0 {
		fmt.Println(common.ErrWorkflowCountMustBePositive)
		SysOutCall()
		osExit(1)
		return
	}

	if count > maxWorkflowCount {
		fmt.Printf(common.ErrWorkflowCountExceedsLimit+"\n", maxWorkflowCount)
		SysOutCall()
		osExit(1)
		return
	}

	outputDir := os.Args[1] // Don't clean the path before validation

	// Get temporary directory as base for path validation
	tempDir := os.TempDir()

	// First validate the output directory path
	if err := validatePath(tempDir, outputDir); err != nil {
		fmt.Printf("invalid directory path: %v\n", err)
		SysOutCall()
		osExit(1)
		return
	}

	// Clean the path after validation
	outputDir = filepath.Clean(outputDir)

	// Create output directory structure using common file utilities
	workflowDir := filepath.Join(outputDir, ".github", "workflows")
	fileOptions := common.FileOptions{
		BaseDir:    tempDir,
		CreateDirs: true,
		Mode:       0750,
		ValidateOptions: common.PathValidationOptions{
			AllowNonExistent: true,
			CheckSymlinks:    true,
		},
	}

	// Create a dummy file to ensure all directories are created
	dummyFile := filepath.Join(workflowDir, ".gitkeep")
	if err := common.WriteFileWithOptions(dummyFile, []byte(""), fileOptions); err != nil {
		fmt.Printf("error creating directories: %v\n", err)
		SysOutCall()
		osExit(1)
		return
	}

	// Remove the dummy file
	if err := os.Remove(dummyFile); err != nil {
		fmt.Printf(common.ErrCouldNotRemoveDummyFile+"\n", err)
	}

	// Parse template
	tmpl, err := template.New("workflow").Option("missingkey=error").Parse(workflowTemplate)
	if err != nil {
		fmt.Printf("error generating test data: %v\n", err)
		SysOutCall()
		osExit(1)
		return
	}

	// Generate workflow files
	// workflowDir is already defined above
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
			fmt.Printf("invalid file path: %v\n", err)
			SysOutCall()
			osExit(1)
			return
		}

		// Use a buffer to execute the template
		var buffer strings.Builder
		if err := tmpl.Execute(&buffer, data); err != nil {
			fmt.Printf("error generating test data: %v\n", err)
			SysOutCall()
			osExit(1)
			return
		}

		// Check if the file exists and is read-only
		if info, err := os.Stat(filename); err == nil {
			if info.Mode().Perm()&0200 == 0 {
				fmt.Println("error creating destination file: file exists and is read-only")
				SysOutCall()
				osExit(1)
				return
			}
		}

		// Write the file using common file utilities
		fileOptions := common.FileOptions{
			BaseDir: outputDir,
			Mode:    0400,
			ValidateOptions: common.PathValidationOptions{
				AllowNonExistent: true,
				CheckSymlinks:    true,
			},
		}

		if err := common.WriteFileWithOptions(filename, []byte(buffer.String()), fileOptions); err != nil {
			fmt.Printf("error creating destination file: %v\n", err)
			SysOutCall()
			osExit(1)
			return
		}

		successCount++
		// Clear used actions for the next workflow
		usedActions = make(map[string]bool)
	}

	fmt.Printf("Generated %d workflow files in %s\n", successCount, workflowDir)
	SysOutCall()
}
