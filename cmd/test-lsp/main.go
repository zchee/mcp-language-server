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
	if len(parts) == 0 {
		return LSPCommand{Command: "gopls"} // default to gopls with no args
	}
	return LSPCommand{
		WorkspaceDir: parts[0],
		Command:      parts[1],
		Args:         parts[2:],
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
		debug         bool
	)

	flag.StringVar(&workspaceDir, "workspace", "", "Path to workspace directory (optional)")
	flag.StringVar(&lspCommandStr, "lsp", "gopls", "LSP command to run (e.g., 'gopls -remote=auto')")
	flag.StringVar(&filePath, "file", "", "File to analyze (optional)")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
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
	if debug {
		fmt.Printf("Server capabilities: %+v\n\n", initResult.Capabilities)
	}

	if initResult.Capabilities.DocumentSymbolProvider == nil {
		log.Fatal("Server does not support document symbols")
	}

	// Send initialized notification
	err = wrapper.Initialized(protocol.InitializedParams{})
	if err != nil {
		log.Fatalf("Initialized notification failed: %v", err)
	}

	// Send a virtual file to trigger analysis
	virtualURI := protocol.DocumentURI("file://" + filepath.Join(workspaceDir, "__virtual_init__.py"))
	err = wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        virtualURI,
			LanguageID: "python",
			Version:    1,
			Text:       "# Virtual file to trigger pyright analysis\n",
		},
	})

	if err != nil {
		log.Fatalf("TextDocuemntDidOpen failed: %v", err)
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

	err = wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
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
	symbols, err := wrapper.TextDocumentDocumentSymbol(protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		log.Fatalf("TextDocumentDocumentSymbol failed: %v", err)
	}

	if symbols == nil {
		fmt.Println("No symbols returned")
	} else {
		fmt.Printf("Raw symbol response type: %T\n", symbols)

		// Try to handle both possible response types
		switch v := symbols.(type) {
		case []protocol.DocumentSymbol:
			fmt.Println("Got DocumentSymbol response:")
			for _, sym := range v {
				fmt.Printf("- %s (%v) at line %d\n", sym.Name, sym.Kind, sym.Range.Start.Line+1)
			}
		case []protocol.SymbolInformation:
			fmt.Println("Got SymbolInformation response:")
			for _, sym := range v {
				fmt.Printf("- %s (%v) at line %d\n", sym.Name, sym.Kind, sym.Location.Range.Start.Line+1)
			}
		default:
			fmt.Printf("Unexpected symbol response type: %T\n", symbols)
			// Print the raw response for debugging
			if debug {
				jsonBytes, _ := json.MarshalIndent(symbols, "", "  ")
				fmt.Printf("Raw response:\n%s\n", string(jsonBytes))
			}
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
