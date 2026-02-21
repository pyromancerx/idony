package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/db"
)

type MemoryTool struct {
	store *db.Store
}

func NewMemoryTool(store *db.Store) *MemoryTool {
	return &MemoryTool{store: store}
}

func (m *MemoryTool) Name() string {
	return "remember"
}

func (m *MemoryTool) Description() string {
	return `Stores a fact, preference, or observation in long-term memory.
Input: {"content": "The user likes blue", "type": "preference|fact", "tags": "user,color"}`
}

func (m *MemoryTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Content string `json:"content"`
		Type    string `json:"type"`
		Tags    string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		// Fallback for simple string input
		req.Content = input
		req.Type = "fact"
	}

	if req.Content == "" {
		return "", fmt.Errorf("content is required")
	}

	err := m.store.SaveMemory(req.Content, req.Type, req.Tags)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Memory saved: [%s] %s", req.Type, req.Content), nil
}

func (m *MemoryTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Save Memory",
		"fields": []map[string]interface{}{
			{"name": "content", "label": "Memory Content", "type": "longtext", "required": true},
			{"name": "type", "label": "Type", "type": "choice", "options": []string{"fact", "preference", "observation"}},
			{"name": "tags", "label": "Tags (comma-separated)", "type": "string"},
		},
	}
}

// RecallTool allows manual memory search
type RecallTool struct {
	store *db.Store
}

func NewRecallTool(store *db.Store) *RecallTool {
	return &RecallTool{store: store}
}

func (r *RecallTool) Name() string {
	return "recall"
}

func (r *RecallTool) Description() string {
	return "Searches long-term memory. Input: search query string."
}

func (r *RecallTool) Execute(ctx context.Context, input string) (string, error) {
	memories, err := r.store.SearchMemories(input, 10)
	if err != nil {
		return "", err
	}

	if len(memories) == 0 {
		return "No relevant memories found.", nil
	}

	var sb strings.Builder
	sb.WriteString("Found Memories:\n")
	for _, m := range memories {
		sb.WriteString(fmt.Sprintf("- [%s] %s (Tags: %s)\n", m.Type, m.Content, m.Tags))
	}
	return sb.String(), nil
}

func (r *RecallTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Recall Memory",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Search Query", "type": "string", "required": true},
		},
	}
}
