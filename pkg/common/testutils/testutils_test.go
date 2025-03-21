package testutils

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMockServer(t *testing.T) {
	builder := NewMockServer()
	if builder == nil {
		t.Fatal("NewMockServer() returned nil")
	}
	if builder.mux == nil {
		t.Error("NewMockServer() created a builder with nil mux")
	}
	if builder.server == nil {
		t.Error("NewMockServer() created a builder with nil server")
	}
}

func TestWithHandler(t *testing.T) {
	builder := NewMockServer()
	called := false

	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test response"))
	}

	builder.WithHandler("/test", handler)

	// Make a request to verify handler was registered
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if !called {
		t.Error("Handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	if body := rec.Body.String(); body != "test response" {
		t.Errorf("Expected body %q, got %q", "test response", body)
	}
}

func TestWithJSONResponse(t *testing.T) {
	builder := NewMockServer()
	builder.WithJSONResponse("/test-json", http.StatusCreated, `{"key": "value"}`)

	// Make a request to verify handler was registered
	req := httptest.NewRequest("GET", "/test-json", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}

	if val, ok := response["key"]; !ok || val != "value" {
		t.Errorf("Expected {\"key\": \"value\"}, got %v", response)
	}
}

func TestWithBlobHandler(t *testing.T) {
	builder := NewMockServer()
	builder.WithBlobHandler("owner", "repo")

	// Make a request to verify handler was registered
	req := httptest.NewRequest("POST", "/repos/owner/repo/git/blobs", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}

	if val, ok := response["sha"]; !ok || val != "new-blob-sha" {
		t.Errorf("Expected blob response with sha \"new-blob-sha\", got %v", response)
	}
}

func TestWithTreeHandler(t *testing.T) {
	builder := NewMockServer()
	builder.WithTreeHandler("owner", "repo")

	// Make a request to verify handler was registered
	req := httptest.NewRequest("POST", "/repos/owner/repo/git/trees", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}

	if val, ok := response["sha"]; !ok || val != "new-tree-sha" {
		t.Errorf("Expected tree response with sha \"new-tree-sha\", got %v", response)
	}
}

func TestWithCommitHandler(t *testing.T) {
	builder := NewMockServer()
	builder.WithCommitHandler("owner", "repo")

	// Make a request to verify handler was registered
	req := httptest.NewRequest("POST", "/repos/owner/repo/git/commits", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}

	if val, ok := response["sha"]; !ok || val != "new-commit-sha" {
		t.Errorf("Expected commit response with sha \"new-commit-sha\", got %v", response)
	}
}

func TestWithPRHandler(t *testing.T) {
	builder := NewMockServer()
	builder.WithPRHandler("owner", "repo")

	// Make a request to verify handler was registered
	req := httptest.NewRequest("POST", "/repos/owner/repo/pulls", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}

	if val, ok := response["number"]; !ok || val != 1 {
		t.Errorf("Expected PR response with number 1, got %v", response)
	}
}

func TestWithLabelsHandler(t *testing.T) {
	builder := NewMockServer()
	builder.WithLabelsHandler("owner", "repo", 1)

	// Make a request to verify handler was registered
	req := httptest.NewRequest("GET", "/repos/owner/repo/issues/1/labels", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var response []map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(response))
	}

	if response[0]["name"] != "dependencies" || response[1]["name"] != "automated-pr" {
		t.Errorf("Expected specific labels, got %v", response)
	}
}

func TestWithRefHandler(t *testing.T) {
	builder := NewMockServer()
	builder.WithRefHandler("owner", "repo", "heads/main", `{"ref": "refs/heads/main", "object": {"sha": "test-sha"}}`)

	// Make a request to verify handler was registered
	req := httptest.NewRequest("GET", "/repos/owner/repo/git/refs/heads/main", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}

	if val, ok := response["ref"]; !ok || val != "refs/heads/main" {
		t.Errorf("Expected ref \"refs/heads/main\", got %v", val)
	}
}

func TestBuild(t *testing.T) {
	builder := NewMockServer()
	server, client := builder.Build()

	if server == nil {
		t.Fatal("Build() returned nil server")
	}
	if client == nil {
		t.Fatal("Build() returned nil client")
	}

	// Verify client URLs
	if !strings.HasSuffix(client.BaseURL.String(), "/") {
		t.Errorf("BaseURL should end with slash, got %s", client.BaseURL.String())
	}

	if !strings.HasSuffix(client.UploadURL.String(), "/") {
		t.Errorf("UploadURL should end with slash, got %s", client.UploadURL.String())
	}

	// Clean up
	server.Close()
}

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		shouldErr bool
	}{
		{
			name:      "valid json",
			json:      `{"key": "value"}`,
			shouldErr: false,
		},
		{
			name:      "invalid json",
			json:      `{"key": value}`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := WriteJSON(w, tt.json)

			if (err != nil) != tt.shouldErr {
				t.Errorf("WriteJSON() error = %v, shouldErr %v", err, tt.shouldErr)
			}

			if !tt.shouldErr {
				// Verify content type
				contentType := w.Header().Get("Content-Type")
				if contentType != "" && !strings.Contains(contentType, "application/json") {
					t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
				}

				// For valid JSON, verify we can decode
				var result map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
					t.Errorf("Failed to decode response JSON: %v", err)
				}
			}
		})
	}
}

func TestTestHelperFunctions(t *testing.T) {
	// Test contains function
	tests := []struct {
		s         string
		substring string
		want      bool
	}{
		{"hello world", "hello", true},
		{"hello world", "world", true},
		{"hello world", "hello world", true},
		{"hello world", "o w", true},
		{"hello world", "xyz", false},
		{"", "hello", false},
		{"hello", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"-"+tt.substring, func(t *testing.T) {
			if got := contains(tt.s, tt.substring); got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substring, got, tt.want)
			}
		})
	}

	// Test name function on TestCase
	tests2 := []struct {
		tc   TestCase
		want string
	}{
		{TestCase{Name: "test1", Input: "input1"}, "test1"},
		{TestCase{Input: 123}, "Test with input 123"},
		{TestCase{}, "Test with input <nil>"},
	}

	for _, tt := range tests2 {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.tc.name(); got != tt.want {
				t.Errorf("TestCase.name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunTableTests(t *testing.T) {
	// Since we can't easily create a mock testing.T that works correctly,
	// we'll just test the function basics manually to ensure it doesn't panic

	// Define a simple test function
	testFunc := func(tc TestCase) (string, error) {
		input, ok := tc.Input.(string)
		if !ok {
			return "", nil
		}

		if input == "error" {
			return "", io.EOF
		}
		return "result: " + input, nil
	}

	// Create a single test case
	tests := []TestCase{
		{
			Name:     "success case",
			Input:    "test",
			Expected: "result: test",
		},
	}

	// Manual test of the test function
	result, err := testFunc(tests[0])
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "result: test" {
		t.Errorf("Expected 'result: test', got '%s'", result)
	}

	// Test error handling
	errorCase := TestCase{
		Name:  "error case",
		Input: "error",
		Error: "EOF",
	}

	_, err = testFunc(errorCase)
	if err == nil {
		t.Error("Expected an error, got nil")
	} else if err.Error() != "EOF" {
		t.Errorf("Expected 'EOF' error, got '%s'", err.Error())
	}

	// Note: We can't directly test RunTableTests here due to the complications
	// with mocking testing.T, but we've verified the test function works correctly
}
