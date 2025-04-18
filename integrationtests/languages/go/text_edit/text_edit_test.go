package text_edit_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/isaacphi/mcp-language-server/integrationtests/languages/common"
	"github.com/isaacphi/mcp-language-server/integrationtests/languages/go/internal"
	"github.com/isaacphi/mcp-language-server/internal/tools"
)

// TestApplyTextEdits tests the ApplyTextEdits tool with various edit scenarios
func TestApplyTextEdits(t *testing.T) {
	suite := internal.GetTestSuite(t)

	ctx, cancel := context.WithTimeout(suite.Context, 10*time.Second)
	defer cancel()

	// Create a test file with known content we can edit
	testFileName := "edit_test.go"
	testFilePath := filepath.Join(suite.WorkspaceDir, testFileName)

	initialContent := `package main

import "fmt"

// TestFunction is a function we will edit
func TestFunction() {
	fmt.Println("Hello, world!")
	fmt.Println("This is a test function")
	fmt.Println("With multiple lines")
}

// AnotherFunction is another function that will be edited
func AnotherFunction() {
	fmt.Println("This is another function")
	fmt.Println("That we can modify")
}
`

	// Write the test file using the suite's method to ensure proper handling
	err := suite.WriteFile(testFileName, initialContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		edits        []tools.TextEdit
		snapshotName string
	}{
		{
			name: "Replace single line",
			edits: []tools.TextEdit{
				{
					Type:      tools.Replace,
					StartLine: 7,
					EndLine:   7,
					NewText:   `	fmt.Println("Modified line")`,
				},
			},
			snapshotName: "replace_single_line",
		},
		{
			name: "Replace multiple lines",
			edits: []tools.TextEdit{
				{
					Type:      tools.Replace,
					StartLine: 6,
					EndLine:   9,
					NewText: `func TestFunction() {
	fmt.Println("This is a completely modified function")
	fmt.Println("With fewer lines")
}`,
				},
			},
			snapshotName: "replace_multiple_lines",
		},
		{
			name: "Insert line",
			edits: []tools.TextEdit{
				{
					Type:      tools.Insert,
					StartLine: 8,
					EndLine:   8,
					NewText:   `	fmt.Println("This is an inserted line")`,
				},
			},
			snapshotName: "insert_line",
		},
		{
			name: "Delete line",
			edits: []tools.TextEdit{
				{
					Type:      tools.Delete,
					StartLine: 8,
					EndLine:   8,
					NewText:   "",
				},
			},
			snapshotName: "delete_line",
		},
		{
			name: "Multiple edits in same file",
			edits: []tools.TextEdit{
				{
					Type:      tools.Replace,
					StartLine: 7,
					EndLine:   7,
					NewText:   `	fmt.Println("First modification")`,
				},
				{
					Type:      tools.Replace,
					StartLine: 14,
					EndLine:   14,
					NewText:   `	fmt.Println("Second modification")`,
				},
			},
			snapshotName: "multiple_edits",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the file before each test
			err := suite.WriteFile(testFileName, initialContent)
			if err != nil {
				t.Fatalf("Failed to reset test file: %v", err)
			}

			// Call the ApplyTextEdits tool with the non-URL file path
			result, err := tools.ApplyTextEdits(ctx, suite.Client, testFilePath, tc.edits)
			if err != nil {
				t.Fatalf("Failed to apply text edits: %v", err)
			}

			// Verify the result message
			if !strings.Contains(result, "Successfully applied text edits") {
				t.Errorf("Result does not contain success message: %s", result)
			}

			// Use snapshot testing to verify the text edit operation output
			common.SnapshotTest(t, "go", "text_edit", tc.snapshotName, result)
		})
	}
}

// TestApplyTextEditsWithBorderCases tests edge cases for the ApplyTextEdits tool
func TestApplyTextEditsWithBorderCases(t *testing.T) {
	suite := internal.GetTestSuite(t)

	ctx, cancel := context.WithTimeout(suite.Context, 10*time.Second)
	defer cancel()

	// Create a test file with known content we can edit
	testFileName := "edge_case_test.go"
	testFilePath := filepath.Join(suite.WorkspaceDir, testFileName)

	initialContent := `package main

import "fmt"

// EmptyFunction is an empty function we will edit
func EmptyFunction() {
}

// SingleLineFunction is a single line function
func SingleLineFunction() { fmt.Println("Single line") }

// LastFunction is the last function in the file
func LastFunction() {
	fmt.Println("Last function")
}
`

	// Write the test file using the suite's method
	err := suite.WriteFile(testFileName, initialContent)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name         string
		edits        []tools.TextEdit
		snapshotName string
	}{
		{
			name: "Edit empty function",
			edits: []tools.TextEdit{
				{
					Type:      tools.Replace,
					StartLine: 6,
					EndLine:   7,
					NewText: `func EmptyFunction() {
	fmt.Println("No longer empty")
}`,
				},
			},
			snapshotName: "edit_empty_function",
		},
		{
			name: "Edit single line function",
			edits: []tools.TextEdit{
				{
					Type:      tools.Replace,
					StartLine: 10,
					EndLine:   10,
					NewText: `func SingleLineFunction() { 
	fmt.Println("Now a multi-line function") 
}`,
				},
			},
			snapshotName: "edit_single_line_function",
		},
		{
			name: "Append to end of file",
			edits: []tools.TextEdit{
				{
					Type:      tools.Insert,
					StartLine: 15,
					EndLine:   15,
					NewText: `

// NewFunction is a new function at the end of the file
func NewFunction() {
	fmt.Println("This is a new function")
}`,
				},
			},
			snapshotName: "append_to_file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the file before each test
			err := suite.WriteFile(testFileName, initialContent)
			if err != nil {
				t.Fatalf("Failed to reset test file: %v", err)
			}

			// Call the ApplyTextEdits tool
			result, err := tools.ApplyTextEdits(ctx, suite.Client, testFilePath, tc.edits)
			if err != nil {
				t.Fatalf("Failed to apply text edits: %v", err)
			}

			// Verify the result message
			if !strings.Contains(result, "Successfully applied text edits") {
				t.Errorf("Result does not contain success message: %s", result)
			}

			// Use snapshot testing to verify the text edit operation output
			common.SnapshotTest(t, "go", "text_edit", tc.snapshotName, result)
		})
	}
}
