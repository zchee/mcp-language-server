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
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func main() {
	println("Hello, World!")
}
`), 0644)
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

	// Test 1: Initialize
	fmt.Println("Testing initialize...")
	initParams := &protocol.InitializeParams{
		XInitializeParams: protocol.XInitializeParams{
			ProcessID: int32(os.Getpid()),
			RootURI:   protocol.DocumentURI("file://" + tmpDir),
			Capabilities: protocol.ClientCapabilities{
				Workspace: protocol.WorkspaceClientCapabilities{
					WorkspaceFolders: true,
				},
				TextDocument: protocol.TextDocumentClientCapabilities{
					Completion: protocol.CompletionClientCapabilities{
						CompletionItem: protocol.ClientCompletionItemOptions{
							SnippetSupport: true,
						},
					},
				},
			},
		},
		WorkspaceFoldersInitializeParams: protocol.WorkspaceFoldersInitializeParams{
			WorkspaceFolders: []protocol.WorkspaceFolder{
				{
					URI:  "file://" + tmpDir,
					Name: "root",
				},
			},
		},
	}

	var initResult protocol.InitializeResult
	if err := client.Call("initialize", initParams, &initResult); err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}
	fmt.Printf("Server capabilities received: %+v\n\n", initResult.Capabilities)

	// Test 2: Initialized notification
	fmt.Println("Testing initialized notification...")
	err = wrapper.Initialized(protocol.InitializedParams{})
	if err != nil {
		log.Fatalf("Initialized notification failed: %v", err)
	}
	fmt.Println("Initialized notification sent successfully\n")

	// Test 3: Shutdown
	fmt.Println("Testing shutdown...")
	err = wrapper.Shutdown()
	if err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}
	fmt.Println("Shutdown successful\n")

	// Test 4: Exit
	fmt.Println("Testing exit...")
	err = wrapper.Exit()
	if err != nil {
		log.Fatalf("Exit failed: %v", err)
	}
	fmt.Println("Exit notification sent successfully")
}
