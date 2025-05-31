package updater

import (
	"io"
	"net/http"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"golang.org/x/oauth2"
)

func TestNewDefaultVersionChecker(t *testing.T) {
	testCases := GetAuthTestCases()

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			checker := NewDefaultVersionChecker(tt.Token)

			if checker == nil {
				t.Fatal(common.ErrVersionCheckerNil)
			}

			if checker.client == nil {
				t.Fatal(common.ErrVersionCheckerClientNil)
			}

			transport := checker.client.Client().Transport
			if tt.WantAuth {
				if _, ok := transport.(*oauth2.Transport); !ok {
					t.Error(common.ErrExpectedAuthClient)
				}
			} else {
				if _, ok := transport.(*oauth2.Transport); ok {
					t.Error(common.ErrExpectedUnauthClient)
				}
			}
		})
	}
}

func TestIsNewer(t *testing.T) {
	testCases := GetVersionComparisonTestCases()

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			versions := tt.Input.([]string)
			v1, v2 := versions[0], versions[1]
			want := tt.Want.(bool)

			if got := IsNewer(v1, v2); got != want {
				t.Errorf("IsNewer() = %v, want %v", got, want)
			}
		})
	}
}

func TestDefaultVersionChecker_GetLatestVersion(t *testing.T) {
	testCases := []VersionTestCase{
		{
			Name:        "successful latest release",
			Action:      CreateSimpleAction(""),
			ServerType:  NormalVersionServer,
			WantVersion: "v2.0.0",
			WantHash:    "abc123",
			WantError:   false,
		},
		{
			Name:        "no releases but has tags",
			Action:      CreateSimpleAction(""),
			ServerType:  EmptyReleaseServer,
			WantVersion: "v1.0.0",
			WantHash:    "def456",
			WantError:   false,
		},
		{
			Name:       "no releases and tags error",
			Action:     CreateSimpleAction(""),
			ServerType: ErrorTagsServer,
			WantError:  true,
		},
		{
			Name:       "no releases and empty tags",
			Action:     CreateSimpleAction(""),
			ServerType: EmptyTagsServer,
			WantError:  true,
		},
	}

	runner := NewTestCaseRunner(t)
	for _, tc := range testCases {
		runner.RunGetLatestVersionTest(tc)
	}
}

func TestDefaultVersionChecker_IsUpdateAvailable(t *testing.T) {
	testCases := []VersionTestCase{
		{
			Name:          "update available with version comparison",
			Action:        CreateSimpleAction("v1.0.0"),
			ServerType:    NormalVersionServer,
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name:          "error getting latest version",
			Action:        CreateSimpleAction("v1.0.0"),
			ServerType:    ErrorReleaseServer,
			WantAvailable: false,
			WantVersion:   "",
			WantHash:      "",
			WantError:     true,
		},
		{
			Name:          "no releases but has tags",
			Action:        CreateSimpleAction("v0.9.0"),
			ServerType:    EmptyReleaseServer,
			WantAvailable: true,
			WantVersion:   "v1.0.0",
			WantHash:      "def456",
			WantError:     false,
		},
		{
			Name:          "no update available - empty tags",
			Action:        CreateSimpleAction("v1.0.0"),
			ServerType:    EmptyTagsServer,
			WantAvailable: false,
			WantError:     true,
		},
		{
			Name:          "version is a commit SHA - update available when different",
			Action:        CreateSimpleAction("0123456789abcdef0123456789abcdef01234567"),
			ServerType:    NormalVersionServer,
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name:          "version is a commit SHA - no update when same",
			Action:        CreateSimpleAction("abc123"),
			ServerType:    NormalVersionServer,
			WantAvailable: false,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name:          "using commit hash comparison when available",
			Action:        CreateActionWithHash("v2.0.0", "xyz789"),
			ServerType:    NormalVersionServer,
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name:          "using commit hash - no update when same hash",
			Action:        CreateActionWithHash("v1.0.0", "abc123"),
			ServerType:    NormalVersionServer,
			WantAvailable: false,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
	}

	// Add edge cases for SHA validation
	shaEdgeCases := []VersionTestCase{
		{
			Name:          "edge case - 5 character SHA (too short)",
			Action:        CreateSimpleAction("abc12"),
			ServerType:    NormalVersionServer,
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name:          "edge case - 41 character string (too long for SHA)",
			Action:        CreateSimpleAction("01234567890123456789012345678901234567890"),
			ServerType:    NormalVersionServer,
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name:          "edge case - 7 character SHA prefix match",
			Action:        CreateSimpleAction("abc1234"),
			ServerType:    NormalVersionServer,
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name:          "edge case - non-hex characters in SHA-length string",
			Action:        CreateSimpleAction("xyz123gh"),
			ServerType:    NormalVersionServer,
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
	}

	// Combine all test cases
	allTestCases := append(testCases, shaEdgeCases...)

	runner := NewTestCaseRunner(t)
	for _, tc := range allTestCases {
		runner.RunIsUpdateAvailableTest(tc)
	}
}

// TestVersionHelperFunctions validates the test helper functions themselves
func TestVersionHelperFunctions(t *testing.T) {
	// Define server test configuration
	serverTestConfigs := map[VersionTestServerType][]string{
		InvalidRefServer: {
			"/repos/test-owner/test-repo/releases/latest",
			"/repos/test-owner/test-repo/git/ref/tags/v1.0.0",
		},
		MissingObjectServer: {
			"/repos/test-owner/test-repo/releases/latest",
			"/repos/test-owner/test-repo/git/ref/tags/v4.0.0",
		},
		MissingSHAServer: {
			"/repos/test-owner/test-repo/releases/latest",
			"/repos/test-owner/test-repo/git/ref/tags/v5.0.0",
		},
		AnnotatedTagServer: {
			"/repos/test-owner/test-repo/releases/latest",
			"/repos/test-owner/test-repo/git/ref/tags/v2.0.0",
			"/repos/test-owner/test-repo/git/tags/tag123",
		},
		AnnotatedTagErrorServer: {
			"/repos/test-owner/test-repo/releases/latest",
			"/repos/test-owner/test-repo/git/ref/tags/v6.0.0",
			"/repos/test-owner/test-repo/git/tags/tag456",
		},
		MissingTagObjectServer: {
			"/repos/test-owner/test-repo/releases/latest",
			"/repos/test-owner/test-repo/git/ref/tags/v7.0.0",
			"/repos/test-owner/test-repo/git/tags/tag789",
		},
	}

	// Test each server type
	for serverType, testPaths := range serverTestConfigs {
		t.Run(string(serverType), func(t *testing.T) {
			server, _ := SetupVersionTestServer(t, serverType)
			defer server.Close()

			// Test each endpoint for this server type
			for _, path := range testPaths {
				t.Run(path, func(t *testing.T) {
					req, _ := http.NewRequest("GET", server.URL+path, nil)
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						t.Fatalf("Error making request: %v", err)
					}
					defer func(Body io.ReadCloser) {
						err := Body.Close()
						if err != nil {
							t.Fatalf(common.ErrFailedToCloseBody, err)
						}
					}(resp.Body)

					if resp.StatusCode == http.StatusNotFound {
						t.Errorf("Endpoint %s not found", path)
					}
				})
			}
		})
	}
}

// TestUnknownServerType verifies that unknown server types return empty configs
func TestUnknownServerType(t *testing.T) {
	// Test getServerConfig with unknown type
	config := getServerConfig("unknown-server-type", "owner", "repo")
	if config.LatestRelease.Path != "" {
		t.Errorf("Expected empty config for unknown server type, got path: %s", config.LatestRelease.Path)
	}

	// Test SetupVersionTestServer with unknown type
	t.Run("SetupWithUnknownType", func(t *testing.T) {
		server, checker := SetupVersionTestServer(t, "unknown-server-type")
		defer server.Close()

		if checker == nil {
			t.Error("Expected non-nil checker even with unknown server type")
		}

		// Test that endpoints are not configured (should return 404)
		req, _ := http.NewRequest("GET", server.URL+"/repos/test-owner/test-repo/releases/latest", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Error making request: %v", err)
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				t.Fatalf(common.ErrFailedToCloseBody, err)
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for unknown server type, got %d", resp.StatusCode)
		}
	})
}

func TestDefaultVersionChecker_GetCommitHash(t *testing.T) {
	commitHashTests := []struct {
		name       string
		version    string
		serverType VersionTestServerType
		wantHash   string
		wantErr    bool
	}{
		{"commit reference", "v2.0.0", NormalVersionServer, "abc123", false},
		{"annotated tag", "v2.0.0", AnnotatedTagServer, "commit123", false},
		{"ref not found error", "v3.0.0", InvalidRefServer, "", true},
		{"missing object in ref", "v4.0.0", MissingObjectServer, "", true},
		{"missing SHA in ref object", "v5.0.0", MissingSHAServer, "", true},
		{"annotated tag error", "v6.0.0", AnnotatedTagErrorServer, "", true},
		{"missing object in annotated tag", "v7.0.0", MissingTagObjectServer, "", true},
	}

	runner := NewTestCaseRunner(t)
	action := CreateSimpleAction("")

	for _, tt := range commitHashTests {
		runner.RunGetCommitHashTest(tt.name, action, tt.version, tt.serverType, tt.wantHash, tt.wantErr)
	}
}

func TestIsHexString(t *testing.T) {
	testCases := GetHexStringTestCases()

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			input := tt.Input.(string)
			want := tt.Want.(bool)

			if got := isHexString(input); got != want {
				t.Errorf("isHexString() = %v, want %v", got, want)
			}
		})
	}
}
