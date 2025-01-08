package tools

import (
	"fmt"
	"os"
	"strings"

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
