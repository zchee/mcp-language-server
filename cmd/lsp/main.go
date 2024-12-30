package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
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

	// Create document manager
	docManager := lsp.NewDocumentManager(client)

	// Example: Open and analyze this file
	filepath := "/Users/phil/dev/mcp-language-server/cmd/lsp/main.go"
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Open document
	uri := protocol.DocumentURI("file://" + filepath)
	err = docManager.OpenDocument(uri, "go", string(content))
	if err != nil {
		log.Fatalf("Failed to open document: %v", err)
	}

	// Get document symbols
	symbols, err := docManager.GetDocumentSymbols(uri)
	if err != nil {
		log.Fatalf("Failed to get document symbols: %v", err)
	}

	// Print symbols
	fmt.Println("\nDocument symbols:")
	printSymbols(symbols, 0)
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
