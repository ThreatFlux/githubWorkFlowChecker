package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scanner handles scanning and parsing of workflow files
type Scanner struct{}

// parseActionReference parses an action reference string (e.g., "actions/checkout@v2" or "actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675")
func parseActionReference(ref string, path string, comments []string) (*ActionReference, error) {
	parts := strings.Split(ref, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid action reference format: %s", ref)
	}

	nameParts := strings.Split(parts[0], "/")
	if len(nameParts) != 2 {
		return nil, fmt.Errorf("invalid action name format: %s", parts[0])
	}

	version := parts[1]
	var commitHash string

	// If the reference is a commit hash (40 character hex string)
	if len(version) == 40 && isHexString(version) {
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
		Owner:      nameParts[0],
		Name:       nameParts[1],
		Version:    version,
		CommitHash: commitHash,
		Path:       path,
		Comments:   comments,
	}, nil
}

// NewScanner creates a new Scanner instance
func NewScanner() *Scanner {
	return &Scanner{}
}

// ScanWorkflows finds all GitHub Actions workflow files in the repository
func (s *Scanner) ScanWorkflows(dir string) ([]string, error) {
	// Check if workflows directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflows directory not found at %s", dir)
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

		// Check for YAML files
		if strings.HasSuffix(info.Name(), ".yml") || strings.HasSuffix(info.Name(), ".yaml") {
			workflows = append(workflows, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning workflows: %w", err)
	}

	return workflows, nil
}

// ParseActionReferences extracts action references from a workflow file
func (s *Scanner) ParseActionReferences(path string) ([]ActionReference, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading workflow file: %w", err)
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
		return nil, fmt.Errorf("error parsing workflow YAML: %w", err)
	}

	// The document node should have one child which is the root mapping
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty YAML document")
	}

	actions := make([]ActionReference, 0)
	err = s.parseNode(doc.Content[0], path, &actions, lineComments)
	if err != nil {
		return nil, fmt.Errorf("error parsing workflow content: %w", err)
	}

	return actions, nil
}

// parseNode recursively traverses the YAML structure looking for action references
func (s *Scanner) parseNode(node *yaml.Node, path string, actions *[]ActionReference, lineComments map[int][]string) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]

			if key.Value == "uses" {
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
				*actions = append(*actions, *action)
			} else {
				if err := s.parseNode(value, path, actions, lineComments); err != nil {
					return err
				}
			}
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if err := s.parseNode(item, path, actions, lineComments); err != nil {
				return err
			}
		}
	case yaml.DocumentNode:
		for _, item := range node.Content {
			if err := s.parseNode(item, path, actions, lineComments); err != nil {
				return err
			}
		}
	}
	return nil
}
