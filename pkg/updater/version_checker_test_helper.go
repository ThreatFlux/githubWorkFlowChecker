package updater

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v58/github"
)

// VersionTestServerType defines types of version checker test servers
type VersionTestServerType string

const (
	// NormalVersionServer is a server that returns valid version data
	NormalVersionServer VersionTestServerType = "normal"
	// EmptyReleaseServer returns no releases but valid tags
	EmptyReleaseServer VersionTestServerType = "empty_release"
	// ErrorReleaseServer returns errors for release endpoint
	ErrorReleaseServer VersionTestServerType = "error_release"
	// EmptyTagsServer returns no tags data
	EmptyTagsServer VersionTestServerType = "empty_tags"
	// ErrorTagsServer returns errors for tag endpoint
	ErrorTagsServer VersionTestServerType = "error_tags"
	// InvalidRefServer returns invalid ref data
	InvalidRefServer VersionTestServerType = "invalid_ref"
	// AnnotatedTagServer returns tag object for annotated tags
	AnnotatedTagServer VersionTestServerType = "annotated_tag"
	// MissingObjectServer returns ref without object
	MissingObjectServer VersionTestServerType = "missing_object"
	// MissingSHAServer returns object without SHA
	MissingSHAServer VersionTestServerType = "missing_sha"
	// AnnotatedTagErrorServer returns error for annotated tag requests
	AnnotatedTagErrorServer VersionTestServerType = "annotated_tag_error"
	// MissingTagObjectServer returns annotated tag without object
	MissingTagObjectServer VersionTestServerType = "missing_tag_object"
)

// EndpointConfig represents configuration for a specific endpoint
type EndpointConfig struct {
	Path       string
	StatusCode int
	Response   string
}

// VersionServerConfig represents configuration for all endpoints in a version test server
type VersionServerConfig struct {
	LatestRelease EndpointConfig
	TagsList      *EndpointConfig // Optional depending on server type
	TagRef        EndpointConfig
	AnnotatedTag  *EndpointConfig // Optional for annotated tags
}

// writeJSONResponse sets headers and writes the JSON response
func writeJSONResponse(w http.ResponseWriter, config EndpointConfig) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(config.StatusCode)
	_, _ = w.Write([]byte(config.Response))
}

// createReleasePath creates the releases API path
func createReleasePath(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo)
}

// createTagsPath creates the tags list API path
func createTagsPath(owner, repo string) string {
	return fmt.Sprintf("/repos/%s/%s/tags", owner, repo)
}

// createTagRefPath creates the tag reference API path
func createTagRefPath(owner, repo, version string) string {
	return fmt.Sprintf("/repos/%s/%s/git/ref/tags/%s", owner, repo, version)
}

// createAnnotatedTagPath creates the annotated tag object API path
func createAnnotatedTagPath(owner, repo, sha string) string {
	return fmt.Sprintf("/repos/%s/%s/git/tags/%s", owner, repo, sha)
}

// createReleaseConfig creates a standardized release endpoint config
func createReleaseConfig(owner, repo string, status int, tagName string) EndpointConfig {
	var response string
	if status == http.StatusOK {
		response = fmt.Sprintf(`{"tag_name": "%s"}`, tagName)
	} else {
		response = `{"message": "Not Found"}`
	}

	return EndpointConfig{
		Path:       createReleasePath(owner, repo),
		StatusCode: status,
		Response:   response,
	}
}

// createTagsListConfig creates a standardized tags list endpoint config
func createTagsListConfig(owner, repo string, status int, isEmpty bool) *EndpointConfig {
	var response string
	if isEmpty {
		response = `[]`
	} else {
		response = `[{"name": "v1.0.0"}]`
	}

	if status != http.StatusOK {
		response = `{"message": "Server Error"}`
	}

	config := EndpointConfig{
		Path:       createTagsPath(owner, repo),
		StatusCode: status,
		Response:   response,
	}
	return &config
}

// createSimpleTagRefConfig creates a tag reference endpoint config with basic options
func createSimpleTagRefConfig(owner, repo, version, sha string) EndpointConfig {
	return EndpointConfig{
		Path:       createTagRefPath(owner, repo, version),
		StatusCode: http.StatusOK,
		Response:   fmt.Sprintf(`{"object": {"sha": "%s", "type": "commit"}}`, sha),
	}
}

// createAnnotatedTagRefConfig creates a tag reference config for annotated tags
func createAnnotatedTagRefConfig(owner, repo, version, sha string) EndpointConfig {
	return EndpointConfig{
		Path:       createTagRefPath(owner, repo, version),
		StatusCode: http.StatusOK,
		Response: fmt.Sprintf(`{
			"ref": "refs/tags/%s",
			"object": {
				"sha": "%s",
				"type": "tag"
			}
		}`, version, sha),
	}
}

// createAnnotatedTagObjectConfig creates a tag object config for annotated tags
func createAnnotatedTagObjectConfig(owner, repo, tagSha, commitSha string, status int, includeObject bool) *EndpointConfig {
	var response string
	if includeObject {
		response = fmt.Sprintf(`{
			"sha": "%s",
			"object": {
				"sha": "%s",
				"type": "commit"
			}
		}`, tagSha, commitSha)
	} else {
		response = fmt.Sprintf(`{"sha": "%s"}`, tagSha)
	}

	if status != http.StatusOK {
		response = `{"message": "Server error"}`
	}

	config := EndpointConfig{
		Path:       createAnnotatedTagPath(owner, repo, tagSha),
		StatusCode: status,
		Response:   response,
	}
	return &config
}

// getServerConfig returns the configuration for the specific server type
func getServerConfig(serverType VersionTestServerType, owner, repo string) VersionServerConfig {
	switch serverType {
	case NormalVersionServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusOK, "v2.0.0"),
			TagRef:        createSimpleTagRefConfig(owner, repo, "v2.0.0", "abc123"),
		}

	case EmptyReleaseServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusNotFound, ""),
			TagsList:      createTagsListConfig(owner, repo, http.StatusOK, false),
			TagRef:        createSimpleTagRefConfig(owner, repo, "v1.0.0", "def456"),
		}

	case ErrorReleaseServer:
		return VersionServerConfig{
			LatestRelease: EndpointConfig{
				Path:       createReleasePath(owner, repo),
				StatusCode: http.StatusInternalServerError,
				Response:   `{"message": "Server Error"}`,
			},
			TagRef: EndpointConfig{}, // Empty but required for the struct
		}

	case EmptyTagsServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusNotFound, ""),
			TagsList:      createTagsListConfig(owner, repo, http.StatusOK, true),
			TagRef:        EndpointConfig{}, // Empty but required for the struct
		}

	case ErrorTagsServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusNotFound, ""),
			TagsList:      createTagsListConfig(owner, repo, http.StatusInternalServerError, false),
			TagRef:        EndpointConfig{}, // Empty but required for the struct
		}

	case InvalidRefServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusOK, "v1.0.0"),
			TagRef: EndpointConfig{
				Path:       createTagRefPath(owner, repo, "v1.0.0"),
				StatusCode: http.StatusOK,
				Response:   `{"ref": "refs/tags/v1.0.0"}`,
			},
		}

	case AnnotatedTagServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusOK, "v2.0.0"),
			TagRef:        createAnnotatedTagRefConfig(owner, repo, "v2.0.0", "tag123"),
			AnnotatedTag:  createAnnotatedTagObjectConfig(owner, repo, "tag123", "commit123", http.StatusOK, true),
		}

	case MissingObjectServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusOK, "v4.0.0"),
			TagRef: EndpointConfig{
				Path:       createTagRefPath(owner, repo, "v4.0.0"),
				StatusCode: http.StatusOK,
				Response:   `{"ref": "refs/tags/v4.0.0"}`,
			},
		}

	case MissingSHAServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusOK, "v5.0.0"),
			TagRef: EndpointConfig{
				Path:       createTagRefPath(owner, repo, "v5.0.0"),
				StatusCode: http.StatusOK,
				Response: `{
					"ref": "refs/tags/v5.0.0",
					"object": {
						"type": "commit"
					}
				}`,
			},
		}

	case AnnotatedTagErrorServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusOK, "v6.0.0"),
			TagRef:        createAnnotatedTagRefConfig(owner, repo, "v6.0.0", "tag456"),
			AnnotatedTag:  createAnnotatedTagObjectConfig(owner, repo, "tag456", "", http.StatusInternalServerError, false),
		}

	case MissingTagObjectServer:
		return VersionServerConfig{
			LatestRelease: createReleaseConfig(owner, repo, http.StatusOK, "v7.0.0"),
			TagRef:        createAnnotatedTagRefConfig(owner, repo, "v7.0.0", "tag789"),
			AnnotatedTag:  createAnnotatedTagObjectConfig(owner, repo, "tag789", "", http.StatusOK, false),
		}

	default:
		return VersionServerConfig{} // Empty config for unknown cases
	}
}

// setupVersionEndpoints configures the server endpoints based on the provided configuration
func setupVersionEndpoints(mux *http.ServeMux, config VersionServerConfig) {
	// Setup latest release endpoint if path is not empty
	if config.LatestRelease.Path != "" {
		mux.HandleFunc(config.LatestRelease.Path, func(w http.ResponseWriter, r *http.Request) {
			writeJSONResponse(w, config.LatestRelease)
		})
	}

	// Setup tags list endpoint if present
	if config.TagsList != nil && config.TagsList.Path != "" {
		mux.HandleFunc(config.TagsList.Path, func(w http.ResponseWriter, r *http.Request) {
			writeJSONResponse(w, *config.TagsList)
		})
	}

	// Setup tag ref endpoint if path is not empty (needed for most server types)
	if config.TagRef.Path != "" {
		mux.HandleFunc(config.TagRef.Path, func(w http.ResponseWriter, r *http.Request) {
			writeJSONResponse(w, config.TagRef)
		})
	}

	// Setup annotated tag endpoint if present
	if config.AnnotatedTag != nil && config.AnnotatedTag.Path != "" {
		mux.HandleFunc(config.AnnotatedTag.Path, func(w http.ResponseWriter, r *http.Request) {
			writeJSONResponse(w, *config.AnnotatedTag)
		})
	}
}

// SetupVersionTestServer creates a test server for version checker tests
func SetupVersionTestServer(t *testing.T, serverType VersionTestServerType) (*httptest.Server, *DefaultVersionChecker) {
	owner := "test-owner"
	repo := "test-repo"

	// Create mock server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Get configuration for this server type
	config := getServerConfig(serverType, owner, repo)

	// Setup endpoints based on configuration
	setupVersionEndpoints(mux, config)

	// Create GitHub client and version checker
	client := github.NewClient(nil)
	client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")
	checker := &DefaultVersionChecker{client: client}

	return server, checker
}

// VersionTestCase represents a test case for version checker tests
type VersionTestCase struct {
	Name          string
	Action        ActionReference
	ServerType    VersionTestServerType
	WantVersion   string
	WantHash      string
	WantAvailable bool
	WantError     bool
}
