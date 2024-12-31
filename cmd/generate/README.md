# LSP Method Generator

This tool generates type-safe wrappers for LSP protocol methods based on the protocol definitions from `github.com/kralicky/tools-lite/gopls/pkg/protocol`.

## Usage

```bash
go run ./cmd/generate
```

This will generate:
1. `internal/lsp/methods/generated.go` - Contains type-safe method wrappers
2. `internal/lsp/methods/registry.go` - Contains method registry and constants

## Generated Code Structure

The generator creates:

1. Type-safe wrappers for each LSP method:
   ```go
   func (c *Client) TextDocumentDidOpen(params protocol.DidOpenTextDocumentParams) error
   func (c *Client) TextDocumentCompletion(params protocol.CompletionParams) (protocol.CompletionList, error)
   ```

2. Method registry for runtime capability checking:
   ```go
   registry := methods.NewRegistry()
   registry.RegisterMethod(methods.TextDocumentDidOpenMethod)
   if registry.HasMethod(methods.TextDocumentCompletionMethod) {
       // Use completion
   }
   ```

## Adding New Methods

Add new method definitions to the `methodDefs` slice in `main.go`:

```go
{
    Name:         "textDocument/newMethod",
    RequestType:  "NewMethodParams",
    ResponseType: "NewMethodResponse",  // Leave empty for notifications
    IsNotification: false,
    Category:     "TextDocument",
},
```

## Customizing Templates

The code generation templates are in `template.go`. You can modify them to:
- Add documentation
- Change method signatures
- Add validation
- Add logging or metrics
