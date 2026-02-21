package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pyromancer/idony/internal/db"
)

type KnowledgeStore interface {
	SaveKnowledge(k db.KnowledgeEntry) error
	GetKnowledge(key string) (*db.KnowledgeEntry, error)
	SearchKnowledge(query string) ([]db.KnowledgeEntry, error)
	ListKnowledgeKeys() ([]string, error)
}

type KnowledgeTool struct {
	store      KnowledgeStore
	exportPath string
}

func NewKnowledgeTool(s KnowledgeStore, exportPath string) *KnowledgeTool {
	return &KnowledgeTool{
		store:      s,
		exportPath: exportPath,
	}
}

func (k *KnowledgeTool) Name() string {
	return "knowledge"
}

func (k *KnowledgeTool) Description() string {
	return `Manages the persistent knowledge base. Actions: "save", "get", "search", "list", "export".
JSON Input: {"action": "save|get|search|list|export", "key": "unique_id", "content": "data to store", "category": "topic", "tags": "tag1,tag2", "query": "search term"}`
}

func (k *KnowledgeTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action   string `json:"action"`
		Key      string `json:"key"`
		Content  string `json:"content"`
		Category string `json:"category"`
		Tags     string `json:"tags"`
		Query    string `json:"query"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	switch req.Action {
	case "save":
		if req.Key == "" || req.Content == "" {
			return "", fmt.Errorf("key and content are required for save")
		}
		entry := db.KnowledgeEntry{
			Key:      req.Key,
			Category: req.Category,
			Content:  req.Content,
			Tags:     req.Tags,
		}
		if err := k.store.SaveKnowledge(entry); err != nil {
			return "", err
		}
		k.syncToFile(entry)
		return fmt.Sprintf("Knowledge saved and synced to disk: %s", req.Key), nil

	case "get":
		entry, err := k.store.GetKnowledge(req.Key)
		if err != nil {
			return "", err
		}
		if entry == nil {
			return "Knowledge entry not found.", nil
		}
		return fmt.Sprintf("Category: %s\nTags: %s\n\n%s", entry.Category, entry.Tags, entry.Content), nil

	case "search":
		entries, err := k.store.SearchKnowledge(req.Query)
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Search results for '%s':\n", req.Query))
		for _, e := range entries {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.Key, e.Category))
		}
		if len(entries) == 0 { return "No matches found.", nil }
		return sb.String(), nil

	case "list":
		keys, err := k.store.ListKnowledgeKeys()
		if err != nil {
			return "", err
		}
		if len(keys) == 0 { return "Knowledge base is empty.", nil }
		return "Known Topics:\n- " + strings.Join(keys, "\n- "), nil

	case "export":
		return k.exportAll()

	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}

func (k *KnowledgeTool) syncToFile(e db.KnowledgeEntry) {
	os.MkdirAll(k.exportPath, 0755)
	filename := filepath.Join(k.exportPath, e.Key+".md")
	header := fmt.Sprintf("---\nCategory: %s\nTags: %s\nUpdated: %s\n---\n\n", e.Category, e.Tags, time.Now().Format(time.RFC3339))
	os.WriteFile(filename, []byte(header+e.Content), 0644)
}

func (k *KnowledgeTool) exportAll() (string, error) {
	keys, _ := k.store.ListKnowledgeKeys()
	for _, key := range keys {
		entry, _ := k.store.GetKnowledge(key)
		if entry != nil {
			k.syncToFile(*entry)
		}
	}
	return fmt.Sprintf("All knowledge entries exported to: %s", k.exportPath), nil
}

func (k *KnowledgeTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Knowledge Base",
		"actions": []map[string]interface{}{
			{
				"name":  "save",
				"label": "Save Knowledge",
				"fields": []map[string]interface{}{
					{"name": "key", "label": "Key (Unique ID)", "type": "string", "required": true},
					{"name": "category", "label": "Category", "type": "string", "hint": "General"},
					{"name": "tags", "label": "Tags (comma-separated)", "type": "string"},
					{"name": "content", "label": "Content", "type": "longtext", "required": true},
				},
			},
			{
				"name":  "get",
				"label": "Get Knowledge",
				"fields": []map[string]interface{}{
					{"name": "key", "label": "Key", "type": "string", "required": true},
				},
			},
			{
				"name":  "search",
				"label": "Search Knowledge",
				"fields": []map[string]interface{}{
					{"name": "query", "label": "Search Term", "type": "string", "required": true},
				},
			},
			{
				"name":  "list",
				"label": "List All Keys",
				"fields": []map[string]interface{}{},
			},
			{
				"name":  "export",
				"label": "Export to Markdown",
				"fields": []map[string]interface{}{},
			},
		},
	}
}
