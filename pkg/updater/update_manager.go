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
func (m *DefaultUpdateManager) CreateUpdate(ctx context.Context, file string, action ActionReference, latestVersion string) (*Update, error) {
	if action.Version == latestVersion {
		return nil, nil
	}

	return &Update{
		Action:      action,
		OldVersion:  action.Version,
		NewVersion:  latestVersion,
		FilePath:    file,
		LineNumber:  action.Line,
		Description: fmt.Sprintf("Update %s/%s from %s to %s", action.Owner, action.Name, action.Version, latestVersion),
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

		// Get the line and replace the version
		line := lines[update.LineNumber-1]
		oldRef := fmt.Sprintf("%s/%s@%s", update.Action.Owner, update.Action.Name, update.OldVersion)
		newRef := fmt.Sprintf("%s/%s@%s", update.Action.Owner, update.Action.Name, update.NewVersion)
		lines[update.LineNumber-1] = strings.Replace(line, oldRef, newRef, 1)
	}

	// Join lines back together
	newContent := strings.Join(lines, "\n")

	// Write updated content back to file
	if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	return nil
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
