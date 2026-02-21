package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type OllamaLibraryTool struct{}

func (t *OllamaLibraryTool) Name() string {
	return "ollama_library"
}

func (t *OllamaLibraryTool) Description() string {
	return `Explores the official Ollama master model list.
Input: {"query": "search term (e.g. vision, llama3)", "filter": "popular|newest"}`
}

func (t *OllamaLibraryTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Query  string `json:"query"`
		Filter string `json:"filter"`
	}

	// Try to parse as JSON, fallback to raw string as query
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		req.Query = input
	}

	url := "https://ollama.com/library"
	if req.Query != "" {
		url += "?q=" + req.Query
	}

	hReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	hReq.Header.Set("User-Agent", "Mozilla/5.0 (Idony AI Explorer)")

	resp, err := (&http.Client{}).Do(hReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch library: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Extract model names and descriptions using Regex (lighter than a full HTML parser for this use case)
	// Pattern for model links: <a href="/library/model-name" ...>
	reModel := regexp.MustCompile(`<a href="/library/([^"]+)"`)
	matches := reModel.FindAllStringSubmatch(string(body), -1)

	if len(matches) == 0 {
		return "No models found matching your query in the Ollama library.", nil
	}

	uniqueModels := make(map[string]bool)
	var models []string
	for _, m := range matches {
		name := m[1]
		if !uniqueModels[name] {
			uniqueModels[name] = true
			models = append(models, name)
		}
	}

	return fmt.Sprintf("Found %d models in the Ollama library:\n- %s", len(models), strings.Join(models, "\n- ")), nil
}

func (t *OllamaLibraryTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Ollama Library Explorer",
		"fields": []map[string]interface{}{
			{"name": "query", "label": "Search Query", "type": "string", "hint": "vision, coding, llama3"},
			{"name": "filter", "label": "Sort By", "type": "choice", "options": []string{"popular", "newest"}},
		},
	}
}
