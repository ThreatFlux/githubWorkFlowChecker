package common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v72/github"
)

func TestValidateTokenScopes(t *testing.T) {
	tests := []struct {
		name          string
		setupServer   func() *httptest.Server
		wantErr       bool
		expectedError string
	}{
		{
			name: "Valid token with required scopes",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.Header().Set("X-OAuth-Scopes", "repo, workflow")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"login": "testuser", "id": 1}`))
					}
				}))
			},
			wantErr: false,
		},
		{
			name: "Valid token with public_repo and workflow",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.Header().Set("X-OAuth-Scopes", "public_repo, workflow")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"login": "testuser", "id": 1}`))
					}
				}))
			},
			wantErr: false,
		},
		{
			name: "Invalid token - 401 Unauthorized",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.WriteHeader(http.StatusUnauthorized)
						_, _ = w.Write([]byte(`{"message": "Bad credentials"}`))
					}
				}))
			},
			wantErr:       true,
			expectedError: "invalid GitHub token",
		},
		{
			name: "Token missing repo scope",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.Header().Set("X-OAuth-Scopes", "workflow")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"login": "testuser", "id": 1}`))
					}
				}))
			},
			wantErr:       true,
			expectedError: "token missing required scope: repo or public_repo",
		},
		{
			name: "Token missing workflow scope",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.Header().Set("X-OAuth-Scopes", "repo")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"login": "testuser", "id": 1}`))
					}
				}))
			},
			wantErr:       true,
			expectedError: "token missing required scope: workflow",
		},
		{
			name: "Unauthenticated client (no login)",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{}`))
					}
				}))
			},
			wantErr: false, // Unauthenticated clients are allowed
		},
		{
			name: "GitHub App token (no scopes header)",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						// GitHub App tokens don't have X-OAuth-Scopes header
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"login": "testuser", "id": 1}`))
					}
				}))
			},
			wantErr: false, // GitHub App tokens are allowed
		},
		{
			name: "Server error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte(`{"message": "Internal server error"}`))
					}
				}))
			},
			wantErr:       true,
			expectedError: "failed to validate token",
		},
		{
			name: "Token with extra scopes",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/user" {
						w.Header().Set("X-OAuth-Scopes", "repo, workflow, user, admin:org")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"login": "testuser", "id": 1}`))
					}
				}))
			},
			wantErr: false, // Extra scopes are fine
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			// Create a client that points to our test server
			client := github.NewClient(nil)
			client.BaseURL, _ = client.BaseURL.Parse(server.URL + "/")

			ctx := context.Background()
			err := ValidateTokenScopes(ctx, client)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTokenScopes() error = nil, wantErr %v", tt.wantErr)
				} else if tt.expectedError != "" && !contains(err.Error(), tt.expectedError) {
					t.Errorf("ValidateTokenScopes() error = %v, expected to contain %v", err, tt.expectedError)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTokenScopes() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
