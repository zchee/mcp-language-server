package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

type LSPCommand struct {
	WorkspaceDir string
	Command      string
	Args         []string
}

// Parse a command string into command and arguments
func parseLSPCommand(cmdStr string) LSPCommand {
	parts := strings.Fields(cmdStr)
	return LSPCommand{
		Command: parts[0],
		Args:    parts[1:],
	}
}

func detectLanguageID(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".py":
		return "python"
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	default:
		return "plaintext"
	}
}

func main() {
	// Define command line flags
	var (
		workspaceDir  string
		lspCommandStr string
		filePath      string
	)

	flag.StringVar(&workspaceDir, "workspace", "", "Path to workspace directory (optional)")
	flag.StringVar(&lspCommandStr, "lsp", "gopls", "LSP command to run (e.g., 'gopls -remote=auto')")
	flag.StringVar(&filePath, "file", "", "File to analyze (optional)")
	flag.Parse()

	if filePath == "" {
		log.Fatal("File path is required. Use -file flag.")
	}

	// Convert file path to absolute path
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for file: %v", err)
	}

	// Handle workspace directory
	if workspaceDir == "" {
		workspaceDir = filepath.Dir(absFilePath)
	} else {
		// Convert workspace to absolute path if provided
		workspaceDir, err = filepath.Abs(workspaceDir)
		if err != nil {
			log.Fatalf("Failed to get absolute path for workspace: %v", err)
		}
	}

	// Validate workspace directory
	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		log.Fatalf("Workspace directory does not exist: %s", workspaceDir)
	}

	// Validate file exists
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %s", absFilePath)
	}

	// Parse LSP command
	lspCmd := parseLSPCommand(lspCommandStr)

	// Verify the LSP command exists
	if _, err := exec.LookPath(lspCmd.Command); err != nil {
		log.Fatalf("LSP command not found: %s", lspCmd.Command)
	}

	// Change to the workspace directory
	if err := os.Chdir(workspaceDir); err != nil {
		log.Fatalf("Failed to change to workspace directory: %v", err)
	}

	fmt.Printf("Using workspace: %s\n", workspaceDir)
	fmt.Printf("Analyzing file: %s\n", absFilePath)

	// Create a new LSP client
	fmt.Printf("Starting %s...\n", lspCmd.Command)
	client, err := lsp.NewClient(lspCmd.Command, lspCmd.Args...)
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

	// Initialize
	ctx := context.Background()
	initResult, err := client.InitializeLSPClient(ctx)
	if err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}
	fmt.Printf("Server capabilities: %+v\n\n", initResult.Capabilities)

	if initResult.Capabilities.DocumentSymbolProvider == nil {
		log.Fatal("Server does not support document symbols")
	}

	// Send initialized notification
	err = client.Initialized(ctx, protocol.InitializedParams{})
	if err != nil {
		log.Fatalf("Initialized notification failed: %v", err)
	}

	// Read the file content
	fileContent, err := os.ReadFile(absFilePath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Test textDocument/didOpen
	fmt.Println("Opening document...")
	uri := protocol.DocumentURI("file://" + absFilePath)
	languageID := protocol.LanguageKind(detectLanguageID(absFilePath))
	fmt.Printf("Using language ID: %s\n", languageID)

	err = client.DidOpen(ctx, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    1,
			Text:       string(fileContent),
		},
	})
	if err != nil {
		log.Fatalf("TextDocumentDidOpen failed: %v", err)
	}
	fmt.Println("Document opened successfully")

	// Test textDocument/documentSymbol
	fmt.Println("\nGetting document symbols...")
	symbols, err := client.DocumentSymbol(ctx, protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		log.Fatalf("TextDocumentDocumentSymbol failed: %v", err)
	}

	// Now check the resulting type
	switch v := symbols.Value.(type) {
	case []protocol.SymbolInformation:
		fmt.Println("Got SymbolInformation response:")
		for _, sym := range v {
			fmt.Printf("- %s (%v) at line %d\n", sym.Name, sym.Kind, sym.Location.Range.Start.Line+1)
		}
	case []protocol.DocumentSymbol:
		fmt.Println("Got DocumentSymbol response:")
		printDocumentSymbols(v, "  ")
	default:
		fmt.Printf("Unexpected symbol response type: %T\n", symbols.Value)
		jsonBytes, _ := json.MarshalIndent(symbols, "", "  ")
		fmt.Printf("Raw response:\n%s\n", string(jsonBytes))
	}

	// Test textDocument/didClose
	fmt.Println("\nClosing document...")
	err = client.DidClose(ctx, protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		log.Fatalf("TextDocumentDidClose failed: %v", err)
	}
	fmt.Println("Document closed successfully")

	// Cleanup
	fmt.Println("\nShutting down...")
	err = client.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}

	err = client.Exit(ctx)
	if err != nil {
		log.Fatalf("Exit failed: %v", err)
	}
	fmt.Println("Server shut down successfully")
}

func printDocumentSymbols(symbols []protocol.DocumentSymbol, indent string) {
	for _, sym := range symbols {
		// Print current symbol with indentation
		fmt.Printf("%s%s (%v) line %d",
			indent,
			sym.Name,
			sym.Kind,
			sym.Range.Start.Line+1,
		)

		// Print detail if available
		if sym.Detail != "" {
			fmt.Printf(" - %s", sym.Detail)
		}

		// Print deprecated status if true
		if sym.Deprecated {
			fmt.Printf(" (deprecated)")
		}

		// Print tags if any
		if len(sym.Tags) > 0 {
			fmt.Printf(" tags: %v", sym.Tags)
		}

		fmt.Println() // End the line

		// Recursively print children with increased indentation
		if len(sym.Children) > 0 {
			printDocumentSymbols(sym.Children, indent+"  ")
		}
	}
}
