package methods

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ReturnType represents a possible return type for an LSP method
type ReturnType struct {
	Type          string // e.g. "DocumentSymbol"
	IsSlice       bool   // whether this is []Type
	NeedsConvert  bool   // whether this type needs conversion to primary type
	FieldMappings []FieldMapping
}

// Used when conversion is needed
type FieldMapping struct {
	SourceField string // field path in source type e.g. "Location.Range"
	DestField   string // field name in dest type e.g. "Range"
	SourceType  string // type of source field e.g. "protocol.Range"
	DestType    string // type of dest field e.g. "protocol.Range"
}

// MethodDef defines an LSP method
type MethodDef struct {
	Name           string       // e.g. "textDocument/didOpen"
	RequestType    string       // e.g. "DidOpenTextDocumentParams"
	ResponseTypes  []ReturnType // Multiple possible return types
	IsNotification bool
	Category       string // e.g. "TextDocument", "Workspace", etc.
}

// Returns type as it should appear in Go code
func (rt ReturnType) GoType() string {
	if rt.IsSlice {
		return "[]protocol." + rt.Type
	}
	return "protocol." + rt.Type
}

// Transforms LSP method name to Go method name
func (m MethodDef) GoName() string {
	parts := strings.Split(m.Name, "/")
	var result strings.Builder
	for _, part := range parts {
		result.WriteString(cases.Title(language.English, cases.NoLower).String(part))
	}
	return result.String()
}
