package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/db"
)

type MessagingTool struct {
	store *db.Store
}

func NewMessagingTool(store *db.Store) *MessagingTool {
	return &MessagingTool{store: store}
}

func (m *MessagingTool) Name() string {
	return "send_message"
}

func (m *MessagingTool) Description() string {
	return `Sends a message to another agent. Input: {"to": "agent_name", "content": "..."}`
}

func (m *MessagingTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		To      string `json:"to"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", err
	}

	_, err := m.store.DB.Exec("INSERT INTO agent_messages (from_agent, to_agent, content) VALUES (?, ?, ?)", "main", req.To, req.Content)
	if err != nil {
		return "", err
	}
	return "Message sent.", nil
}

func (m *MessagingTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Send Message",
		"fields": []map[string]interface{}{
			{"name": "to", "label": "Recipient", "type": "string", "required": true},
			{"name": "content", "label": "Message", "type": "longtext", "required": true},
		},
	}
}

// InboxTool
type InboxTool struct {
	store *db.Store
}

func NewInboxTool(store *db.Store) *InboxTool {
	return &InboxTool{store: store}
}

func (i *InboxTool) Name() string {
	return "check_inbox"
}

func (i *InboxTool) Description() string {
	return "Checks messages for a specific agent. Input: agent_name"
}

func (i *InboxTool) Execute(ctx context.Context, input string) (string, error) {
	rows, err := i.store.DB.Query("SELECT from_agent, content, created_at FROM agent_messages WHERE to_agent = ? AND read = 0", input)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var sb strings.Builder
	count := 0
	for rows.Next() {
		var from, content, date string
		rows.Scan(&from, &content, &date)
		sb.WriteString(fmt.Sprintf("From %s (%s): %s\n", from, date, content))
		count++
	}
	
	if count == 0 {
		return "No new messages.", nil
	}

	// Mark as read
	i.store.DB.Exec("UPDATE agent_messages SET read = 1 WHERE to_agent = ? AND read = 0", input)
	return sb.String(), nil
}

func (i *InboxTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Check Inbox",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Agent Name", "type": "string", "required": true},
		},
	}
}
