package updater

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// NewVersionChecker creates a new version checker instance
func NewVersionChecker(token string) VersionChecker {
	client := github.NewClient(nil)
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		client = github.NewClient(oauth2.NewClient(context.Background(), ts))
	}
	return &DefaultVersionChecker{client: client}
}

func BenchmarkScanWorkflows(b *testing.B) {
	// Set up test data directory
	testDataDir := filepath.Join(b.TempDir(), "test-repo")

	// Run the test with different workflow counts
	for _, count := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("workflows-%d", count), func(b *testing.B) {
			// Generate test workflows
			generateTestWorkflows(b, testDataDir, count)

			scanner := NewScanner()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				workflows, err := scanner.ScanWorkflows(filepath.Join(testDataDir, ".github", "workflows"))
				if err != nil {
					b.Fatal(err)
				}
				if len(workflows) != count {
					b.Fatalf("expected %d workflows, got %d", count, len(workflows))
				}

				// Parse action references
				for _, workflow := range workflows {
					_, err := scanner.ParseActionReferences(workflow)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkVersionChecker(b *testing.B) {
	// Create mock release
	tagName := "v4"
	mockRelease := &github.RepositoryRelease{
		TagName: &tagName,
	}

	// Create version checker with mock
	checker := &DefaultVersionChecker{
		client: github.NewClient(nil),
		mockGetLatestRelease: func(ctx context.Context, owner, repo string) (*github.RepositoryRelease, *github.Response, error) {
			return mockRelease, &github.Response{Response: &http.Response{StatusCode: http.StatusOK}}, nil
		},
	}

	action := ActionReference{
		Owner:   "actions",
		Name:    "checkout",
		Version: "v3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := checker.IsUpdateAvailable(context.Background(), action)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// mockTransport implements http.RoundTripper for testing
type mockTransport struct {
	Response *http.Response
	Error    error
}

func (t *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return t.Response, t.Error
}

func BenchmarkMemoryUsage(b *testing.B) {
	testDataDir := filepath.Join(b.TempDir(), "test-repo")
	generateTestWorkflows(b, testDataDir, 1000)

	b.ResetTimer()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	initialAlloc := m.Alloc

	scanner := NewScanner()
	workflows, err := scanner.ScanWorkflows(filepath.Join(testDataDir, ".github", "workflows"))
	if err != nil {
		b.Fatal(err)
	}

	runtime.ReadMemStats(&m)
	finalAlloc := m.Alloc

	b.ReportMetric(float64(finalAlloc-initialAlloc)/1024/1024, "MB_allocated")
	b.ReportMetric(float64(len(workflows)), "workflows_processed")
}

func BenchmarkConcurrentOperations(b *testing.B) {
	testDataDir := filepath.Join(b.TempDir(), "test-repo")
	generateTestWorkflows(b, testDataDir, 100)

	scanner := NewScanner()
	workflows, err := scanner.ScanWorkflows(filepath.Join(testDataDir, ".github", "workflows"))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for _, numGoroutines := range []int{1, 5, 10, 20} {
		b.Run(fmt.Sprintf("goroutines-%d", numGoroutines), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				done := make(chan bool)
				errors := make(chan error, numGoroutines)

				// Process workflows concurrently
				batchSize := len(workflows) / numGoroutines
				for j := 0; j < numGoroutines; j++ {
					start := j * batchSize
					end := start + batchSize
					if j == numGoroutines-1 {
						end = len(workflows)
					}

					go func(files []string) {
						for _, file := range files {
							_, err := scanner.ParseActionReferences(file)
							if err != nil {
								errors <- err
								return
							}
						}
						done <- true
					}(workflows[start:end])
				}

				// Wait for all goroutines
				for j := 0; j < numGoroutines; j++ {
					select {
					case err := <-errors:
						b.Fatal(err)
					case <-done:
						continue
					case <-time.After(5 * time.Second):
						b.Fatal("timeout waiting for goroutine")
					}
				}
			}
		})
	}
}

func generateTestWorkflows(b *testing.B, dir string, count int) {
	cmd := exec.Command("go", "run", "../../tools/generate-test-data.go", dir, fmt.Sprint(count))
	output, err := cmd.CombinedOutput()
	if err != nil {
		b.Fatalf("failed to generate test data: %v\nOutput: %s", err, output)
	}
}

func init() {
	// Enable CPU and memory profiling
	if os.Getenv("BENCH_PROFILE") == "1" {
		f, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		go func() {
			http.ListenAndServe("localhost:6060", nil)
		}()
	}
}
