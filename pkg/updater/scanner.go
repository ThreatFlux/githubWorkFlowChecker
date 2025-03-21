package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"gopkg.in/yaml.v3"
)

// Scanner handles scanning and parsing of workflow files
type Scanner struct {
	rateLimit    int
	rateDuration time.Duration
	lastOp       time.Time
	opCount      int
	mu           sync.Mutex
	baseDir      string // Base directory for path validation
}

// validatePath ensures the path is within the allowed directory
func (s *Scanner) validatePath(path string) error {
	if s.baseDir == "" {
		return fmt.Errorf(common.ErrBaseDirectoryNotSet)
	}

	// Use the common path validation utility
	return common.ValidatePathWithDefaults(s.baseDir, path)
}

// parseActionReference parses an action reference string (e.g., "actions/checkout@v2" or "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675")
func parseActionReference(ref string, path string, comments []string) (*ActionReference, error) {
	parts := strings.Split(ref, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf(common.ErrInvalidActionRefFormat, ref)
	}

	nameParts := strings.Split(parts[0], "/")
	if len(nameParts) < 2 {
		return nil, fmt.Errorf(common.ErrInvalidActionNameFormat, parts[0])
	}

	// For actions with more than two parts (e.g., github/codeql-action/init)
	// we'll consider the first part as the owner and join the rest as the name
	owner := nameParts[0]
	name := strings.Join(nameParts[1:], "/")

	version := parts[1]
	if version == "" {
		return nil, fmt.Errorf(common.ErrInvalidActionRefFormat, ref)
	}

	var commitHash string

	// If the reference is a commit hash (40 character hex string)
	if len(version) == 40 && common.IsHexString(version) {
		commitHash = version
		// Look for version in comments
		for _, comment := range comments {
			if strings.Contains(comment, "Original version:") {
				parts := strings.SplitN(comment, ":", 2)
				if len(parts) == 2 {
					version = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	return &ActionReference{
		Owner:      owner,
		Name:       name,
		Version:    version,
		CommitHash: commitHash,
		Path:       path,
		Comments:   comments,
	}, nil
}

// NewScanner creates a new Scanner instance
func NewScanner(baseDir string) *Scanner {
	return &Scanner{
		rateLimit:    60,          // Default to 60 operations
		rateDuration: time.Minute, // Per minute
		baseDir:      filepath.Clean(baseDir),
	}
}

// SetRateLimit configures the rate limiting for the scanner
func (s *Scanner) SetRateLimit(limit int, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rateLimit = limit
	s.rateDuration = duration
}

// checkRateLimit ensures operations don't exceed the configured rate limit
func (s *Scanner) checkRateLimit(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check context cancellation first
	if err := ctx.Err(); err != nil {
		return err
	}

	now := time.Now()
	windowStart := now.Add(-s.rateDuration)

	// If this is our first operation or we're in a new time window
	if s.lastOp.IsZero() || s.lastOp.Before(windowStart) {
		s.opCount = 1
		s.lastOp = now
		return nil
	}

	// If we've exceeded the rate limit
	if s.opCount >= s.rateLimit {
		return context.DeadlineExceeded
	}

	// We're within the rate limit
	s.opCount++
	return nil
}

// checkTimeout verifies if an operation has exceeded its timeout
func (s *Scanner) checkTimeout(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// ScanWorkflows finds all GitHub Actions workflow files in the repository
func (s *Scanner) ScanWorkflows(dir string) ([]string, error) {
	// Validate the directory path
	if err := s.validatePath(dir); err != nil {
		return nil, fmt.Errorf(common.ErrInvalidDirectoryPath, err)
	}

	// Check if workflows directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf(common.ErrWorkflowDirNotFound, dir)
	}

	var workflows []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Validate each file path
		if err := s.validatePath(path); err != nil {
			return err
		}

		// Check for YAML files
		if strings.HasSuffix(info.Name(), ".yml") || strings.HasSuffix(info.Name(), ".yaml") {
			// Check if file is readable
			if _, err := common.ReadFile(path); err != nil {
				return err
			}
			workflows = append(workflows, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf(common.ErrScanningWorkflows, err)
	}

	return workflows, nil
}

// ParseActionReferences extracts action references from a workflow file
func (s *Scanner) ParseActionReferences(path string) ([]ActionReference, error) {
	// Validate the file path
	if err := s.validatePath(path); err != nil {
		return nil, fmt.Errorf(common.ErrInvalidFilePath, err)
	}

	// Read the file using the common utility
	content, err := common.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(common.ErrReadingWorkflowFile, err)
	}

	// Split content into lines to preserve comments
	lines := strings.Split(string(content), "\n")
	lineComments := make(map[int][]string)
	var currentComments []string

	// Extract comments for each line
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			currentComments = append(currentComments, trimmed)
		} else if strings.Contains(line, "uses:") {
			if len(currentComments) > 0 {
				lineComments[i] = currentComments
				currentComments = nil
			}
		} else if !strings.Contains(line, "uses:") && len(strings.TrimSpace(line)) > 0 {
			currentComments = nil
		}
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf(common.ErrParsingWorkflowYAML, err)
	}

	// The document node should have one child which is the root mapping
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf(common.ErrEmptyYAMLDocument)
	}

	actions := make([]ActionReference, 0)
	seen := make(map[string]bool) // Track unique action references by line
	err = s.parseNode(doc.Content[0], path, &actions, lineComments, seen)
	if err != nil {
		return nil, fmt.Errorf(common.ErrParsingWorkflowContent, err)
	}

	return actions, nil
}

// parseNode recursively traverses the YAML structure looking for action references
func (s *Scanner) parseNode(node *yaml.Node, path string, actions *[]ActionReference, lineComments map[int][]string, seen map[string]bool) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			if key.Value == "uses" && value.Kind == yaml.ScalarNode {
				// Skip if it's inside a run command
				if i >= 2 && node.Content[i-2].Value == "run" {
					continue
				}

				// Handle template expressions
				if strings.Contains(value.Value, "${{") && strings.Contains(value.Value, "}}") {
					// For matrix expressions, we want to count them as one reference
					if strings.Contains(value.Value, "matrix.action") {
						lineNumber := value.Line
						comments := lineComments[lineNumber]
						if lineNumber > 0 && lineComments[lineNumber-1] != nil {
							comments = append(lineComments[lineNumber-1], comments...)
						}

						// Create a placeholder action reference for matrix usage
						action := &ActionReference{
							Owner:    "matrix",
							Name:     "action",
							Version:  "dynamic",
							Path:     path,
							Line:     lineNumber,
							Comments: comments,
						}
						*actions = append(*actions, *action)
					}
					continue
				}

				lineNumber := value.Line
				comments := lineComments[lineNumber]
				if lineNumber > 0 && lineComments[lineNumber-1] != nil {
					comments = append(lineComments[lineNumber-1], comments...)
				}

				action, err := parseActionReference(value.Value, path, comments)
				if err != nil {
					return err
				}
				action.Line = lineNumber
				action.Comments = comments

				// Include line number in the key to handle same action used in different places
				// Use the full action name (which may include multiple path segments)
				actionFullName := action.Owner + "/" + action.Name
				key := fmt.Sprintf("%s@%s:%d", actionFullName, action.Version, lineNumber)
				if !seen[key] {
					seen[key] = true
					*actions = append(*actions, *action)
				}
			} else if key.Value == "steps" {
				// Special handling for steps with aliases
				if value.Kind == yaml.AliasNode {
					// Get the actual node that this alias refers to
					aliasedNode := value.Alias
					if aliasedNode != nil {
						// Create a copy of the aliased node with the current line number
						aliasLine := value.Line
						err := s.parseAliasedNode(aliasedNode, aliasLine, path, actions, lineComments, seen)
						if err != nil {
							return err
						}
					}
				} else {
					if err := s.parseNode(value, path, actions, lineComments, seen); err != nil {
						return err
					}
				}
			} else if key.Value != "run" { // Skip parsing inside run commands
				if err := s.parseNode(value, path, actions, lineComments, seen); err != nil {
					return err
				}
			}
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if err := s.parseNode(item, path, actions, lineComments, seen); err != nil {
				return err
			}
		}
	case yaml.DocumentNode:
		for _, item := range node.Content {
			if err := s.parseNode(item, path, actions, lineComments, seen); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		return nil
	default:
		// Return nil for unhandled node types instead of panicking
		return nil
	}
	return nil
}

// parseAliasedNode parses a node that is referenced by an alias, using the alias's line number
func (s *Scanner) parseAliasedNode(node *yaml.Node, aliasLine int, path string, actions *[]ActionReference, lineComments map[int][]string, seen map[string]bool) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.MappingNode:
		// First check if this node has a run command
		hasRunCommand := false
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == "run" {
				hasRunCommand = true
				break
			}
		}

		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			if key.Value == "uses" && value.Kind == yaml.ScalarNode {
				// Skip if it's inside a run command
				if hasRunCommand {
					continue
				}

				// Use the alias line number instead of the original node's line number
				comments := lineComments[aliasLine]
				if aliasLine > 0 && lineComments[aliasLine-1] != nil {
					comments = append(lineComments[aliasLine-1], comments...)
				}

				action, err := parseActionReference(value.Value, path, comments)
				if err != nil {
					return err
				}
				action.Line = aliasLine
				action.Comments = comments

				// Include line number in the key to handle same action used in different places
				// Use the full action name (which may include multiple path segments)
				actionFullName := action.Owner + "/" + action.Name
				key := fmt.Sprintf("%s@%s:%d", actionFullName, action.Version, aliasLine)
				if !seen[key] {
					seen[key] = true
					*actions = append(*actions, *action)
				}
			} else if key.Value != "run" { // Skip parsing inside run commands
				if err := s.parseNode(value, path, actions, lineComments, seen); err != nil {
					return err
				}
			}
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if err := s.parseAliasedNode(item, aliasLine, path, actions, lineComments, seen); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		return nil
	default:
		// Return nil for unhandled node types instead of panicking
		return nil
	}
	return nil
}
