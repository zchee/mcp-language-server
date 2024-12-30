package fileutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

// ReplaceLocationContent replaces the content between the start and end positions with new content
func ReplaceLocationContent(loc protocol.Location, newContent string) error {
	// Convert URI to filepath
	filepath := strings.TrimPrefix(string(loc.URI), "file://")

	// Read entire file
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert positions
	startLine := safeUint32ToInt(loc.Range.Start.Line)
	endLine := safeUint32ToInt(loc.Range.End.Line)
	startChar := safeUint32ToInt(loc.Range.Start.Character)
	endChar := safeUint32ToInt(loc.Range.End.Character)

	if startLine > endLine || (startLine == endLine && startChar > endChar) {
		return fmt.Errorf("invalid range: start position must precede end position")
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")
	if startLine >= len(lines) || endLine >= len(lines) {
		return fmt.Errorf("line range exceeds file bounds")
	}

	// Handle single line case
	if startLine == endLine {
		line := lines[startLine]
		if startChar > len(line) || endChar > len(line) {
			return fmt.Errorf("character range exceeds line length")
		}
		lines[startLine] = line[:startChar] + newContent + line[endChar:]
	} else {
		// Handle multi-line case
		startLine := lines[startLine]
		endLine := lines[endLine]

		if startChar > len(startLine) {
			return fmt.Errorf("start character exceeds line length")
		}
		if endChar > len(endLine) {
			endChar = len(endLine) // Clamp to line length
		}

		// Replace the content
		lines[loc.Range.Start.Line] = startLine[:startChar] + newContent
		// Remove the lines in between
		lines = append(lines[:loc.Range.Start.Line+1], lines[loc.Range.End.Line+1:]...)
	}

	// Write back to file
	err = os.WriteFile(filepath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
