package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

// GetContentInfo reads the source code definition of a symbol (function, type, constant, etc.) at the specified position
func GetContentInfo(ctx context.Context, client *lsp.Client, filePath string, line, column int) (string, error) {
	// Open the file if not already open
	err := client.OpenFile(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file: %v", err)
	}

	// Convert 1-indexed line/column to 0-indexed for LSP protocol
	position := protocol.Position{
		Line:      uint32(line - 1),
		Character: uint32(column - 1),
	}

	location := protocol.Location{
		URI: protocol.DocumentUri("file://" + filePath),
		Range: protocol.Range{
			Start: position,
			End:   position,
		},
	}

	definition, loc, symbol, err := GetFullDefinition(ctx, client, location)
	if err != nil {
		return "", err
	}

	locationInfo := fmt.Sprintf(
		"Symbol: %s\n"+
			"File: %s\n"+
			"Range: L%d:C%d - L%d:C%d\n\n",
		symbol.GetName(),
		strings.TrimPrefix(string(loc.URI), "file://"),
		loc.Range.Start.Line+1,
		loc.Range.Start.Character+1,
		loc.Range.End.Line+1,
		loc.Range.End.Character+1,
	)

	definition = addLineNumbers(definition, int(loc.Range.Start.Line)+1)

	return locationInfo + definition, nil
}
