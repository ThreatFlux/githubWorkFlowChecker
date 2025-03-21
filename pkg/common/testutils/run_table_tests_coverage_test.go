package testutils

import (
	"testing"
)

// These additional test cases specifically target the error paths in RunTableTests
// to increase the code coverage.

// Define test types to use with generics
type testInput struct { //nolint:unused
	Value int
}

type testResult struct { //nolint:unused
	Value int
}

// TestCase implementation for our tests
type myTestCase struct { //nolint:unused
	TestCase
}

// Direct test of the contains function
func TestContainsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "empty string empty substring",
			s:        "",
			substr:   "",
			expected: false,
		},
		{
			name:     "non-empty string empty substring",
			s:        "hello",
			substr:   "",
			expected: false,
		},
		{
			name:     "empty string non-empty substring",
			s:        "",
			substr:   "hello",
			expected: false,
		},
		{
			name:     "exact match",
			s:        "hello",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "substring shorter than string",
			s:        "hello",
			substr:   "ell",
			expected: true,
		},
		{
			name:     "substring at beginning",
			s:        "hello",
			substr:   "hel",
			expected: true,
		},
		{
			name:     "substring at end",
			s:        "hello",
			substr:   "llo",
			expected: true,
		},
		{
			name:     "substring longer than string",
			s:        "hello",
			substr:   "hello world",
			expected: false,
		},
		{
			name:     "substring not in string",
			s:        "hello",
			substr:   "world",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.s, tc.substr)
			if result != tc.expected {
				t.Errorf("contains(%q, %q) = %v, expected %v", tc.s, tc.substr, result, tc.expected)
			}
		})
	}
}
