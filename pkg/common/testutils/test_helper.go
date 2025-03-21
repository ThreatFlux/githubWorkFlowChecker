package testutils

import (
	"fmt"
	"testing"
)

// TestCase represents a generic test case structure
type TestCase struct {
	Name     string
	Input    interface{}
	Expected interface{}
	Error    string
}

// RunTableTests runs a set of table-driven tests with standard error handling
func RunTableTests[T any, E any](t *testing.T, tests []TestCase, testFunc func(tc TestCase) (T, error)) {
	if t == nil {
		panic("testing.T cannot be nil")
	}

	if testFunc == nil {
		panic("test function cannot be nil")
	}

	for _, tc := range tests {
		tcCopy := tc // Create a copy to avoid issues with loop variable capture
		t.Run(tcCopy.name(), func(t *testing.T) {
			result, err := testFunc(tcCopy)

			// Check error expectations
			if tcCopy.Error != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tcCopy.Error)
					return
				}
				if errMsg := err.Error(); !contains(errMsg, tcCopy.Error) {
					t.Errorf("Expected error containing %q, got %q", tcCopy.Error, errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check result expectations if provided
			if tcCopy.Expected != nil {
				expected, ok := tcCopy.Expected.(E)
				if !ok {
					t.Errorf("Expected value is not of the right type")
					return
				}

				// Use type assertion to compare
				// This is a simple comparison, might need to be extended for more complex types
				if fmt.Sprintf("%v", result) != fmt.Sprintf("%v", expected) {
					t.Errorf("Expected %v, got %v", expected, result)
				}
			}
		})
	}
}

// name returns the test case name, or generates a default one
func (tc TestCase) name() string {
	if tc.Name != "" {
		return tc.Name
	}
	return fmt.Sprintf("Test with input %v", tc.Input)
}

// contains checks if substring is contained in s
func contains(s, substring string) bool {
	// Handle empty strings
	if s == "" || substring == "" {
		return false
	}

	// Check if s is long enough to contain substring
	if len(s) < len(substring) {
		return false
	}

	// Simple case: exact match
	if s == substring {
		return true
	}

	// General case: search for substring within s
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}

	return false
}
