package main

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// runWithExit captures stdout and returns the exit code and output
func runWithExit(f func()) (int, string) {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	// Create a WaitGroup to ensure we capture all output
	var wg sync.WaitGroup
	wg.Add(1)

	// Create a buffer to store the output
	var buf bytes.Buffer

	// Start a goroutine to read from the pipe
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, r)
	}()

	// Set stdout to the pipe writer
	os.Stdout = w

	// Save original osExit
	oldOsExit := osExit

	// Create a channel to receive the exit code
	exitCode := make(chan int, 1)

	// Override osExit to capture the exit code
	osExit = func(code int) {
		exitCode <- code
		// Don't actually exit, but do cleanup
		_ = w.Close()
		os.Stdout = oldStdout
		osExit = oldOsExit
	}

	// Run the function
	f()

	// If the function didn't call exit, clean up
	select {
	case code := <-exitCode:
		// Wait for all output to be read
		wg.Wait()
		// Close the reader
		_ = r.Close()
		// Return the exit code and output
		return code, buf.String()
	default:
		// Close the writer and restore stdout
		_ = w.Close()
		os.Stdout = oldStdout
		// Wait for all output to be read
		wg.Wait()
		// Close the reader
		_ = r.Close()
		// Restore original osExit
		osExit = oldOsExit
		// Return success and output
		return 0, buf.String()
	}
}
