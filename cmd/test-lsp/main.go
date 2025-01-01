package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/lsp/methods"
	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

func main() {
	// Create a temporary workspace
	tmpDir, err := os.MkdirTemp("", "lsp-test-*")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a simple Go file
	testFile := filepath.Join(tmpDir, "main.go")
	fileContent := []byte(`package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func add(a, b int) int {
	return a + b
}
`)
	err = os.WriteFile(testFile, fileContent, 0644)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}

	// Change to the workspace directory
	if err := os.Chdir(tmpDir); err != nil {
		log.Fatalf("Failed to change directory: %v", err)
	}

	// Create a new LSP client for gopls
	fmt.Println("Starting gopls...")
	client, err := lsp.NewClient("gopls")
	if err != nil {
		log.Fatalf("Failed to create LSP client: %v", err)
	}
	defer client.Close()

	// Create the wrapper for type-safe method calls
	wrapper := methods.NewWrapper(client)

	// Register notification handler for window/showMessage
	client.RegisterNotificationHandler("window/showMessage", func(method string, params json.RawMessage) {
		var msg struct {
			Type    int    `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &msg); err == nil {
			fmt.Printf("Server message: %s\n", msg.Message)
		}
	})

	// Initialize
	initResult, err := client.Initialize()
	if err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}
	fmt.Printf("Server initialized with capabilities: %+v\n\n", initResult.Capabilities)

	// Send initialized notification
	err = wrapper.Initialized(protocol.InitializedParams{})
	if err != nil {
		log.Fatalf("Initialized notification failed: %v", err)
	}

	// Test textDocument/didOpen
	fmt.Println("Testing textDocument/didOpen...")
	uri := protocol.DocumentURI("file://" + testFile)
	err = wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       string(fileContent),
		},
	})
	if err != nil {
		log.Fatalf("TextDocumentDidOpen failed: %v", err)
	}
	fmt.Println("TextDocumentDidOpen successful")

	// Test textDocument/documentSymbol
	fmt.Println("\nTesting textDocument/documentSymbol...")
	symbols, err := wrapper.TextDocumentDocumentSymbol(protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		log.Fatalf("TextDocumentDocumentSymbol failed: %v", err)
	}
	documentSymbols, ok := symbols.([]protocol.DocumentSymbol)
	if !ok {
		fmt.Println("Got SymbolInformation instead of DocumentSymbol")
	} else {
		fmt.Println("Document symbols:")
		for _, sym := range documentSymbols {
			fmt.Printf("- %s (%s) at line %d\n", sym.Name, sym.Kind, sym.Range.Start.Line+1)
		}
	}

	// Test textDocument/hover
	fmt.Println("\nTesting textDocument/hover...")
	hover, err := wrapper.TextDocumentHover(protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:    protocol.Position{Line: 8, Character: 5}, // Position on 'add' function
		},
	})
	if err != nil {
		log.Fatalf("TextDocumentHover failed: %v", err)
	}
	fmt.Printf("Hover result: %+v\n", hover)

	// Test textDocument/didClose
	fmt.Println("\nTesting textDocument/didClose...")
	err = wrapper.TextDocumentDidClose(protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		log.Fatalf("TextDocumentDidClose failed: %v", err)
	}
	fmt.Println("TextDocumentDidClose successful")

	// Cleanup
	fmt.Println("\nShutting down...")
	err = wrapper.Shutdown()
	if err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}

	err = wrapper.Exit()
	if err != nil {
		log.Fatalf("Exit failed: %v", err)
	}
	fmt.Println("Server shut down successfully")
}
