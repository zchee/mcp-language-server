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

// JSON-RPC 2.0 message
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

	// Read response using the new readResponseMessage function
	reader := bufio.NewReader(stdout)
	response, err := readResponseMessage(reader)
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

	// 1. Read the file contents
	filepath := "/Users/phil/dev/mcp-language-server/cmd/lsp/main.go"
	content, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// 2. Send didOpen with the file content
	uri := protocol.DocumentURI("file://" + filepath)
	didOpenParams := protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       string(content),
		},
	}

	err = writeRequest(stdin, 0, "textDocument/didOpen", didOpenParams)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Send document symbol request
	symbolParams := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri,
		},
	}

	err = writeRequest(stdin, 2, "textDocument/documentSymbol", symbolParams)
	if err != nil {
		log.Fatal(err)
	}

	// Read response using the new readResponseMessage function
	response, err = readResponseMessage(reader)
	if err != nil {
		log.Fatal(err)
	}

	// Check if we got an error response
	if response.Error != nil {
		log.Fatalf("Server returned error: %v (code: %d)",
			response.Error.Message,
			response.Error.Code)
	}

	if response.Result == nil {
		log.Fatal("No result in response")
	}

	// fmt.Println(string(response.Result))

	// Parse symbols
	// Try DocumentSymbol format first
	var symbols []protocol.DocumentSymbol
	if err := json.Unmarshal(response.Result, &symbols); err == nil {
		// Additional validation: check if we got actual DocumentSymbol data
		// A valid DocumentSymbol should at least have a name and valid range
		isDocumentSymbol := len(symbols) > 0 && symbols[0].Name != "" &&
			(symbols[0].Range.End.Line > 0 || symbols[0].Range.End.Character > 0)

		if isDocumentSymbol {
			fmt.Println("Found DocumentSymbol format:")
			for _, symbol := range symbols {
				printDocumentSymbol(symbol, 0)
			}
			return
		}
	}

	// If not DocumentSymbol, try SymbolInformation
	var symbolInfo []protocol.SymbolInformation
	if err := json.Unmarshal(response.Result, &symbolInfo); err != nil {
		log.Fatalf("Failed to parse symbols response: %v", err)
	}

	fmt.Println("Found SymbolInformation format:")
	for _, info := range symbolInfo {
		fmt.Printf("- %s (%s) at line %d - %d\n",
			info.Name,
			info.Kind,
			info.Location.Range.Start.Line+1,
			info.Location.Range.End.Line+1,
		)
	}

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

// Helper function to print DocumentSymbol with proper indentation
func printDocumentSymbol(symbol protocol.DocumentSymbol, level int) {
	fmt.Printf("%v", symbol)
	indent := strings.Repeat("  ", level)
	fmt.Printf("%s- %s (%s) at line %d\n",
		indent,
		symbol.Name,
		symbol.Kind,
		symbol.Range.Start.Line)

	// Print children recursively
	for _, child := range symbol.Children {
		printDocumentSymbol(child, level+1)
	}
}

func readMessage(r *bufio.Reader) (*Message, error) {
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

// readResponseMessage reads messages until it finds a response (a message with a Result or Error)
func readResponseMessage(r *bufio.Reader) (*Message, error) {
	for {
		msg, err := readMessage(r)
		if err != nil {
			return nil, err
		}

		// If this is a notification (has Method but no ID), process it and continue
		if msg.Method != "" && msg.ID == 0 {
			// Handle notification (you can add logging or process it as needed)
			fmt.Printf("Received notification: %s\n", msg.Method)
			if msg.Method == "window/showMessage" {
				var params struct {
					Type    int    `json:"type"`
					Message string `json:"message"`
				}
				if err := json.Unmarshal(msg.Params, &params); err == nil {
					fmt.Printf("Server message: %s\n", params.Message)
				}
			}
			continue
		}

		// If this is a response (has ID and either Result or Error), return it
		if msg.ID != 0 {
			return msg, nil
		}
	}
}
