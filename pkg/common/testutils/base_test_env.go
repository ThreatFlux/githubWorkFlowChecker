package testutils

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
)

// BaseTestEnvironment provides common utilities for all test environments
type BaseTestEnvironment struct {
	WorkDir      string
	T            *testing.T
	Commands     []*exec.Cmd
	ctx          context.Context
	cancel       context.CancelFunc
	cleanupFuncs []func() error // For deferred cleanup operations
}

// NewBaseTestEnvironment creates a new base test environment
func NewBaseTestEnvironment(t *testing.T, prefix string) *BaseTestEnvironment {
	// Create a temporary working directory
	workDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf(common.ErrFailedToCreateTempDir, err)
	}

	// Set appropriate permissions
	// #nosec G302 - Test directories need group/other access for test commands to work
	if err := os.Chmod(workDir, 0700); err != nil {
		_ = os.RemoveAll(workDir)
		t.Fatalf(common.ErrWrongDirectoryPermissions, err, 0700)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	return &BaseTestEnvironment{
		WorkDir:      workDir,
		T:            t,
		Commands:     make([]*exec.Cmd, 0),
		ctx:          ctx,
		cancel:       cancel,
		cleanupFuncs: make([]func() error, 0),
	}
}

// Cleanup removes the temporary directory and terminates any running processes
func (env *BaseTestEnvironment) Cleanup() {
	// Run deferred cleanup functions in reverse order
	for i := len(env.cleanupFuncs) - 1; i >= 0; i-- {
		if err := env.cleanupFuncs[i](); err != nil {
			env.T.Errorf("Cleanup function error: %v", err)
		}
	}

	// Cancel the context if it exists
	if env.cancel != nil {
		env.cancel()
	}

	// Terminate any running processes
	for _, cmd := range env.Commands {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}

	// Remove the temporary directory
	if env.WorkDir != "" {
		if err := os.RemoveAll(env.WorkDir); err != nil {
			env.T.Errorf(common.ErrFailedToRemoveTempDir, err)
		}
	}
}

// AddCleanupFunc adds a cleanup function to be executed during Cleanup
func (env *BaseTestEnvironment) AddCleanupFunc(fn func() error) {
	env.cleanupFuncs = append(env.cleanupFuncs, fn)
}

// Context returns the test environment context
func (env *BaseTestEnvironment) Context() context.Context {
	return env.ctx
}

// CreateCommand creates an exec.Cmd and tracks it for cleanup
func (env *BaseTestEnvironment) CreateCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	env.Commands = append(env.Commands, cmd)
	return cmd
}

// CreateCommandContext creates an exec.Cmd with context and tracks it for cleanup
func (env *BaseTestEnvironment) CreateCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	env.Commands = append(env.Commands, cmd)
	return cmd
}

// WithWorkingDir changes to the specified directory, executes the function,
// and returns to the original directory
func (env *BaseTestEnvironment) WithWorkingDir(dir string, fn func() error) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := os.Chdir(dir); err != nil {
		return err
	}

	defer func() {
		if err := os.Chdir(currentDir); err != nil {
			env.T.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	return fn()
}

// CreateSubDir creates a subdirectory in the test environment
func (env *BaseTestEnvironment) CreateSubDir(subdirPath string) (string, error) {
	fullPath := filepath.Join(env.WorkDir, subdirPath)
	if err := os.MkdirAll(fullPath, 0700); err != nil {
		return "", err
	}
	return fullPath, nil
}

// CreateFile creates a file with given content and permissions
func (env *BaseTestEnvironment) CreateFile(filePath string, content []byte, permissions os.FileMode) (string, error) {
	fullPath := filepath.Join(env.WorkDir, filePath)

	// Ensure parent directory exists
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return "", err
	}

	// Write the file
	if err := os.WriteFile(fullPath, content, permissions); err != nil {
		return "", err
	}

	return fullPath, nil
}

// MustCreateSubDir creates a subdirectory or fails the test
func (env *BaseTestEnvironment) MustCreateSubDir(subdirPath string) string {
	path, err := env.CreateSubDir(subdirPath)
	if err != nil {
		env.T.Fatalf(common.ErrFailedToCreateSubdir, err)
	}
	return path
}

// MustCreateFile creates a file or fails the test
func (env *BaseTestEnvironment) MustCreateFile(filePath string, content []byte, permissions os.FileMode) string {
	path, err := env.CreateFile(filePath, content, permissions)
	if err != nil {
		env.T.Fatalf(common.ErrFailedToCreateTestFile, err)
	}
	return path
}

// CreateWorkflowsDir creates the .github/workflows directory and returns its path
func (env *BaseTestEnvironment) CreateWorkflowsDir() string {
	return env.MustCreateSubDir(filepath.Join(".github", "workflows"))
}

// GetWorkflowsPath returns the full path to the .github/workflows directory
func (env *BaseTestEnvironment) GetWorkflowsPath() string {
	return filepath.Join(env.WorkDir, ".github", "workflows")
}
