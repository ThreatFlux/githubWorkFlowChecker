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

// parseActionReference parses an action reference string (e.g., "actions/checkout@v2")
func parseActionReference(ref string, path string) (*ActionReference, error) {
	parts := strings.Split(ref, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid action reference format: %s", ref)
	}

	nameParts := strings.Split(parts[0], "/")
	if len(nameParts) != 2 {
		return nil, fmt.Errorf("invalid action name format: %s", parts[0])
	}

	return &ActionReference{
		Owner:   nameParts[0],
		Name:    nameParts[1],
		Version: parts[1],
		Path:    path,
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

	var workflow map[string]interface{}
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return nil, fmt.Errorf("error parsing workflow YAML: %w", err)
	}

	actions := make([]ActionReference, 0)
	err = s.parseNode(workflow, path, &actions)
	if err != nil {
		return nil, fmt.Errorf("error parsing workflow content: %w", err)
	}

	return actions, nil
}

// parseNode recursively traverses the YAML structure looking for action references
func (s *Scanner) parseNode(node interface{}, path string, actions *[]ActionReference) error {
	switch v := node.(type) {
	case map[string]interface{}:
		for key, value := range v {
			if key == "uses" {
				if actionRef, ok := value.(string); ok {
					action, err := parseActionReference(actionRef, path)
					if err != nil {
						return err
					}
					*actions = append(*actions, *action)
				}
			} else {
				if err := s.parseNode(value, path, actions); err != nil {
					return err
				}
			}
		}
	case []interface{}:
		for _, item := range v {
			if err := s.parseNode(item, path, actions); err != nil {
				return err
			}
		}
	}
	return nil
}
