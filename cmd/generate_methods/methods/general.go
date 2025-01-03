package methods

// GeneralMethods returns the method definitions for general LSP methods
func GeneralMethods() []MethodDef {
	return []MethodDef{
		{
			Name:        "initialize",
			RequestType: "InitializeParams",
			ResponseTypes: []ReturnType{
				{Type: "InitializeResult"},
			},
			Category: "General",
		},
		{
			Name:           "initialized",
			RequestType:    "InitializedParams",
			IsNotification: true,
			Category:       "General",
		},
		{
			Name:        "shutdown",
			RequestType:  "struct{}", // Empty request
			// No ResponseTypes means void return
			Category:    "General",
		},
		{
			Name:           "exit",
			RequestType:    "struct{}",
			IsNotification: true,
			Category:       "General",
		},
	}
}
