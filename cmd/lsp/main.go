package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

// Message represents a JSON-RPC 2.0 message
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func writeRequest(w io.Writer, id int, method string, params interface{}) error {
	// Convert params to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	msg := Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}

	// Marshal the entire message
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write the Content-Length header
	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(msgJSON))
	if err != nil {
		return err
	}

	// Write the JSON message
	_, err = w.Write(msgJSON)
	return err
}

func readResponse(r *bufio.Reader) (*Message, error) {
	// Read headers
	var contentLength int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)

		if line == "" {
			break // End of headers
		}

		if strings.HasPrefix(line, "Content-Length: ") {
			_, err := fmt.Sscanf(line, "Content-Length: %d", &contentLength)
			if err != nil {
				return nil, err
			}
		}
	}

	// Read the JSON content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(r, content)
	if err != nil {
		return nil, err
	}

	// Parse the message
	var msg Message
	err = json.Unmarshal(content, &msg)
	return &msg, err
}

func main() {
	cmd := exec.Command(os.Getenv("LSP_COMMAND"))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	// Send initialize request
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

	err = writeRequest(stdin, 1, "initialize", initParams)
	if err != nil {
		log.Fatal(err)
	}

	// Read response
	reader := bufio.NewReader(stdout)
	response, err := readResponse(reader)
	if err != nil {
		log.Fatal(err)
	}

	// Parse into protocol-defined result type
	var result protocol.InitializeResult
	err = json.Unmarshal(response.Result, &result)
	if err != nil {
		log.Fatal(err)
	}

	// Pretty print the capabilities
	prettyJSON, err := json.MarshalIndent(result.Capabilities, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server capabilities:\n%s\n", string(prettyJSON))

	// Send initialized notification
	err = writeRequest(stdin, 0, "initialized", struct{}{})
	if err != nil {
		log.Fatal(err)
	}

	// Wait for the LSP server to exit
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
