package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v58/github"
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
				t.Fatal("NewDefaultVersionChecker() returned nil")
			}

			if checker.client == nil {
				t.Fatal("NewDefaultVersionChecker() client is nil")
			}

			// Check if client is authenticated when token is provided
			transport := checker.client.Client().Transport
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHexString(tt.input); got != tt.want {
				t.Errorf("isHexString() = %v, want %v", got, tt.want)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNewer(tt.v1, tt.v2); got != tt.want {
				t.Errorf("IsNewer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultVersionChecker(t *testing.T) {
	// Set up test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a client that points to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	checker := &DefaultVersionChecker{client: client}

	// Test cases
	tests := []struct {
		name          string
		action        ActionReference
		mockResponse  interface{}
		mockStatus    int
		wantVersion   string
		wantAvailable bool
		wantErr       bool
	}{
		{
			name: "latest release available",
			action: ActionReference{
				Owner: "actions",
				Name:  "checkout",
			},
			mockResponse: &github.RepositoryRelease{
				TagName: github.String("v3.0.0"),
			},
			mockStatus:    http.StatusOK,
			wantVersion:   "v3.0.0",
			wantAvailable: true,
			wantErr:       false,
		},
		{
			name: "no releases but tags available",
			action: ActionReference{
				Owner: "actions",
				Name:  "setup-go",
			},
			mockResponse: []*github.RepositoryTag{
				{Name: github.String("v2.0.0")},
			},
			mockStatus:    http.StatusNotFound,
			wantVersion:   "v2.0.0",
			wantAvailable: true,
			wantErr:       false,
		},
		{
			name: "no releases or tags",
			action: ActionReference{
				Owner: "actions",
				Name:  "nonexistent",
			},
			mockResponse:  nil,
			mockStatus:    http.StatusNotFound,
			wantVersion:   "",
			wantAvailable: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock handlers
			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/releases/latest", tt.action.Owner, tt.action.Name),
				func(w http.ResponseWriter, r *http.Request) {
					if tt.mockStatus != http.StatusOK {
						http.Error(w, "Not found", tt.mockStatus)
						return
					}
					release := tt.mockResponse.(*github.RepositoryRelease)
					fmt.Fprintf(w, `{"tag_name":"%s"}`, *release.TagName)
				})

			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/tags", tt.action.Owner, tt.action.Name),
				func(w http.ResponseWriter, r *http.Request) {
					if tt.mockStatus != http.StatusNotFound {
						http.Error(w, "Unexpected", http.StatusInternalServerError)
						return
					}
					if tags, ok := tt.mockResponse.([]*github.RepositoryTag); ok && len(tags) > 0 {
						fmt.Fprintf(w, `[{"name":"%s"}]`, *tags[0].Name)
					} else {
						fmt.Fprintf(w, `[]`)
					}
				})

			// Mock the Git.GetRef endpoint for the latest version
			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/%s", tt.action.Owner, tt.action.Name, tt.wantVersion),
				func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintf(w, `{
						"ref": "refs/tags/%s",
						"object": {
							"sha": "abcdef1234567890",
							"type": "commit"
						}
					}`, tt.wantVersion)
				})

			// Mock the Git.GetRef endpoint for the old version
			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v1.0.0", tt.action.Owner, tt.action.Name),
				func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintf(w, `{
						"ref": "refs/tags/v1.0.0",
						"object": {
							"sha": "0987654321fedcba",
							"type": "commit"
						}
					}`)
				})

			// Test GetLatestVersion
			version, hash, err := checker.GetLatestVersion(context.Background(), tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if version != tt.wantVersion {
					t.Errorf("GetLatestVersion() version = %v, want %v", version, tt.wantVersion)
				}
				if hash != "abcdef1234567890" {
					t.Errorf("GetLatestVersion() hash = %v, want abcdef1234567890", hash)
				}
			}

			// Test IsUpdateAvailable
			if !tt.wantErr {
				tt.action.Version = "v1.0.0"              // Set an old version
				tt.action.CommitHash = "0987654321fedcba" // Set old hash
				available, newVersion, newHash, err := checker.IsUpdateAvailable(context.Background(), tt.action)
				if err != nil {
					t.Errorf("IsUpdateAvailable() error = %v", err)
					return
				}
				if available != tt.wantAvailable {
					t.Errorf("IsUpdateAvailable() available = %v, want %v", available, tt.wantAvailable)
				}
				if tt.wantAvailable {
					if newVersion != tt.wantVersion {
						t.Errorf("IsUpdateAvailable() version = %v, want %v", newVersion, tt.wantVersion)
					}
					if newHash != "abcdef1234567890" {
						t.Errorf("IsUpdateAvailable() hash = %v, want abcdef1234567890", newHash)
					}
				}
			}
		})
	}
}

type MockVersionChecker struct {
	latestVersion string
	latestHash    string
	err           error
}

func NewMockVersionChecker(latestVersion string, latestHash string, err error) *MockVersionChecker {
	return &MockVersionChecker{
		latestVersion: latestVersion,
		latestHash:    latestHash,
		err:           err,
	}
}

func (m *MockVersionChecker) GetLatestVersion(ctx context.Context, action ActionReference) (string, string, error) {
	return m.latestVersion, m.latestHash, m.err
}

func (m *MockVersionChecker) IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, string, error) {
	if m.err != nil {
		return false, "", "", m.err
	}
	if IsNewer(m.latestVersion, action.Version) {
		return true, m.latestVersion, m.latestHash, nil
	}
	return false, m.latestVersion, m.latestHash, nil
}

func (m *MockVersionChecker) GetCommitHash(ctx context.Context, action ActionReference, version string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.latestHash, nil
}

func TestMockVersionChecker(t *testing.T) {
	ctx := context.Background()
	action := ActionReference{
		Owner:   "actions",
		Name:    "checkout",
		Version: "v2",
	}

	tests := []struct {
		name          string
		latestVersion string
		latestHash    string
		currentAction ActionReference
		wantUpdate    bool
		wantVersion   string
		wantHash      string
		wantErr       bool
	}{
		{
			name:          "update available",
			latestVersion: "v3",
			latestHash:    "abc123",
			currentAction: action,
			wantUpdate:    true,
			wantVersion:   "v3",
			wantHash:      "abc123",
			wantErr:       false,
		},
		{
			name:          "no update needed",
			latestVersion: "v1",
			latestHash:    "def456",
			currentAction: action,
			wantUpdate:    false,
			wantVersion:   "v1",
			wantHash:      "def456",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewMockVersionChecker(tt.latestVersion, tt.latestHash, nil)

			// Test GetLatestVersion
			version, hash, err := checker.GetLatestVersion(ctx, tt.currentAction)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if version != tt.latestVersion {
				t.Errorf("GetLatestVersion() version = %v, want %v", version, tt.latestVersion)
			}
			if hash != tt.latestHash {
				t.Errorf("GetLatestVersion() hash = %v, want %v", hash, tt.latestHash)
			}

			// Test IsUpdateAvailable
			gotUpdate, gotVersion, gotHash, err := checker.IsUpdateAvailable(ctx, tt.currentAction)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsUpdateAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUpdate != tt.wantUpdate {
				t.Errorf("IsUpdateAvailable() update = %v, want %v", gotUpdate, tt.wantUpdate)
			}
			if gotVersion != tt.wantVersion {
				t.Errorf("IsUpdateAvailable() version = %v, want %v", gotVersion, tt.wantVersion)
			}
			if gotHash != tt.wantHash {
				t.Errorf("IsUpdateAvailable() hash = %v, want %v", gotHash, tt.wantHash)
			}

			// Test GetCommitHash
			hash, err = checker.GetCommitHash(ctx, tt.currentAction, tt.latestVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if hash != tt.latestHash {
				t.Errorf("GetCommitHash() = %v, want %v", hash, tt.latestHash)
			}
		})
	}
}

func TestGetCommitHash(t *testing.T) {
	// Set up test server
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a client that points to test server
	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	checker := &DefaultVersionChecker{client: client}

	tests := []struct {
		name     string
		action   ActionReference
		version  string
		wantHash string
		mockType string // "commit" or "tag"
		wantErr  bool
	}{
		{
			name: "commit reference",
			action: ActionReference{
				Owner: "actions",
				Name:  "checkout",
			},
			version:  "v3",
			wantHash: "abc123def456",
			mockType: "commit",
			wantErr:  false,
		},
		{
			name: "annotated tag",
			action: ActionReference{
				Owner: "actions",
				Name:  "setup-go",
			},
			version:  "v2",
			wantHash: "def456abc123",
			mockType: "tag",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the Git.GetRef endpoint for the version
			mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/%s", tt.action.Owner, tt.action.Name, tt.version),
				func(w http.ResponseWriter, r *http.Request) {
					fmt.Fprintf(w, `{
						"ref": "refs/tags/%s",
						"object": {
							"sha": "%s",
							"type": "%s"
						}
					}`, tt.version, tt.wantHash, tt.mockType)
				})

			// For annotated tags, mock the Git.GetTag endpoint
			if tt.mockType == "tag" {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/tags/%s", tt.action.Owner, tt.action.Name, tt.wantHash),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{
							"sha": "%s",
							"object": {
								"sha": "%s",
								"type": "commit"
							}
						}`, tt.wantHash, tt.wantHash)
					})
			}

			hash, err := checker.GetCommitHash(context.Background(), tt.action, tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if hash != tt.wantHash {
				t.Errorf("GetCommitHash() = %v, want %v", hash, tt.wantHash)
			}
		})
	}
}
