package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
)

type OptimizeMemoryTool struct {
	store  *db.Store
	client Summarizer
}

func NewOptimizeMemoryTool(store *db.Store, client Summarizer) *OptimizeMemoryTool {
	return &OptimizeMemoryTool{store: store, client: client}
}

func (o *OptimizeMemoryTool) Name() string {
	return "optimize_memory"
}

func (o *OptimizeMemoryTool) Description() string {
	return "Analyzes stored memories to merge duplicates and remove contradictions. Input: ignored."
}

func (o *OptimizeMemoryTool) Execute(ctx context.Context, input string) (string, error) {
	memories, err := o.store.GetAllMemories()
	if err != nil {
		return "", err
	}
	if len(memories) < 2 {
		return "Not enough memories to optimize.", nil
	}

	var content strings.Builder
	for _, m := range memories {
		content.WriteString(fmt.Sprintf("ID: %d | Type: %s | Content: %s\n", m.ID, m.Type, m.Content))
	}

	prompt := fmt.Sprintf(`Analyze the following list of memories. Identify duplicates, redundancies, or contradictions.
Return a JSON object with:
1. "delete": list of IDs to remove.
2. "merge": list of objects {"ids": [id1, id2], "new_content": "merged content"} to replace multiple memories with one.

Memories:
%s`, content.String())

	resp, err := o.client.GenerateResponse(ctx, []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return "", err
	}

	// Simple extraction of JSON if wrapped in markdown
	jsonStr := resp
	if start := strings.Index(resp, "{"); start != -1 {
		if end := strings.LastIndex(resp, "}"); end != -1 {
			jsonStr = resp[start : end+1]
		}
	}

	var plan struct {
		Delete []int `json:"delete"`
		Merge  []struct {
			IDs        []int  `json:"ids"`
			NewContent string `json:"new_content"`
		} `json:"merge"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return fmt.Sprintf("Failed to parse optimization plan: %v\nRaw: %s", err, resp), nil
	}

	deletedCount := 0
	mergedCount := 0

	// Process deletions
	if len(plan.Delete) > 0 {
		// We don't have DeleteMemories yet, assumed DeleteMessages logic works or we add it
		// Using a loop for now or add DeleteMemory func
		for _, id := range plan.Delete {
			o.store.DB.Exec("DELETE FROM memories WHERE id = ?", id)
		}
		deletedCount += len(plan.Delete)
	}

	// Process merges
	for _, m := range plan.Merge {
		for _, id := range m.IDs {
			o.store.DB.Exec("DELETE FROM memories WHERE id = ?", id)
		}
		o.store.SaveMemory(m.NewContent, "fact", "merged")
		mergedCount++
	}

	return fmt.Sprintf("Optimization complete. Deleted: %d, Merged: %d", deletedCount, mergedCount), nil
}

func (o *OptimizeMemoryTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Optimize Memory",
		"fields": []map[string]interface{}{},
	}
}
