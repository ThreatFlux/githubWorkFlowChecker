package main

import (
	"fmt"
	"path/filepath"
)

// Helper to restore original Abs function
func restoreAbs() {
	absFunc = filepath.Abs
}

// Helper to mock Abs function
func mockAbsWithError() {
	absFunc = func(path string) (string, error) {
		return "", fmt.Errorf("mock Abs error")
	}
}
