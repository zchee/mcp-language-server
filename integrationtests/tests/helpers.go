package tests

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Logger is an interface for logging in tests
type Logger interface {
	Printf(format string, v ...interface{})
}

// Helper to copy directories recursively
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err = copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err = copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper to copy a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// logWriter implements io.Writer to capture and log output
type logWriter struct {
	logger Logger
	prefix string
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.logger.Printf("%s: %s", w.prefix, string(p))
	return len(p), nil
}

// CleanupTestSuites is a helper to clean up all test suites in a test
func CleanupTestSuites(suites ...*TestSuite) {
	for _, suite := range suites {
		if suite != nil {
			suite.Cleanup()
		}
	}
}

// normalizePaths replaces absolute paths in the result with placeholder paths for consistent snapshots
func normalizePaths(t *testing.T, input string) string {
	// No need to get the repo root - we're just looking for patterns
	
	// Simple approach: just replace any path segments that contain workspace/
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		// Any line containing a path to a workspace file needs normalization
		if strings.Contains(line, "/workspace/") {
			// Extract everything after /workspace/
			parts := strings.Split(line, "/workspace/")
			if len(parts) > 1 {
				// Replace with a simple placeholder path
				lines[i] = "/TEST_OUTPUT/workspace/" + parts[1]
			}
		}
	}
	
	return strings.Join(lines, "\n")
}

// SnapshotTest compares the actual result against an expected result file
// If the file doesn't exist or UPDATE_SNAPSHOTS=true env var is set, it will update the snapshot
func SnapshotTest(t *testing.T, snapshotName, actualResult string) {
	// Normalize paths in the result to avoid system-specific paths in snapshots
	actualResult = normalizePaths(t, actualResult)
	
	// Get the absolute path to the snapshots directory
	repoRoot, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("Failed to get repo root: %v", err)
	}

	snapshotDir := filepath.Join(repoRoot, "integrationtests", "fixtures", "snapshots")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatalf("Failed to create snapshots directory: %v", err)
	}

	snapshotFile := filepath.Join(snapshotDir, snapshotName+".snap")

	// Use a package-level flag to control snapshot updates
	updateFlag := os.Getenv("UPDATE_SNAPSHOTS") == "true"

	// If snapshot doesn't exist or update flag is set, write the snapshot
	_, err = os.Stat(snapshotFile)
	if os.IsNotExist(err) || updateFlag {
		if err := os.WriteFile(snapshotFile, []byte(actualResult), 0644); err != nil {
			t.Fatalf("Failed to write snapshot: %v", err)
		}
		if os.IsNotExist(err) {
			t.Logf("Created new snapshot: %s", snapshotFile)
		} else {
			t.Logf("Updated snapshot: %s", snapshotFile)
		}
		return
	}

	// Read the expected result
	expectedBytes, err := os.ReadFile(snapshotFile)
	if err != nil {
		t.Fatalf("Failed to read snapshot: %v", err)
	}
	expected := string(expectedBytes)

	// Compare the results
	if expected != actualResult {
		t.Errorf("Result doesn't match snapshot.\nExpected:\n%s\n\nActual:\n%s", expected, actualResult)

		// Create a diff file for debugging
		diffFile := snapshotFile + ".diff"
		diffContent := fmt.Sprintf("=== Expected ===\n%s\n\n=== Actual ===\n%s", expected, actualResult)
		if err := os.WriteFile(diffFile, []byte(diffContent), 0644); err != nil {
			t.Logf("Failed to write diff file: %v", err)
		} else {
			t.Logf("Wrote diff to: %s", diffFile)
		}
	}
}