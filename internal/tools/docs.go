package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DocsTool allows Idony to read its own documentation.
type DocsTool struct {
	docsPath string
}

func NewDocsTool(path string) *DocsTool {
	return &DocsTool{docsPath: path}
}

func (d *DocsTool) Name() string {
	return "help"
}

func (d *DocsTool) Description() string {
	return "Retrieves information about Idony's capabilities and commands. Input: keyword or filename (e.g., 'capabilities', 'commands')."
}

func (d *DocsTool) Execute(ctx context.Context, input string) (string, error) {
	if input == "" {
		input = "capabilities"
	}

	filename := strings.TrimSpace(input)
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	fullPath := filepath.Join(d.docsPath, filename)
	
	// Security: Ensure the path is within the docs directory
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	absDocs, _ := filepath.Abs(d.docsPath)
	if !strings.HasPrefix(absPath, absDocs) {
		return "Access denied: outside documentation directory.", nil
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		// Try to list available docs if file not found
		files, _ := os.ReadDir(d.docsPath)
		var available []string
		for _, f := range files {
			if !f.IsDir() {
				available = append(available, strings.TrimSuffix(f.Name(), ".md"))
			}
		}
		return fmt.Sprintf("Topic '%s' not found. Available topics: %s", input, strings.Join(available, ", ")), nil
	}

	return string(content), nil
}

func (d *DocsTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Documentation",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Topic/Filename", "type": "string", "hint": "capabilities"},
		},
	}
}
