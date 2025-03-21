package updater

import (
	"net/http/httptest"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common/testutils"
)

// PRTestServerType defines the type of test server to create
type PRTestServerType string

const (
	// NormalServer represents a standard server with all endpoints working correctly
	NormalServer PRTestServerType = "normal"
	// ErrorServer represents a server with repo endpoint returning an error
	ErrorServer PRTestServerType = "error"
	// BranchErrorServer represents a server with branch creation returning an error
	BranchErrorServer PRTestServerType = "branch_error"
	// ContentsErrorServer represents a server with contents endpoint returning an error
	ContentsErrorServer PRTestServerType = "contents_error"
	// BlobErrorServer represents a server with blob creation returning an error
	BlobErrorServer PRTestServerType = "blob_error"
	// PRErrorServer represents a server with PR creation returning an error
	PRErrorServer PRTestServerType = "pr_error"
)

// SetupPRTestServer creates a test server and PR creator for testing based on the provided type
func SetupPRTestServer(t *testing.T, serverType PRTestServerType) (*httptest.Server, *DefaultPRCreator) {
	const owner = "test-owner"
	const repo = "test-repo"

	options := testutils.DefaultServerOptions(owner, repo)

	// Configure options based on server type
	switch serverType {
	case ErrorServer:
		options.ErrorMode = "repo"
	case BranchErrorServer:
		options.ErrorMode = "branch"
	case ContentsErrorServer:
		options.ErrorMode = "contents"
		// Let test_fixtures.go handle this with its special error handler
	case BlobErrorServer:
		options.ErrorMode = "blob"
	case PRErrorServer:
		options.ErrorMode = "pr"
	case NormalServer:
		// Default configuration is already set
	default:
		t.Errorf("Unknown server type: %s", serverType)
	}

	// Create the fixture
	fixture := testutils.NewGitHubServerFixture(options)

	// Create PR creator
	creator := &DefaultPRCreator{
		client: fixture.Client,
		owner:  owner,
		repo:   repo,
	}

	return fixture.Server, creator
}

// CreateTestUpdate creates a test update object with the given parameters
func CreateTestUpdate(owner, name, oldVersion, newVersion, filePath string) *Update {
	return &Update{
		Action: ActionReference{
			Owner:   owner,
			Name:    name,
			Version: oldVersion,
		},
		OldVersion:  oldVersion,
		NewVersion:  newVersion,
		OldHash:     "def456",
		NewHash:     "abc123",
		FilePath:    filePath,
		LineNumber:  7,
		Description: "Update " + owner + "/" + name + " from " + oldVersion + " to " + newVersion,
	}
}

// CreateTestUpdates creates a slice of test updates with the given parameters
func CreateTestUpdates(count int, owner, name, oldVersion, newVersion, filePath string) []*Update {
	updates := make([]*Update, count)
	for i := 0; i < count; i++ {
		updates[i] = CreateTestUpdate(owner, name, oldVersion, newVersion, filePath)
	}
	return updates
}
