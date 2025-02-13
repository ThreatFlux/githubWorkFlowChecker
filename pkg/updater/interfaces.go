package updater

import "context"

// ActionReference represents a GitHub Action reference in a workflow file
type ActionReference struct {
	Owner   string
	Name    string
	Version string
	Path    string
	Line    int
}

// Update represents a pending update for a GitHub Action
type Update struct {
	Action      ActionReference
	OldVersion  string
	NewVersion  string
	FilePath    string
	LineNumber  int
	Description string
}

// VersionChecker checks for newer versions of GitHub Actions
type VersionChecker interface {
	// GetLatestVersion returns the latest version for a given action
	GetLatestVersion(ctx context.Context, action ActionReference) (string, error)

	// IsUpdateAvailable checks if a newer version is available
	IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, error)
}

// PRCreator creates pull requests for GitHub Action updates
type PRCreator interface {
	// CreatePR creates a pull request with the given updates
	CreatePR(ctx context.Context, updates []*Update) error
}

// UpdateManager manages the process of updating GitHub Actions
type UpdateManager interface {
	// CreateUpdate creates an update for a given action and its latest version
	CreateUpdate(ctx context.Context, file string, action ActionReference, latestVersion string) (*Update, error)

	// ApplyUpdates applies the given updates to workflow files
	ApplyUpdates(ctx context.Context, updates []*Update) error
}
