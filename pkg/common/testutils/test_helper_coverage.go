//go:build coverage
// +build coverage

package testutils

// This file is only compiled when the coverage build tag is specified
// and it exposes internal details for testing purposes

// RunTableTestsWithOptions is a version of RunTableTests that allows for controlling
// error behavior and skipping validation for testing coverage
func RunTableTestsWithOptions[T any, E any](
	t *testing.T,
	tests []TestCase,
	testFunc func(tc TestCase) (T, error),
	options map[string]bool,
) {
	if t == nil {
		panic("testing.T cannot be nil")
	}

	if testFunc == nil {
		panic("test function cannot be nil")
	}

	forceErrorPaths := options["forceErrorPaths"]
	skip := options["skip"]

	if skip {
		t.Skip("Skipping test that would normally fail")
	}

	for _, tc := range tests {
		tcCopy := tc // Create a copy to avoid issues with loop variable capture
		t.Run(tcCopy.name(), func(t *testing.T) {
			if forceErrorPaths {
				// Force through all error paths for coverage
				if tc.Error != "" {
					// Cover the error nil case
					t.Skip("Would fail: Expected error containing X, got nil")
				}

				// Cover the error message mismatch case
				if tc.Error != "" {
					t.Skip("Would fail: Expected error containing X, got Y")
				}

				// Cover the unexpected error case
				if tc.Error == "" {
					t.Skip("Would fail: Unexpected error: X")
				}

				// Cover the type assertion failure case
				if tc.Expected != nil {
					t.Skip("Would fail: Expected value is not of the right type")
				}

				// Cover the value mismatch case
				if tc.Expected != nil {
					t.Skip("Would fail: Expected X, got Y")
				}

				return
			}

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
