# GitHub Actions Workflow Checker API Documentation

## Package `updater`

The `updater` package provides the core functionality for scanning, checking, and updating GitHub Actions workflow files. It offers a modular architecture through well-defined interfaces that handle different aspects of the update process.

## Core Types

### ActionReference

```go
type ActionReference struct {
    Owner   string // GitHub owner/organization of the action
    Name    string // Name of the action
    Version string // Current version of the action
    Path    string // Path to the workflow file
    Line    int    // Line number in the workflow file
}
```

Represents a GitHub Action reference found in a workflow file. This structure contains all necessary information to identify and locate an action within a workflow.

### Update

```go
type Update struct {
    Action      ActionReference // The action to be updated
    OldVersion  string         // Current version of the action
    NewVersion  string         // Version to update to
    FilePath    string         // Path to the workflow file
    LineNumber  int           // Line number where the update should occur
    Description string        // Human-readable description of the update
}
```

Represents a pending update for a GitHub Action, containing all information needed to perform the update and create a pull request.

## Core Interfaces

### VersionChecker

```go
type VersionChecker interface {
    GetLatestVersion(ctx context.Context, action ActionReference) (string, error)
    IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, error)
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
    CreateUpdate(ctx context.Context, file string, action ActionReference, latestVersion string) (*Update, error)
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
    available, latestVersion, err := checker.IsUpdateAvailable(ctx, action)
    if err != nil {
        log.Fatalf("Error checking for updates: %v", err)
    }
    
    if available {
        fmt.Printf("Update available for %s/%s: %s -> %s\n",
            action.Owner, action.Name, action.Version, latestVersion)
    }
}
```

### Creating and Applying Updates

```go
func updateWorkflows(manager UpdateManager, creator PRCreator, actions []ActionReference) {
    ctx := context.Background()
    var updates []*Update
    
    // Create updates for each action
    for _, action := range actions {
        update, err := manager.CreateUpdate(ctx, action.Path, action, "v2.0.0")
        if err != nil {
            log.Printf("Error creating update for %s/%s: %v",
                action.Owner, action.Name, err)
            continue
        }
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
   - Handle both tag-based versions and commit SHAs
   - Consider version constraints and compatibility

4. **Pull Request Creation**
   - Group related updates into a single PR
   - Provide clear descriptions of changes
   - Include version change details in PR description

5. **Update Application**
   - Validate workflow files after applying updates
   - Maintain original file formatting
   - Handle concurrent update scenarios properly
