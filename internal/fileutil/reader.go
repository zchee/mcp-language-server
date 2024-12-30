package fileutil

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

// safeUint32ToInt converts a uint32 to int, capping at math.MaxInt32
func safeUint32ToInt(u uint32) int {
	if u > uint32(math.MaxInt32) {
		return math.MaxInt32
	}
	return int(u)
}

// ReadLocationContent reads the content between the start and end positions defined in the Location
func ReadLocationContent(loc protocol.Location) (string, error) {
	// Convert URI to filepath
	filepath := strings.TrimPrefix(string(loc.URI), "file://")

	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Convert uint32 positions to int safely
	startLine := safeUint32ToInt(loc.Range.Start.Line)
	endLine := safeUint32ToInt(loc.Range.End.Line)
	startChar := safeUint32ToInt(loc.Range.Start.Character)
	endChar := safeUint32ToInt(loc.Range.End.Character)

	if startLine > endLine || (startLine == endLine && startChar > endChar) {
		return "", fmt.Errorf("invalid range: start position must precede end position")
	}

	// Read the whole file into lines for easier manipulation
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if startLine >= len(allLines) || endLine >= len(allLines) {
		return "", fmt.Errorf("line range exceeds file bounds")
	}

	// Extract the lines we want
	lines := allLines[startLine : endLine+1]
	if len(lines) == 0 {
		return "", fmt.Errorf("no content found in specified range")
	}

	// Handle single line case
	if startLine == endLine {
		line := lines[0]
		if startChar > len(line) {
			return "", fmt.Errorf("start character position %d exceeds line length %d", startChar, len(line))
		}
		if endChar > len(line) {
			endChar = len(line) // Clamp to line length
		}
		return line[startChar:endChar], nil
	}

	// Handle multi-line case
	if startChar > len(lines[0]) {
		return "", fmt.Errorf("start character position %d exceeds first line length %d", startChar, len(lines[0]))
	}
	if endChar > len(lines[len(lines)-1]) {
		endChar = len(lines[len(lines)-1]) // Clamp to line length
	}

	// Apply character offsets to first and last lines
	lines[0] = lines[0][startChar:]
	lines[len(lines)-1] = lines[len(lines)-1][:endChar]

	return strings.Join(lines, "\n"), nil
}
