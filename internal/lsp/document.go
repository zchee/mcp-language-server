package lsp

import (
	"fmt"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

// DocumentManager handles document-related operations
type DocumentManager struct {
	client *Client
}

// NewDocumentManager creates a new document manager
func NewDocumentManager(client *Client) *DocumentManager {
	return &DocumentManager{client: client}
}

// OpenDocument notifies the server that a document has been opened
func (d *DocumentManager) OpenDocument(uri protocol.DocumentURI, languageID protocol.LanguageKind, content string) error {
	params := protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: languageID,
			Version:    1,
			Text:       content,
		},
	}

	return d.client.Notify("textDocument/didOpen", params)
}

// GetDocumentSymbols retrieves the symbol information for a document
func (d *DocumentManager) GetDocumentSymbols(uri protocol.DocumentURI) ([]protocol.DocumentSymbol, error) {
	params := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri,
		},
	}

	var symbols []protocol.DocumentSymbol
	if err := d.client.Call("textDocument/documentSymbol", params, &symbols); err != nil {
		// Try SymbolInformation format if DocumentSymbol fails
		var symbolInfo []protocol.SymbolInformation
		if err := d.client.Call("textDocument/documentSymbol", params, &symbolInfo); err != nil {
			return nil, fmt.Errorf("failed to get document symbols: %w", err)
		}

		// Convert SymbolInformation to DocumentSymbol
		return convertSymbolInformation(symbolInfo), nil
	}

	return symbols, nil
}

// Helper function to convert SymbolInformation to DocumentSymbol
func convertSymbolInformation(info []protocol.SymbolInformation) []protocol.DocumentSymbol {
	symbols := make([]protocol.DocumentSymbol, len(info))
	for i, si := range info {
		symbols[i] = protocol.DocumentSymbol{
			Name:     si.Name,
			Kind:     si.Kind,
			Range:    si.Location.Range,
			Detail:   si.ContainerName,
			Children: nil, // SymbolInformation doesn't have hierarchy information
		}
	}
	return symbols
}

// DidChange notifies the server that a document has changed
func (d *DocumentManager) DidChange(uri protocol.DocumentURI, version int, changes []protocol.TextDocumentContentChangeEvent) error {
	params := protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                int32(version),
		},
		ContentChanges: changes,
	}

	return d.client.Notify("textDocument/didChange", params)
}

// DidClose notifies the server that a document has been closed
func (d *DocumentManager) DidClose(uri protocol.DocumentURI) error {
	params := protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri,
		},
	}

	return d.client.Notify("textDocument/didClose", params)
}
