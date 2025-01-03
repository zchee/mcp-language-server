package methods

import (
	"testing"

	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

func TestTextDocumentDidOpen(t *testing.T) {
	ts := newTestServer(t)
	ts.initialize()

	uri := ts.createFile("main.go", sampleGoFile)

	err := ts.wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       sampleGoFile,
		},
	})

	if err != nil {
		t.Errorf("TextDocumentDidOpen failed: %v", err)
	}
}

func TestTextDocumentDidChange(t *testing.T) {
	ts := newTestServer(t)
	ts.initialize()

	uri := ts.createFile("main.go", sampleGoFile)

	// First open the document
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

	// Test document changes
	changes := []protocol.TextDocumentContentChangeEvent{
		{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 5, Character: 1},
				End:   protocol.Position{Line: 5, Character: 6},
			},
			Text: "fmt.Printf",
		},
	}

	err = ts.wrapper.TextDocumentDidChange(protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                2,
		},
		ContentChanges: changes,
	})

	if err != nil {
		t.Errorf("TextDocumentDidChange failed: %v", err)
	}
}

func TestTextDocumentCompletion(t *testing.T) {
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

	// Test completion at the 'fmt.' position
	result, err := ts.wrapper.TextDocumentCompletion(protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 5, Character: 5},
		},
		Context: protocol.CompletionContext{
			TriggerKind: protocol.Invoked,
		},
	})

	if err != nil {
		t.Errorf("TextDocumentCompletion failed: %v", err)
	}

	// Check that we got some completions back
	switch v := result.(type) {
	case protocol.CompletionList:
		if len(v.Items) == 0 {
			t.Error("Expected non-empty completion list")
		}
	case []protocol.CompletionItem:
		if len(v) == 0 {
			t.Error("Expected non-empty completion items")
		}
	default:
		t.Errorf("Unexpected completion result type: %T", result)
	}
}

func TestTextDocumentHover(t *testing.T) {
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

	// Test hover over the 'add' function
	hover, err := ts.wrapper.TextDocumentHover(protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 8, Character: 5}, // Position on 'add' function
		},
	})
	if err != nil {
		t.Errorf("TextDocumentHover failed: %v", err)
	}

	if hover.Contents.Value == "" {
		t.Error("Expected non-empty hover content")
	}
}

func TestTextDocumentFormatting(t *testing.T) {
	ts := newTestServer(t)
	ts.initialize()

	// Create file with intentionally bad formatting
	badFormatting := `package main
import "fmt"
func main(){
fmt.Println("Hello, World!")
}
`
	uri := ts.createFile("main.go", badFormatting)

	// Open the document first
	err := ts.wrapper.TextDocumentDidOpen(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "go",
			Version:    1,
			Text:       badFormatting,
		},
	})
	if err != nil {
		t.Fatalf("TextDocumentDidOpen failed: %v", err)
	}

	// Test formatting
	edits, err := ts.wrapper.TextDocumentFormatting(protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Options: protocol.FormattingOptions{
			TabSize:      4,
			InsertSpaces: true,
		},
	})

	if err != nil {
		t.Errorf("TextDocumentFormatting failed: %v", err)
	}

	if len(edits) == 0 {
		t.Error("Expected non-empty formatting edits for badly formatted file")
	}
}

func TestTextDocumentDefinition(t *testing.T) {
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

	// Test getting definition of 'add' function when used
	definition, err := ts.wrapper.TextDocumentDefinition(protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 8, Character: 5}, // Position on 'add' function
		},
	})

	if err != nil {
		t.Errorf("TextDocumentDefinition failed: %v", err)
	}

	// Check we got some location back
	switch v := definition.(type) {
	case protocol.Location:
		if v.URI == "" {
			t.Error("Expected non-empty location URI")
		}
	case []protocol.Location:
		if len(v) == 0 {
			t.Error("Expected non-empty location array")
		}
	case []protocol.DefinitionLink:
		if len(v) == 0 {
			t.Error("Expected non-empty definition link array")
		}
	default:
		t.Errorf("Unexpected definition result type: %T", definition)
	}
}

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

func TestTextDocumentDidClose(t *testing.T) {
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

	// Test closing the document
	err = ts.wrapper.TextDocumentDidClose(protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Errorf("TextDocumentDidClose failed: %v", err)
	}
}
