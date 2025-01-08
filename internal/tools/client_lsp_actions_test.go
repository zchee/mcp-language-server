package tools

import (
	"os"
	"testing"

	"github.com/isaacphi/mcp-language-server/internal/protocol"
	"github.com/stretchr/testify/assert"
)

func createTempFileWithContent(t *testing.T, content string) string {
	t.Helper()
	tmpfile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpfile.Name()
}

func TestReadLocation(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		location    protocol.Location
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:    "single line - middle of text",
			content: "Hello, World!\nSecond line\nThird line",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 7},
					End:   protocol.Position{Line: 0, Character: 12},
				},
			},
			want: "World",
		},
		{
			name:    "multi line - simple",
			content: "First line\nSecond line\nThird line",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 6},
					End:   protocol.Position{Line: 1, Character: 6},
				},
			},
			want: "line\nSecond",
		},
		{
			name:    "single line - full line",
			content: "Hello\nWorld\nTest",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 1, Character: 0},
					End:   protocol.Position{Line: 1, Character: 5},
				},
			},
			want: "World",
		},
		{
			name:    "multi line - three lines",
			content: "First line\nSecond line\nThird line\nFourth line",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 6},
					End:   protocol.Position{Line: 2, Character: 5},
				},
			},
			want: "line\nSecond line\nThird",
		},
		{
			name:    "error - invalid start line",
			content: "Hello\nWorld",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 5, Character: 0},
					End:   protocol.Position{Line: 5, Character: 5},
				},
			},
			wantErr:     true,
			errContains: "invalid Location",
		},
		{
			name:    "error - invalid end line",
			content: "Hello\nWorld",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 5, Character: 5},
				},
			},
			wantErr:     true,
			errContains: "invalid Location",
		},
		{
			name:    "error - invalid start character",
			content: "Hello\nWorld",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 10},
					End:   protocol.Position{Line: 0, Character: 12},
				},
			},
			wantErr:     true,
			errContains: "invalid Location",
		},
		{
			name:    "error - invalid end character",
			content: "Hello\nWorld",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 10},
				},
			},
			wantErr:     true,
			errContains: "invalid Location",
		},
		{
			name:    "zero-width selection",
			content: "Hello, World!",
			location: protocol.Location{
				URI: protocol.DocumentURI("test.txt"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 5},
					End:   protocol.Position{Line: 0, Character: 5},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the test content
			filename := createTempFileWithContent(t, tt.content)
			defer os.Remove(filename)

			// Update the location URI to point to our temporary file
			tt.location.URI = protocol.DocumentURI(filename)

			// Call the function
			got, err := ReadLocation(tt.location)

			// Check error cases
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			// Check success cases
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadLocation_FileErrors(t *testing.T) {
	loc := protocol.Location{
		URI: protocol.DocumentURI("nonexistent-file.txt"),
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 5},
		},
	}

	_, err := ReadLocation(loc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}
