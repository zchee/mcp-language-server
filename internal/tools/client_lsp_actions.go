package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

func ExtractTextFromLocation(loc protocol.Location) (string, error) {
	path := strings.TrimPrefix(string(loc.URI), "file://")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

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

func GetFullDefinition(ctx context.Context, client *lsp.Client, symbol protocol.WorkspaceSymbolResult) (string, error) {
	symParams := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: symbol.GetLocation().URI,
		},
	}

	symResult, err := client.DocumentSymbol(ctx, symParams)
	if err != nil {
		return "", fmt.Errorf("failed to get document symbols: %w", err)
	}

	symbols, err := symResult.Results()
	if err != nil {
		return "", fmt.Errorf("failed to process document symbols: %w", err)
	}

	var symbolRange protocol.Range
	found := false

	// Need to check all document symbols because WorkspaceSymbolResult's range
	// only contains the definition but the document symbols's range has
	// the full definition
	var searchSymbols func(symbols []protocol.DocumentSymbolResult) bool
	searchSymbols = func(symbols []protocol.DocumentSymbolResult) bool {
		for _, sym := range symbols {
			if containsPosition(sym.GetRange(), symbol.GetLocation().Range.Start) {
				symbolRange = sym.GetRange()
				found = true
				return true
			}
			// Handle nested symbols if it's a DocumentSymbol
			if ds, ok := sym.(*protocol.DocumentSymbol); ok && len(ds.Children) > 0 {
				childSymbols := make([]protocol.DocumentSymbolResult, len(ds.Children))
				for i := range ds.Children {
					childSymbols[i] = &ds.Children[i]
				}
				if searchSymbols(childSymbols) {
					return true
				}
			}
		}
		return false
	}

	searchSymbols(symbols)

	if !found {
		// Fall back to the original location if we can't find a better range
		symbolRange = symbol.GetLocation().Range
	}

	return ExtractTextFromLocation(protocol.Location{
		URI:   symbol.GetLocation().URI,
		Range: symbolRange,
	})
}
