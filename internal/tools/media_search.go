package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/db"
)

type MediaSearchTool struct {
	store *db.Store
}

func NewMediaSearchTool(store *db.Store) *MediaSearchTool {
	return &MediaSearchTool{store: store}
}

func (m *MediaSearchTool) Name() string {
	return "media_search"
}

func (m *MediaSearchTool) Description() string {
	return "Searches indexed images, videos, and audio transcripts. Input: search query."
}

func (m *MediaSearchTool) Execute(ctx context.Context, input string) (string, error) {
	entries, err := m.store.SearchMedia(input, 5)
	if err != nil {
		return "", err
	}

	if len(entries) == 0 {
		return "No relevant media found.", nil
	}

	var sb strings.Builder
	sb.WriteString("Found Media:\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("- [%s] %s: %s (ID: %d)\n", e.MediaType, e.FilePath, e.Description[:min(len(e.Description), 100)]+"...", e.ID))
	}
	return sb.String(), nil
}

func (m *MediaSearchTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Search Media Index",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Search Query", "type": "string", "required": true},
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
