package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/isaacphi/mcp-language-server/internal/logging"
	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/watcher"
)

// LSPTestConfig defines configuration for a language server test
type LSPTestConfig struct {
	Name             string   // Name of the language server
	Command          string   // Command to run
	Args             []string // Arguments
	WorkspaceDir     string   // Template workspace directory
	InitializeTimeMs int      // Time to wait after initialization in ms
}

// TestSuite contains everything needed for running integration tests
type TestSuite struct {
	Config       LSPTestConfig
	Client       *lsp.Client
	WorkspaceDir string
	TempDir      string
	Context      context.Context
	Cancel       context.CancelFunc
	Watcher      *watcher.WorkspaceWatcher
	initialized  bool
	cleanupOnce  sync.Once
	logFile      string
	t            *testing.T
}

// NewTestSuite creates a new test suite for the given language server
func NewTestSuite(t *testing.T, config LSPTestConfig) *TestSuite {
	ctx, cancel := context.WithCancel(context.Background())
	return &TestSuite{
		Config:      config,
		Context:     ctx,
		Cancel:      cancel,
		initialized: false,
		t:           t,
	}
}

// Setup initializes the test suite, copies the workspace, and starts the LSP
func (ts *TestSuite) Setup() error {
	if ts.initialized {
		return fmt.Errorf("test suite already initialized")
	}

	// Create test output directory in the repo
	
	// Navigate to the repo root (assuming tests run from within the repo)
	// The executable is in a temporary directory, so find the repo root based on the package path
	pkgDir, err := filepath.Abs("../../")
	if err != nil {
		return fmt.Errorf("failed to get absolute path to repo root: %w", err)
	}
	
	testOutputDir := filepath.Join(pkgDir, "test-output")
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create test-output directory: %w", err)
	}
	
	// Create a consistent directory for this language server
	// Extract the language name from the config
	langName := ts.Config.Name
	if langName == "" {
		langName = "unknown"
	}
	
	// Use a consistent directory name based on the language
	tempDir := filepath.Join(testOutputDir, langName)
	
	// Clean up previous test output
	if _, err := os.Stat(tempDir); err == nil {
		ts.t.Logf("Cleaning up previous test directory: %s", tempDir)
		if err := os.RemoveAll(tempDir); err != nil {
			ts.t.Logf("Warning: Failed to clean up previous test directory: %v", err)
		}
	}
	
	// Create a fresh directory
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create test directory: %w", err)
	}
	ts.TempDir = tempDir
	ts.t.Logf("Created test directory: %s", tempDir)

	// Set up logging
	logsDir := filepath.Join(tempDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Configure logging to write to a file
	ts.logFile = filepath.Join(logsDir, "test.log")
	if err := logging.SetupFileLogging(ts.logFile); err != nil {
		return fmt.Errorf("failed to set up logging: %w", err)
	}

	// Set log levels based on test configuration
	logging.SetGlobalLevel(logging.LevelInfo)

	// Enable debug logging for specific components
	if os.Getenv("DEBUG_LSP") == "true" {
		logging.SetLevel(logging.LSP, logging.LevelDebug)
	}
	if os.Getenv("DEBUG_LSP_WIRE") == "true" {
		logging.SetLevel(logging.LSPWire, logging.LevelDebug)
	}
	if os.Getenv("DEBUG_LSP_PROCESS") == "true" {
		logging.SetLevel(logging.LSPProcess, logging.LevelDebug)
	}
	if os.Getenv("DEBUG_WATCHER") == "true" {
		logging.SetLevel(logging.Watcher, logging.LevelDebug)
	}

	ts.t.Logf("Logs will be written to: %s", ts.logFile)

	// Copy workspace template
	workspaceDir := filepath.Join(tempDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	if err := copyDir(ts.Config.WorkspaceDir, workspaceDir); err != nil {
		return fmt.Errorf("failed to copy workspace template: %w", err)
	}
	ts.WorkspaceDir = workspaceDir
	ts.t.Logf("Copied workspace from %s to %s", ts.Config.WorkspaceDir, workspaceDir)

	// Create and initialize LSP client
	// TODO: Extend lsp.Client to support custom IO for capturing logs
	client, err := lsp.NewClient(ts.Config.Command, ts.Config.Args...)
	if err != nil {
		return fmt.Errorf("failed to create LSP client: %w", err)
	}
	ts.Client = client
	ts.t.Logf("Started LSP: %s %v", ts.Config.Command, ts.Config.Args)

	// Initialize LSP and set up file watcher
	initResult, err := client.InitializeLSPClient(ts.Context, workspaceDir)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}
	ts.t.Logf("LSP initialized with capabilities: %+v", initResult.Capabilities)

	ts.Watcher = watcher.NewWorkspaceWatcher(client)
	go ts.Watcher.WatchWorkspace(ts.Context, workspaceDir)

	if err := client.WaitForServerReady(ts.Context); err != nil {
		return fmt.Errorf("server failed to become ready: %w", err)
	}

	// Give watcher time to set up and scan workspace
	initializeTime := 1000 // Default 1 second
	if ts.Config.InitializeTimeMs > 0 {
		initializeTime = ts.Config.InitializeTimeMs
	}
	ts.t.Logf("Waiting %d ms for LSP to initialize", initializeTime)
	time.Sleep(time.Duration(initializeTime) * time.Millisecond)

	ts.initialized = true
	return nil
}

// Cleanup stops the LSP and cleans up resources
func (ts *TestSuite) Cleanup() {
	ts.cleanupOnce.Do(func() {
		ts.t.Logf("Cleaning up test suite")

		// Cancel context to stop watchers
		ts.Cancel()

		// Shutdown LSP
		if ts.Client != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			ts.t.Logf("Shutting down LSP client")
			err := ts.Client.Shutdown(shutdownCtx)
			if err != nil {
				ts.t.Logf("Shutdown failed: %v", err)
			}

			err = ts.Client.Exit(shutdownCtx)
			if err != nil {
				ts.t.Logf("Exit failed: %v", err)
			}

			err = ts.Client.Close()
			if err != nil {
				ts.t.Logf("Close failed: %v", err)
			}
		}

		// No need to close log files explicitly, logging package handles that

		ts.t.Logf("Test artifacts are in: %s", ts.TempDir)
		ts.t.Logf("To clean up, run: rm -rf %s", ts.TempDir)
	})
}

// ReadFile reads a file from the workspace
func (ts *TestSuite) ReadFile(relPath string) (string, error) {
	path := filepath.Join(ts.WorkspaceDir, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(data), nil
}

// WriteFile writes content to a file in the workspace
func (ts *TestSuite) WriteFile(relPath, content string) error {
	path := filepath.Join(ts.WorkspaceDir, relPath)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	// Give the watcher time to detect the file change
	time.Sleep(500 * time.Millisecond)
	return nil
}
