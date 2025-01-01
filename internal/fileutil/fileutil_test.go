package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kralicky/tools-lite/gopls/pkg/protocol"
)

func setupTestFile(t *testing.T) (string, func()) {
	t.Helper()

	content := []byte(`package test

func example() {
	fmt.Println("Hello, World!")
	fmt.Println("Another line")
}
`)

	dir := t.TempDir()
	filename := filepath.Join(dir, "test.go")

	err := os.WriteFile(filename, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cleanup := func() {
		os.Remove(filename)
	}

	return filename, cleanup
}

// Helper function to print string as bytes
func debugString(s string) string {
	var bytes []string
	for i := 0; i < len(s); i++ {
		bytes = append(bytes, fmt.Sprintf("%02x", s[i]))
	}
	return fmt.Sprintf("len=%d bytes=[%s]", len(s), strings.Join(bytes, " "))
}

func TestReadLocationContent(t *testing.T) {
	filename, cleanup := setupTestFile(t)
	defer cleanup()

	tests := []struct {
		name     string
		location protocol.Location
		want     string
		wantErr  bool
	}{
		{
			name: "single line",
			location: protocol.Location{
				URI: protocol.DocumentURI("file://" + filename),
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 1},
					End:   protocol.Position{Line: 3, Character: 29},
				},
			},
			want:    `fmt.Println("Hello, World!")`,
			wantErr: false,
		},
		{
			name: "multiple lines",
			location: protocol.Location{
				URI: protocol.DocumentURI("file://" + filename),
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 1},
					End:   protocol.Position{Line: 4, Character: 28}, // Changed to include closing parenthesis
				},
			},
			want:    "fmt.Println(\"Hello, World!\")\n\tfmt.Println(\"Another line\")",
			wantErr: false,
		},
		{
			name: "end character beyond line length",
			location: protocol.Location{
				URI: protocol.DocumentURI("file://" + filename),
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 1},
					End:   protocol.Position{Line: 3, Character: 100}, // Deliberately too long
				},
			},
			want:    `fmt.Println("Hello, World!")`,
			wantErr: false,
		},
		{
			name: "invalid range - start after end line",
			location: protocol.Location{
				URI: protocol.DocumentURI("file://" + filename),
				Range: protocol.Range{
					Start: protocol.Position{Line: 4, Character: 0},
					End:   protocol.Position{Line: 3, Character: 29},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid range - start after end character same line",
			location: protocol.Location{
				URI: protocol.DocumentURI("file://" + filename),
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 20},
					End:   protocol.Position{Line: 3, Character: 10},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "non-existent line",
			location: protocol.Location{
				URI: protocol.DocumentURI("file://" + filename),
				Range: protocol.Range{
					Start: protocol.Position{Line: 100, Character: 0},
					End:   protocol.Position{Line: 101, Character: 10},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "character position past line end",
			location: protocol.Location{
				URI: protocol.DocumentURI("file://" + filename),
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 50},
					End:   protocol.Position{Line: 3, Character: 60},
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadLocationContent(tt.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadLocationContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ReadLocationContent()\nGot:  %s\nWant: %s\nGot bytes:  %s\nWant bytes: %s",
					got, tt.want, debugString(got), debugString(tt.want))
			}
		})
	}
}

func TestReplaceLocationContent(t *testing.T) {
	tests := []struct {
		name       string
		location   protocol.Location
		newContent string
		wantErr    bool
		wantFile   string
	}{
		{
			name: "replace single line",
			location: protocol.Location{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 1},
					End:   protocol.Position{Line: 3, Character: 29},
				},
			},
			newContent: `fmt.Println("Goodbye, World!")`,
			wantErr:    false,
			wantFile: `package test

func example() {
	fmt.Println("Goodbye, World!")
	fmt.Println("Another line")
}
`,
		},
		{
			name: "replace multiple lines",
			location: protocol.Location{
				Range: protocol.Range{
					Start: protocol.Position{Line: 3, Character: 1},
					End:   protocol.Position{Line: 4, Character: 28}, // Changed to match the read test
				},
			},
			newContent: `fmt.Println("Single line now")`,
			wantErr:    false,
			wantFile: `package test

func example() {
	fmt.Println("Single line now")
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename, cleanup := setupTestFile(t)
			defer cleanup()

			tt.location.URI = protocol.DocumentURI("file://" + filename)

			err := ReplaceLocationContent(tt.location, tt.newContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceLocationContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Read file and compare content
				got, err := os.ReadFile(filename)
				if err != nil {
					t.Fatalf("Failed to read result file: %v", err)
				}

				// Normalize line endings for comparison
				gotStr := strings.ReplaceAll(string(got), "\r\n", "\n")
				wantStr := strings.ReplaceAll(tt.wantFile, "\r\n", "\n")

				if gotStr != wantStr {
					t.Errorf("File content = %q, want %q", gotStr, wantStr)
				}
			}
		})
	}
}
