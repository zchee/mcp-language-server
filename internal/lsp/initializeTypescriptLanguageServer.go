package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// typescript-language-server requires files to be opened in order to load the project
func initializeTypescriptLanguageServer(ctx context.Context, client *Client, rootDir string) error {
	tsFiles, err := findAllTypeScriptFiles(rootDir)
	if err != nil {
		return fmt.Errorf("error finding TypeScript files: %w", err)
	}

	if len(tsFiles) == 0 {
		return fmt.Errorf("no TypeScript files found in %s", rootDir)
	}

	// Open all the TypeScript files
	for _, file := range tsFiles {
		err = client.OpenFile(ctx, file)
		if err != nil {
			return fmt.Errorf("error opening TypeScript file %s: %w", file, err)
		}
	}

	return nil
}

func findAllTypeScriptFiles(rootDir string) ([]string, error) {
	var tsFiles []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		// Check for walk errors
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip node_modules directories
			if info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file has .ts or .tsx extension
		ext := filepath.Ext(path)
		if ext == ".ts" || ext == ".tsx" {
			tsFiles = append(tsFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return tsFiles, nil
}

