package methods

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

type testServer struct {
	t           *testing.T
	client      *lsp.Client
	wrapper     *Wrapper
	workDir     string
	initialized bool
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	// Create temporary workspace
	tmpDir, err := os.MkdirTemp("", "lsp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create client
	client, err := lsp.NewClient("gopls")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create LSP client: %v", err)
	}

	ts := &testServer{
		t:       t,
		client:  client,
		wrapper: NewWrapper(client),
		workDir: tmpDir,
	}

	t.Cleanup(func() {
		if ts.initialized {
			if err := ts.wrapper.Shutdown(); err != nil {
				t.Errorf("Shutdown failed: %v", err)
			}
			if err := ts.wrapper.Exit(); err != nil {
				t.Errorf("Exit failed: %v", err)
			}
		}
		ts.client.Close()
		os.RemoveAll(tmpDir)
	})

	return ts
}

func (ts *testServer) initialize() {
	ts.t.Helper()

	if ts.initialized {
		return
	}

	// Initialize
	_, err := ts.client.Initialize()
	if err != nil {
		ts.t.Fatalf("Initialize failed: %v", err)
	}

	// Send initialized notification
	err = ts.wrapper.Initialized(protocol.InitializedParams{})
	if err != nil {
		ts.t.Fatalf("Initialized notification failed: %v", err)
	}

	ts.initialized = true
}

func (ts *testServer) createFile(name, content string) protocol.DocumentURI {
	ts.t.Helper()

	fullPath := filepath.Join(ts.workDir, name)
	err := os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		ts.t.Fatalf("Failed to create test file: %v", err)
	}

	return protocol.DocumentURI("file://" + fullPath)
}

const sampleGoFile = `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func add(a, b int) int {
	return a + b
}
`

