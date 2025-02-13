package updater

import "context"

// WorkflowScanner defines the interface for scanning GitHub Actions workflow files
type WorkflowScanner interface {
	// ScanWorkflows scans the repository for workflow files and returns their paths
	ScanWorkflows(ctx context.Context, repoPath string) ([]string, error)
	// ParseWorkflow parses a workflow file and returns the actions it uses
	ParseWorkflow(ctx context.Context, path string) ([]ActionReference, error)
}

// VersionChecker defines the interface for checking action versions
type VersionChecker interface {
	// GetLatestVersion returns the latest version for a given action
	GetLatestVersion(ctx context.Context, action ActionReference) (string, error)
	// IsUpdateAvailable checks if a newer version is available
	IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, error)
}

// UpdateManager defines the interface for managing workflow updates
type UpdateManager interface {
	// CreateUpdate creates an update for an action in a workflow file
	CreateUpdate(ctx context.Context, path string, action ActionReference, newVersion string) (*Update, error)
	// ApplyUpdates applies multiple updates to workflow files
	ApplyUpdates(ctx context.Context, updates []*Update) error
}

// PRCreator defines the interface for creating pull requests
type PRCreator interface {
	// CreatePR creates a pull request with the given updates
	CreatePR(ctx context.Context, updates []*Update) error
}

// ActionReference represents a GitHub Action reference in a workflow
type ActionReference struct {
	Owner   string // e.g., "actions" in actions/checkout
	Name    string // e.g., "checkout" in actions/checkout
	Version string // e.g., "v2" or commit SHA
	Path    string // Path to the workflow file containing this reference
	Line    int    // Line number where this reference appears
}

// Update represents a version update for an action
type Update struct {
	Action      ActionReference
	OldVersion  string
	NewVersion  string
	FilePath    string
	LineNumber  int
	Description string
}
