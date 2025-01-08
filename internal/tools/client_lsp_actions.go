package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

// ExtractTextFromLocation extracts text from a file given an LSP Location
func ExtractTextFromLocation(loc protocol.Location) (string, error) {
	// Convert URI to filesystem path
	path := strings.TrimPrefix(string(loc.URI), "file://")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Validate line numbers
	startLine := int(loc.Range.Start.Line)
	endLine := int(loc.Range.End.Line)
	if startLine < 0 || startLine >= len(lines) || endLine < 0 || endLine >= len(lines) {
		return "", fmt.Errorf("invalid Location range: %v", loc.Range)
	}

	// Handle single-line case
	if startLine == endLine {
		line := lines[startLine]
		startChar := int(loc.Range.Start.Character)
		endChar := int(loc.Range.End.Character)

		if startChar < 0 || startChar > len(line) || endChar < 0 || endChar > len(line) {
			return "", fmt.Errorf("invalid character range: %v", loc.Range)
		}

		return line[startChar:endChar], nil
	}

	// Handle multi-line case
	var result strings.Builder

	// First line
	firstLine := lines[startLine]
	startChar := int(loc.Range.Start.Character)
	if startChar < 0 || startChar > len(firstLine) {
		return "", fmt.Errorf("invalid start character: %v", loc.Range.Start)
	}
	result.WriteString(firstLine[startChar:])

	// Middle lines
	for i := startLine + 1; i < endLine; i++ {
		result.WriteString("\n")
		result.WriteString(lines[i])
	}

	// Last line
	lastLine := lines[endLine]
	endChar := int(loc.Range.End.Character)
	if endChar < 0 || endChar > len(lastLine) {
		return "", fmt.Errorf("invalid end character: %v", loc.Range.End)
	}
	result.WriteString("\n")
	result.WriteString(lastLine[:endChar])

	return result.String(), nil
}

// GetFullDefinition gets the complete definition of a symbol using LSP
func GetFullDefinition(ctx context.Context, client lsp.Client, symbol protocol.WorkspaceSymbolResult) (string, error) {
	// First, get the symbol's definition location
	defParams := protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: symbol.GetLocation().URI,
			},
			Position: symbol.GetLocation().Range.Start,
		},
	}

	var defResult protocol.Or_Result_textDocument_definition
	err := client.Call(ctx, "textDocument/definition", defParams, &defResult)
	if err != nil {
		return "", fmt.Errorf("failed to get definition: %w", err)
	}

	// Handle response
	var locations []protocol.Location
	switch v := defResult.Value.(type) {
	case protocol.Or_Definition:
		if locs, ok := v.Value.([]protocol.Location); ok {
			locations = locs
		} else if loc, ok := v.Value.(protocol.Location); ok {
			locations = []protocol.Location{loc}
		} else {
			return "", fmt.Errorf("unexpected definition value type: %T", v.Value)
		}
	default:
		return "", fmt.Errorf("unexpected definition result type: %T", v)
	}

	if len(locations) == 0 {
		return "", fmt.Errorf("no definition found for symbol %s", symbol.GetName())
	}

	// Now get the full range of the definition using documentSymbol request
	symParams := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: locations[0].URI,
		},
	}

	var symResult protocol.Or_Result_textDocument_documentSymbol
	err = client.Call(ctx, "textDocument/documentSymbol", symParams, &symResult)
	if err != nil {
		return "", fmt.Errorf("failed to get document symbols: %w", err)
	}

	// Find the symbol at the definition location
	var symbolRange protocol.Range
	found := false

	var searchSymbols func(symbols []protocol.DocumentSymbol) bool
	searchSymbols = func(symbols []protocol.DocumentSymbol) bool {
		for _, sym := range symbols {
			if containsPosition(sym.Range, locations[0].Range.Start) {
				symbolRange = sym.Range
				found = true
				return true
			}
			if len(sym.Children) > 0 && searchSymbols(sym.Children) {
				return true
			}
		}
		return false
	}

	switch v := symResult.Value.(type) {
	case []protocol.DocumentSymbol:
		searchSymbols(v)
	case []protocol.SymbolInformation:
		for _, sym := range v {
			if sym.Location.URI == locations[0].URI &&
				containsPosition(sym.Location.Range, locations[0].Range.Start) {
				symbolRange = sym.Location.Range
				found = true
				break
			}
		}
	}

	if !found {
		// Fall back to the original location if we can't find a better range
		symbolRange = locations[0].Range
	}

	// Extract the text using the full symbol range
	return ExtractTextFromLocation(protocol.Location{
		URI:   locations[0].URI,
		Range: symbolRange,
	})
}

// containsPosition checks if a range contains a position
func containsPosition(r protocol.Range, p protocol.Position) bool {
	if r.Start.Line > p.Line || r.End.Line < p.Line {
		return false
	}
	if r.Start.Line == p.Line && r.Start.Character > p.Character {
		return false
	}
	if r.End.Line == p.Line && r.End.Character < p.Character {
		return false
	}
	return true
}
