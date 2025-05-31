package testutils

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/google/go-github/v72/github"
)

// TestFixture represents a reusable test fixture for GitHub API testing
type TestFixture struct {
	Server *httptest.Server
	Client *github.Client
	Mux    *http.ServeMux
}

// GitHubServerOptions contains configuration for mock GitHub server setup
type GitHubServerOptions struct {
	Owner           string
	Repo            string
	DefaultBranch   string
	WorkflowContent string
	SetupRepoInfo   bool
	SetupRefs       bool
	SetupContents   bool
	SetupBlobs      bool
	SetupTrees      bool
	SetupCommits    bool
	SetupPRs        bool
	SetupLabels     bool
	ErrorMode       string // Empty or one of: "repo", "branch", "contents", "blob", "pr"
}

// DefaultServerOptions returns standard options for a test server
func DefaultServerOptions(owner, repo string) *GitHubServerOptions {
	return &GitHubServerOptions{
		Owner:           owner,
		Repo:            repo,
		DefaultBranch:   "main",
		WorkflowContent: defaultWorkflowContent(),
		SetupRepoInfo:   true,
		SetupRefs:       true,
		SetupContents:   true,
		SetupBlobs:      true,
		SetupTrees:      true,
		SetupCommits:    true,
		SetupPRs:        true,
		SetupLabels:     true,
		ErrorMode:       "",
	}
}

// NewGitHubServerFixture creates a test fixture with a mock GitHub server
func NewGitHubServerFixture(options *GitHubServerOptions) *TestFixture {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	client := github.NewClient(nil)
	urlParse, _ := url.Parse(server.URL + "/")
	client.BaseURL = urlParse
	client.UploadURL = urlParse

	fixture := &TestFixture{
		Server: server,
		Client: client,
		Mux:    mux,
	}

	// Setup basic endpoints based on options
	if options.SetupRepoInfo && options.ErrorMode != "repo" {
		setupRepoInfoEndpoint(fixture, options)
	} else if options.ErrorMode == "repo" {
		setupErrorEndpoint(fixture, fmt.Sprintf("/repos/%s/%s", options.Owner, options.Repo))
	}

	if options.SetupRefs && options.ErrorMode != "branch" {
		setupRefsEndpoints(fixture, options)
	} else if options.ErrorMode == "branch" {
		setupErrorEndpoint(fixture, fmt.Sprintf("/repos/%s/%s/git/refs", options.Owner, options.Repo))
	}

	if options.SetupContents && options.ErrorMode != "contents" {
		setupContentsEndpoint(fixture, options)
	}

	// Special handling for contents errors to ensure they're caught
	if options.ErrorMode == "contents" {
		// Handle any contents path, not just the base pattern
		fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/contents/", options.Owner, options.Repo),
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				// Return a 500 error instead of 404 since PR creator handles 404 as a special case
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"Internal Server Error","documentation_url":"https://docs.github.com/rest/reference/repos#get-repository-content"}`))
			})
	}

	if options.SetupBlobs && options.ErrorMode != "blob" {
		setupBlobEndpoint(fixture, options)
	} else if options.ErrorMode == "blob" {
		setupErrorEndpoint(fixture, fmt.Sprintf("/repos/%s/%s/git/blobs", options.Owner, options.Repo))
	}

	if options.SetupTrees && options.ErrorMode != "trees" {
		setupTreeEndpoint(fixture, options)
	} else if options.ErrorMode == "trees" {
		setupErrorEndpoint(fixture, fmt.Sprintf("/repos/%s/%s/git/trees", options.Owner, options.Repo))
	}

	if options.SetupCommits && options.ErrorMode != "commits" {
		setupCommitEndpoint(fixture, options)
	} else if options.ErrorMode == "commits" {
		setupErrorEndpoint(fixture, fmt.Sprintf("/repos/%s/%s/git/commits", options.Owner, options.Repo))
	}

	if options.SetupPRs && options.ErrorMode != "pr" {
		setupPREndpoint(fixture, options)
	} else if options.ErrorMode == "pr" {
		setupErrorEndpoint(fixture, fmt.Sprintf("/repos/%s/%s/pulls", options.Owner, options.Repo))
	}

	if options.SetupLabels {
		setupLabelsEndpoint(fixture, options)
	}

	return fixture
}

// SetupCustomHandler adds a custom handler to the mock server
func (f *TestFixture) SetupCustomHandler(path string, handler http.HandlerFunc) {
	f.Mux.HandleFunc(path, handler)
}

// Close shuts down the test server
func (f *TestFixture) Close() {
	f.Server.Close()
}

// GitHubClientForFixture returns a GitHub client configured for the fixture
func (f *TestFixture) GitHubClientForFixture() *github.Client {
	return f.Client
}

// ClientContext returns a context for use with the client
func (f *TestFixture) ClientContext() context.Context {
	return context.Background()
}

// Helper functions for setting up endpoints

func setupRepoInfoEndpoint(fixture *TestFixture, options *GitHubServerOptions) {
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprintf(w, `{"default_branch": "%s"}`, options.DefaultBranch)
			if err != nil {
				return
			}
		})
}

func setupRefsEndpoints(fixture *TestFixture, options *GitHubServerOptions) {
	// Main branch reference
	mainRefResponse := fmt.Sprintf(`{
		"ref": "refs/heads/%s",
		"object": {
			"sha": "test-sha",
			"type": "commit"
		}
	}`, options.DefaultBranch)

	// Update branch reference
	updateRefResponse := `{
		"ref": "refs/heads/action-updates",
		"object": {
			"sha": "test-sha",
			"type": "commit"
		}
	}`

	// Updated reference with new commit SHA
	updatedRefResponse := `{
		"ref": "refs/heads/action-updates",
		"object": {
			"sha": "new-commit-sha",
			"type": "commit"
		}
	}`

	// Handler for branch references
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, fmt.Sprintf("/ref/heads/%s", options.DefaultBranch)) ||
				strings.Contains(r.URL.Path, fmt.Sprintf("/refs/heads/%s", options.DefaultBranch)) {
				w.WriteHeader(http.StatusOK)
				if err := WriteJSON(w, mainRefResponse); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else if strings.Contains(r.URL.Path, "/ref/heads/action-updates-") ||
				strings.Contains(r.URL.Path, "/refs/heads/action-updates-") {
				switch r.Method {
				case "GET":
					w.WriteHeader(http.StatusOK)
					if err := WriteJSON(w, updateRefResponse); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				case "PATCH":
					w.WriteHeader(http.StatusOK)
					if err := WriteJSON(w, updatedRefResponse); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				}
			} else if strings.Contains(r.URL.Path, "/ref/tags/") {
				// For version checker tests
				parts := strings.Split(r.URL.Path, "/")
				tagName := parts[len(parts)-1]

				if strings.Contains(tagName, "v") {
					w.WriteHeader(http.StatusOK)
					tagRefResponse := fmt.Sprintf(`{
						"ref": "refs/tags/%s",
						"object": {
							"sha": "abc123",
							"type": "commit"
						}
					}`, tagName)
					if err := WriteJSON(w, tagRefResponse); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
					}
				} else {
					http.Error(w, "not found", http.StatusNotFound)
				}
			}
		})

	// Add ref creation endpoint
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/refs", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			if err := WriteJSON(w, updateRefResponse); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func setupContentsEndpoint(fixture *TestFixture, options *GitHubServerOptions) {
	content := base64.StdEncoding.EncodeToString([]byte(options.WorkflowContent))

	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/contents/", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			response := fmt.Sprintf(`{
				"type": "file",
				"encoding": "base64",
				"content": "%s"
			}`, content)
			if err := WriteJSON(w, response); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
}

func setupBlobEndpoint(fixture *TestFixture, options *GitHubServerOptions) {
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/blobs", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := fmt.Fprintf(w, `{"sha": "new-blob-sha"}`)
			if err != nil {
				return
			}
		})
}

func setupTreeEndpoint(fixture *TestFixture, options *GitHubServerOptions) {
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/trees", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := fmt.Fprintf(w, `{
			"sha": "new-tree-sha",
			"tree": [{"path": "test.yml", "mode": "100644", "type": "blob", "sha": "new-blob-sha"}]
		}`)
			if err != nil {
				return
			}
		})
}

func setupCommitEndpoint(fixture *TestFixture, options *GitHubServerOptions) {
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/commits", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := fmt.Fprintf(w, `{"sha": "new-commit-sha"}`)
			if err != nil {
				return
			}
		})
}

func setupPREndpoint(fixture *TestFixture, options *GitHubServerOptions) {
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/pulls", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := fmt.Fprintf(w, `{"number": 1}`)
			if err != nil {
				return
			}
		})
}

func setupLabelsEndpoint(fixture *TestFixture, options *GitHubServerOptions) {
	fixture.Mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/issues/1/labels", options.Owner, options.Repo),
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprintf(w, `[{"name": "dependencies"}, {"name": "automated-pr"}]`)
			if err != nil {
				return
			}
		})
}

func setupErrorEndpoint(fixture *TestFixture, path string) {
	fixture.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Not Found"}`))
	})
}

func defaultWorkflowContent() string {
	return `name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@abc123  # v2`
}
