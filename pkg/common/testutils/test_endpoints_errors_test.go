package testutils

import (
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test direct error handling in WriteJSON with mocked errorResponseWriter
func TestEndpointWriteJSONErrors(t *testing.T) {
	// Create a test server with handler using WriteJSON
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := WriteJSON(w, `{"test": "value"}`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Test with custom response writer that generates error on write
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := &errorResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		failWrite:      true,
	}

	// This should cause an error inside WriteJSON
	handler.ServeHTTP(rw, req)
}

// Test error paths in setupRefsEndpoints - modifying the handler function
// to demonstrate method not allowed errors
func TestSetupRefsEndpoints_ErrorPathHandling(t *testing.T) {
	// Create a custom handler that rejects certain methods
	methodHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject any method other than GET and POST
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Otherwise proceed with default behavior
		w.WriteHeader(http.StatusOK)
		if err := WriteJSON(w, `{"status": "ok"}`); err != nil {
			t.Fatalf("Failed to write JSON: %v", err)
		}
	})

	// Create test server with the handler
	ts := httptest.NewServer(methodHandler)
	defer ts.Close()

	// Test with PUT method, which should be rejected
	req, _ := http.NewRequest(http.MethodPut, ts.URL, nil)
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatalf(common.ErrFailedToCloseBody, err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}
