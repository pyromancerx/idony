package tools

import (
	"context"
	"fmt"

	"github.com/pyromancer/idony/internal/db"
	"github.com/pyromancer/idony/internal/llm"
)

type Summarizer interface {
	GenerateResponse(ctx context.Context, messages []llm.Message) (string, error)
}

type CompactTool struct {
	store  *db.Store
	client Summarizer
}

func NewCompactTool(store *db.Store, client Summarizer) *CompactTool {
	return &CompactTool{store: store, client: client}
}

func (c *CompactTool) Name() string {
	return "compact"
}

func (c *CompactTool) Description() string {
	return "Summarizes older conversation history to save tokens. Input: ignored."
}

func (c *CompactTool) Execute(ctx context.Context, input string) (string, error) {
	// 1. Fetch oldest 10 messages (arbitrary chunk size)
	msgs, err := c.store.GetOldestMessages(10)
	if err != nil {
		return "", err
	}
	if len(msgs) < 5 {
		return "History is too short to compact.", nil
	}

	// 2. Format for summarization
	var transcript string
	var ids []int
	for _, m := range msgs {
		transcript += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
		ids = append(ids, m.ID)
	}

	// 3. Ask LLM to summarize
	prompt := fmt.Sprintf("Summarize the following conversation segment concisely, preserving key facts and context:\n\n%s", transcript)
	
	summary, err := c.client.GenerateResponse(ctx, []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return "", fmt.Errorf("summarization failed: %w", err)
	}

	// 4. Delete old messages
	err = c.store.DeleteMessages(ids)
	if err != nil {
		return "", fmt.Errorf("failed to delete old messages: %w", err)
	}

	// 5. Insert summary as a system/context message (or just a user message saying "Previous context: ...")
	// We'll use 'system' role if supported, or 'assistant'
	err = c.store.SaveMessage("system", fmt.Sprintf("Summary of previous conversation: %s", summary))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Compacted %d messages into summary: %s", len(msgs), summary), nil
}

func (c *CompactTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Compact History",
		"fields": []map[string]interface{}{},
	}
}
