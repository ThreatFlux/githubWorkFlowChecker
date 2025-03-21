package testutils

import (
	"net/http/httptest"
	"testing"
)

// Test WriteJSON with error conditions
func TestWriteJSON_WithErrors(t *testing.T) {
	// Create a normal response recorder
	w := httptest.NewRecorder()

	// Test with invalid JSON
	err := WriteJSON(w, "{{invalid json")
	if err == nil {
		t.Errorf("Expected error with invalid JSON, got nil")
	}

	// Test with error response writer - but we need to properly initialize it first
	errorWriter := &errorResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		failWrite:      true,
	}

	err = WriteJSON(errorWriter, `{"test": "value"}`)
	if err == nil {
		t.Errorf("Expected error with failing writer, got nil")
	}
}
