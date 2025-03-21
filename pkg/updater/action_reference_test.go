package updater

import (
	"testing"
)

// TestFormatActionReference tests the formatActionReference function
func TestFormatActionReference(t *testing.T) {
	creator := &DefaultPRCreator{}

	testCases := []struct {
		name     string
		update   *Update
		expected string
	}{
		{
			name: "basic action reference",
			update: &Update{
				Action: ActionReference{
					Owner:   "actions",
					Name:    "checkout",
					Version: "v2",
				},
				NewHash:    "abc123",
				NewVersion: "v3",
			},
			expected: "actions/checkout@abc123  # v3",
		},
		{
			name: "action with no version comment",
			update: &Update{
				Action: ActionReference{
					Owner:   "actions",
					Name:    "checkout",
					Version: "v2",
				},
				NewHash:    "abc123",
				NewVersion: "", // Empty version
			},
			expected: "actions/checkout@abc123",
		},
		{
			name: "multi-part action name",
			update: &Update{
				Action: ActionReference{
					Owner:   "github",
					Name:    "codeql-action/init",
					Version: "v2",
				},
				NewHash:    "def456",
				NewVersion: "v2.1.0",
			},
			expected: "github/codeql-action/init@def456  # v2.1.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := creator.formatActionReference(tc.update)

			if result != tc.expected {
				t.Errorf("formatActionReference() = %q, want %q", result, tc.expected)
			}
		})
	}
}
