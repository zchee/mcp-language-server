package methods

import (
	"testing"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

func TestTextDocumentDocumentSymbol(t *testing.T) {
	ts := newTestServer(t)
	ts.initialize()

	uri := ts.createFile("main.go", sampleGoFile)

	// Open the document first
	err := ts.wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       sampleGoFile,
		},
	})
	if err != nil {
		t.Fatalf("TextDocumentDidOpen failed: %v", err)
	}

	// Now try the normal call
	symbols, err := ts.wrapper.TextDocumentDocumentSymbol(protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Errorf("TextDocumentDocumentSymbol failed: %v", err)
	}

	documentSymbols, ok := symbols.([]protocol.DocumentSymbol)
	if !ok {
		t.Errorf("Got unexpected symbol type, expected DocumentSymbol, got %T", symbols)
		return
	}

	// Define expected symbols with their exact positions
	expectedSymbols := []struct {
		name      string
		kind      protocol.SymbolKind
		startLine uint32
		endLine   uint32
	}{
		{
			name:      "main",
			kind:      protocol.Function,
			startLine: 4,
			endLine:   6,
		},
		{
			name:      "add",
			kind:      protocol.Function,
			startLine: 8,
			endLine:   10,
		},
	}

	if len(documentSymbols) != len(expectedSymbols) {
		t.Errorf("Got %d symbols, expected %d", len(documentSymbols), len(expectedSymbols))
	}

	// Create a map for easier lookup
	symbolMap := make(map[string]protocol.DocumentSymbol)
	for _, sym := range documentSymbols {
		symbolMap[sym.Name] = sym
	}

	// Verify each expected symbol
	for _, expected := range expectedSymbols {
		actual, ok := symbolMap[expected.name]
		if !ok {
			t.Errorf("Missing expected symbol: %s", expected.name)
			continue
		}

		if actual.Kind != expected.kind {
			t.Errorf("Symbol %s: expected kind %v, got %v", expected.name, expected.kind, actual.Kind)
		}

		if actual.Range.Start.Line != expected.startLine {
			t.Errorf("Symbol %s: expected start line %d, got %d", expected.name, expected.startLine, actual.Range.Start.Line)
		}

		if actual.Range.End.Line != expected.endLine {
			t.Errorf("Symbol %s: expected end line %d, got %d", expected.name, expected.endLine, actual.Range.End.Line)
		}
	}
}

