package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

// Client represents an LSP client instance
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	// Request ID counter
	nextID atomic.Int32

	// Response handlers
	handlers   map[int32]chan *Message
	handlersMu sync.RWMutex

	// Notification handlers
	notificationHandlers map[string]NotificationHandler
	notificationMu       sync.RWMutex
}

// NotificationHandler is called when a notification is received
type NotificationHandler func(method string, params json.RawMessage)

// NewClient creates a new LSP client
func NewClient(command string) (*Client, error) {
	cmd := exec.Command(command)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	client := &Client{
		cmd:                  cmd,
		stdin:                stdin,
		stdout:               bufio.NewReader(stdout),
		handlers:             make(map[int32]chan *Message),
		notificationHandlers: make(map[string]NotificationHandler),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	// Start message handling loop
	go client.handleMessages()

	return client, nil
}

// RegisterNotificationHandler registers a handler for a specific notification method
func (c *Client) RegisterNotificationHandler(method string, handler NotificationHandler) {
	c.notificationMu.Lock()
	defer c.notificationMu.Unlock()
	c.notificationHandlers[method] = handler
}

// Initialize initializes the LSP connection
func (c *Client) Initialize() (*protocol.InitializeResult, error) {
	initParams := protocol.InitializeParams{
		XInitializeParams: protocol.XInitializeParams{
			ProcessID: int32(os.Getpid()),
			RootURI:   "file:///",
			Capabilities: protocol.ClientCapabilities{
				TextDocument: protocol.TextDocumentClientCapabilities{
					Completion: protocol.CompletionClientCapabilities{
						CompletionItem: protocol.ClientCompletionItemOptions{
							SnippetSupport: true,
						},
					},
				},
			},
		},
	}

	var result protocol.InitializeResult
	if err := c.Call("initialize", initParams, &result); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	// Send initialized notification
	if err := c.Notify("initialized", struct{}{}); err != nil {
		return nil, fmt.Errorf("initialized notification failed: %w", err)
	}

	return &result, nil
}

// Close closes the LSP client and terminates the server
func (c *Client) Close() error {
	if err := c.stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	if err := c.cmd.Wait(); err != nil {
		return fmt.Errorf("server process error: %w", err)
	}

	return nil
}
