package updater

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"golang.org/x/oauth2"
)

func TestNewDefaultVersionChecker(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantAuth bool
	}{
		{
			name:     "with token",
			token:    "test-token",
			wantAuth: true,
		},
		{
			name:     "without token",
			token:    "",
			wantAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewDefaultVersionChecker(tt.token)

			if checker == nil {
				t.Fatal(common.ErrVersionCheckerNil)
			}

			if checker.client == nil {
				t.Fatal(common.ErrVersionCheckerClientNil)
			}

			transport := checker.client.Client().Transport
			if tt.wantAuth {
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
	tests := []struct {
		name string
		v1   string
		v2   string
		want bool
	}{
		{
			name: "newer major version",
			v1:   "v2.0.0",
			v2:   "v1.0.0",
			want: true,
		},
		{
			name: "older major version",
			v1:   "v1.0.0",
			v2:   "v2.0.0",
			want: false,
		},
		{
			name: "newer minor version",
			v1:   "v1.1.0",
			v2:   "v1.0.0",
			want: true,
		},
		{
			name: "newer patch version",
			v1:   "v1.0.1",
			v2:   "v1.0.0",
			want: true,
		},
		{
			name: "same version",
			v1:   "v1.0.0",
			v2:   "v1.0.0",
			want: false,
		},
		{
			name: "without v prefix",
			v1:   "2.0.0",
			v2:   "1.0.0",
			want: true,
		},
		{
			name: "mixed v prefix",
			v1:   "v2.0.0",
			v2:   "1.0.0",
			want: true,
		},
		{
			name: "longer version",
			v1:   "v1.0.0.1",
			v2:   "v1.0.0",
			want: true,
		},
		{
			name: "shorter version",
			v1:   "v1.0",
			v2:   "v1.0.0",
			want: false,
		},
		{
			name: "alpha versions",
			v1:   "v1.0.0-alpha.2",
			v2:   "v1.0.0-alpha.1",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNewer(tt.v1, tt.v2); got != tt.want {
				t.Errorf("IsNewer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultVersionChecker_GetLatestVersion(t *testing.T) {
	tests := []struct {
		name        string
		action      ActionReference
		serverType  VersionTestServerType
		wantVersion string
		wantHash    string
		wantErr     bool
	}{
		{
			name: "successful latest release",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			serverType:  NormalVersionServer,
			wantVersion: "v2.0.0",
			wantHash:    "abc123",
			wantErr:     false,
		},
		{
			name: "no releases but has tags",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			serverType:  EmptyReleaseServer,
			wantVersion: "v1.0.0",
			wantHash:    "def456",
			wantErr:     false,
		},
		{
			name: "no releases and tags error",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			serverType: ErrorTagsServer,
			wantErr:    true,
		},
		{
			name: "no releases and empty tags",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			serverType: EmptyTagsServer,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server with the appropriate configuration
			server, checker := SetupVersionTestServer(t, tt.serverType)
			defer server.Close()
			gotVersion, gotHash, err := checker.GetLatestVersion(context.Background(), tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotVersion != tt.wantVersion {
					t.Errorf("GetLatestVersion() version = %v, want %v", gotVersion, tt.wantVersion)
				}
				if gotHash != tt.wantHash {
					t.Errorf("GetLatestVersion() hash = %v, want %v", gotHash, tt.wantHash)
				}
			}
		})
	}
}

func TestDefaultVersionChecker_IsUpdateAvailable(t *testing.T) {
	// First, let's enhance our version_checker_test_helper.go to support these tests
	// For now, we'll use the standard test cases that we can support with the current helpers

	tests := []VersionTestCase{
		{
			Name: "update available with version comparison",
			Action: ActionReference{
				Owner:   "test-owner",
				Name:    "test-repo",
				Version: "v1.0.0",
			},
			ServerType:    NormalVersionServer, // This server returns v2.0.0 as latest version
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name: "error getting latest version",
			Action: ActionReference{
				Owner:   "test-owner",
				Name:    "test-repo",
				Version: "v1.0.0",
			},
			ServerType:    ErrorReleaseServer,
			WantAvailable: false,
			WantVersion:   "",
			WantHash:      "",
			WantError:     true,
		},
		{
			Name: "no releases but has tags",
			Action: ActionReference{
				Owner:   "test-owner",
				Name:    "test-repo",
				Version: "v0.9.0", // Older than v1.0.0 returned by EmptyReleaseServer
			},
			ServerType:    EmptyReleaseServer,
			WantAvailable: true,
			WantVersion:   "v1.0.0",
			WantHash:      "def456",
			WantError:     false,
		},
		{
			Name: "no update available - empty tags",
			Action: ActionReference{
				Owner:   "test-owner",
				Name:    "test-repo",
				Version: "v1.0.0",
			},
			ServerType:    EmptyTagsServer,
			WantAvailable: false,
			WantError:     true,
		},
		{
			Name: "version is a commit SHA - update available when different",
			Action: ActionReference{
				Owner:   "test-owner",
				Name:    "test-repo",
				Version: "0123456789abcdef0123456789abcdef01234567", // Full length SHA string
			},
			ServerType:    NormalVersionServer, // Server returns abc123 as latest hash
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name: "version is a commit SHA - no update when same",
			Action: ActionReference{
				Owner:   "test-owner",
				Name:    "test-repo",
				Version: "abc123", // Same as what the server will return
			},
			ServerType:    NormalVersionServer,
			WantAvailable: false,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name: "using commit hash comparison when available",
			Action: ActionReference{
				Owner:      "test-owner",
				Name:       "test-repo",
				Version:    "v2.0.0", // Same version as latest
				CommitHash: "xyz789", // Different hash than latest
			},
			ServerType:    NormalVersionServer, // Server returns abc123 as latest hash
			WantAvailable: true,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
		{
			Name: "using commit hash - no update when same hash",
			Action: ActionReference{
				Owner:      "test-owner",
				Name:       "test-repo",
				Version:    "v1.0.0", // Different version
				CommitHash: "abc123", // Same hash as latest
			},
			ServerType:    NormalVersionServer,
			WantAvailable: false,
			WantVersion:   "v2.0.0",
			WantHash:      "abc123",
			WantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			// Create a test server with the appropriate configuration
			server, checker := SetupVersionTestServer(t, tt.ServerType)
			defer server.Close()
			gotAvailable, gotVersion, gotHash, err := checker.IsUpdateAvailable(context.Background(), tt.Action)
			if (err != nil) != tt.WantError {
				t.Errorf("IsUpdateAvailable() error = %v, wantErr %v", err, tt.WantError)
				return
			}
			if !tt.WantError {
				if gotAvailable != tt.WantAvailable {
					t.Errorf("IsUpdateAvailable() available = %v, want %v", gotAvailable, tt.WantAvailable)
				}
				if gotVersion != tt.WantVersion {
					t.Errorf("IsUpdateAvailable() version = %v, want %v", gotVersion, tt.WantVersion)
				}
				if gotHash != tt.WantHash {
					t.Errorf("IsUpdateAvailable() hash = %v, want %v", gotHash, tt.WantHash)
				}
			}
		})
	}
}

// TestVersionHelperFunctions validates the test helper functions themselves
func TestVersionHelperFunctions(t *testing.T) {
	serverTypes := []struct {
		name       string
		serverType VersionTestServerType
		testPaths  []string // Paths to test for this server type
	}{
		{
			name:       "InvalidRefServer",
			serverType: InvalidRefServer,
			testPaths: []string{
				"/repos/test-owner/test-repo/releases/latest",
				"/repos/test-owner/test-repo/git/ref/tags/v1.0.0",
			},
		},
		{
			name:       "MissingObjectServer",
			serverType: MissingObjectServer,
			testPaths: []string{
				"/repos/test-owner/test-repo/releases/latest",
				"/repos/test-owner/test-repo/git/ref/tags/v4.0.0",
			},
		},
		{
			name:       "MissingSHAServer",
			serverType: MissingSHAServer,
			testPaths: []string{
				"/repos/test-owner/test-repo/releases/latest",
				"/repos/test-owner/test-repo/git/ref/tags/v5.0.0",
			},
		},
		{
			name:       "AnnotatedTagServer",
			serverType: AnnotatedTagServer,
			testPaths: []string{
				"/repos/test-owner/test-repo/releases/latest",
				"/repos/test-owner/test-repo/git/ref/tags/v2.0.0",
				"/repos/test-owner/test-repo/git/tags/tag123",
			},
		},
		{
			name:       "AnnotatedTagErrorServer",
			serverType: AnnotatedTagErrorServer,
			testPaths: []string{
				"/repos/test-owner/test-repo/releases/latest",
				"/repos/test-owner/test-repo/git/ref/tags/v6.0.0",
				"/repos/test-owner/test-repo/git/tags/tag456",
			},
		},
		{
			name:       "MissingTagObjectServer",
			serverType: MissingTagObjectServer,
			testPaths: []string{
				"/repos/test-owner/test-repo/releases/latest",
				"/repos/test-owner/test-repo/git/ref/tags/v7.0.0",
				"/repos/test-owner/test-repo/git/tags/tag789",
			},
		},
	}

	for _, st := range serverTypes {
		t.Run(st.name, func(t *testing.T) {
			// Create test server using our helper function
			server, _ := SetupVersionTestServer(t, st.serverType)
			defer server.Close()

			// Test each path for this server type
			for _, path := range st.testPaths {
				t.Run(path, func(t *testing.T) {
					// Make a request to specific endpoint
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

					// All helper functions should set up the endpoints
					// We don't care what the response is, just that it exists
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
	// Call getServerConfig directly with an unknown server type
	config := getServerConfig("unknown-server-type", "owner", "repo")
	// Config should be empty but successfully returned
	if config.LatestRelease.Path != "" {
		t.Errorf("Expected empty config for unknown server type, got path: %s", config.LatestRelease.Path)
	}

	// Also test the default case in SetupVersionTestServer
	t.Run("SetupWithUnknownType", func(t *testing.T) {
		server, checker := SetupVersionTestServer(t, "unknown-server-type")
		defer server.Close()

		// Should still have a valid checker and server
		if checker == nil {
			t.Error("Expected non-nil checker even with unknown server type")
		}

		// Try making a request to server endpoint (it should be 404 since no endpoints are configured)
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

		// Since the config is empty, we expect a 404 as no endpoints were setup
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 for unknown server type, got %d", resp.StatusCode)
		}
	})
}

func TestDefaultVersionChecker_GetCommitHash(t *testing.T) {
	tests := []struct {
		name       string
		action     ActionReference
		version    string
		serverType VersionTestServerType
		wantHash   string
		wantErr    bool
	}{
		{
			name: "commit reference",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			version:    "v2.0.0",
			serverType: NormalVersionServer,
			wantHash:   "abc123",
			wantErr:    false,
		},
		{
			name: "annotated tag",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			version:    "v2.0.0",
			serverType: AnnotatedTagServer,
			wantHash:   "commit123",
			wantErr:    false,
		},
		{
			name: "ref not found error",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			version:    "v3.0.0",
			serverType: InvalidRefServer,
			wantErr:    true,
		},
		{
			name: "missing object in ref",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			version:    "v4.0.0",
			serverType: MissingObjectServer,
			wantErr:    true,
		},
		{
			name: "missing SHA in ref object",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			version:    "v5.0.0",
			serverType: MissingSHAServer,
			wantErr:    true,
		},
		{
			name: "annotated tag error",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			version:    "v6.0.0",
			serverType: AnnotatedTagErrorServer,
			wantErr:    true,
		},
		{
			name: "missing object in annotated tag",
			action: ActionReference{
				Owner: "test-owner",
				Name:  "test-repo",
			},
			version:    "v7.0.0",
			serverType: MissingTagObjectServer,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server with the appropriate configuration
			server, checker := SetupVersionTestServer(t, tt.serverType)
			defer server.Close()

			gotHash, err := checker.GetCommitHash(context.Background(), tt.action, tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gotHash != tt.wantHash {
				t.Errorf("GetCommitHash() = %v, want %v", gotHash, tt.wantHash)
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid SHA",
			input: "a81bbbf8298c0fa03ea29cdc473d45769f953675",
			want:  true,
		},
		{
			name:  "invalid characters",
			input: "not-a-hex-string",
			want:  false,
		},
		{
			name:  "mixed case valid",
			input: "A81BBbf8298c0fa03ea29cdc473d45769f953675",
			want:  true,
		},
		{
			name:  "empty string",
			input: "",
			want:  true,
		},
		{
			name:  "short valid hex",
			input: "abc123",
			want:  true,
		},
		{
			name:  "invalid character in middle",
			input: "a81bbbf8298c0fa03ea29cdc473g45769f953675",
			want:  false,
		},
		{
			name:  "spaces in string",
			input: "a81b bbf8 298c",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHexString(tt.input); got != tt.want {
				t.Errorf("isHexString() = %v, want %v", got, tt.want)
			}
		})
	}
}
