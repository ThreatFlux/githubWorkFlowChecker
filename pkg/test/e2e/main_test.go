package e2e

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup any global test requirements
	if os.Getenv("GITHUB_TOKEN") == "" {
		fmt.Println("GITHUB_TOKEN environment variable is required for e2e tests")
		os.Exit(1)
	}

	os.Exit(m.Run())
}
