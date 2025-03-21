package testutils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Custom response writer for simulating error conditions
type errorResponseWriter struct {
	http.ResponseWriter
	failHeader bool
	failWrite  bool
}

func (e *errorResponseWriter) Header() http.Header {
	if e.failHeader {
		// Return nil header to simulate error
		return nil
	}
	return e.ResponseWriter.Header()
}

func (e *errorResponseWriter) Write(data []byte) (int, error) {
	if e.failWrite {
		return 0, &mockWriteError{}
	}
	return e.ResponseWriter.Write(data)
}

// Mock error for testing
type mockWriteError struct{}

func (m *mockWriteError) Error() string {
	return "simulated write error"
}

// Test response writer error path
func TestMockResponseWriter(t *testing.T) {
	w := &errorResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		failWrite:      true,
	}

	// Try to write to the error-producing writer
	_, err := w.Write([]byte("test"))

	if err == nil {
		t.Errorf("Expected error from Write, got nil")
	}

	// Test header failure too
	w.failHeader = true
	headers := w.Header()

	if headers != nil {
		t.Errorf("Expected nil headers, got %v", headers)
	}
}
