package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/lsp/methods"
	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

func main() {
	// Create a new LSP client
	command := os.Getenv("LSP_COMMAND")
	if command == "" {
		log.Fatal("LSP_COMMAND environment variable not set")
	}

	client, err := lsp.NewClient(command)
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

	// Initialize the client
	result, err := client.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize LSP client: %v", err)
	}

	// Pretty print the capabilities
	prettyJSON, err := json.MarshalIndent(result.Capabilities, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Server capabilities:\n%s\n", string(prettyJSON))

	// Example: Open and analyze this file
	filepath := "/Users/phil/dev/mcp-language-server/cmd/lsp/main.go"
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Open document using the wrapper
	uri := protocol.DocumentURI("file://" + filepath)
	err = wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       string(content),
		},
	})
	if err != nil {
		log.Fatalf("Failed to open document: %v", err)
	}

	// Get document symbols using the wrapper
	symbols, err := wrapper.TextDocumentDocumentSymbol(protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		log.Fatalf("Failed to get document symbols: %v", err)
	}

	// Convert interface{} to []protocol.DocumentSymbol
	documentSymbols, ok := symbols.([]protocol.DocumentSymbol)
	if !ok {
		log.Fatalf("Failed to convert symbols to DocumentSymbol slice")
	}

	// Print symbols
	fmt.Println("\nDocument symbols:")
	printSymbols(documentSymbols, 0)
}

// Helper function to print symbols with proper indentation
func printSymbols(symbols []protocol.DocumentSymbol, level int) {
	indent := strings.Repeat("  ", level)
	for _, symbol := range symbols {
		fmt.Printf("%s- %s (%s) at line %d\n",
			indent,
			symbol.Name,
			symbol.Kind,
			symbol.Range.Start.Line+1)

		if len(symbol.Children) > 0 {
			printSymbols(symbol.Children, level+1)
		}
	}
}
