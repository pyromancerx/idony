package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Helper to enforce path safety
func isAllowedPath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", err
	}
	cwd, _ := os.Getwd()
	if !strings.HasPrefix(absPath, cwd) {
		return "", fmt.Errorf("access denied: path must be within the project directory")
	}
	return absPath, nil
}

// ListFilesTool wraps 'ls -la'
type ListFilesTool struct{}

func (t *ListFilesTool) Name() string { return "ls" }
func (t *ListFilesTool) Description() string {
	return "Lists files in a directory. Input: directory path (e.g., '.')."
}
func (t *ListFilesTool) Execute(ctx context.Context, input string) (string, error) {
	if input == "" {
		input = "."
	}
	path, err := isAllowedPath(input)
	if err != nil {
		return "", err
	}
	// Use Go's ReadDir instead of exec ls for safety/portability
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, e := range entries {
		info, _ := e.Info()
		sb.WriteString(fmt.Sprintf("%s %d %s\n", info.Mode(), info.Size(), e.Name()))
	}
	return sb.String(), nil
}

func (t *ListFilesTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "List Files",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Directory Path", "type": "string", "hint": "."},
		},
	}
}

// ReadFileTool wraps 'cat'
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "cat" }
func (t *ReadFileTool) Description() string {
	return "Reads the content of a file. Input: file path."
}
func (t *ReadFileTool) Execute(ctx context.Context, input string) (string, error) {
	path, err := isAllowedPath(strings.TrimSpace(input))
	if err != nil {
		return "", err
	}
	
	// Check size limit (e.g., 1MB)
	info, err := os.Stat(path)
	if err != nil { return "", err }
	if info.Size() > 1024*1024 {
		return "", fmt.Errorf("file too large (>1MB)")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (t *ReadFileTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Read File",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "File Path", "type": "string", "required": true},
		},
	}
}

// WriteFileTool allows writing content to a file
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Writes content to a file. Input format: 'path|content'."
}
func (t *WriteFileTool) Execute(ctx context.Context, input string) (string, error) {
	parts := strings.SplitN(input, "|", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid format, use 'path|content'")
	}
	
	path, err := isAllowedPath(strings.TrimSpace(parts[0]))
	if err != nil {
		return "", err
	}

	content := parts[1]
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

func (t *WriteFileTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Write File",
		"fields": []map[string]interface{}{
			{"name": "path", "label": "File Path", "type": "string", "required": true},
			{"name": "content", "label": "Content", "type": "longtext", "required": true},
		},
	}
}

// DeleteFileTool allows deleting a file
type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string { return "rm" }
func (t *DeleteFileTool) Description() string { return "Deletes a file. Input: file path." }
func (t *DeleteFileTool) Execute(ctx context.Context, input string) (string, error) {
	path, err := isAllowedPath(strings.TrimSpace(input))
	if err != nil { return "", err }
	err = os.Remove(path)
	if err != nil { return "", err }
	return fmt.Sprintf("Deleted %s", path), nil
}
func (t *DeleteFileTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Delete File",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "File Path", "type": "string", "required": true},
		},
	}
}

// SearchFileTool allows finding files by pattern
type SearchFileTool struct{}

func (t *SearchFileTool) Name() string { return "find" }
func (t *SearchFileTool) Description() string { return "Finds files matching a glob pattern. Input: pattern (e.g. *.go)." }
func (t *SearchFileTool) Execute(ctx context.Context, input string) (string, error) {
	// Glob doesn't support recursive ** well in standard lib, but we can do a simple Walk
	// Or just use filepath.Glob for current dir. Let's use Walk for power.
	pattern := strings.TrimSpace(input)
	var matches []string
	
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if matched, _ := filepath.Match(pattern, info.Name()); matched {
			matches = append(matches, path)
		}
		return nil
	})
	
	if err != nil { return "", err }
	if len(matches) == 0 { return "No matches found.", nil }
	if len(matches) > 50 { matches = matches[:50]; matches = append(matches, "...(truncated)") }
	return strings.Join(matches, "\n"), nil
}
func (t *SearchFileTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Find Files",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Glob Pattern", "type": "string", "required": true},
		},
	}
}

// ShellExecTool allows executing arbitrary shell commands with safety
type ShellExecTool struct{}

func (t *ShellExecTool) Name() string { return "exec" }
func (t *ShellExecTool) Description() string {
	return "Executes an arbitrary shell command with a 30s timeout. Blocked: rm -rf, sudo."
}
func (t *ShellExecTool) Execute(ctx context.Context, input string) (string, error) {
	cmdStr := strings.TrimSpace(input)
	
	// Basic blocklist
	blocked := []string{"rm -rf", "sudo", "mkfs", ":(){:|:&};:"}
	for _, b := range blocked {
		if strings.Contains(cmdStr, b) {
			return "", fmt.Errorf("command blocked for safety")
		}
	}

	// 30s timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command timed out")
	}
	if err != nil {
		return fmt.Sprintf("Error: %v\nOutput: %s", err, string(out)), nil
	}
	return string(out), nil
}

func (t *ShellExecTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Execute Command",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Shell Command", "type": "string", "required": true},
		},
	}
}
