package updater

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

func TestScannerRateLimiting(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "scanner-rate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set secure permissions
	if err := os.Chmod(tempDir, 0750); err != nil {
		t.Fatalf("Failed to set temp dir permissions: %v", err)
	}

	tests := []struct {
		name         string
		rateLimit    int
		rateDuration time.Duration
		operations   int
		minSuccesses int
		maxSuccesses int
	}{
		{
			name:         "basic rate limiting",
			rateLimit:    2,
			rateDuration: time.Second,
			operations:   3,
			minSuccesses: 2,
			maxSuccesses: 2,
		},
		{
			name:         "exceed rate limit",
			rateLimit:    1,
			rateDuration: time.Second * 2,
			operations:   3,
			minSuccesses: 1,
			maxSuccesses: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tempDir)
			scanner.SetRateLimit(tt.rateLimit, tt.rateDuration)

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()

			var (
				successCount int
				mu           sync.Mutex
				wg           sync.WaitGroup
			)

			// Launch operations with a small delay between them
			for i := 0; i < tt.operations; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := scanner.checkRateLimit(ctx)
					if err == nil {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
				}()
				// Small delay to ensure operations are properly sequenced
				time.Sleep(time.Millisecond * 10)
			}

			// Wait for all operations to complete
			wg.Wait()

			if successCount < tt.minSuccesses {
				t.Errorf("Got %d successes, want at least %d", successCount, tt.minSuccesses)
			}
			if successCount > tt.maxSuccesses {
				t.Errorf("Got %d successes, want at most %d", successCount, tt.maxSuccesses)
			}
		})
	}
}

func TestScannerTimeout(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "scanner-timeout-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set secure permissions
	if err := os.Chmod(tempDir, 0750); err != nil {
		t.Fatalf("Failed to set temp dir permissions: %v", err)
	}

	tests := []struct {
		name        string
		timeout     time.Duration
		sleepTime   time.Duration
		expectError bool
	}{
		{
			name:        "operation within timeout",
			timeout:     time.Second * 2,
			sleepTime:   time.Second,
			expectError: false,
		},
		{
			name:        "operation exceeds timeout",
			timeout:     time.Second,
			sleepTime:   time.Second * 2,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			done := make(chan error)
			go func() {
				// Simulate a long-running operation
				time.Sleep(tt.sleepTime)
				done <- scanner.checkTimeout(ctx)
			}()

			err := <-done
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestScannerConcurrentOperations(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "scanner-concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set secure permissions
	if err := os.Chmod(tempDir, 0750); err != nil {
		t.Fatalf("Failed to set temp dir permissions: %v", err)
	}

	scanner := NewScanner(tempDir)
	scanner.SetRateLimit(5, time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const numOperations = 10
	errChan := make(chan error, numOperations)

	// Start multiple concurrent operations
	for i := 0; i < numOperations; i++ {
		go func() {
			err := scanner.checkRateLimit(ctx)
			if err == nil {
				// Simulate some work
				time.Sleep(time.Millisecond * 100)
			}
			errChan <- err
		}()
	}

	// Collect results
	var (
		successCount int
		timeoutCount int
	)
	for i := 0; i < numOperations; i++ {
		err := <-errChan
		if err == nil {
			successCount++
		} else if err == context.DeadlineExceeded {
			timeoutCount++
		}
	}

	// We expect some operations to succeed and some to be rate limited
	if successCount == 0 {
		t.Error("Expected some operations to succeed")
	}
	if timeoutCount == 0 {
		t.Error("Expected some operations to be rate limited")
	}
	if successCount+timeoutCount != numOperations {
		t.Errorf("Expected %d total operations, got %d", numOperations, successCount+timeoutCount)
	}
}
