package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// BrowserTool interfaces with the idony-browser CLI.
type BrowserTool struct {
	binPath string
}

func NewBrowserTool(binPath string) *BrowserTool {
	return &BrowserTool{binPath: binPath}
}

func (b *BrowserTool) Name() string {
	return "browser"
}

func (b *BrowserTool) Description() string {
	return `Allows Idony to search and surf the web. 
Input must be a JSON object: {"action": "search|scrape", "query": "search query", "url": "url to scrape"}`
}

func (b *BrowserTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action string `json:"action"`
		Query  string `json:"query"`
		URL    string `json:"url"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	var args []string
	switch req.Action {
	case "search":
		if req.Query == "" {
			return "", fmt.Errorf("query is required for search")
		}
		args = []string{"search", "--query", req.Query}
	case "scrape":
		if req.URL == "" {
			return "", fmt.Errorf("url is required for scrape")
		}
		args = []string{"scrape", "--url", req.URL}
	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}

	cmd := exec.CommandContext(ctx, b.binPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing browser tool: %v\nOutput: %s", err, string(output)), nil
	}

	return string(output), nil
}
