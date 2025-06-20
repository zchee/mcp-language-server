package content_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/isaacphi/mcp-language-server/integrationtests/tests/common"
	"github.com/isaacphi/mcp-language-server/integrationtests/tests/go/internal"
	"github.com/isaacphi/mcp-language-server/internal/tools"
)

func TestContent(t *testing.T) {
	suite := internal.GetTestSuite(t)

	ctx, cancel := context.WithTimeout(suite.Context, 10*time.Second)
	defer cancel()

	tests := []struct {
		name         string
		file         string
		line         int
		column       int
		expectedText string
		snapshotName string
	}{
		{
			name:         "Function",
			file:         filepath.Join(suite.WorkspaceDir, "clean.go"),
			line:         32,
			column:       1,
			expectedText: "func TestFunction()",
			snapshotName: "test_function",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call the ReadDefinition tool
			result, err := tools.GetContentInfo(ctx, suite.Client, tc.file, tc.line, tc.column)
			if err != nil {
				t.Fatalf("Failed to read content: %v", err)
			}

			// Check that the result contains relevant information
			if !strings.Contains(result, tc.expectedText) {
				t.Errorf("Content does not contain expected text: %s", tc.expectedText)
			}

			// Use snapshot testing to verify exact output
			common.SnapshotTest(t, "go", "content", tc.snapshotName, result)
		})
	}
}
