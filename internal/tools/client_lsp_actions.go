package tools

import (
	"context"
	"fmt"
	"io/ioutil"
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

// GetFullDefinition gets the complete definition of a symbol using LSP
func GetFullDefinition(ctx context.Context, client lsp.Client, symbol protocol.WorkspaceSymbolResult) (string, error) {
	// Convert symbol location to TextDocumentPositionParams
	params := protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: symbol.GetLocation().URI,
			},
			Position: symbol.GetLocation().Range.Start,
		},
	}

	// Get definition location
	var defResult protocol.Or_Result_textDocument_definition
	err := client.Call(ctx, "textDocument/definition", params, &defResult)
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
			return "", fmt.Errorf("unexpected Or_Definition value type: %T", v.Value)
		}
	default:
		return "", fmt.Errorf("unexpected definition result type: %T", v)
	}

	if len(locations) == 0 {
		return "", fmt.Errorf("no definition locations found")
	}

	if len(locations) == 0 {
		return "", fmt.Errorf("no definition found for symbol %s", symbol.GetName())
	}

	// Get the first location (most relevant)
	loc := locations[0]

	// Read the file content
	content, err := ReadFileFromURI(string(loc.URI))
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Extract the definition range
	fullRange, err := ExpandDefinitionRange(content, loc.Range)
	if err != nil {
		return "", fmt.Errorf("failed to expand definition range: %w", err)
	}

	// Extract the text from the expanded range
	definition := ExtractRangeText(content, fullRange)
	return definition, nil
}

// ExpandDefinitionRange expands the initial range to include the full definition
func ExpandDefinitionRange(content string, initial protocol.Range) (protocol.Range, error) {
	lines := strings.Split(content, "\n")

	// Start with the initial range
	expanded := initial

	// Scan forward to find the end of the definition
	line := initial.Start.Line
	indent := getIndentation(lines[line])
	for int(line) < len(lines) {
		// Break if we hit an empty line or a line with less indentation
		if line > initial.Start.Line {
			currIndent := getIndentation(lines[line])
			if strings.TrimSpace(lines[line]) == "" || currIndent < indent {
				break
			}
		}
		expanded.End.Line = line + 1
		expanded.End.Character = uint32(len(lines[line]))
		line++
	}

	return expanded, nil
}

// getIndentation returns the number of leading spaces in a string
func getIndentation(s string) int {
	return len(s) - len(strings.TrimLeft(s, " \t"))
}

// ReadFileFromURI reads a file from a LSP URI (file://)
func ReadFileFromURI(uri string) (string, error) {
	// Remove the file:// prefix
	path := strings.TrimPrefix(uri, "file://")

	// Read the file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return string(content), nil
}

func ExtractRangeText(content string, rng protocol.Range) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return ""
	}

	// Validate range bounds
	startLine := int(rng.Start.Line)
	endLine := int(rng.End.Line)
	if startLine >= len(lines) || startLine < 0 {
		return ""
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// Handle single-line case
	if startLine == endLine {
		line := lines[startLine]
		startChar := int(rng.Start.Character)
		endChar := int(rng.End.Character)
		if startChar >= len(line) {
			return ""
		}
		if endChar > len(line) {
			endChar = len(line)
		}
		return line[startChar:endChar]
	}

	// Handle multi-line case
	var result []string

	// First line
	firstLine := lines[startLine]
	startChar := int(rng.Start.Character)
	if startChar < len(firstLine) {
		result = append(result, firstLine[startChar:])
	}

	// Middle lines
	for i := startLine + 1; i < endLine; i++ {
		result = append(result, lines[i])
	}

	// Last line
	if endLine < len(lines) {
		lastLine := lines[endLine]
		endChar := int(rng.End.Character)
		if endChar > len(lastLine) {
			endChar = len(lastLine)
		}
		result = append(result, lastLine[:endChar])
	}

	return strings.Join(result, "\n")
}
