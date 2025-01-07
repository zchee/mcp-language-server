package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
	"github.com/isaacphi/mcp-language-server/internal/tools"
	"github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

type GetDefinitionArgs struct {
	SymbolName string `json:"symbolName" jsonschema:"required,description=The exact name of the symbol (function, class or something else) to fetch."`
}

type LSPCommand struct {
	WorkspaceDir string
	Command      string
	Args         []string
}

func parseLSPCommand(cmdStr string) LSPCommand {
	parts := strings.Fields(cmdStr)
	return LSPCommand{
		Command: parts[0],
		Args:    parts[1:],
	}
}

func main() {
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

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for file: %v", err)
	}

	if workspaceDir == "" {
		workspaceDir = filepath.Dir(absFilePath)
	} else {
		// Convert workspace to absolute path if provided
		workspaceDir, err = filepath.Abs(workspaceDir)
		if err != nil {
			log.Fatalf("Failed to get absolute path for workspace: %v", err)
		}
	}

	if _, err := os.Stat(workspaceDir); os.IsNotExist(err) {
		log.Fatalf("Workspace directory does not exist: %s", workspaceDir)
	}
	if _, err := os.Stat(absFilePath); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %s", absFilePath)
	}

	lspCmd := parseLSPCommand(lspCommandStr)

	if _, err := exec.LookPath(lspCmd.Command); err != nil {
		log.Fatalf("LSP command not found: %s", lspCmd.Command)
	}

	if err := os.Chdir(workspaceDir); err != nil {
		log.Fatalf("Failed to change to workspace directory: %v", err)
	}

	log.Printf("Using workspace: %s\n", workspaceDir)
	log.Printf("Analyzing file: %s\n", absFilePath)

	log.Printf("Starting %s...\n", lspCmd.Command)
	client, err := lsp.NewClient(lspCmd.Command, lspCmd.Args...)
	if err != nil {
		log.Fatalf("Failed to create LSP client: %v", err)
	}
	defer client.Close()

	// Initialize
	initResult, err := client.InitializeLSPClient()
	if err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}
	if debug {
		log.Printf("Server capabilities: %+v\n\n", initResult.Capabilities)
	}

	if initResult.Capabilities.DocumentSymbolProvider == nil {
		log.Fatal("Server does not support document symbols")
	}

	// Send initialized notification
	err = client.Initialized(client.Ctx, protocol.InitializedParams{})
	if err != nil {
		log.Fatalf("Initialized notification failed: %v", err)
	}

	done := make(chan struct{})

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	err = server.RegisterTool(
		"Get definition",
		"Read the source code for a given symbol from the codebase",
		func(args GetDefinitionArgs) (*mcp_golang.ToolResponse, error) {
			content, err := tools.GetDefinition(args.SymbolName)
			if err != nil {
				return nil, err
			}
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(content)), nil
		})
	if err != nil {
		panic(err)
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	<-done
}
