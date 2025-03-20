package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxPathLength defines the maximum allowed path length
	MaxPathLength = 255
)

// PathValidationOptions provides configuration options for path validation
type PathValidationOptions struct {
	// RequireRegularFile if true, validates that the path points to a regular file
	RequireRegularFile bool
	// AllowNonExistent if true, allows paths that don't exist yet
	AllowNonExistent bool
	// CheckSymlinks if true, validates that symlinks don't point outside the base directory
	CheckSymlinks bool
	// MaxPathLength specifies the maximum allowed path length (defaults to 255 if not set)
	MaxPathLength int
}

// DefaultPathValidationOptions returns the default options for path validation
func DefaultPathValidationOptions() PathValidationOptions {
	return PathValidationOptions{
		RequireRegularFile: false,
		AllowNonExistent:   true,
		CheckSymlinks:      true,
		MaxPathLength:      MaxPathLength,
	}
}

// ValidatePath ensures the path is safe and within the allowed directory
// baseDir is the root directory that all paths must be contained within
// path is the path to validate
// options provides configuration for the validation process
func ValidatePath(baseDir, path string, options PathValidationOptions) error {
	// Check for empty paths
	if baseDir == "" {
		return fmt.Errorf("base directory not set")
	}

	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is empty")
	}

	// Check for null bytes in both base and path
	if strings.ContainsRune(baseDir, 0) || strings.ContainsRune(path, 0) {
		return fmt.Errorf("path contains null bytes")
	}

	// Check path length
	maxLength := options.MaxPathLength
	if maxLength <= 0 {
		maxLength = MaxPathLength
	}
	if len(path) > maxLength {
		return fmt.Errorf("path exceeds maximum length of %d characters", maxLength)
	}

	// Clean and resolve both paths
	cleanBase := filepath.Clean(baseDir)
	cleanPath := filepath.Clean(path)

	// Convert to absolute paths
	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if path is within base directory
	if !strings.HasPrefix(absPath, absBase) {
		return fmt.Errorf("path is outside of allowed directory: %s", path)
	}

	// Check for path traversal attempts
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return fmt.Errorf("failed to determine relative path: %w", err)
	}

	if strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "/") {
		return fmt.Errorf("path traversal attempt detected")
	}

	// Check if the path is a symlink first using Lstat (doesn't follow symlinks)
	if options.CheckSymlinks {
		lstatInfo, lstatErr := os.Lstat(path)
		if lstatErr == nil && lstatInfo.Mode()&os.ModeSymlink != 0 {
			// It's a symlink, evaluate it
			evalPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("failed to evaluate symlink: %w", err)
			}

			// Evaluate the base directory as well to ensure consistent path comparison
			evalBase, err := filepath.EvalSymlinks(baseDir)
			if err != nil {
				return fmt.Errorf("failed to evaluate base directory: %w", err)
			}

			// Convert both to absolute paths
			absEvalPath, err := filepath.Abs(evalPath)
			if err != nil {
				return fmt.Errorf("failed to resolve symlink target path: %w", err)
			}

			absEvalBase, err := filepath.Abs(evalBase)
			if err != nil {
				return fmt.Errorf("failed to resolve evaluated base path: %w", err)
			}

			// Check if the resolved symlink target is within the resolved base directory
			if !strings.HasPrefix(absEvalPath, absEvalBase) {
				return fmt.Errorf("symlink points outside allowed directory: path is outside of allowed directory: %s", path)
			}
		}
	}

	// Check file existence and type if needed
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if !options.AllowNonExistent {
				return fmt.Errorf("path does not exist: %s", path)
			}
		} else {
			return fmt.Errorf("failed to access path: %w", err)
		}
	} else if options.RequireRegularFile && !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}

	return nil
}

// ValidatePathWithDefaults validates a path using default options
func ValidatePathWithDefaults(baseDir, path string) error {
	return ValidatePath(baseDir, path, DefaultPathValidationOptions())
}

// IsPathSafe is a simplified version that just checks if a path is within a base directory
func IsPathSafe(baseDir, path string) bool {
	err := ValidatePathWithDefaults(baseDir, path)
	return err == nil
}

// JoinAndValidatePath joins path elements and validates the result
func JoinAndValidatePath(baseDir string, elements ...string) (string, error) {
	if len(elements) == 0 {
		return "", fmt.Errorf("path is empty")
	}

	path := filepath.Join(elements...)

	// For relative paths, join with the base directory for validation
	if !filepath.IsAbs(path) {
		fullPath := filepath.Join(baseDir, path)
		if err := ValidatePathWithDefaults(baseDir, fullPath); err != nil {
			return "", err
		}
		return path, nil
	}

	// For absolute paths, validate directly
	if err := ValidatePathWithDefaults(baseDir, path); err != nil {
		return "", err
	}
	return path, nil
}

// SafeAbs returns the absolute path if it's safe, otherwise returns an error
func SafeAbs(baseDir, path string) (string, error) {
	if err := ValidatePathWithDefaults(baseDir, path); err != nil {
		return "", err
	}
	return filepath.Abs(path)
}
