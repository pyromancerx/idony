package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pyromancer/idony/internal/db"
)

type SubAgentSpawnManager interface {
	Spawn(ctx context.Context, prompt string) (string, error)
	List() ([]db.SubAgentTask, error)
}

// SubAgentTool allows Idony to spawn background tasks.
type SubAgentTool struct {
	manager SubAgentSpawnManager
}

func NewSubAgentTool(m SubAgentSpawnManager) *SubAgentTool {
	return &SubAgentTool{manager: m}
}

func (s *SubAgentTool) Name() string {
	return "subagent"
}

func (s *SubAgentTool) Description() string {
	return `Manages sub-agents. Input must be a JSON object: 
{"action": "spawn|list|result", "prompt": "prompt for spawn", "id": "id for result retrieval"}`
}

func (s *SubAgentTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action string `json:"action"`
		Prompt string `json:"prompt"`
		ID     string `json:"id"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	switch req.Action {
	case "spawn":
		if req.Prompt == "" {
			return "", fmt.Errorf("prompt is required for spawn")
		}
		id, err := s.manager.Spawn(ctx, req.Prompt)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Spawned sub-agent with ID: %s", id), nil
	case "list":
		tasks, err := s.manager.List()
		if err != nil {
			return "", err
		}
		var res string
		for _, t := range tasks {
			res += fmt.Sprintf("[%s] %s: %s\n", t.ID, t.Status, t.Prompt)
		}
		if res == "" {
			return "No sub-agents found.", nil
		}
		return res, nil
	case "result":
		if req.ID == "" {
			return "", fmt.Errorf("id is required for result action")
		}
		tasks, err := s.manager.List()
		if err != nil {
			return "", err
		}
		for _, t := range tasks {
			if t.ID == req.ID {
				if t.Status == "running" {
					return fmt.Sprintf("Sub-agent %s is still running.", req.ID), nil
				}
				return fmt.Sprintf("Sub-agent %s result: %s", req.ID, t.Result), nil
			}
		}
		return fmt.Sprintf("Sub-agent with ID %s not found.", req.ID), nil
	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}
