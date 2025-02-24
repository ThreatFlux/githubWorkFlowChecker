package updater

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

// DefaultUpdateManager implements the UpdateManager interface
type DefaultUpdateManager struct {
	fileLocks sync.Map // Map of file paths to sync.Mutex
	baseDir   string   // Base directory for path validation
}

// validatePath ensures the path is within the allowed directory and has proper permissions
func (m *DefaultUpdateManager) validatePath(path string) error {
	if m.baseDir == "" {
		return fmt.Errorf("base directory not set")
	}

	// Use the common path validation utility
	options := common.PathValidationOptions{
		RequireRegularFile: true,
		AllowNonExistent:   true,
		CheckSymlinks:      true,
	}
	return common.ValidatePath(m.baseDir, path, options)
}

// NewUpdateManager creates a new instance of DefaultUpdateManager
func NewUpdateManager(baseDir string) *DefaultUpdateManager {
	// If baseDir is empty, preserve it as empty to trigger validation error
	if baseDir == "" {
		return &DefaultUpdateManager{
			baseDir: "",
		}
	}
	return &DefaultUpdateManager{
		baseDir: filepath.Clean(baseDir),
	}
}

// CreateUpdate creates an update for a given action and its latest version
func (m *DefaultUpdateManager) CreateUpdate(ctx context.Context, file string, action ActionReference, latestVersion string, commitHash string) (*Update, error) {
	if action.Version == latestVersion && action.CommitHash == commitHash {
		return nil, nil
	}
	if ctx == nil {
		log.Printf("context is nil")
	}
	// Preserve existing comments
	comments := m.PreserveComments(action)

	// Determine the original version (use commit hash if available)
	originalVersion := action.Version
	if action.CommitHash != "" {
		originalVersion = action.CommitHash
	}

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
	// If ctx is empty, log a warning
	if ctx == nil {
		log.Println("context is nil")
	}
	// Group updates by file
	fileUpdates := make(map[string][]*Update)
	for _, update := range updates {
		fileUpdates[update.FilePath] = append(fileUpdates[update.FilePath], update)
	}

	// Process each file with proper locking
	for fileN, updates := range fileUpdates {
		// Get or create mutex for this file
		lockInterface, _ := m.fileLocks.LoadOrStore(fileN, &sync.Mutex{})
		lock := lockInterface.(*sync.Mutex)

		// Lock the file for exclusive access
		lock.Lock()
		err := m.applyFileUpdates(fileN, updates)
		lock.Unlock()

		if err != nil {
			return fmt.Errorf("error updating file %s: %w", fileN, err)
		}
	}

	return nil
}
func (m *DefaultUpdateManager) applyFileUpdates(fileN string, updates []*Update) error {
	// Validate file path
	if err := m.validatePath(fileN); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Read file content using common utility
	content, err := common.ReadFile(fileN)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Convert content to string and split into lines
	lines := strings.Split(string(content), "\n")

	// Sort updates by line number in descending order
	sortUpdatesByLine(updates)

	// Track line number adjustments
	lineAdjustments := make(map[int]int)

	// Apply each update
	for _, update := range updates {
		// Adjust the line number based on previous updates
		adjustedLineNumber := update.LineNumber
		for origLine, adjustment := range lineAdjustments {
			if update.LineNumber > origLine {
				adjustedLineNumber += adjustment
			}
		}

		if adjustedLineNumber <= 0 || adjustedLineNumber > len(lines) {
			return fmt.Errorf("invalid line number %d (adjusted from %d) for file %s",
				adjustedLineNumber, update.LineNumber, fileN)
		}

		// Get the line and preserve indentation
		line := lines[adjustedLineNumber-1]
		indentation := ""
		if idx := strings.Index(line, "-"); idx >= 0 {
			indentation = line[:idx]
		}

		// Apply the update (same as before)
		parts := strings.SplitN(line, "#", 2)
		mainPart := strings.TrimSpace(parts[0])

		oldRefBase := fmt.Sprintf("%s/%s@", update.Action.Owner, update.Action.Name)
		idx := strings.Index(mainPart, oldRefBase)
		if idx >= 0 {
			endIdx := strings.Index(mainPart[idx:], " ")
			if endIdx == -1 {
				endIdx = len(mainPart[idx:])
			}
			mainPart = mainPart[:idx] + oldRefBase + update.NewHash + mainPart[idx+endIdx:]
		}

		var newLine string
		if update.VersionComment != "" {
			newLine = fmt.Sprintf("%s%s  %s", indentation, strings.TrimSpace(mainPart),
				update.VersionComment)
		} else {
			newLine = fmt.Sprintf("%s%s  # %s", indentation, strings.TrimSpace(mainPart),
				update.NewVersion)
		}

		// Update the lines array
		newLines := make([]string, 0, len(lines))
		newLines = append(newLines, lines[:adjustedLineNumber-1]...)
		newLines = append(newLines, newLine)
		if adjustedLineNumber < len(lines) {
			newLines = append(newLines, lines[adjustedLineNumber:]...)
		}
		lines = newLines

		lineAdjustments[update.LineNumber] = len(lines) - len(newLines)
	}

	// Write updated content back to file using common utility
	fileContent := strings.Join(lines, "\n")
	if err := common.WriteFileString(fileN, fileContent); err != nil {
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
	if len(updates) <= 1 {
		return
	}

	sort.Slice(updates, func(i, j int) bool {
		return updates[i].LineNumber > updates[j].LineNumber
	})
}
