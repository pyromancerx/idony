package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"github.com/pyromancer/idony/internal/db"
)

type SubAgentSpawnManager interface {
	Spawn(ctx context.Context, prompt string, images []string) (string, error)
	SpawnNamed(ctx context.Context, agentName, prompt string, images []string) (string, error)
	List() ([]db.SubAgentTask, error)
	ListDefinitions() ([]db.SubAgentDefinition, error)
	DefineAgent(name, personality, tools, model string) error
	GetAvailableTools() []string
}

type ContextImagesProvider interface {
	GetLastUserImages() []string
}

// SubAgentTool allows Idony to spawn background tasks.
type SubAgentTool struct {
	manager SubAgentSpawnManager
	context ContextImagesProvider
}

func NewSubAgentTool(m SubAgentSpawnManager, c ContextImagesProvider) *SubAgentTool {
	return &SubAgentTool{manager: m, context: c}
}

func (s *SubAgentTool) Name() string {
	return "subagent"
}

func (s *SubAgentTool) Description() string {
	return `Manages sub-agents. 
Actions: 
- "spawn": Starts a new task with a prompt and optional images.
- "spawn_named": Starts a task using a pre-defined agent's personality and tools.
- "list": Shows all tasks and their IDs.
- "result": Retrieves the final output of a completed task (requires "id").
- "define": Creates a new specialized agent definition.
- "list_definitions": Lists all available specialized agents.
Input MUST be a JSON object: {"action": "spawn|spawn_named|list|result|define", "prompt": "...", "images": ["base64..."], "id": "task_id", "name": "agent_name"}.
If "action" is omitted, "spawn" is assumed. If "images" is omitted, current context images are used.`
}

func (s *SubAgentTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action      string   `json:"action"`
		Prompt      string   `json:"prompt"`
		Images      []string `json:"images,omitempty"`
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Personality string   `json:"personality"`
		Tools       string   `json:"tools"`
		Model       string   `json:"model"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	if req.Action == "" {
		req.Action = "spawn"
	}

	// Heuristic: if LLM provides a prompt but asks for 'result' or 'list', it probably meant 'spawn'
	if req.Prompt != "" && (req.Action == "result" || req.Action == "list") && req.ID == "" {
		req.Action = "spawn"
	}

	// Fallback to context images if none provided in request
	if len(req.Images) == 0 && s.context != nil {
		req.Images = s.context.GetLastUserImages()
	}

	switch req.Action {
	case "get_capabilities":
		tools := s.manager.GetAvailableTools()
		return fmt.Sprintf("Available tools for sub-agents: %s", strings.Join(tools, ", ")), nil
	case "spawn":
		// If name and personality are provided, it's a "define and spawn" request
		if req.Name != "" && req.Personality != "" {
			err := s.manager.DefineAgent(req.Name, req.Personality, req.Tools, req.Model)
			if err != nil {
				return fmt.Sprintf("Error defining agent during spawn: %v", err), nil
			}
			if req.Prompt == "" {
				return fmt.Sprintf("Agent '%s' defined. What should I task it with?", req.Name), nil
			}
			id, err := s.manager.SpawnNamed(ctx, req.Name, req.Prompt, req.Images)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Spawned specialized sub-agent '%s' with ID: %s. You MUST wait for it to complete or check /subagent list for progress.", req.Name, id), nil
		}

		if req.Prompt == "" {
			return "Error: 'prompt' is required for spawn action.", nil
		}
		id, err := s.manager.Spawn(ctx, req.Prompt, req.Images)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Spawned generic sub-agent with ID: %s. You MUST wait for it to complete or check /subagent list for progress.", id), nil
	case "spawn_named":
		if req.Name == "" {
			return "Error: 'name' is required for spawn_named action.", nil
		}
		// If personality is provided, it's a "define and spawn" request
		if req.Personality != "" {
			err := s.manager.DefineAgent(req.Name, req.Personality, req.Tools, req.Model)
			if err != nil {
				return fmt.Sprintf("Error defining agent during spawn: %v", err), nil
			}
		}
		
		if req.Prompt == "" {
			return fmt.Sprintf("Agent '%s' defined/verified. What should I task it with?", req.Name), nil
		}

		id, err := s.manager.SpawnNamed(ctx, req.Name, req.Prompt, req.Images)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Spawned specialized sub-agent '%s' with ID: %s. You MUST wait for it to complete or check /subagent list for progress.", req.Name, id), nil
	case "define":
		if req.Name == "" || req.Personality == "" {
			return "Error: Both 'name' and 'personality' are required to define an agent.", nil
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
			res += fmt.Sprintf("[%s] %s: %s (Model: %s)\n", t.ID, t.Status, t.Prompt, t.Model)
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
			return "Error: 'id' is required for result action. Use /subagent list to find the ID.", nil
		}
		fmt.Printf("[SubAgentTool]: Checking result for ID %s\n", req.ID)
		tasks, err := s.manager.List()
		if err != nil {
			return "", err
		}
		for _, t := range tasks {
			if t.ID == req.ID {
				if t.Status == "running" {
					return fmt.Sprintf("Sub-agent %s is still running. Progress: %d%%. Please wait.", req.ID, t.Progress), nil
				}
				return fmt.Sprintf("Sub-agent %s result: %s", req.ID, t.Result), nil
			}
		}
		return fmt.Sprintf("Error: Sub-agent with ID %s not found.", req.ID), nil
	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}

func (s *SubAgentTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Sub-Agent Manager",
		"actions": []map[string]interface{}{
			{
				"name":  "spawn",
				"label": "Spawn Agent",
				"fields": []map[string]interface{}{
					{"name": "prompt", "label": "Task Prompt", "type": "longtext", "required": true},
					{"name": "images", "label": "Attach Images", "type": "image_list"},
				},
			},
			{
				"name":  "spawn_named",
				"label": "Spawn Specialized Agent",
				"fields": []map[string]interface{}{
					{"name": "name", "label": "Agent Name", "type": "string", "required": true},
					{"name": "prompt", "label": "Task Prompt", "type": "longtext", "required": true},
				},
			},
			{
				"name":  "define",
				"label": "Define New Agent",
				"fields": []map[string]interface{}{
					{"name": "name", "label": "Name", "type": "string", "required": true},
					{"name": "personality", "label": "Personality", "type": "longtext", "required": true},
					{"name": "tools", "label": "Tools (comma-separated)", "type": "string", "hint": "time,email,shell"},
					{"name": "model", "label": "Model Override", "type": "string", "hint": "llama3.1"},
				},
			},
			{
				"name":  "result",
				"label": "Get Result",
				"fields": []map[string]interface{}{
					{"name": "id", "label": "Task ID", "type": "string", "required": true},
				},
			},
			{
				"name":  "list",
				"label": "List All Tasks",
				"fields": []map[string]interface{}{},
			},
		},
	}
}
