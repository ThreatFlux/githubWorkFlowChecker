package updater

import "context"

// ActionReference represents a GitHub Action reference in a workflow file
type ActionReference struct {
	Owner           string
	Name            string
	Version         string
	CommitHash      string
	Path            string
	Line            int
	Comments        []string
	VersionComment  string // Comment indicating version (e.g., "# v3")
	OriginalVersion string // For tracking version history
}

// Update represents a pending update for a GitHub Action
type Update struct {
	Action          ActionReference
	OldVersion      string
	NewVersion      string
	OldHash         string
	NewHash         string
	FilePath        string
	LineNumber      int
	Description     string
	Comments        []string // Preserved comments
	VersionComment  string   // New version comment
	OriginalVersion string   // For tracking version history
}

// VersionChecker checks for newer versions of GitHub Actions
type VersionChecker interface {
	// GetLatestVersion returns the latest version and its commit hash for a given action
	GetLatestVersion(ctx context.Context, action ActionReference) (version string, hash string, err error)

	// IsUpdateAvailable checks if a newer version is available
	// Returns: available, new version, new hash, error
	IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, string, error)

	// GetCommitHash returns the commit hash for a specific version of an action
	GetCommitHash(ctx context.Context, action ActionReference, version string) (string, error)
}

// PRCreator creates pull requests for GitHub Action updates
type PRCreator interface {
	// CreatePR creates a pull request with the given updates
	CreatePR(ctx context.Context, updates []*Update) error
}

// UpdateManager manages the process of updating GitHub Actions
type UpdateManager interface {
	// CreateUpdate creates an update for a given action and its latest version
	// Now includes commit hash and preserves comments
	CreateUpdate(ctx context.Context, file string, action ActionReference, latestVersion string, commitHash string) (*Update, error)

	// ApplyUpdates applies the given updates to workflow files
	// Now handles comment preservation and hash updates
	ApplyUpdates(ctx context.Context, updates []*Update) error

	// PreserveComments preserves existing comments when updating an action
	PreserveComments(action ActionReference) []string
}
