package diagnostics_test

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

// TestDiagnostics tests diagnostics functionality with the Go language server
func TestDiagnostics(t *testing.T) {
	// Test with a clean file
	t.Run("CleanFile", func(t *testing.T) {
		// Get a test suite with clean code
		suite := internal.GetTestSuite(t)

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

		common.SnapshotTest(t, "go", "diagnostics", "clean", result)
	})

	// Test with a file containing an error
	t.Run("FileWithError", func(t *testing.T) {
		// Get a test suite with code that contains errors
		suite := internal.GetErrorTestSuite(t)

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

		common.SnapshotTest(t, "go", "diagnostics", "unreachable", result)
	})
}
