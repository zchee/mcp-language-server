package lsp

import (
	"encoding/json"
)

type WorkspaceConfigurationHandler struct{}

func (h *WorkspaceConfigurationHandler) Handle(params json.RawMessage) (interface{}, error) {
	return []map[string]interface{}{{}}, nil
}

type RegisterCapabilityHandler struct{}

func (h *RegisterCapabilityHandler) Handle(params json.RawMessage) (interface{}, error) {
	return nil, nil
}
