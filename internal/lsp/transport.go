package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Write writes an LSP message to the given writer
func WriteMessage(w io.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data))
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// ReadMessage reads a single LSP message from the given reader
func ReadMessage(r *bufio.Reader) (*Message, error) {
	// Read headers
	var contentLength int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %w", err)
		}
		line = strings.TrimSpace(line)

		if line == "" {
			break // End of headers
		}

		if strings.HasPrefix(line, "Content-Length: ") {
			_, err := fmt.Sscanf(line, "Content-Length: %d", &contentLength)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
	}

	// Read content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(r, content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Parse message
	var msg Message
	if err := json.Unmarshal(content, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// handleMessages reads and dispatches messages in a loop
func (c *Client) handleMessages() {
	for {
		msg, err := ReadMessage(c.stdout)
		if err != nil {
			// Handle error (possibly reconnect or notify error handlers)
			return
		}

		// Handle notification
		if msg.Method != "" && msg.ID == 0 {
			c.notificationMu.RLock()
			handler, ok := c.notificationHandlers[msg.Method]
			c.notificationMu.RUnlock()
			
			if ok {
				go handler(msg.Method, msg.Params)
			}
			continue
		}

		// Handle response
		if msg.ID != 0 {
			c.handlersMu.RLock()
			ch, ok := c.handlers[msg.ID]
			c.handlersMu.RUnlock()

			if ok {
				ch <- msg
			}
		}
	}
}

// Call makes a request and waits for the response
func (c *Client) Call(method string, params interface{}, result interface{}) error {
	id := c.nextID.Add(1)
	
	msg, err := NewRequest(id, method, params)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Create response channel
	ch := make(chan *Message, 1)
	c.handlersMu.Lock()
	c.handlers[id] = ch
	c.handlersMu.Unlock()

	defer func() {
		c.handlersMu.Lock()
		delete(c.handlers, id)
		c.handlersMu.Unlock()
	}()

	// Send request
	if err := WriteMessage(c.stdin, msg); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	resp := <-ch

	if resp.Error != nil {
		return fmt.Errorf("request failed: %s (code: %d)", resp.Error.Message, resp.Error.Code)
	}

	if result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

// Notify sends a notification (a request without an ID that doesn't expect a response)
func (c *Client) Notify(method string, params interface{}) error {
	msg, err := NewNotification(method, params)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	if err := WriteMessage(c.stdin, msg); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
