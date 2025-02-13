package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v57/github"
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

			// Test GetLatestVersion
			version, err := checker.GetLatestVersion(context.Background(), tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && version != tt.wantVersion {
				t.Errorf("GetLatestVersion() = %v, want %v", version, tt.wantVersion)
			}

			// Test IsUpdateAvailable
			if !tt.wantErr {
				tt.action.Version = "v1.0.0" // Set an old version
				available, newVersion, err := checker.IsUpdateAvailable(context.Background(), tt.action)
				if err != nil {
					t.Errorf("IsUpdateAvailable() error = %v", err)
					return
				}
				if available != tt.wantAvailable {
					t.Errorf("IsUpdateAvailable() available = %v, want %v", available, tt.wantAvailable)
				}
				if tt.wantAvailable && newVersion != tt.wantVersion {
					t.Errorf("IsUpdateAvailable() version = %v, want %v", newVersion, tt.wantVersion)
				}
			}
		})
	}
}

type MockVersionChecker struct {
	latestVersion string
	err           error
}

func NewMockVersionChecker(latestVersion string, err error) *MockVersionChecker {
	return &MockVersionChecker{
		latestVersion: latestVersion,
		err:           err,
	}
}

func (m *MockVersionChecker) GetLatestVersion(ctx context.Context, action ActionReference) (string, error) {
	return m.latestVersion, m.err
}

func (m *MockVersionChecker) IsUpdateAvailable(ctx context.Context, action ActionReference) (bool, string, error) {
	if m.err != nil {
		return false, "", m.err
	}
	if IsNewer(m.latestVersion, action.Version) {
		return true, m.latestVersion, nil
	}
	return false, "", nil
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
		currentAction ActionReference
		wantUpdate    bool
		wantVersion   string
		wantErr       bool
	}{
		{
			name:          "update available",
			latestVersion: "v3",
			currentAction: action,
			wantUpdate:    true,
			wantVersion:   "v3",
			wantErr:       false,
		},
		{
			name:          "no update needed",
			latestVersion: "v1",
			currentAction: action,
			wantUpdate:    false,
			wantVersion:   "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewMockVersionChecker(tt.latestVersion, nil)
			gotUpdate, gotVersion, err := checker.IsUpdateAvailable(ctx, tt.currentAction)
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
		})
	}
}
