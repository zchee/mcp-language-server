package methods

import (
	"testing"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

func TestInitialize(t *testing.T) {
	ts := newTestServer(t)

	result, err := ts.client.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if result.Capabilities.TextDocumentSync == nil {
		t.Error("Expected TextDocumentSync capabilities")
	}

	err = ts.wrapper.Initialized(protocol.InitializedParams{})
	if err != nil {
		t.Errorf("Initialized notification failed: %v", err)
	}

	ts.initialized = true
}

func TestShutdownExit(t *testing.T) {
	ts := newTestServer(t)
	ts.initialize()

	err := ts.wrapper.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	err = ts.wrapper.Exit()
	if err != nil {
		t.Errorf("Exit failed: %v", err)
	}

	ts.initialized = false // Prevent double shutdown in cleanup
}