package tools

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

type match struct {
	Symbol protocol.DocumentSymbolResult
	Range  protocol.Range
}

func identifyOverlappingSymbols(ctx context.Context, client *lsp.Client, startLocation protocol.Location) ([]match, error) {
	symParams := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: startLocation.URI,
		},
	}

	// Get all symbols in document
	symResult, err := client.DocumentSymbol(ctx, symParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get document symbols: %w", err)
	}

	symbols, err := symResult.Results()
	if err != nil {
		return nil, fmt.Errorf("failed to process document symbols: %w", err)
	}

	// Search for symbol at startLocation
	// - multiple symbols might match (for example, a C++ namespace) so find
	//   all of the matching symbols and use the smallest one (or the first one
	//   if there is a tie)
	var matchingSymbols []match

	var searchSymbols func(symbols []protocol.DocumentSymbolResult)
	searchSymbols = func(symbols []protocol.DocumentSymbolResult) {
		for _, sym := range symbols {
			if containsPosition(sym.GetRange(), startLocation.Range.Start) {
				matchingSymbols = append(matchingSymbols, match{sym, sym.GetRange()})
			}

			// Handle nested symbols if it's a DocumentSymbol
			if ds, ok := sym.(*protocol.DocumentSymbol); ok && len(ds.Children) > 0 {
				childSymbols := make([]protocol.DocumentSymbolResult, len(ds.Children))
				for i := range ds.Children {
					childSymbols[i] = &ds.Children[i]
				}
				searchSymbols(childSymbols)
			}
		}
	}

	searchSymbols(symbols)
	return matchingSymbols, nil
}

// Gets the full code block surrounding the start of the input location
func GetFullDefinition(ctx context.Context, client *lsp.Client, startLocation protocol.Location) (string, protocol.Location, protocol.DocumentSymbolResult, error) {

	matchingSymbols, err := identifyOverlappingSymbols(ctx, client, startLocation)
	if err != nil {
		return "", protocol.Location{}, nil, err
	}

	// Identify the smallest overlapping symbol
	slices.SortStableFunc(matchingSymbols, func(a, b match) int {
		return int(a.Range.End.Line-a.Range.Start.Line) - int(b.Range.End.Line-b.Range.Start.Line)
	})

	if len(matchingSymbols) > 0 {
		symbol := matchingSymbols[0].Symbol
		symbolRange := matchingSymbols[0].Range

		// Convert URI to filesystem path
		filePath, err := url.PathUnescape(strings.TrimPrefix(string(startLocation.URI), "file://"))
		if err != nil {
			return "", protocol.Location{}, nil, fmt.Errorf("failed to unescape URI: %w", err)
		}

		// Read the file to get the full lines of the definition
		// because we may have a start and end column
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", protocol.Location{}, nil, fmt.Errorf("failed to read file: %w", err)
		}

		lines := strings.Split(string(content), "\n")

		// Extend start to beginning of line
		symbolRange.Start.Character = 0

		// Get the line at the end of the range
		if int(symbolRange.End.Line) >= len(lines) {
			return "", protocol.Location{}, nil, fmt.Errorf("line number out of range")
		}

		line := lines[symbolRange.End.Line]
		trimmedLine := strings.TrimSpace(line)

		// In some cases (python), constant definitions do not include the full body and instead
		// end with an opening bracket. In this case, parse the file until the closing bracket
		if len(trimmedLine) > 0 {
			lastChar := trimmedLine[len(trimmedLine)-1]
			if lastChar == '(' || lastChar == '[' || lastChar == '{' || lastChar == '<' {
				// Find matching closing bracket
				bracketStack := []rune{rune(lastChar)}
				lineNum := symbolRange.End.Line + 1

				for lineNum < uint32(len(lines)) {
					line := lines[lineNum]
					for pos, char := range line {
						if char == '(' || char == '[' || char == '{' || char == '<' {
							bracketStack = append(bracketStack, char)
						} else if char == ')' || char == ']' || char == '}' || char == '>' {
							if len(bracketStack) > 0 {
								lastOpen := bracketStack[len(bracketStack)-1]
								if (lastOpen == '(' && char == ')') ||
									(lastOpen == '[' && char == ']') ||
									(lastOpen == '{' && char == '}') ||
									(lastOpen == '<' && char == '>') {
									bracketStack = bracketStack[:len(bracketStack)-1]
									if len(bracketStack) == 0 {
										// Found matching bracket - update range
										symbolRange.End.Line = lineNum
										symbolRange.End.Character = uint32(pos + 1)
										goto foundClosing
									}
								}
							}
						}
					}
					lineNum++
				}
			foundClosing:
			}
		}

		// Return the text within the range
		if int(symbolRange.End.Line) >= len(lines) {
			return "", protocol.Location{}, nil, fmt.Errorf("end line out of range")
		}

		selectedLines := lines[symbolRange.Start.Line : symbolRange.End.Line+1]
		return strings.Join(selectedLines, "\n"), protocol.Location{URI: startLocation.URI, Range: symbolRange}, symbol, nil
	}

	return "", protocol.Location{}, nil, fmt.Errorf("symbol not found")
}

// GetLineRangesToDisplay determines which lines should be displayed for a set of locations
func GetLineRangesToDisplay(ctx context.Context, client *lsp.Client, locations []protocol.Location, totalLines int, contextLines int) (map[int]bool, error) {
	// Set to track which lines need to be displayed
	linesToShow := make(map[int]bool)

	// For each location, get its container and add relevant lines
	for _, loc := range locations {
		// Use GetFullDefinition to find container
		matchingSymbols, _ := identifyOverlappingSymbols(ctx, client, loc)
		if len(matchingSymbols) == 0 {
			// If container not found, just use the location's line
			refLine := int(loc.Range.Start.Line)
			linesToShow[refLine] = true

			// Add context lines
			for i := refLine - contextLines; i <= refLine+contextLines; i++ {
				if i >= 0 && i < totalLines {
					linesToShow[i] = true
				}
			}
			continue
		}

		containerRange := matchingSymbols[0].Range

		// Add container start and end lines
		containerStart := int(containerRange.Start.Line)
		containerEnd := int(containerRange.End.Line)
		linesToShow[containerStart] = true
		// linesToShow[containerEnd] = true

		// Add the reference line
		refLine := int(loc.Range.Start.Line)
		linesToShow[refLine] = true

		// Add context lines around the reference
		for i := refLine - contextLines; i <= refLine+contextLines; i++ {
			if i >= 0 && i < totalLines && i >= containerStart && i <= containerEnd {
				linesToShow[i] = true
			}
		}
	}

	return linesToShow, nil
}
