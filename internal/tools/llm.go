package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/llm"
)

type ModelListTool struct {
	client *llm.OllamaClient
}

func NewModelListTool(c *llm.OllamaClient) *ModelListTool {
	return &ModelListTool{client: c}
}

func (m *ModelListTool) Name() string {
	return "list_models"
}

func (m *ModelListTool) Description() string {
	return "Lists all available LLM models on the Ollama server."
}

func (m *ModelListTool) Execute(ctx context.Context, input string) (string, error) {
	models, err := m.client.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch models: %w", err)
	}

	if len(models) == 0 {
		return "No models found on the server.", nil
	}

	return "Available Models:\n- " + strings.Join(models, "\n- "), nil
}

func (m *ModelListTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title":  "Available Models",
		"fields": []map[string]interface{}{},
	}
}
