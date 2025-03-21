package updater

import (
	"testing"
)

// TestScannerValidatePathEdgeCases tests edge cases for the validatePath function
func TestScannerValidatePathEdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		baseDir   string
		path      string
		expectErr bool
	}{
		{
			name:      "empty base directory",
			baseDir:   "",                          // Note: Scanner uses this on initialization, but we need to explicitly check it
			path:      "../workflows/workflow.yml", // Using a path with traversal to ensure it fails
			expectErr: true,
		},
		{
			name:      "very long path",
			baseDir:   "/tmp",
			path:      "/" + string(make([]byte, 4096)),
			expectErr: true,
		},
		{
			name:      "path with non-printable characters",
			baseDir:   "/tmp",
			path:      string([]byte{7, 8, 9}), // Bell, backspace, tab
			expectErr: true,
		},
		{
			name:      "path traversal attempt",
			baseDir:   "/tmp",
			path:      "../../../etc/passwd",
			expectErr: true,
		},
		{
			name:      "valid path within base directory",
			baseDir:   "/tmp",
			path:      "/tmp/workflow.yml",
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := &Scanner{
				baseDir: tc.baseDir,
			}

			err := scanner.validatePath(tc.path)
			if tc.expectErr && err == nil {
				t.Errorf("Expected error for path %q, but got none", tc.path)
			} else if !tc.expectErr && err != nil {
				t.Errorf("Unexpected error for path %q: %v", tc.path, err)
			}
		})
	}
}
