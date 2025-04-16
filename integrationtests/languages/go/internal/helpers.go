// Package internal contains shared helpers for Go tests
package internal

import (
	"path/filepath"
	"testing"

	"github.com/isaacphi/mcp-language-server/integrationtests/languages/common"
)

// GetTestSuite returns a test suite for Go language server tests
func GetTestSuite(t *testing.T) *common.TestSuite {
	// Configure Go LSP
	repoRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	config := common.LSPTestConfig{
		Name:             "go",
		Command:          "gopls",
		Args:             []string{},
		WorkspaceDir:     filepath.Join(repoRoot, "integrationtests/workspaces/go"),
		InitializeTimeMs: 2000, // 2 seconds
	}

	// Create a test suite
	suite := common.NewTestSuite(t, config)

	// Set up the suite
	err = suite.Setup()
	if err != nil {
		t.Fatalf("Failed to set up test suite: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		suite.Cleanup()
	})

	return suite
}

// GetErrorTestSuite returns a test suite for Go files with errors
func GetErrorTestSuite(t *testing.T) *common.TestSuite {
	// Configure Go LSP with error workspace
	repoRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	config := common.LSPTestConfig{
		Name:             "go_with_errors",
		Command:          "gopls",
		Args:             []string{},
		WorkspaceDir:     filepath.Join(repoRoot, "integrationtests/workspaces/go/with_errors"),
		InitializeTimeMs: 2000, // 2 seconds
	}

	// Create a test suite
	suite := common.NewTestSuite(t, config)

	// Set up the suite
	err = suite.Setup()
	if err != nil {
		t.Fatalf("Failed to set up test suite: %v", err)
	}

	// Register cleanup
	t.Cleanup(func() {
		suite.Cleanup()
	})

	return suite
}
