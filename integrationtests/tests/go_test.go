package tests

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/isaacphi/mcp-language-server/internal/tools"
)

// TestGoLanguageServer runs a series of tests against the Go language server
// using a shared LSP instance to avoid startup overhead between tests
func TestGoLanguageServer(t *testing.T) {
	// Configure Go LSP
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	config := LSPTestConfig{
		Name:             "go",
		Command:          "gopls",
		Args:             []string{},
		WorkspaceDir:     filepath.Join(repoRoot, "integrationtests/workspaces/go"),
		InitializeTimeMs: 2000, // 2 seconds
	}

	// Create a shared test suite for all subtests
	suite := NewTestSuite(t, config)
	t.Cleanup(func() {
		suite.Cleanup()
	})

	// Initialize just once for all tests
	err = suite.Setup()
	if err != nil {
		t.Fatalf("Failed to set up test suite: %v", err)
	}

	// Run tests that share the same LSP instance
	t.Run("ReadDefinition", func(t *testing.T) {
		testGoReadDefinition(t, suite)
	})

	t.Run("Diagnostics", func(t *testing.T) {
		testGoDiagnostics(t, suite)
	})
}

// Test the ReadDefinition tool with the Go language server
func testGoReadDefinition(t *testing.T, suite *TestSuite) {
	ctx, cancel := context.WithTimeout(suite.Context, 5*time.Second)
	defer cancel()

	// Call the ReadDefinition tool
	result, err := tools.ReadDefinition(ctx, suite.Client, "FooBar", true)
	if err != nil {
		t.Fatalf("ReadDefinition failed: %v", err)
	}

	// Verify the result
	if result == "FooBar not found" {
		t.Errorf("FooBar function not found")
	}

	// Check that the result contains relevant function information
	if !strings.Contains(result, "func FooBar()") {
		t.Errorf("Definition does not contain expected function signature")
	}

	// Use snapshot testing to verify exact output
	SnapshotTest(t, "go_definition_foobar", result)
}

// Test diagnostics functionality with the Go language server
func testGoDiagnostics(t *testing.T, suite *TestSuite) {
	// First test with a clean file
	t.Run("CleanFile", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(suite.Context, 5*time.Second)
		defer cancel()

		filePath := filepath.Join(suite.WorkspaceDir, "main.go")
		result, err := tools.GetDiagnosticsForFile(ctx, suite.Client, filePath, true, true)
		if err != nil {
			t.Fatalf("GetDiagnosticsForFile failed: %v", err)
		}

		// Verify we have no diagnostics
		if !strings.Contains(result, "No diagnostics found") {
			t.Errorf("Expected no diagnostics but got: %s", result)
		}

		SnapshotTest(t, "go_diagnostics_clean", result)
	})

	// Test with a file containing an error
	t.Run("FileWithError", func(t *testing.T) {
		// Write a file with an error
		badCode := `package main

import "fmt"

// FooBar is a simple function for testing
func FooBar() string {
	return "Hello, World!"
	fmt.Println("Unreachable code") // This is unreachable code
}

func main() {
	fmt.Println(FooBar())
}
`
		err := suite.WriteFile("main.go", badCode)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Wait for diagnostics to be generated
		time.Sleep(2 * time.Second)

		ctx, cancel := context.WithTimeout(suite.Context, 5*time.Second)
		defer cancel()

		filePath := filepath.Join(suite.WorkspaceDir, "main.go")
		result, err := tools.GetDiagnosticsForFile(ctx, suite.Client, filePath, true, true)
		if err != nil {
			t.Fatalf("GetDiagnosticsForFile failed: %v", err)
		}

		// Verify we have diagnostics about unreachable code
		if strings.Contains(result, "No diagnostics found") {
			t.Errorf("Expected diagnostics but got none")
		}

		if !strings.Contains(result, "unreachable") {
			t.Errorf("Expected unreachable code error but got: %s", result)
		}

		SnapshotTest(t, "go_diagnostics_unreachable", result)

		// Restore the original file for other tests
		cleanCode := `package main

import "fmt"

// FooBar is a simple function for testing
func FooBar() string {
	return "Hello, World!"
}

func main() {
	fmt.Println(FooBar())
}
`
		err = suite.WriteFile("main.go", cleanCode)
		if err != nil {
			t.Fatalf("Failed to restore clean file: %v", err)
		}

		// Wait for diagnostics to be cleared
		time.Sleep(2 * time.Second)
	})
}
