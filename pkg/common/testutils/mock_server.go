// Package testutils provides common testing utilities for the application
package testutils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-github/v58/github"
)

// MockServerBuilder helps build a mock HTTP server for testing GitHub API interactions
type MockServerBuilder struct {
	mux    *http.ServeMux
	server *httptest.Server
}

// NewMockServer creates a new MockServerBuilder
func NewMockServer() *MockServerBuilder {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	return &MockServerBuilder{
		mux:    mux,
		server: server,
	}
}

// WithHandler adds a custom handler function to the mock server
func (b *MockServerBuilder) WithHandler(path string, handler http.HandlerFunc) *MockServerBuilder {
	b.mux.HandleFunc(path, handler)
	return b
}

// WithJSONResponse adds a handler that returns a JSON response
func (b *MockServerBuilder) WithJSONResponse(path string, statusCode int, responseBody string) *MockServerBuilder {
	b.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if err := WriteJSON(w, responseBody); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	return b
}

// WithBlobHandler adds a handler for GitHub blob creation endpoint
func (b *MockServerBuilder) WithBlobHandler(owner, repo string) *MockServerBuilder {
	path := "/repos/" + owner + "/" + repo + "/git/blobs"
	return b.WithJSONResponse(path, http.StatusCreated, `{"sha": "new-blob-sha"}`)
}

// WithTreeHandler adds a handler for GitHub tree creation endpoint
func (b *MockServerBuilder) WithTreeHandler(owner, repo string) *MockServerBuilder {
	path := "/repos/" + owner + "/" + repo + "/git/trees"
	treeJSON := `{
		"sha": "new-tree-sha",
		"tree": [{"path": "test.yml", "mode": "100644", "type": "blob", "sha": "new-blob-sha"}]
	}`
	return b.WithJSONResponse(path, http.StatusCreated, treeJSON)
}

// WithCommitHandler adds a handler for GitHub commit creation endpoint
func (b *MockServerBuilder) WithCommitHandler(owner, repo string) *MockServerBuilder {
	path := "/repos/" + owner + "/" + repo + "/git/commits"
	return b.WithJSONResponse(path, http.StatusCreated, `{"sha": "new-commit-sha"}`)
}

// WithPRHandler adds a handler for GitHub pull request creation endpoint
func (b *MockServerBuilder) WithPRHandler(owner, repo string) *MockServerBuilder {
	path := "/repos/" + owner + "/" + repo + "/pulls"
	return b.WithJSONResponse(path, http.StatusCreated, `{"number": 1}`)
}

// WithLabelsHandler adds a handler for GitHub labels endpoint
func (b *MockServerBuilder) WithLabelsHandler(owner, repo string, issueNumber int) *MockServerBuilder {
	path := "/repos/" + owner + "/" + repo + "/issues/" + strconv.Itoa(issueNumber) + "/labels"
	return b.WithJSONResponse(path, http.StatusOK, `[{"name": "dependencies"}, {"name": "automated-pr"}]`)
}

// WithRefHandler adds a handler for GitHub reference endpoint
func (b *MockServerBuilder) WithRefHandler(owner, repo, refName string, refResponse string) *MockServerBuilder {
	path := "/repos/" + owner + "/" + repo + "/git/refs/" + refName
	return b.WithJSONResponse(path, http.StatusOK, refResponse)
}

// Build finalizes the server setup and returns the test server and a GitHub client
func (b *MockServerBuilder) Build() (*httptest.Server, *github.Client) {
	client := github.NewClient(nil)
	url := b.server.URL + "/"
	client.BaseURL, _ = client.BaseURL.Parse(url)
	client.UploadURL, _ = client.UploadURL.Parse(url)

	return b.server, client
}

// WriteJSON writes the given JSON string to the ResponseWriter
func WriteJSON(w http.ResponseWriter, jsonStr string) error {
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(jsonObj)
}
