package methods

// TextDocumentMethods returns method definitions for textDocument/* LSP methods
func TextDocumentMethods() []MethodDef {
	return []MethodDef{
		// Document Sync
		{
			Name:           "textDocument/didOpen",
			RequestType:    "DidOpenTextDocumentParams",
			IsNotification: true,
			Category:       "TextDocument",
		},
		{
			Name:           "textDocument/didChange",
			RequestType:    "DidChangeTextDocumentParams",
			IsNotification: true,
			Category:       "TextDocument",
		},
		{
			Name:           "textDocument/didClose",
			RequestType:    "DidCloseTextDocumentParams",
			IsNotification: true,
			Category:       "TextDocument",
		},
		{
			Name:           "textDocument/willSave",
			RequestType:    "WillSaveTextDocumentParams",
			IsNotification: true,
			Category:       "TextDocument",
		},
		{
			Name:        "textDocument/willSaveWaitUntil",
			RequestType: "WillSaveTextDocumentParams",
			ResponseTypes: []ReturnType{
				{Type: "TextEdit", IsSlice: true},
			},
			Category: "TextDocument",
		},
		{
			Name:           "textDocument/didSave",
			RequestType:    "DidSaveTextDocumentParams",
			IsNotification: true,
			Category:       "TextDocument",
		},

		// Language Features
		{
			Name:        "textDocument/completion",
			RequestType: "CompletionParams",
			ResponseTypes: []ReturnType{
				{Type: "CompletionList"},
				{Type: "CompletionItem", IsSlice: true},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/hover",
			RequestType: "HoverParams",
			ResponseTypes: []ReturnType{
				{Type: "Hover"},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/signatureHelp",
			RequestType: "SignatureHelpParams",
			ResponseTypes: []ReturnType{
				{Type: "SignatureHelp"},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/definition",
			RequestType: "DefinitionParams",
			ResponseTypes: []ReturnType{
				{Type: "Location"},
				{Type: "Location", IsSlice: true},
				{Type: "DefinitionLink", IsSlice: true},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/declaration",
			RequestType: "DeclarationParams",
			ResponseTypes: []ReturnType{
				{Type: "Location"},
				{Type: "Location", IsSlice: true},
				{Type: "DeclarationLink", IsSlice: true},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/references",
			RequestType: "ReferenceParams",
			ResponseTypes: []ReturnType{
				{Type: "Location", IsSlice: true},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/documentHighlight",
			RequestType: "DocumentHighlightParams",
			ResponseTypes: []ReturnType{
				{Type: "DocumentHighlight", IsSlice: true},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/documentSymbol",
			RequestType: "DocumentSymbolParams",
			ResponseTypes: []ReturnType{
				{Type: "DocumentSymbol", IsSlice: true},
				{Type: "SymbolInformation", IsSlice: true, NeedsConvert: true},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/formatting",
			RequestType: "DocumentFormattingParams",
			ResponseTypes: []ReturnType{
				{Type: "TextEdit", IsSlice: true},
			},
			Category: "TextDocument",
		},
		{
			Name:        "textDocument/rangeFormatting",
			RequestType: "DocumentRangeFormattingParams",
			ResponseTypes: []ReturnType{
				{Type: "TextEdit", IsSlice: true},
			},
			Category: "TextDocument",
		},
	}
}
