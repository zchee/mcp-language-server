package main

import (
	"github.com/isaacphi/mcp-language-server/internal/tools"
	"github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

type GetDefinitionArgs struct {
	SymbolName string `json:"symbolName" jsonschema:"required,description=The exact name of the symbol (function, class or something else) to fetch."`
}

func main() {
	done := make(chan struct{})

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	err := server.RegisterTool(
		"Get definition",
		"Read the source code for a given symbol from the codebase",
		func(args GetDefinitionArgs) (*mcp_golang.ToolResponse, error) {
			content, err := tools.GetDefinition(args.SymbolName)
			if err != nil {
				return nil, err
			}
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(content)), nil
		})
	if err != nil {
		panic(err)
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	<-done
}
