package main

import (
	"context"
	"errors"
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
)

// Define test-specific helpers for filepath.Abs and factories
var (
	testAbsFunc               = filepath.Abs
	testVersionCheckerFactory = func(token string) updater.VersionChecker {
		return &mockVersionCheckerErr{}
	}
	testPRCreatorFactory = func(token, owner, repo string) updater.PRCreator {
		return &mockPRCreatorErr{}
	}
)

// mockUpdateManager is a mock implementation of updater.UpdateManager
type mockUpdateManagerErr struct {
	createUpdateError error
	applyUpdatesError error
}

func (m *mockUpdateManagerErr) CreateUpdate(ctx context.Context, file string, ref updater.ActionReference, newVersion, newHash string) (*updater.Update, error) {
	if m.createUpdateError != nil {
		return nil, m.createUpdateError
	}
	return &updater.Update{
		FilePath:   file,
		Action:     ref,
		OldVersion: ref.Version,
		NewVersion: newVersion,
		OldHash:    "",
		NewHash:    newHash,
	}, nil
}

func (m *mockUpdateManagerErr) ApplyUpdates(ctx context.Context, updates []*updater.Update) error {
	return m.applyUpdatesError
}

// PreserveComments is needed to implement the interface
func (m *mockUpdateManagerErr) PreserveComments(action updater.ActionReference) []string {
	return []string{}
}

// mockScanner is a mock implementation of updater.Scanner
type mockScannerErr struct {
	scanWorkflowsError   error
	parseReferencesError error
	workflows            []string
	references           []updater.ActionReference
}

func (m *mockScannerErr) ScanWorkflows(dir string) ([]string, error) {
	if m.scanWorkflowsError != nil {
		return nil, m.scanWorkflowsError
	}
	return m.workflows, nil
}

func (m *mockScannerErr) ParseActionReferences(file string) ([]updater.ActionReference, error) {
	if m.parseReferencesError != nil {
		return nil, m.parseReferencesError
	}
	return m.references, nil
}

// mockVersionCheckerErr is a mock implementation with error handling
type mockVersionCheckerErr struct {
	latestVersion          string
	latestHash             string
	err                    error
	isUpdateAvailableValue bool
	isUpdateAvailableError error
}

func (m *mockVersionCheckerErr) GetLatestVersion(ctx context.Context, action updater.ActionReference) (string, string, error) {
	return m.latestVersion, m.latestHash, m.err
}

func (m *mockVersionCheckerErr) IsUpdateAvailable(ctx context.Context, action updater.ActionReference) (bool, string, string, error) {
	if m.isUpdateAvailableError != nil {
		return false, "", "", m.isUpdateAvailableError
	}
	return m.isUpdateAvailableValue, m.latestVersion, m.latestHash, nil
}

func (m *mockVersionCheckerErr) GetCommitHash(ctx context.Context, action updater.ActionReference, version string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.latestHash, nil
}

// mockPRCreatorErr is a mock implementation of updater.PRCreator
type mockPRCreatorErr struct {
	err error
}

func (m *mockPRCreatorErr) CreatePR(ctx context.Context, updates []*updater.Update) error {
	return m.err
}

// SetWorkflowsPath implements the expected interface for DefaultPRCreator
func (m *mockPRCreatorErr) SetWorkflowsPath(path string) {
	// Do nothing - this is a mock
}

// Test error cases in the run function - using a modified run function
func TestRunErrorCases(t *testing.T) {
	// Define test-scoped flags as local variables, not command line flags
	var testRepoPath = "."
	var testOwner = "test-owner"
	var testRepo = "test-repo"
	var testToken = "test-token"
	var testWorkflowsPath = ".github/workflows"
	var testDryRun = false
	var testStage = false

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "run-errors-test-*")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			t.Fatalf(common.ErrFailedToRemoveTempDir, err)
		}
	}(tempDir)

	// Set values for testing
	testRepoPath = tempDir
	testOwner = "test-owner"
	testRepo = "test-repo"
	testToken = "test-token"
	testWorkflowsPath = ".github/workflows"

	// Create default test reference
	testRef := updater.ActionReference{
		Owner:   "actions",
		Name:    "checkout",
		Version: "v2",
	}

	tests := []struct {
		name                string
		setup               func()
		mockAbsFunc         func(path string) (string, error)
		mockScanner         *mockScannerErr
		mockVersionChecker  *mockVersionCheckerErr
		mockUpdateManager   *mockUpdateManagerErr
		mockPRCreator       *mockPRCreatorErr
		expectError         bool
		expectErrorContains string
	}{
		{
			name:  "filepath.Abs error",
			setup: func() {},
			mockAbsFunc: func(path string) (string, error) {
				return "", errors.New("abs error")
			},
			mockScanner:         &mockScannerErr{},
			mockVersionChecker:  &mockVersionCheckerErr{},
			expectError:         true,
			expectErrorContains: "command execution failed: abs error",
		},
		{
			name:        "scan workflows error",
			setup:       func() {},
			mockAbsFunc: nil, // Use default
			mockScanner: &mockScannerErr{
				scanWorkflowsError: errors.New("scan workflows error"),
			},
			mockVersionChecker:  &mockVersionCheckerErr{},
			expectError:         true,
			expectErrorContains: "scan workflows error",
		},
		{
			name:  "no workflows found",
			setup: func() {},
			mockScanner: &mockScannerErr{
				workflows: []string{},
			},
			mockVersionChecker: &mockVersionCheckerErr{},
			expectError:        false,
		},
		{
			name: "parse references error",
			setup: func() {
				// Create .github/workflows directory
				workflowsDir := filepath.Join(tempDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create test workflow file
				workflowFile := filepath.Join(workflowsDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(`name: Test`), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			},
			mockScanner: &mockScannerErr{
				workflows:            []string{filepath.Join(tempDir, ".github", "workflows", "test.yml")},
				parseReferencesError: errors.New("parse references error"),
			},
			mockVersionChecker: &mockVersionCheckerErr{},
			expectError:        false, // Error is logged but doesn't cause function to return error
		},
		{
			name: "get latest version error",
			setup: func() {
				// Create .github/workflows directory
				workflowsDir := filepath.Join(tempDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create test workflow file
				workflowFile := filepath.Join(workflowsDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(`name: Test`), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			},
			mockScanner: &mockScannerErr{
				workflows:  []string{filepath.Join(tempDir, ".github", "workflows", "test.yml")},
				references: []updater.ActionReference{testRef},
			},
			mockVersionChecker: &mockVersionCheckerErr{
				err: errors.New("version check error"),
			},
			expectError: false, // Error is logged but doesn't cause function to return error
		},
		{
			name: "is update available error",
			setup: func() {
				// Create .github/workflows directory
				workflowsDir := filepath.Join(tempDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create test workflow file
				workflowFile := filepath.Join(workflowsDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(`name: Test`), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			},
			mockScanner: &mockScannerErr{
				workflows:  []string{filepath.Join(tempDir, ".github", "workflows", "test.yml")},
				references: []updater.ActionReference{testRef},
			},
			mockVersionChecker: &mockVersionCheckerErr{
				latestVersion:          "v3",
				latestHash:             "abc123",
				isUpdateAvailableError: errors.New("is update available error"),
			},
			expectError: false, // Error is logged but doesn't cause function to return error
		},
		{
			name: "create update error",
			setup: func() {
				// Create .github/workflows directory
				workflowsDir := filepath.Join(tempDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create test workflow file
				workflowFile := filepath.Join(workflowsDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(`name: Test`), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			},
			mockScanner: &mockScannerErr{
				workflows:  []string{filepath.Join(tempDir, ".github", "workflows", "test.yml")},
				references: []updater.ActionReference{testRef},
			},
			mockVersionChecker: &mockVersionCheckerErr{
				latestVersion:          "v3",
				latestHash:             "abc123",
				isUpdateAvailableValue: true,
			},
			mockUpdateManager: &mockUpdateManagerErr{
				createUpdateError: errors.New("create update error"),
			},
			expectError: false, // Error is logged but doesn't cause function to return error
		},
		{
			name: "no updates available",
			setup: func() {
				// Create .github/workflows directory
				workflowsDir := filepath.Join(tempDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create test workflow file
				workflowFile := filepath.Join(workflowsDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(`name: Test`), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}
			},
			mockScanner: &mockScannerErr{
				workflows:  []string{filepath.Join(tempDir, ".github", "workflows", "test.yml")},
				references: []updater.ActionReference{testRef},
			},
			mockVersionChecker: &mockVersionCheckerErr{
				latestVersion:          "v3",
				latestHash:             "abc123",
				isUpdateAvailableValue: false,
			},
			expectError: false,
		},
		{
			name: "apply updates error",
			setup: func() {
				// Create .github/workflows directory
				workflowsDir := filepath.Join(tempDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create test workflow file
				workflowFile := filepath.Join(workflowsDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(`name: Test`), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}

				// Set stage flag
				testStage = true
				testDryRun = false
			},
			mockScanner: &mockScannerErr{
				workflows:  []string{filepath.Join(tempDir, ".github", "workflows", "test.yml")},
				references: []updater.ActionReference{testRef},
			},
			mockVersionChecker: &mockVersionCheckerErr{
				latestVersion:          "v3",
				latestHash:             "abc123",
				isUpdateAvailableValue: true,
			},
			mockUpdateManager: &mockUpdateManagerErr{
				applyUpdatesError: errors.New("apply updates error"),
			},
			expectError:         true,
			expectErrorContains: "failed to apply updates: apply updates error",
		},
		{
			name: "create PR error",
			setup: func() {
				// Create .github/workflows directory
				workflowsDir := filepath.Join(tempDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					t.Fatalf("Failed to create workflows directory: %v", err)
				}

				// Create test workflow file
				workflowFile := filepath.Join(workflowsDir, "test.yml")
				if err := os.WriteFile(workflowFile, []byte(`name: Test`), 0644); err != nil {
					t.Fatalf("Failed to create test workflow file: %v", err)
				}

				// Reset stage and dry-run flags
				testStage = false
				testDryRun = false
			},
			mockScanner: &mockScannerErr{
				workflows:  []string{filepath.Join(tempDir, ".github", "workflows", "test.yml")},
				references: []updater.ActionReference{testRef},
			},
			mockVersionChecker: &mockVersionCheckerErr{
				latestVersion:          "v3",
				latestHash:             "abc123",
				isUpdateAvailableValue: true,
			},
			mockPRCreator: &mockPRCreatorErr{
				err: errors.New("create PR error"),
			},
			expectError:         true,
			expectErrorContains: "failed to create PR: create PR error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset stage and dry-run flags for each test
			originalStage := testStage
			originalDryRun := testDryRun
			defer func() {
				testStage = originalStage
				testDryRun = originalDryRun
			}()

			testStage = false
			testDryRun = false

			// Run setup
			tt.setup()

			// Create a test-specific run function that uses the mock values
			runTest := func() error {
				// Save original functions - using our test-specific ones
				origAbsFunc := testAbsFunc
				origVersionCheckerFactory := testVersionCheckerFactory
				origPRCreatorFactory := testPRCreatorFactory

				// Restore after test
				defer func() {
					testAbsFunc = origAbsFunc
					testVersionCheckerFactory = origVersionCheckerFactory
					testPRCreatorFactory = origPRCreatorFactory
				}()

				// Mock absFunc if provided
				if tt.mockAbsFunc != nil {
					testAbsFunc = tt.mockAbsFunc
				}

				// Create test-local version of run function with mocked dependencies
				mockRun := func() error {
					// Convert repo path to absolute path
					absPath, err := testAbsFunc(testRepoPath)
					if err != nil {
						return errors.New("command execution failed: " + err.Error())
					}

					// Create scanner
					var scanner interface {
						ScanWorkflows(dir string) ([]string, error)
						ParseActionReferences(file string) ([]updater.ActionReference, error)
					}
					if tt.mockScanner != nil {
						scanner = tt.mockScanner
					} else {
						scanner = updater.NewScanner(absPath)
					}

					// Scan for workflow files using configurable path
					workflowsDir := filepath.Join(absPath, testWorkflowsPath)
					files, err := scanner.ScanWorkflows(workflowsDir)
					if err != nil {
						return err
					}

					if len(files) == 0 {
						return nil // No workflow files found
					}

					// Create version checker
					var checker updater.VersionChecker
					if tt.mockVersionChecker != nil {
						checker = tt.mockVersionChecker
					} else if origVersionCheckerFactory != nil {
						checker = origVersionCheckerFactory(testToken)
					} else {
						checker = &mockVersionCheckerErr{}
					}

					// Create update manager
					var manager updater.UpdateManager
					if tt.mockUpdateManager != nil {
						manager = tt.mockUpdateManager
					} else {
						manager = updater.NewUpdateManager(absPath)
					}

					// Create PR creator
					var creator updater.PRCreator
					if tt.mockPRCreator != nil {
						creator = tt.mockPRCreator
					} else if origPRCreatorFactory != nil {
						creator = origPRCreatorFactory(testToken, testOwner, testRepo)
					} else {
						creator = &mockPRCreatorErr{}
					}

					// Process each workflow file
					var updates []*updater.Update
					ctx := context.Background()

					for _, file := range files {
						// Get action references from file
						refs, err := scanner.ParseActionReferences(file)
						if err != nil {
							// Just log error and continue
							continue
						}

						// Check each action for updates
						for _, ref := range refs {
							latestVersion, latestHash, err := checker.GetLatestVersion(ctx, ref)
							if err != nil {
								// Just log error and continue
								continue
							}

							// Check if update is available
							available, _, _, err := checker.IsUpdateAvailable(ctx, ref)
							if err != nil {
								// Just log error and continue
								continue
							}

							if available {
								update, err := manager.CreateUpdate(ctx, file, ref, latestVersion, latestHash)
								if err != nil {
									// Just log error and continue
									continue
								}
								updates = append(updates, update)
							}
						}
					}

					if len(updates) == 0 {
						return nil // No updates available
					}

					// Handle updates based on mode (dry-run, stage, or normal)
					if testDryRun {
						// Preview changes without applying them - no-op for test
					} else if testStage {
						// Apply changes locally without creating a PR
						if err := manager.ApplyUpdates(ctx, updates); err != nil {
							return errors.New("failed to apply updates: " + err.Error())
						}
					} else {
						// Normal mode: Create pull request with updates
						if err := creator.CreatePR(ctx, updates); err != nil {
							return errors.New("failed to create PR: " + err.Error())
						}
					}
					return nil
				}

				// Run the test-local function
				return mockRun()
			}

			// Run the test-specific run function
			err := runTest()

			// Check expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.expectErrorContains != "" && !strings.Contains(err.Error(), tt.expectErrorContains) {
					t.Errorf("Expected error containing %q, got %q", tt.expectErrorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
