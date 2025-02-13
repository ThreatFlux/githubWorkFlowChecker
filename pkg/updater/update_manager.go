package updater

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// DefaultUpdateManager implements the UpdateManager interface
type DefaultUpdateManager struct{}

// NewUpdateManager creates a new instance of DefaultUpdateManager
func NewUpdateManager() *DefaultUpdateManager {
	return &DefaultUpdateManager{}
}

// CreateUpdate creates an update for a given action and its latest version
func (m *DefaultUpdateManager) CreateUpdate(ctx context.Context, file string, action ActionReference, latestVersion string, commitHash string) (*Update, error) {
	if action.Version == latestVersion && action.CommitHash == commitHash {
		return nil, nil
	}

	// Preserve existing comments
	comments := m.PreserveComments(action)

	// Determine the original version (use commit hash if available)
	originalVersion := action.Version
	if action.CommitHash != "" {
		originalVersion = action.CommitHash
	}

		// Add version history comments
		comments = append(comments,
			fmt.Sprintf("# Using older hash from %s", originalVersion),
			fmt.Sprintf("# Original version: %s", originalVersion),
		)

	return &Update{
		Action:          action,
		OldVersion:      action.Version,
		NewVersion:      latestVersion,
		OldHash:         action.CommitHash,
		NewHash:         commitHash,
		FilePath:        file,
		LineNumber:      action.Line,
		Comments:        comments,
		VersionComment:  fmt.Sprintf("# %s", latestVersion),
		OriginalVersion: originalVersion,
		Description:     fmt.Sprintf("Update %s/%s from %s to %s", action.Owner, action.Name, originalVersion, latestVersion),
	}, nil
}

// ApplyUpdates applies the given updates to workflow files
func (m *DefaultUpdateManager) ApplyUpdates(ctx context.Context, updates []*Update) error {
	// Group updates by file
	fileUpdates := make(map[string][]*Update)
	for _, update := range updates {
		fileUpdates[update.FilePath] = append(fileUpdates[update.FilePath], update)
	}

	// Process each file
	for file, updates := range fileUpdates {
		if err := m.applyFileUpdates(file, updates); err != nil {
			return fmt.Errorf("error updating file %s: %w", file, err)
		}
	}

	return nil
}

// applyFileUpdates applies updates to a single file
func (m *DefaultUpdateManager) applyFileUpdates(file string, updates []*Update) error {
	// Read file content
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Convert content to string and split into lines
	lines := strings.Split(string(content), "\n")

	// Sort updates by line number in descending order to avoid line number changes
	sortUpdatesByLine(updates)

	// Apply each update
	for _, update := range updates {
		if update.LineNumber <= 0 || update.LineNumber > len(lines) {
			return fmt.Errorf("invalid line number %d for file %s", update.LineNumber, file)
		}

		// Get the line and extract any comments
		line := lines[update.LineNumber-1]
		parts := strings.SplitN(line, "#", 2)
		mainPart := strings.TrimSpace(parts[0])

		// Always use hash references
		oldRef := fmt.Sprintf("%s/%s@%s", update.Action.Owner, update.Action.Name, update.OldHash)
		if update.OldHash == "" {
			oldRef = fmt.Sprintf("%s/%s@%s", update.Action.Owner, update.Action.Name, update.OldVersion)
		}
		newRef := fmt.Sprintf("%s/%s@%s", update.Action.Owner, update.Action.Name, update.NewHash)
		mainPart = strings.Replace(mainPart, oldRef, newRef, -1)

		// Use the update's comments if available, otherwise generate them
		var comments []string
		if len(update.Comments) > 0 {
			comments = update.Comments
		} else {
			comments = []string{
				fmt.Sprintf("# Using older hash from %s", update.OriginalVersion),
				fmt.Sprintf("# Original version: %s", update.OriginalVersion),
			}
		}

		// Reconstruct the line with comments and version comment
		var newLine string
		if update.VersionComment != "" {
			newLine = fmt.Sprintf("%s  %s", strings.TrimSpace(mainPart), update.VersionComment)
		} else {
			newLine = fmt.Sprintf("%s  # %s", strings.TrimSpace(mainPart), update.NewVersion)
		}

		// Insert version history comments before the line
		newLines := make([]string, 0, len(lines)+2)
		newLines = append(newLines, lines[:update.LineNumber-1]...)
		newLines = append(newLines, comments...)
		newLines = append(newLines, newLine)
		if update.LineNumber < len(lines) {
			newLines = append(newLines, lines[update.LineNumber:]...)
		}
		lines = newLines
	}

	// Join lines back together
	newContent := strings.Join(lines, "\n")

	// Write updated content back to file
	if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
}

// PreserveComments preserves existing comments when updating an action
func (m *DefaultUpdateManager) PreserveComments(action ActionReference) []string {
	if len(action.Comments) == 0 {
		return nil
	}

	// Keep all comments except the version comment we'll update
	var preserved []string
	for _, comment := range action.Comments {
		if !strings.Contains(comment, "Original version:") {
			preserved = append(preserved, comment)
		}
	}
	return preserved
}

// sortUpdatesByLine sorts updates by line number in descending order
func sortUpdatesByLine(updates []*Update) {
	for i := 0; i < len(updates)-1; i++ {
		for j := i + 1; j < len(updates); j++ {
			if updates[i].LineNumber < updates[j].LineNumber {
				updates[i], updates[j] = updates[j], updates[i]
			}
		}
	}
}
