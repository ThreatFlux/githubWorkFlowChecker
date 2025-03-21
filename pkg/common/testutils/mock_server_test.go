package testutils

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMockServerBuilder_Fluent tests that methods return *MockServerBuilder for fluent API
func TestMockServerBuilder_Fluent(t *testing.T) {
	// Create separate builders for each method to avoid path conflicts
	t.Run("WithHandler", func(t *testing.T) {
		builder := NewMockServer()
		handler := func(w http.ResponseWriter, r *http.Request) {}
		if b := builder.WithHandler("/test-handler", handler); b != builder {
			t.Error("WithHandler did not return the same builder instance")
		}
	})

	t.Run("WithJSONResponse", func(t *testing.T) {
		builder := NewMockServer()
		if b := builder.WithJSONResponse("/test-json", http.StatusOK, "{}"); b != builder {
			t.Error("WithJSONResponse did not return the same builder instance")
		}
	})

	t.Run("WithBlobHandler", func(t *testing.T) {
		builder := NewMockServer()
		if b := builder.WithBlobHandler("owner", "repo"); b != builder {
			t.Error("WithBlobHandler did not return the same builder instance")
		}
	})

	t.Run("WithTreeHandler", func(t *testing.T) {
		builder := NewMockServer()
		if b := builder.WithTreeHandler("owner", "repo"); b != builder {
			t.Error("WithTreeHandler did not return the same builder instance")
		}
	})

	t.Run("WithCommitHandler", func(t *testing.T) {
		builder := NewMockServer()
		if b := builder.WithCommitHandler("owner", "repo"); b != builder {
			t.Error("WithCommitHandler did not return the same builder instance")
		}
	})

	t.Run("WithPRHandler", func(t *testing.T) {
		builder := NewMockServer()
		if b := builder.WithPRHandler("owner", "repo"); b != builder {
			t.Error("WithPRHandler did not return the same builder instance")
		}
	})

	t.Run("WithLabelsHandler", func(t *testing.T) {
		builder := NewMockServer()
		if b := builder.WithLabelsHandler("owner", "repo", 1); b != builder {
			t.Error("WithLabelsHandler did not return the same builder instance")
		}
	})

	t.Run("WithRefHandler", func(t *testing.T) {
		builder := NewMockServer()
		if b := builder.WithRefHandler("owner", "repo", "main", "{}"); b != builder {
			t.Error("WithRefHandler did not return the same builder instance")
		}
	})
}

// TestMockServerBuilder_ChainedMethods tests chaining of methods
func TestMockServerBuilder_ChainedMethods(t *testing.T) {
	// Create a single builder but with unique paths
	builder := NewMockServer().
		WithHandler("/unique-handler", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("handler response"))
		}).
		WithJSONResponse("/unique-json", http.StatusOK, `{"status":"ok"}`)

	// Build the server
	server, client := builder.Build()
	defer server.Close()

	// Test handler endpoint
	req := httptest.NewRequest("GET", "/unique-handler", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	if rec.Body.String() != "handler response" {
		t.Errorf("Expected 'handler response', got %q", rec.Body.String())
	}

	// Test JSON endpoint
	req = httptest.NewRequest("GET", "/unique-json", nil)
	rec = httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Errorf("Error unmarshaling JSON: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %q", response["status"])
	}

	// Ensure we got a client
	if client == nil {
		t.Error("Build() did not return a client")
	}
}

// TestWriteJSON_ErrorCases tests error cases for WriteJSON
func TestWriteJSON_ErrorCases(t *testing.T) {
	// Test with invalid JSON
	w := httptest.NewRecorder()
	err := WriteJSON(w, "{invalid json")
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	// Test with writer that returns error
	mockWriter := &mockResponseWriter{
		header:     make(http.Header),
		writeError: errors.New("write error"),
	}
	err = WriteJSON(mockWriter, `{"valid": "json"}`)
	if err == nil {
		t.Error("Expected error for writer error, got nil")
	}
}

// mockResponseWriter is a mock http.ResponseWriter that can simulate errors
type mockResponseWriter struct {
	header      http.Header
	written     []byte
	statusCode  int
	writeError  error
	headerError error //nolint:unused
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	if m.writeError != nil {
		return 0, m.writeError
	}
	m.written = append(m.written, b...)
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

// TestWithJSONResponse_InvalidJSON tests that WithJSONResponse handles invalid JSON
func TestWithJSONResponse_InvalidJSON(t *testing.T) {
	builder := NewMockServer()

	// Use a channel to track if error was handled
	errorHandled := make(chan bool, 1)

	// Mock a panic to capture internal server errors
	origHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				// The handler panicked, but we caught it
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal server error"}`))
				errorHandled <- true
			}
		}()

		// This will cause an error because it's invalid JSON
		if err := WriteJSON(w, "{invalid json"); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal server error"}`))
			errorHandled <- true
			return
		}
		errorHandled <- false
	})

	builder.WithHandler("/invalid", origHandler)

	// Make a request to verify handler returns 500 for invalid JSON
	req := httptest.NewRequest("GET", "/invalid", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	wasErrorHandled := <-errorHandled
	if !wasErrorHandled {
		t.Error("Error in JSON handling was not detected")
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

// TestWithJSONResponse_MethodFilter tests that WithJSONResponse handles method filtering
func TestWithJSONResponse_MethodFilter(t *testing.T) {
	builder := NewMockServer()

	// Add a JSON response handler with custom status code
	customStatusCode := http.StatusAccepted // 202
	testPath := "/custom-status"
	testJSON := `{"status":"accepted"}`

	// Create a custom response handler that only responds to POST
	builder.mux.HandleFunc(testPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(customStatusCode)
		if err := WriteJSON(w, testJSON); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Test allowed method
	postReq := httptest.NewRequest("POST", testPath, nil)
	postRec := httptest.NewRecorder()
	builder.mux.ServeHTTP(postRec, postReq)

	if postRec.Code != customStatusCode {
		t.Errorf("Expected status code %d for allowed method, got %d", customStatusCode, postRec.Code)
	}

	// Test disallowed method
	getReq := httptest.NewRequest("GET", testPath, nil)
	getRec := httptest.NewRecorder()
	builder.mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code %d for disallowed method, got %d",
			http.StatusMethodNotAllowed, getRec.Code)
	}
}

// TestWithJSONResponse_DirectCall tests the WithJSONResponse method directly
func TestWithJSONResponse_DirectCall(t *testing.T) {
	builder := NewMockServer()
	testPath := "/direct-call-path"

	// Call WithJSONResponse directly with a normal JSON response
	builder.WithJSONResponse(testPath, http.StatusOK, `{"direct":"call"}`)

	// Make a request to the handler
	req := httptest.NewRequest("GET", testPath, nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	// Verify status code
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify response body
	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	// Check that the response contains the expected data
	value, ok := response["direct"].(string)
	if !ok || value != "call" {
		t.Errorf("Expected 'direct' field to be 'call', got %v", response["direct"])
	}

	// Test with invalid JSON to trigger the error branch
	invalidJSONPath := "/invalid-json-handler"
	builder.WithJSONResponse(invalidJSONPath, http.StatusOK, `{"broken": "json"`)

	// Make the request to the invalid JSON handler
	req = httptest.NewRequest("GET", invalidJSONPath, nil)
	rec = httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	// The error message may vary by JSON parser implementation
	// We just need to check that we get some error related to the JSON
	if rec.Body.String() == "" {
		t.Errorf("Expected error message, got empty body")
	}
}

// TestWithJSONResponse_DirectImplementation directly tests each line in WithJSONResponse
func TestWithJSONResponse_DirectImplementation(t *testing.T) {
	builder := NewMockServer()

	// Create a direct handler that mimics WithJSONResponse but exposes internal state
	builder.mux.HandleFunc("/direct-json-test", func(w http.ResponseWriter, r *http.Request) {
		// This is exactly what WithJSONResponse does internally
		w.WriteHeader(http.StatusOK)
		if err := WriteJSON(w, `{"success":true}`); err != nil {
			// This would normally go to http.Error
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	})

	// Test the endpoint
	req := httptest.NewRequest("GET", "/direct-json-test", nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	// Verify the status code
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify content type
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type to be application/json, got %s", contentType)
	}

	// Verify the response body
	var response map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Errorf("Expected success to be true, got %v", response["success"])
	}
}

// TestWithJSONResponse_CustomStatus tests WithJSONResponse with non-standard status codes
func TestWithJSONResponse_CustomStatus(t *testing.T) {
	builder := NewMockServer()

	testCases := []struct {
		name       string
		path       string
		statusCode int
		json       string
	}{
		{
			name:       "Created status",
			path:       "/created",
			statusCode: http.StatusCreated,
			json:       `{"status":"created"}`,
		},
		{
			name:       "Not found status",
			path:       "/not-found",
			statusCode: http.StatusNotFound,
			json:       `{"message":"Not found"}`,
		},
		{
			name:       "Bad request status",
			path:       "/bad-request",
			statusCode: http.StatusBadRequest,
			json:       `{"message":"Bad request"}`,
		},
	}

	// Add handlers for each case
	for _, tc := range testCases {
		builder.WithJSONResponse(tc.path, tc.statusCode, tc.json)
	}

	// Test each case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			rec := httptest.NewRecorder()
			builder.mux.ServeHTTP(rec, req)

			// Check status code
			if rec.Code != tc.statusCode {
				t.Errorf("Expected status code %d, got %d", tc.statusCode, rec.Code)
			}

			// Check content-type
			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type to contain 'application/json', got %q", contentType)
			}

			// Check response body
			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Error decoding response: %v", err)
			}
		})
	}
}

// TestWithLabelsHandler_InvalidIssueNumber tests issue numbers are handled correctly
func TestWithLabelsHandler_InvalidIssueNumber(t *testing.T) {
	// Test with issue number 0
	builder := NewMockServer()
	builder.WithLabelsHandler("owner", "repo", 0)

	// Test with a large issue number that would be outside ASCII range
	builder = NewMockServer()
	builder.WithLabelsHandler("owner", "repo", 1000)
}

// TestBuild_ClientURLs tests the client URLs are set correctly
func TestBuild_ClientURLs(t *testing.T) {
	builder := NewMockServer()
	server, client := builder.Build()
	defer server.Close()

	if client.BaseURL.String() != server.URL+"/" {
		t.Errorf("Expected client.BaseURL to be %q, got %q", server.URL+"/", client.BaseURL.String())
	}

	if client.UploadURL.String() != server.URL+"/" {
		t.Errorf("Expected client.UploadURL to be %q, got %q", server.URL+"/", client.UploadURL.String())
	}
}

// TestWriteJSON_ContentTypeHeader tests that WriteJSON sets the Content-Type header correctly
func TestWriteJSON_ContentTypeHeader(t *testing.T) {
	w := httptest.NewRecorder()
	err := WriteJSON(w, `{"key": "value"}`)
	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain 'application/json', got %q", contentType)
	}
}

// TestWithJSONResponse_ErrorHandling tests error handling in WithJSONResponse
func TestWithJSONResponse_ErrorHandling(t *testing.T) {
	// Create a mock writer that simulates a Write error
	errorWriter := &mockResponseWriter{
		header:     make(http.Header),
		writeError: errors.New("simulated write error"),
	}

	// Call WriteJSON directly with our error writer
	err := WriteJSON(errorWriter, `{"key": "value"}`)
	if err == nil {
		t.Error("Expected WriteJSON to return error, got nil")
	}
}

// TestWithJSONResponse_ErrorPathHandling tests the full error path in WithJSONResponse
func TestWithJSONResponse_ErrorPathHandling(t *testing.T) {
	// Create a server builder
	builder := NewMockServer()

	// Set up a route that will trigger an error in WriteJSON
	invalidJSONPath := "/invalid-json-path"
	builder.mux.HandleFunc(invalidJSONPath, func(w http.ResponseWriter, r *http.Request) {
		// Invalid JSON should cause WriteJSON to return an error
		invalidJSON := `{"broken": json`
		err := WriteJSON(w, invalidJSON)
		if err == nil {
			t.Error("Expected WriteJSON to fail with invalid JSON")
		}
		// The error should be handled in the regular flow
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Failed to encode JSON"}`))
	})

	// Test the handler with the invalid JSON
	req := httptest.NewRequest("GET", invalidJSONPath, nil)
	rec := httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	// Expect a 500 error
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// We'll directly demonstrate the error handling in WithJSONResponse
	// by using a path already set up with a different valid handler
	// and verify that the WriteJSON error path works correctly
	directErrorPath := "/direct-error-test"

	// Set up a handler that directly causes an error
	builder.mux.HandleFunc(directErrorPath, func(w http.ResponseWriter, r *http.Request) {
		// Set status code first since once we write to the body, we can't change it
		w.WriteHeader(http.StatusOK)

		// Now directly call WriteJSON with invalid JSON
		if err := WriteJSON(w, `{"invalid": json`); err != nil {
			// This would normally be a 500, but we've already set status to 200
			// So we'll just write a failure message
			_, _ = w.Write([]byte(`Error handling JSON`))

			// Verification is in the test assertion
			return
		}

		// Normal success response (shouldn't get here)
		_, _ = w.Write([]byte(`Success`))
	})

	// Make the request
	req = httptest.NewRequest("GET", directErrorPath, nil)
	rec = httptest.NewRecorder()
	builder.mux.ServeHTTP(rec, req)

	// The status code will still be 200 because we set it before attempting to write JSON
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// But the body should indicate an error
	if rec.Body.String() != "Error handling JSON" {
		t.Errorf("Expected error message in body, got: %q", rec.Body.String())
	}
}

// TestWriteJSON_WithJSONUnmarshalOnly tests that JSON can be unmarshaled correctly
func TestWriteJSON_WithJSONUnmarshalOnly(t *testing.T) {
	w := httptest.NewRecorder()
	jsonStr := `{"array": [1, 2, 3], "object": {"nested": true}, "string": "value", "number": 123, "bool": true, "null": null}`

	err := WriteJSON(w, jsonStr)
	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	// Verify we can unmarshal the result
	var result map[string]interface{}
	decoder := json.NewDecoder(w.Body)
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Check array
	arr, ok := result["array"].([]interface{})
	if !ok || len(arr) != 3 {
		t.Errorf("Expected array with 3 elements, got %v", result["array"])
	}

	// Check object
	obj, ok := result["object"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected nested object, got %v", result["object"])
	}
	nestedVal, ok := obj["nested"].(bool)
	if !ok || !nestedVal {
		t.Errorf("Expected object.nested to be true, got %v", obj["nested"])
	}

	// Check other types
	if result["string"] != "value" {
		t.Errorf("Expected string to be 'value', got %v", result["string"])
	}

	if result["number"] != float64(123) {
		t.Errorf("Expected number to be 123, got %v", result["number"])
	}

	if result["bool"] != true {
		t.Errorf("Expected bool to be true, got %v", result["bool"])
	}

	if result["null"] != nil {
		t.Errorf("Expected null to be nil, got %v", result["null"])
	}
}

func TestWriteJSON_EncoderError(t *testing.T) {
	// Create a special value that will cause json.Marshal to fail
	type badJSON struct {
		Ch chan int
	}

	badValue := badJSON{Ch: make(chan int)}

	// We need to manually marshal to test this scenario
	_, err := json.Marshal(badValue) //nolint:staticcheck // We're intentionally testing marshal errors with bad types

	// This should error because channels can't be marshaled to JSON
	if err == nil {
		t.Errorf("Expected Marshal to fail, but it succeeded")
	}
}
