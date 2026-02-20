package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pyromancer/idony/internal/db"
)

type SubAgentSpawnManager interface {
	Spawn(ctx context.Context, prompt string) (string, error)
	SpawnNamed(ctx context.Context, agentName, prompt string) (string, error)
	List() ([]db.SubAgentTask, error)
	ListDefinitions() ([]db.SubAgentDefinition, error)
	DefineAgent(name, personality, tools, model string) error
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
{"action": "spawn|spawn_named|list|result|define|list_definitions", "prompt": "prompt for spawn", "id": "id for result retrieval", "name": "name for define/spawn_named", "personality": "personality for define", "tools": "comma-separated tool names for define (or '*' for all)", "model": "optional model name for define"}`
}

func (s *SubAgentTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action      string `json:"action"`
		Prompt      string `json:"prompt"`
		ID          string `json:"id"`
		Name        string `json:"name"`
		Personality string `json:"personality"`
		Tools       string `json:"tools"`
		Model       string `json:"model"`
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
		return fmt.Sprintf("Spawned generic sub-agent with ID: %s", id), nil
	case "spawn_named":
		if req.Name == "" || req.Prompt == "" {
			return "", fmt.Errorf("name and prompt are required for spawn_named")
		}
		id, err := s.manager.SpawnNamed(ctx, req.Name, req.Prompt)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Spawned specialized sub-agent '%s' with ID: %s", req.Name, id), nil
	case "define":
		if req.Name == "" || req.Personality == "" {
			return "", fmt.Errorf("name and personality are required for define")
		}
		err := s.manager.DefineAgent(req.Name, req.Personality, req.Tools, req.Model)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Successfully defined specialized sub-agent: %s", req.Name), nil
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
	case "list_definitions":
		defs, err := s.manager.ListDefinitions()
		if err != nil {
			return "", err
		}
		var res string
		for _, d := range defs {
			res += fmt.Sprintf("- %s: %s (Tools: %s, Model: %s)\n", d.Name, d.Personality, d.Tools, d.Model)
		}
		if res == "" {
			return "No specialized sub-agents defined yet.", nil
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
