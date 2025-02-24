package common

import (
	"testing"
)

func TestIsHexString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid hex string (lowercase)",
			input:    "abcdef0123456789",
			expected: true,
		},
		{
			name:     "Valid hex string (uppercase)",
			input:    "ABCDEF0123456789",
			expected: true,
		},
		{
			name:     "Valid hex string (mixed case)",
			input:    "aBcDeF0123456789",
			expected: true,
		},
		{
			name:     "Invalid hex string (contains g)",
			input:    "abcdefg0123456789",
			expected: false,
		},
		{
			name:     "Invalid hex string (contains special character)",
			input:    "abcdef-0123456789",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: true, // An empty string technically contains only valid hex characters
		},
		{
			name:     "Typical commit hash",
			input:    "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsHexString(tc.input)
			if result != tc.expected {
				t.Errorf("IsHexString(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		substrings []string
		expected   bool
	}{
		{
			name:       "Contains one substring",
			input:      "Hello, World!",
			substrings: []string{"World"},
			expected:   true,
		},
		{
			name:       "Contains multiple substrings",
			input:      "Hello, World!",
			substrings: []string{"Hello", "World"},
			expected:   true,
		},
		{
			name:       "Contains none of the substrings",
			input:      "Hello, World!",
			substrings: []string{"Foo", "Bar"},
			expected:   false,
		},
		{
			name:       "Empty input string",
			input:      "",
			substrings: []string{"Foo", "Bar"},
			expected:   false,
		},
		{
			name:       "Empty substring list",
			input:      "Hello, World!",
			substrings: []string{},
			expected:   false,
		},
		{
			name:       "Empty substring in list",
			input:      "Hello, World!",
			substrings: []string{""},
			expected:   true, // Empty string is contained in any string
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ContainsAny(tc.input, tc.substrings...)
			if result != tc.expected {
				t.Errorf("ContainsAny(%q, %v) = %v, expected %v", tc.input, tc.substrings, result, tc.expected)
			}
		})
	}
}

func TestContainsAll(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		substrings []string
		expected   bool
	}{
		{
			name:       "Contains all substrings",
			input:      "Hello, World!",
			substrings: []string{"Hello", "World"},
			expected:   true,
		},
		{
			name:       "Contains some substrings",
			input:      "Hello, World!",
			substrings: []string{"Hello", "Foo"},
			expected:   false,
		},
		{
			name:       "Contains none of the substrings",
			input:      "Hello, World!",
			substrings: []string{"Foo", "Bar"},
			expected:   false,
		},
		{
			name:       "Empty input string",
			input:      "",
			substrings: []string{"Foo", "Bar"},
			expected:   false,
		},
		{
			name:       "Empty substring list",
			input:      "Hello, World!",
			substrings: []string{},
			expected:   true, // Vacuously true
		},
		{
			name:       "Empty substring in list",
			input:      "Hello, World!",
			substrings: []string{"Hello", ""},
			expected:   true, // Empty string is contained in any string
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ContainsAll(tc.input, tc.substrings...)
			if result != tc.expected {
				t.Errorf("ContainsAll(%q, %v) = %v, expected %v", tc.input, tc.substrings, result, tc.expected)
			}
		})
	}
}

func TestTrimPrefixAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prefixes []string
		expected string
	}{
		{
			name:     "Has first prefix",
			input:    "prefixText",
			prefixes: []string{"prefix", "pre"},
			expected: "Text",
		},
		{
			name:     "Has second prefix",
			input:    "preText",
			prefixes: []string{"prefix", "pre"},
			expected: "Text",
		},
		{
			name:     "Has no prefix",
			input:    "Text",
			prefixes: []string{"prefix", "pre"},
			expected: "Text",
		},
		{
			name:     "Empty input",
			input:    "",
			prefixes: []string{"prefix", "pre"},
			expected: "",
		},
		{
			name:     "Empty prefixes",
			input:    "Text",
			prefixes: []string{},
			expected: "Text",
		},
		{
			name:     "Empty prefix in list",
			input:    "Text",
			prefixes: []string{"prefix", ""},
			expected: "Text", // Empty prefix doesn't match
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := TrimPrefixAny(tc.input, tc.prefixes...)
			if result != tc.expected {
				t.Errorf("TrimPrefixAny(%q, %v) = %q, expected %q", tc.input, tc.prefixes, result, tc.expected)
			}
		})
	}
}

func TestTrimSuffixAny(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		suffixes []string
		expected string
	}{
		{
			name:     "Has first suffix",
			input:    "TextSuffix",
			suffixes: []string{"Suffix", "fix"},
			expected: "Text",
		},
		{
			name:     "Has second suffix",
			input:    "Textfix",
			suffixes: []string{"Suffix", "fix"},
			expected: "Text",
		},
		{
			name:     "Has no suffix",
			input:    "Text",
			suffixes: []string{"Suffix", "fix"},
			expected: "Text",
		},
		{
			name:     "Empty input",
			input:    "",
			suffixes: []string{"Suffix", "fix"},
			expected: "",
		},
		{
			name:     "Empty suffixes",
			input:    "Text",
			suffixes: []string{},
			expected: "Text",
		},
		{
			name:     "Empty suffix in list",
			input:    "Text",
			suffixes: []string{"Suffix", ""},
			expected: "Text", // Empty suffix doesn't match
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := TrimSuffixAny(tc.input, tc.suffixes...)
			if result != tc.expected {
				t.Errorf("TrimSuffixAny(%q, %v) = %q, expected %q", tc.input, tc.suffixes, result, tc.expected)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "Simple split",
			input:    "a,b,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Split with spaces",
			input:    "a, b, c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Split with extra spaces",
			input:    "  a  ,  b  ,  c  ",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "Empty input",
			input:    "",
			sep:      ",",
			expected: []string{""},
		},
		{
			name:     "No separator in input",
			input:    "abc",
			sep:      ",",
			expected: []string{"abc"},
		},
		{
			name:     "Empty separator",
			input:    "abc",
			sep:      "",
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SplitAndTrim(tc.input, tc.sep)
			if len(result) != len(tc.expected) {
				t.Errorf("SplitAndTrim(%q, %q) = %v, expected %v", tc.input, tc.sep, result, tc.expected)
				return
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("SplitAndTrim(%q, %q)[%d] = %q, expected %q", tc.input, tc.sep, i, v, tc.expected[i])
				}
			}
		})
	}
}

func TestJoinNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		sep      string
		parts    []string
		expected string
	}{
		{
			name:     "All non-empty parts",
			sep:      ",",
			parts:    []string{"a", "b", "c"},
			expected: "a,b,c",
		},
		{
			name:     "Some empty parts",
			sep:      ",",
			parts:    []string{"a", "", "c"},
			expected: "a,c",
		},
		{
			name:     "All empty parts",
			sep:      ",",
			parts:    []string{"", "", ""},
			expected: "",
		},
		{
			name:     "No parts",
			sep:      ",",
			parts:    []string{},
			expected: "",
		},
		{
			name:     "Empty separator",
			sep:      "",
			parts:    []string{"a", "b", "c"},
			expected: "abc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := JoinNonEmpty(tc.sep, tc.parts...)
			if result != tc.expected {
				t.Errorf("JoinNonEmpty(%q, %v) = %q, expected %q", tc.sep, tc.parts, result, tc.expected)
			}
		})
	}
}
