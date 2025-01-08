package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

func ReadLocation(loc protocol.Location) (string, error) {
	// Convert URI to filesystem path by removing the file:// prefix
	path := strings.TrimPrefix(string(loc.URI), "file://")

	fmt.Println("file:", loc.URI)

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Validate line numbers
	startLine := int(loc.Range.Start.Line)
	endLine := int(loc.Range.End.Line)
	if startLine < 0 || startLine >= len(lines) || endLine < 0 || endLine >= len(lines) {
		return "", fmt.Errorf("invalid Location: %v", loc)
	}

	// If it's a single line
	if startLine == endLine {
		line := lines[startLine]
		startChar := int(loc.Range.Start.Character)
		endChar := int(loc.Range.End.Character)

		if startChar < 0 || startChar > len(line) || endChar < 0 || endChar > len(line) {
			return "", fmt.Errorf("invalid Location: %v", loc)
		}

		return line[startChar:endChar], nil
	}

	// Handle multi-line selection
	var result strings.Builder

	// First line (from start character to end of line)
	firstLine := lines[startLine]
	startChar := int(loc.Range.Start.Character)
	if startChar < 0 || startChar > len(firstLine) {
		return "", fmt.Errorf("invalid Location: %v", loc)
	}
	result.WriteString(firstLine[startChar:])

	// Middle lines (complete lines)
	for i := startLine + 1; i < endLine; i++ {
		result.WriteString("\n")
		result.WriteString(lines[i])
	}

	// Last line (from start of line to end character)
	if startLine != endLine {
		lastLine := lines[endLine]
		endChar := int(loc.Range.End.Character)
		if endChar < 0 || endChar > len(lastLine) {
			return "", fmt.Errorf("invalid Location: %v", loc)
		}
		result.WriteString("\n")
		result.WriteString(lastLine[:endChar])
	}

	return result.String(), nil
}

func GetFullDefinition(ctx context.Context, client *lsp.Client, loc protocol.Location) (string, error) {
	params := protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: loc.URI},
			Position:     loc.Range.Start,
		},
	}

	var defLocation protocol.Location
	defResult, err := client.Definition(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to get definition: %w", err)
	}
	fmt.Println(defResult)

	// Handle the LocationOrLocations response
	switch v := defResult.Value.(type) {
	case protocol.Definition:
		switch v := v.Value.(type) {
		case protocol.Location:
			defLocation = v
		case []protocol.Location:
			if len(v) > 0 {
				defLocation = v[0]
			}
		}
	default:
		return "", fmt.Errorf("unexpected definition response type: %T", defResult)
	}

	// Now get the document symbols to find the full range of the definition
	symbolParams := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: defLocation.URI},
	}

	symbols, err := client.DocumentSymbol(ctx, symbolParams)
	if err != nil {
		return "", fmt.Errorf("failed to get document symbols: %w", err)
	}

	// Find the symbol that contains our definition location
	var fullRange protocol.Range
	switch v := symbols.Value.(type) {
	case []protocol.DocumentSymbol:
		fullRange = findContainingSymbolRange(v, defLocation.Range.Start)
	case []protocol.SymbolInformation:
		fullRange = findContainingSymbolInfoRange(v, defLocation.Range.Start)
	}

	// If we found a containing symbol, use its range, otherwise use the original
	if fullRange.Start.Line != 0 || fullRange.End.Line != 0 {
		defLocation.Range = fullRange
	}

	// Read the full definition
	return ReadLocation(defLocation)
}

// Helper function to find the symbol containing a position
func findContainingSymbolRange(symbols []protocol.DocumentSymbol, pos protocol.Position) protocol.Range {
	for _, sym := range symbols {
		if containsPosition(sym.Range, pos) {
			return sym.Range
		}
		// Check children recursively
		if len(sym.Children) > 0 {
			if r := findContainingSymbolRange(sym.Children, pos); r.End.Line != 0 {
				return r
			}
		}
	}
	return protocol.Range{}
}

func findContainingSymbolInfoRange(symbols []protocol.SymbolInformation, pos protocol.Position) protocol.Range {
	for _, sym := range symbols {
		if containsPosition(sym.Location.Range, pos) {
			return sym.Location.Range
		}
	}
	return protocol.Range{}
}

func containsPosition(r protocol.Range, pos protocol.Position) bool {
	return (r.Start.Line < pos.Line || (r.Start.Line == pos.Line && r.Start.Character <= pos.Character)) &&
		(r.End.Line > pos.Line || (r.End.Line == pos.Line && r.End.Character >= pos.Character))
}
