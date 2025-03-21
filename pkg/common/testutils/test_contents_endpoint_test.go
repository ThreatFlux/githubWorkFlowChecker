package testutils

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test direct expected response format from setupContentsEndpoint
func TestGitHubServerFixture_ContentErrors(t *testing.T) {
	// Setup with custom content to verify encoding
	customContent := "name: Custom Workflow"
	encoded := base64.StdEncoding.EncodeToString([]byte(customContent))

	options := &GitHubServerOptions{
		Owner:           "owner",
		Repo:            "repo",
		WorkflowContent: customContent,
	}

	fixture := &TestFixture{
		Mux: http.NewServeMux(),
	}

	// Setup contents endpoint
	setupContentsEndpoint(fixture, options)

	// Test GET request - should return the content
	req := httptest.NewRequest("GET", "/repos/owner/repo/contents/.github/workflows/test.yml", nil)
	w := httptest.NewRecorder()
	fixture.Mux.ServeHTTP(w, req)

	// Verify status and encoded content
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	responseBody := w.Body.String()
	if !strings.Contains(responseBody, encoded) {
		t.Errorf("Expected response to contain encoded content %q, got %q", encoded, responseBody)
	}
}
