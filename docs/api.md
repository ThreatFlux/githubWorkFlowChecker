# GitHub Actions Workflow Checker API Documentation

## Package `updater`

The `updater` package provides the core functionality for scanning, checking, and updating GitHub Actions workflow files. It offers a modular architecture through well-defined interfaces that handle different aspects of the update process.

## CLI Interface

The tool provides a command-line interface with the following flags:

```
-owner string
    Repository owner (required)
-repo-name string
    Repository name (required)
-token string
    GitHub token (required, can also be set via GITHUB_TOKEN environment variable)
-repo string
    Path to repository (default: ".")
-version
    Print version information and exit
```

Version information includes:
- Version number (e.g., "20250215.release.1")
- Commit hash of the build

Example version output:
```
ghactions-updater version 20250215.release.1 (commit: abc123)
```

## Core Types

### ActionReference

```go
type ActionReference struct {
    Owner           string   // GitHub owner/organization of the action
    Name            string   // Name of the action
    Version         string   // Current version of the action
    CommitHash      string   // Commit hash for the action (if using commit reference)
    Path            string   // Path to the workflow file
    Line            int      // Line number in the workflow file
    Comments        []string // Comments associated with the action
    VersionComment  string   // Comment indicating version (e.g., "# v3")
    OriginalVersion string   // For tracking version history
}
```

Represents a GitHub Action reference found in a workflow file. This structure contains all necessary information to identify and locate an action within a workflow.

### Update

```go
type Update struct {
    Action          ActionReference // The action to be updated
    OldVersion      string         // Current version of the action
    NewVersion      string         // Version to update to
    OldHash         string         // Current commit hash (if using commit reference)
    NewHash         string         // New commit hash to update to
    FilePath        string         // Path to the workflow file
    LineNumber      int           // Line number where the update should occur
    Description     string        // Human-readable description of the update
    Comments        []string      // Preserved comments
    VersionComment  string        // New version comment
    OriginalVersion string        // For tracking version history
}
```

Represents a pending update for a GitHub Action, containing all information needed to perform the update and create a pull request.

## Core Interfaces

### VersionChecker

```go
type VersionChecker interface {
    // GetLatestVersion returns the latest version and its commit hash for a given action
    GetLatestVersion(ctx context.Context, action ActionReference) (version string, hash string, error error)

    // IsUpdateAvailable checks if a newer version is available
    // Returns: available, new version, new hash, error
    IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, string, error)

    // GetCommitHash returns the commit hash for a specific version of an action
    GetCommitHash(ctx context.Context, action ActionReference, version string) (string, error)
}
```

Responsible for checking the availability of newer versions for GitHub Actions.

#### Methods

- `GetLatestVersion`: Retrieves the latest available version for a given action.
  - Parameters:
    - `ctx`: Context for the operation
    - `action`: ActionReference to check
  - Returns:
    - `string`: Latest version available
    - `error`: Any error encountered

- `IsUpdateAvailable`: Checks if a newer version exists for a given action.
  - Parameters:
    - `ctx`: Context for the operation
    - `action`: ActionReference to check
  - Returns:
    - `bool`: True if an update is available
    - `string`: Latest version (if available)
    - `error`: Any error encountered

### PRCreator

```go
type PRCreator interface {
    CreatePR(ctx context.Context, updates []*Update) error
}
```

Handles the creation of pull requests for action updates.

#### Methods

- `CreatePR`: Creates a pull request containing the specified updates.
  - Parameters:
    - `ctx`: Context for the operation
    - `updates`: Slice of Update objects to include in the PR
  - Returns:
    - `error`: Any error encountered during PR creation

### UpdateManager

```go
type UpdateManager interface {
    // CreateUpdate creates an update for a given action and its latest version
    // Now includes commit hash and preserves comments
    CreateUpdate(ctx context.Context, file string, action ActionReference, latestVersion string, commitHash string) (*Update, error)
    ApplyUpdates(ctx context.Context, updates []*Update) error
}
```

Manages the overall process of creating and applying updates to workflow files.

#### Methods

- `CreateUpdate`: Creates an Update object for a specific action and version.
  - Parameters:
    - `ctx`: Context for the operation
    - `file`: Path to the workflow file
    - `action`: ActionReference to update
    - `latestVersion`: Version to update to
  - Returns:
    - `*Update`: Created Update object
    - `error`: Any error encountered

- `ApplyUpdates`: Applies a set of updates to workflow files.
  - Parameters:
    - `ctx`: Context for the operation
    - `updates`: Slice of Update objects to apply
  - Returns:
    - `error`: Any error encountered during update application

## Usage Examples

### Checking for Updates

```go
func checkForUpdates(checker VersionChecker, action ActionReference) {
    ctx := context.Background()
    
    // Check if an update is available
    available, latestVersion, latestHash, err := checker.IsUpdateAvailable(ctx, action)
    if err != nil {
        log.Fatalf("Error checking for updates: %v", err)
    }
    
    if available {
        if action.CommitHash != "" {
            fmt.Printf("Update available for %s/%s: %s (%s) -> %s (%s)\n",
                action.Owner, action.Name, 
                action.Version, action.CommitHash,
                latestVersion, latestHash)
        } else {
            fmt.Printf("Update available for %s/%s: %s -> %s (%s)\n",
                action.Owner, action.Name, 
                action.Version, latestVersion, latestHash)
        }
    }
}
```

### Creating and Applying Updates

```go
// Example of updating workflows with standardized comment format
func updateWorkflows(checker VersionChecker, manager UpdateManager, creator PRCreator, actions []ActionReference) {
    ctx := context.Background()
    var updates []*Update
    
    // Create updates for each action
    for _, action := range actions {
        // Get latest version and commit hash
        latestVersion, latestHash, err := checker.GetLatestVersion(ctx, action)
        if err != nil {
            log.Printf("Error getting latest version for %s/%s: %v",
                action.Owner, action.Name, err)
            continue
        }

        // Create update with commit hash and version history
        update, err := manager.CreateUpdate(ctx, action.Path, action, latestVersion, latestHash)
        if err != nil {
            log.Printf("Error creating update for %s/%s: %v",
                action.Owner, action.Name, err)
            continue
        }
        
        // Example of resulting workflow file:
        // # Using older hash from v2
        // # Original version: v2
        // uses: actions/checkout@abc123  # v3
        
        updates = append(updates, update)
    }
    
    // Apply updates and create PR
    if err := manager.ApplyUpdates(ctx, updates); err != nil {
        log.Fatalf("Error applying updates: %v", err)
    }
    
    if err := creator.CreatePR(ctx, updates); err != nil {
        log.Fatalf("Error creating PR: %v", err)
    }
}
```

## Best Practices

1. **Context Usage**
   - Always pass a context to enable timeout and cancellation handling
   - Consider using context with timeout for network operations

2. **Error Handling**
   - Check for errors from all interface methods
   - Provide meaningful error messages that include the action details
   - Consider implementing custom error types for specific failure cases

3. **Version Management**
   - Use semantic versioning when possible
   - Handle both tag-based versions and commit hashes
   - Preserve version information in comments
   - Track original versions for reference
   - Use commit hashes for secure referencing
   - Consider version constraints and compatibility

4. **Comment Preservation**
   - Follow the standardized comment format:
     ```yaml
     # Using older hash from [old-version]
     # Original version: [old-version]
     uses: actions/[name]@[hash]  # [new-version]
     ```
   - Preserve version history in comments
   - Track original versions for reference
   - Document version changes in PR descriptions
   - Maintain consistent comment style

5. **Pull Request Creation**
   - Group related updates into a single PR
   - Provide clear descriptions of changes
   - Include both version and hash changes in PR description
   - Document original versions in PR body
   - Add migration notes if needed

6. **Update Application**
   - Validate workflow files after applying updates
   - Maintain original file formatting and comments
   - Handle concurrent update scenarios properly
   - Verify commit hash validity
   - Ensure comment preservation during updates

## Migration Guide

### Updating from Version Tags to Commit Hashes

1. **Existing Format**
   ```yaml
   uses: actions/checkout@v3
   ```

2. **New Format with Version History**
   ```yaml
   # Using older hash from v3
   # Original version: v3
   uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675  # v4
   ```

3. **Benefits**
   - Improved security through immutable references
   - Version history tracking
   - Clear upgrade path documentation
   - Compatibility information preservation

4. **Automatic Migration**
   The tool will automatically:
   - Convert version tags to commit hashes
   - Add version history comments
   - Preserve original version information
   - Update PR descriptions with hash details

5. **Manual Steps**
   No manual steps required. The tool handles all aspects of migration:
   - Hash resolution
   - Comment generation
   - PR creation with detailed descriptions
   - Version history tracking
