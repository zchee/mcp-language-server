package definition_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/isaacphi/mcp-language-server/integrationtests/languages/common"
	"github.com/isaacphi/mcp-language-server/integrationtests/languages/go/internal"
	"github.com/isaacphi/mcp-language-server/internal/tools"
)

// TestReadDefinition tests the ReadDefinition tool with the Go language server
func TestReadDefinition(t *testing.T) {
	suite := internal.GetTestSuite(t)

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
	common.SnapshotTest(t, "go", "definition", "foobar", result)
}
