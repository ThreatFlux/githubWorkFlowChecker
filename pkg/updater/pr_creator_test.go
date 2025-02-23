package updater

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v58/github"
	"golang.org/x/oauth2"
)

func TestNewPRCreator(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		owner    string
		repo     string
		wantAuth bool
	}{
		{
			name:     "with token",
			token:    "test-token",
			owner:    "test-owner",
			repo:     "test-repo",
			wantAuth: true,
		},
		{
			name:     "without token",
			token:    "",
			owner:    "test-owner",
			repo:     "test-repo",
			wantAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := NewPRCreator(tt.token, tt.owner, tt.repo)

			if creator == nil {
				t.Fatal("NewPRCreator() returned nil")
			}

			if creator.owner != tt.owner {
				t.Errorf("NewPRCreator() owner = %v, want %v", creator.owner, tt.owner)
			}

			if creator.repo != tt.repo {
				t.Errorf("NewPRCreator() repo = %v, want %v", creator.repo, tt.repo)
			}

			// Check if client is authenticated when token is provided
			transport := creator.client.Client().Transport
			if tt.wantAuth {
				if _, ok := transport.(*oauth2.Transport); !ok {
					t.Error("Expected authenticated client, got unauthenticated")
				}
			} else {
				if _, ok := transport.(*oauth2.Transport); ok {
					t.Error("Expected unauthenticated client, got authenticated")
				}
			}
		})
	}
}

func writeJSON(w http.ResponseWriter, data string) error {
	_, err := w.Write([]byte(data))
	return err
}

func setupTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	requestUrl, _ := url.Parse(server.URL + "/")
	client.BaseURL = requestUrl

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{"default_branch": "main"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref endpoints for getting and updating branches
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ref/heads/main") || strings.Contains(r.URL.Path, "/refs/heads/main") {
			w.WriteHeader(http.StatusOK)
			if err := writeJSON(w, `{
				"ref": "refs/heads/main",
				"object": {
					"sha": "test-sha",
					"type": "commit",
					"request_url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
				}
			}`); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if strings.Contains(r.URL.Path, "/ref/heads/action-updates-") || strings.Contains(r.URL.Path, "/refs/heads/action-updates-") {
			switch r.Method {
			case "GET":
				w.WriteHeader(http.StatusOK)
				if err := writeJSON(w, `{
					"ref": "refs/heads/action-updates",
					"object": {
						"sha": "test-sha",
						"type": "commit",
						"request_url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
					}
				}`); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case "PATCH":
				w.WriteHeader(http.StatusOK)
				if err := writeJSON(w, `{
					"ref": "refs/heads/action-updates",
					"object": {
						"sha": "new-commit-sha",
						"type": "commit",
						"request_url": "https://api.github.com/repos/test-owner/test-repo/git/commits/new-commit-sha"
					}
				}`); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	})

	// Mock ref endpoint for creating new branches
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			if err := writeJSON(w, `{
				"ref": "refs/heads/action-updates",
				"object": {
					"sha": "test-sha",
					"type": "commit",
					"request_url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
				}
			}`); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	})

	// Mock contents endpoint with version comments
	mux.HandleFunc("/repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		content := base64.StdEncoding.EncodeToString([]byte(`name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      # Using older hash from v1
      # Original version: v1
      - uses: actions/checkout@abc123  # v2`))
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{
			"type": "file",
			"encoding": "base64",
			"content": "`+content+`"
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock blob creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/blobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{"sha": "new-blob-sha"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock tree creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/trees", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{
			"sha": "new-tree-sha",
			"tree": [{"path": "test.yml", "mode": "100644", "type": "blob", "sha": "new-blob-sha"}]
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock commit creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/commits", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{"sha": "new-commit-sha"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock pull request creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{"number": 1}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock labels endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `[{"name": "dependencies"}, {"name": "automated-pr"}]`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return server, creator
}

func setupErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	// use Surl instead of url to avoid overriding a default
	Surl, _ := url.Parse(server.URL + "/")
	client.BaseURL = Surl

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if err := writeJSON(w, `{"message": "Not Found"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return server, creator
}

func setupBranchErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	// Use Surl instead of Surl to avoid overriding package name
	Surl, _ := url.Parse(server.URL + "/")
	client.BaseURL = Surl

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{"default_branch": "main"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if err := writeJSON(w, `{"message": "Internal Server Error"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return server, creator
}

func setupContentsErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	// Use Surl instead of Surl to avoid package name override
	Surl, _ := url.Parse(server.URL + "/")
	client.BaseURL = Surl

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{"default_branch": "main"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref endpoints
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{
			"ref": "refs/heads/main",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"Surl": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{
			"ref": "refs/heads/action-updates",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"Surl": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock contents endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		if err := writeJSON(w, `{"message": "Not Found"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return server, creator
}

func setupBlobErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	Surl, _ := url.Parse(server.URL + "/")
	client.BaseURL = Surl

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{"default_branch": "main"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref endpoints
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{
			"ref": "refs/heads/main",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"Surl": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{
			"ref": "refs/heads/action-updates",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"Surl": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock contents endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		content := base64.StdEncoding.EncodeToString([]byte(`name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`))
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{
			"type": "file",
			"encoding": "base64",
			"content": "`+content+`"
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock blob creation endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/git/blobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if err := writeJSON(w, `{"message": "Internal Server Error"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return server, creator
}

func setupPRErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	Surl, _ := url.Parse(server.URL + "/")
	client.BaseURL = Surl

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{"default_branch": "main"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref endpoints
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{
			"ref": "refs/heads/main",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"Surl": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock ref creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{
			"ref": "refs/heads/action-updates",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"Surl": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock contents endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		content := base64.StdEncoding.EncodeToString([]byte(`name: Test Workflow
on: [push]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2`))
		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, `{
			"type": "file",
			"encoding": "base64",
			"content": "`+content+`"
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock blob creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/blobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{"sha": "new-blob-sha"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock tree creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/trees", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{
			"sha": "new-tree-sha",
			"tree": [{"path": "test.yml", "mode": "100644", "type": "blob", "sha": "new-blob-sha"}]
		}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock commit creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/commits", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		if err := writeJSON(w, `{"sha": "new-commit-sha"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Mock pull request creation endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := writeJSON(w, `{"message": "Validation Failed"}`); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return server, creator
}

func TestCreatePR(t *testing.T) {
	server, creator := setupTestServer()
	defer server.Close()

	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:      "v2",
			NewVersion:      "v3",
			OldHash:         "def456",
			NewHash:         "abc123",
			FilePath:        ".github/workflows/test.yml",
			LineNumber:      7,
			Description:     "Update actions/checkout from v2 to v3",
			OriginalVersion: "v1",
		},
	}

	err := creator.CreatePR(context.Background(), updates)
	if err != nil {
		t.Errorf("CreatePR() error = %v", err)
	}
}

func TestCreatePR_NoUpdates(t *testing.T) {
	server, creator := setupTestServer()
	defer server.Close()

	err := creator.CreatePR(context.Background(), nil)
	if err != nil {
		t.Errorf("CreatePR() with no updates error = %v", err)
	}

	err = creator.CreatePR(context.Background(), []*Update{})
	if err != nil {
		t.Errorf("CreatePR() with empty updates error = %v", err)
	}
}

func TestCreatePR_RepoError(t *testing.T) {
	server, creator := setupErrorTestServer()
	defer server.Close()

	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			OldHash:     "def456",
			NewHash:     "abc123",
			FilePath:    ".github/workflows/test.yml",
			LineNumber:  7,
			Description: "Update actions/checkout from v2 to v3",
		},
	}

	err := creator.CreatePR(context.Background(), updates)
	if err == nil {
		t.Error("CreatePR() expected error, got nil")
	}
}

func TestCreatePR_BranchError(t *testing.T) {
	server, creator := setupBranchErrorTestServer()
	defer server.Close()

	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			OldHash:     "def456",
			NewHash:     "abc123",
			FilePath:    ".github/workflows/test.yml",
			LineNumber:  7,
			Description: "Update actions/checkout from v2 to v3",
		},
	}

	err := creator.CreatePR(context.Background(), updates)
	if err == nil {
		t.Error("CreatePR() expected error, got nil")
	}
}

func TestCreatePR_ContentsError(t *testing.T) {
	server, creator := setupContentsErrorTestServer()
	defer server.Close()

	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			OldHash:     "def456",
			NewHash:     "abc123",
			FilePath:    ".github/workflows/test.yml",
			LineNumber:  7,
			Description: "Update actions/checkout from v2 to v3",
		},
	}

	err := creator.CreatePR(context.Background(), updates)
	if err == nil {
		t.Error("CreatePR() expected error, got nil")
	}
}

func TestCreatePR_BlobError(t *testing.T) {
	server, creator := setupBlobErrorTestServer()
	defer server.Close()

	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			OldHash:     "def456",
			NewHash:     "abc123",
			FilePath:    ".github/workflows/test.yml",
			LineNumber:  7,
			Description: "Update actions/checkout from v2 to v3",
		},
	}

	err := creator.CreatePR(context.Background(), updates)
	if err == nil {
		t.Error("CreatePR() expected error, got nil")
	}
}

func TestCreatePR_PRError(t *testing.T) {
	server, creator := setupPRErrorTestServer()
	defer server.Close()

	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			OldHash:     "def456",
			NewHash:     "abc123",
			FilePath:    ".github/workflows/test.yml",
			LineNumber:  7,
			Description: "Update actions/checkout from v2 to v3",
		},
	}

	err := creator.CreatePR(context.Background(), updates)
	if err == nil {
		t.Error("CreatePR() expected error, got nil")
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	creator := &DefaultPRCreator{}
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			Description: "Update actions/checkout from v2 to v3",
		},
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "setup-node",
				Version: "v2",
			},
			OldVersion:  "v2",
			NewVersion:  "v3",
			Description: "Update actions/setup-node from v2 to v3",
		},
	}

	message := creator.generateCommitMessage(updates)
	expected := "Update GitHub Actions dependencies\n\n" +
		"* Update actions/checkout from v2 to v3\n" +
		"* Update actions/setup-node from v2 to v3\n"

	if message != expected {
		t.Errorf("generateCommitMessage() = %v, want %v", message, expected)
	}
}

func TestGeneratePRBody(t *testing.T) {
	creator := &DefaultPRCreator{}
	updates := []*Update{
		{
			Action: ActionReference{
				Owner:   "actions",
				Name:    "checkout",
				Version: "v2",
			},
			OldVersion:      "v2",
			NewVersion:      "v3",
			OldHash:         "def456",
			NewHash:         "abc123",
			OriginalVersion: "v1",
		},
	}

	body := creator.generatePRBody(updates)
	expectedContents := []string{
		"actions/checkout",
		"v2 (def456)",
		"v3 (abc123)",
		"Original version: v1",
		"🔒 This PR uses commit hashes",
		"🤖",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(body, expected) {
			t.Errorf("PR body missing expected content: %s", expected)
		}
	}
}
