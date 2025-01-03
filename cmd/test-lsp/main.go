package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/lsp/methods"
	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

type LSPCommand struct {
	Command string
	Args    []string
}

// Parse a command string into command and arguments
func parseLSPCommand(cmdStr string) LSPCommand {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return LSPCommand{Command: "gopls"} // default to gopls with no args
	}
	return LSPCommand{
		Command: parts[0],
		Args:    parts[1:],
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

	// Read the file content
	fileContent, err := os.ReadFile(absFilePath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Test textDocument/didOpen
	fmt.Println("Opening document...")
	uri := protocol.DocumentURI("file://" + absFilePath)
	err = wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go", // Note: This assumes Go files, might want to make this configurable
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

	// Test textDocument/didClose
	fmt.Println("\nClosing document...")
	err = wrapper.TextDocumentDidClose(protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		log.Fatalf("TextDocumentDidClose failed: %v", err)
	}
	fmt.Println("Document closed successfully")

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
