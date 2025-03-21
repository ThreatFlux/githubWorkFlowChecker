package main

import (
	"bytes"
	"os"
	"testing"
)

// TestSysOutCall tests both paths of the SysOutCall function:
// 1. When inTestMode is true (default for tests)
// 2. When inTestMode is false (production mode)
func TestSysOutCall(t *testing.T) {
	// Save original values to restore after test
	origTestMode := inTestMode
	origStdout := os.Stdout

	defer func() {
		// Restore original values
		inTestMode = origTestMode
		os.Stdout = origStdout
	}()

	// First test with inTestMode=true (should be a no-op)
	inTestMode = true
	SysOutCall() // This should be a no-op

	// Now test with inTestMode=false by replacing stdout
	// Create a pipe to capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Replace stdout with our pipe
	os.Stdout = w

	// Set inTestMode to false to execute the sync operation
	inTestMode = false

	// Call SysOutCall which should now call os.Stdout.Sync()
	SysOutCall()

	// Close the write end of the pipe
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe writer: %v", err)
	}

	// Read any output (should be empty since Sync doesn't write anything)
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	// Close the read end of the pipe
	if err := r.Close(); err != nil {
		t.Fatalf("Failed to close pipe reader: %v", err)
	}
}
