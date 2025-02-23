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
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantVersion string
		wantHash    string
		wantErr     bool
	}{
		{
			name: "successful latest release",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/owner/repo/releases/latest":
					fmt.Fprintf(w, `{"tag_name": "v1.0.0"}`)
				case "/repos/owner/repo/git/ref/tags/v1.0.0":
					fmt.Fprintf(w, `{"object": {"sha": "abc123", "type": "commit"}}`)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			wantVersion: "v1.0.0",
			wantHash:    "abc123",
			wantErr:     false,
		},
		{
			name: "no releases but has tags",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/owner/repo/releases/latest":
					http.Error(w, "Not found", http.StatusNotFound)
				case "/repos/owner/repo/tags":
					fmt.Fprintf(w, `[{"name": "v1.0.0"}]`)
				case "/repos/owner/repo/git/ref/tags/v1.0.0":
					fmt.Fprintf(w, `{"object": {"sha": "def456", "type": "commit"}}`)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			wantVersion: "v1.0.0",
			wantHash:    "def456",
			wantErr:     false,
		},
		{
			name: "no releases and tags error",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/owner/repo/releases/latest":
					http.Error(w, "Not found", http.StatusNotFound)
				case "/repos/owner/repo/tags":
					http.Error(w, "Server error", http.StatusInternalServerError)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			wantErr: true,
		},
		{
			name: "no releases and empty tags",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/owner/repo/releases/latest":
					http.Error(w, "Not found", http.StatusNotFound)
				case "/repos/owner/repo/tags":
					fmt.Fprintf(w, `[]`)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new test server for each test case
			server := httptest.NewServer(http.HandlerFunc(tt.mockHandler))
			defer server.Close()

			// Create a client that points to the test server
			client := github.NewClient(nil)
			url, _ := url.Parse(server.URL + "/")
			client.BaseURL = url

			checker := &DefaultVersionChecker{client: client}

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
	tests := []struct {
		name          string
		action        ActionReference
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantAvailable bool
		wantVersion   string
		wantHash      string
		wantErr       bool
	}{
		{
			name: "update available with version comparison",
			action: ActionReference{
				Owner:   "owner",
				Name:    "repo",
				Version: "v1.0.0",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/owner/repo/releases/latest":
					fmt.Fprintf(w, `{"tag_name": "v2.0.0"}`)
				case "/repos/owner/repo/git/ref/tags/v2.0.0":
					fmt.Fprintf(w, `{"object": {"sha": "abc123", "type": "commit"}}`)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			wantAvailable: true,
			wantVersion:   "v2.0.0",
			wantHash:      "abc123",
			wantErr:       false,
		},
		{
			name: "update available with commit hash comparison",
			action: ActionReference{
				Owner:      "owner",
				Name:       "repo",
				Version:    "v1.0.0",
				CommitHash: "old123",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/owner/repo/releases/latest":
					fmt.Fprintf(w, `{"tag_name": "v1.0.0"}`)
				case "/repos/owner/repo/git/ref/tags/v1.0.0":
					fmt.Fprintf(w, `{"object": {"sha": "new123", "type": "commit"}}`)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			wantAvailable: true,
			wantVersion:   "v1.0.0",
			wantHash:      "new123",
			wantErr:       false,
		},
		{
			name: "no update needed - same commit SHA",
			action: ActionReference{
				Owner:   "owner",
				Name:    "repo",
				Version: "abc123def456789abcdef0123456789abcdef012", // 40-char SHA
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/owner/repo/releases/latest":
					fmt.Fprintf(w, `{"tag_name": "v1.0.0"}`)
				case "/repos/owner/repo/git/ref/tags/v1.0.0":
					fmt.Fprintf(w, `{"object": {"sha": "abc123def456789abcdef0123456789abcdef012", "type": "commit"}}`)
				default:
					http.Error(w, "not found", http.StatusNotFound)
				}
			},
			wantAvailable: false,
			wantVersion:   "v1.0.0",
			wantHash:      "abc123def456789abcdef0123456789abcdef012",
			wantErr:       false,
		},
		{
			name: "error getting latest version",
			action: ActionReference{
				Owner:   "owner",
				Name:    "repo",
				Version: "v1.0.0",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Server error", http.StatusInternalServerError)
			},
			wantAvailable: false,
			wantVersion:   "",
			wantHash:      "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new test server for each test case
			server := httptest.NewServer(http.HandlerFunc(tt.mockHandler))
			defer server.Close()

			// Create a client that points to the test server
			client := github.NewClient(nil)
			url, _ := url.Parse(server.URL + "/")
			client.BaseURL = url

			checker := &DefaultVersionChecker{client: client}

			gotAvailable, gotVersion, gotHash, err := checker.IsUpdateAvailable(context.Background(), tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsUpdateAvailable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotAvailable != tt.wantAvailable {
					t.Errorf("IsUpdateAvailable() available = %v, want %v", gotAvailable, tt.wantAvailable)
				}
				if gotVersion != tt.wantVersion {
					t.Errorf("IsUpdateAvailable() version = %v, want %v", gotVersion, tt.wantVersion)
				}
				if gotHash != tt.wantHash {
					t.Errorf("IsUpdateAvailable() hash = %v, want %v", gotHash, tt.wantHash)
				}
			}
		})
	}
}

func TestDefaultVersionChecker_GetCommitHash(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	client := github.NewClient(nil)
	url, _ := url.Parse(server.URL + "/")
	client.BaseURL = url

	checker := &DefaultVersionChecker{client: client}

	tests := []struct {
		name       string
		action     ActionReference
		version    string
		setupMocks func(mux *http.ServeMux, action ActionReference)
		wantHash   string
		wantErr    bool
	}{
		{
			name: "commit reference",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			version: "v1.0.0",
			setupMocks: func(mux *http.ServeMux, action ActionReference) {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v1.0.0", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{
							"ref": "refs/tags/v1.0.0",
							"object": {
								"sha": "abc123",
								"type": "commit"
							}
						}`)
					})
			},
			wantHash: "abc123",
			wantErr:  false,
		},
		{
			name: "annotated tag",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			version: "v2.0.0",
			setupMocks: func(mux *http.ServeMux, action ActionReference) {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v2.0.0", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{
							"ref": "refs/tags/v2.0.0",
							"object": {
								"sha": "tag123",
								"type": "tag"
							}
						}`)
					})
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/tags/tag123", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{
							"sha": "tag123",
							"object": {
								"sha": "commit123",
								"type": "commit"
							}
						}`)
					})
			},
			wantHash: "commit123",
			wantErr:  false,
		},
		{
			name: "ref not found error",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			version: "v3.0.0",
			setupMocks: func(mux *http.ServeMux, action ActionReference) {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v3.0.0", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						http.Error(w, "Not found", http.StatusNotFound)
					})
			},
			wantErr: true,
		},
		{
			name: "missing object in ref",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			version: "v4.0.0",
			setupMocks: func(mux *http.ServeMux, action ActionReference) {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v4.0.0", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{"ref": "refs/tags/v4.0.0"}`)
					})
			},
			wantErr: true,
		},
		{
			name: "missing SHA in ref object",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			version: "v5.0.0",
			setupMocks: func(mux *http.ServeMux, action ActionReference) {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v5.0.0", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{
							"ref": "refs/tags/v5.0.0",
							"object": {
								"type": "commit"
							}
						}`)
					})
			},
			wantErr: true,
		},
		{
			name: "annotated tag error",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			version: "v6.0.0",
			setupMocks: func(mux *http.ServeMux, action ActionReference) {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v6.0.0", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{
							"ref": "refs/tags/v6.0.0",
							"object": {
								"sha": "tag456",
								"type": "tag"
							}
						}`)
					})
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/tags/tag456", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						http.Error(w, "Server error", http.StatusInternalServerError)
					})
			},
			wantErr: true,
		},
		{
			name: "missing object in annotated tag",
			action: ActionReference{
				Owner: "owner",
				Name:  "repo",
			},
			version: "v7.0.0",
			setupMocks: func(mux *http.ServeMux, action ActionReference) {
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/ref/tags/v7.0.0", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{
							"ref": "refs/tags/v7.0.0",
							"object": {
								"sha": "tag789",
								"type": "tag"
							}
						}`)
					})
				mux.HandleFunc(fmt.Sprintf("/repos/%s/%s/git/tags/tag789", action.Owner, action.Name),
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprintf(w, `{"sha": "tag789"}`)
					})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(mux, tt.action)

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
