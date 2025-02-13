package updater

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v57/github"
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

func setupTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"default_branch": "main"}`))
	})

	// Mock ref endpoints for getting and updating branches
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/ref/heads/main") || strings.Contains(r.URL.Path, "/refs/heads/main") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"ref": "refs/heads/main",
				"object": {
					"sha": "test-sha",
					"type": "commit",
					"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
				}
			}`))
		} else if strings.Contains(r.URL.Path, "/ref/heads/action-updates-") || strings.Contains(r.URL.Path, "/refs/heads/action-updates-") {
			switch r.Method {
			case "GET":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"ref": "refs/heads/action-updates",
					"object": {
						"sha": "test-sha",
						"type": "commit",
						"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
					}
				}`))
			case "PATCH":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"ref": "refs/heads/action-updates",
					"object": {
						"sha": "new-commit-sha",
						"type": "commit",
						"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/new-commit-sha"
					}
				}`))
			}
		}
	})

	// Mock ref endpoint for creating new branches
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{
				"ref": "refs/heads/action-updates",
				"object": {
					"sha": "test-sha",
					"type": "commit",
					"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
				}
			}`))
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
		w.Write([]byte(`{
			"type": "file",
			"encoding": "base64",
			"content": "` + content + `"
		}`))
	})

	// Mock blob creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/blobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"sha": "new-blob-sha"}`))
	})

	// Mock tree creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/trees", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"sha": "new-tree-sha",
			"tree": [{"path": "test.yml", "mode": "100644", "type": "blob", "sha": "new-blob-sha"}]
		}`))
	})

	// Mock commit creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/commits", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"sha": "new-commit-sha"}`))
	})

	// Mock pull request creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"number": 1}`))
	})

	// Mock labels endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/issues/1/labels", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"name": "dependencies"}, {"name": "automated-pr"}]`))
	})

	return server, creator
}

func setupErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	})

	return server, creator
}

func setupBranchErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"default_branch": "main"}`))
	})

	// Mock ref endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Internal Server Error"}`))
	})

	return server, creator
}

func setupContentsErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"default_branch": "main"}`))
	})

	// Mock ref endpoints
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ref": "refs/heads/main",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`))
	})

	// Mock ref creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"ref": "refs/heads/action-updates",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`))
	})

	// Mock contents endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	})

	return server, creator
}

func setupBlobErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"default_branch": "main"}`))
	})

	// Mock ref endpoints
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ref": "refs/heads/main",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`))
	})

	// Mock ref creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"ref": "refs/heads/action-updates",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`))
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
		w.Write([]byte(`{
			"type": "file",
			"encoding": "base64",
			"content": "` + content + `"
		}`))
	})

	// Mock blob creation endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/git/blobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Internal Server Error"}`))
	})

	return server, creator
}

func setupPRErrorTestServer() (*httptest.Server, *DefaultPRCreator) {
	// Create test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Create client pointing to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	creator := &DefaultPRCreator{
		client: client,
		owner:  "test-owner",
		repo:   "test-repo",
	}

	// Mock repository endpoint
	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"default_branch": "main"}`))
	})

	// Mock ref endpoints
	mux.HandleFunc("/repos/test-owner/test-repo/git/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"ref": "refs/heads/main",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`))
	})

	// Mock ref creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"ref": "refs/heads/action-updates",
			"object": {
				"sha": "test-sha",
				"type": "commit",
				"url": "https://api.github.com/repos/test-owner/test-repo/git/commits/test-sha"
			}
		}`))
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
		w.Write([]byte(`{
			"type": "file",
			"encoding": "base64",
			"content": "` + content + `"
		}`))
	})

	// Mock blob creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/blobs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"sha": "new-blob-sha"}`))
	})

	// Mock tree creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/trees", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"sha": "new-tree-sha",
			"tree": [{"path": "test.yml", "mode": "100644", "type": "blob", "sha": "new-blob-sha"}]
		}`))
	})

	// Mock commit creation endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/commits", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"sha": "new-commit-sha"}`))
	})

	// Mock pull request creation endpoint with error
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message": "Validation Failed"}`))
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
			OldVersion:  "v2",
			NewVersion:  "v3",
			FilePath:    ".github/workflows/test.yml",
			LineNumber:  7,
			Description: "Update actions/checkout from v2 to v3",
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
			OldVersion: "v2",
			NewVersion: "v3",
		},
	}

	body := creator.generatePRBody(updates)
	if !strings.Contains(body, "actions/checkout") {
		t.Error("PR body does not contain action reference")
	}
	if !strings.Contains(body, "v2 to v3") {
		t.Error("PR body does not contain version update")
	}
	if !strings.Contains(body, "🤖") {
		t.Error("PR body does not contain bot emoji")
	}
}
