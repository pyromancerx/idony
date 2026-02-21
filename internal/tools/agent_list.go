package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/pyromancer/idony/internal/db"
)

type AgentListManager interface {
	ListDefinitions() ([]db.SubAgentDefinition, error)
}

type AgentListTool struct {
	manager AgentListManager
}

func NewAgentListTool(m AgentListManager) *AgentListTool {
	return &AgentListTool{manager: m}
}

func (a *AgentListTool) Name() string {
	return "list_agents"
}

func (a *AgentListTool) Description() string {
	return "Lists all specialized agents currently defined in Idony."
}

func (a *AgentListTool) Execute(ctx context.Context, input string) (string, error) {
	defs, err := a.manager.ListDefinitions()
	if err != nil {
		return "", fmt.Errorf("failed to fetch agent list: %w", err)
	}

	if len(defs) == 0 {
		return "No specialized agents have been defined yet. You can create one by asking me!", nil
	}

	var sb strings.Builder
	sb.WriteString("Available Specialized Agents:\n")
	for _, d := range defs {
		sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", d.Name, d.Model, d.Personality))
	}

	return sb.String(), nil
}

func (a *AgentListTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title":  "Specialized Agents",
		"fields": []map[string]interface{}{},
	}
}
