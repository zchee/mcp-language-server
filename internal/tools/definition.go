package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

func ReadDefinition(ctx context.Context, client *lsp.Client, symbolName string) (string, error) {
	symbolName, results, err := QuerySymbol(ctx, client, symbolName)
	if err != nil {
		return "", err
	}

	var definitions []string
	for _, symbol := range results {
		kind := ""
		container := ""

		// Skip symbols that we are not looking for. workspace/symbol may return
		// a large number of fuzzy matches. This handles BaseSymbolInformation
		doesSymbolMatch := func(vKind protocol.SymbolKind, vContainerName string) bool {
			thisName := symbol.GetName()

			kind = fmt.Sprintf("Kind: %s\n", protocol.TableKindMap[vKind])
			if vContainerName != "" {
				container = fmt.Sprintf("Container Name: %s\n", vContainerName)
			}

			if thisName == symbolName {
				return true
			}

			// Handle different matching strategies based on the search term
			if strings.Contains(symbolName, ".") {
				// For qualified names like "Type.Method", don't do fuzzy match

			} else if vKind == protocol.Method {
				// For methods, only match if the method name matches exactly Type.symbolName or Type::symbolName or symbolName
				if strings.HasSuffix(thisName, "::"+symbolName) || strings.HasSuffix(symbolName, "::"+thisName) {
					return true
				}

				if strings.HasSuffix(thisName, "."+symbolName) || strings.HasSuffix(symbolName, "."+thisName) {
					return true
				}
			}

			return false
		}

		switch v := symbol.(type) {
		case *protocol.SymbolInformation:
			if !doesSymbolMatch(v.Kind, v.ContainerName) {
				continue
			}

		case *protocol.WorkspaceSymbol:
			if !doesSymbolMatch(v.Kind, v.ContainerName) {
				continue
			}
		default:
			if symbol.GetName() != symbolName {
				continue
			}
		}

		toolsLogger.Debug("Found symbol: %s", symbol.GetName())
		loc := symbol.GetLocation()

		err := client.OpenFile(ctx, loc.URI.Path())
		if err != nil {
			toolsLogger.Error("Error opening file: %v", err)
			continue
		}

		banner := "---\n\n"
		definition, loc, _, err := GetFullDefinition(ctx, client, loc)
		locationInfo := fmt.Sprintf(
			"Symbol: %s\n"+
				"File: %s\n"+
				kind+
				container+
				"Range: L%d:C%d - L%d:C%d\n\n",
			symbol.GetName(),
			strings.TrimPrefix(string(loc.URI), "file://"),
			loc.Range.Start.Line+1,
			loc.Range.Start.Character+1,
			loc.Range.End.Line+1,
			loc.Range.End.Character+1,
		)

		if err != nil {
			toolsLogger.Error("Error getting definition: %v", err)
			continue
		}

		definition = addLineNumbers(definition, int(loc.Range.Start.Line)+1)

		definitions = append(definitions, banner+locationInfo+definition+"\n")
	}

	if len(definitions) == 0 {
		return fmt.Sprintf("%s not found", symbolName), nil
	}

	return strings.Join(definitions, ""), nil
}
