package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pyromancer/idony/internal/db"
)

type WebhookTool struct {
	store *db.Store
}

func NewWebhookTool(store *db.Store) *WebhookTool {
	return &WebhookTool{store: store}
}

func (w *WebhookTool) Name() string {
	return "webhook"
}

func (w *WebhookTool) Description() string {
	return "Manage incoming webhooks. Actions: create, list, delete. Payload is passed as {{payload}} in prompt."
}

func (w *WebhookTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action         string `json:"action"`
		Name           string `json:"name"`
		TargetAgent    string `json:"target_agent"`
		PromptTemplate string `json:"prompt_template"`
		ID             string `json:"id"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", err
	}

	switch req.Action {
	case "create":
		id := uuid.New().String()
		if req.TargetAgent == "" { req.TargetAgent = "main" }
		
		wh := db.Webhook{
			ID:             id,
			Name:           req.Name,
			TargetAgent:    req.TargetAgent,
			PromptTemplate: req.PromptTemplate,
		}
		if err := w.store.SaveWebhook(wh); err != nil {
			return "", err
		}
		return fmt.Sprintf("Webhook created. URL: /webhooks/%s", id), nil

	case "list":
		list, err := w.store.ListWebhooks()
		if err != nil { return "", err }
		if len(list) == 0 { return "No webhooks found.", nil }
		
		var sb strings.Builder
		sb.WriteString("Active Webhooks:\n")
		for _, hook := range list {
			sb.WriteString(fmt.Sprintf("- [%s] %s -> %s (Template: %s)\n", hook.ID, hook.Name, hook.TargetAgent, hook.PromptTemplate))
		}
		return sb.String(), nil

	case "delete":
		if err := w.store.DeleteWebhook(req.ID); err != nil { return "", err }
		return "Webhook deleted.", nil

	default:
		return "", fmt.Errorf("unknown action: %s", req.Action)
	}
}

func (w *WebhookTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Webhook Manager",
		"actions": []map[string]interface{}{
			{
				"name": "create",
				"label": "Create Webhook",
				"fields": []map[string]interface{}{
					{"name": "name", "label": "Name", "type": "string", "required": true},
					{"name": "target_agent", "label": "Target Agent", "type": "string", "hint": "main or subagent name"},
					{"name": "prompt_template", "label": "Prompt Template (use {{payload}})", "type": "longtext", "required": true},
				},
			},
			{
				"name": "list",
				"label": "List Webhooks",
				"fields": []map[string]interface{}{},
			},
			{
				"name": "delete",
				"label": "Delete Webhook",
				"fields": []map[string]interface{}{
					{"name": "id", "label": "Webhook ID", "type": "string"},
				},
			},
		},
	}
}
