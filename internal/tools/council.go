package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pyromancer/idony/internal/db"
)

type CouncilInteractionManager interface {
	RunCouncilSession(ctx context.Context, councilName, problem string) (string, error)
	DefineCouncil(name string, members []string) error
	ListCouncils() ([]db.Council, error)
}

// CouncilTool allows Idony to manage councils of sub-agents.
type CouncilTool struct {
	manager CouncilInteractionManager
}

func NewCouncilTool(m CouncilInteractionManager) *CouncilTool {
	return &CouncilTool{manager: m}
}

func (c *CouncilTool) Name() string {
	return "council"
}

func (c *CouncilTool) Description() string {
	return `Manages agent councils. Input must be a JSON object: 
{"action": "define|run|list", "name": "council_name", "members": ["member1", "member2"], "problem": "the problem for the council to solve"}`
}

func (c *CouncilTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action  string   `json:"action"`
		Name    string   `json:"name"`
		Members []string `json:"members"`
		Problem string   `json:"problem"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	switch req.Action {
	case "define":
		if req.Name == "" || len(req.Members) == 0 {
			return "", fmt.Errorf("name and members are required for define")
		}
		err := c.manager.DefineCouncil(req.Name, req.Members)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Successfully defined council: %s", req.Name), nil
	case "run":
		if req.Name == "" || req.Problem == "" {
			return "", fmt.Errorf("name and problem are required for run")
		}
		id, err := c.manager.RunCouncilSession(ctx, req.Name, req.Problem)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Started council session for '%s' with ID: %s", req.Name, id), nil
	case "list":
		councils, err := c.manager.ListCouncils()
		if err != nil {
			return "", err
		}
		var res string
		for _, cn := range councils {
			res += fmt.Sprintf("- %s: Members (%s)\n", cn.Name, cn.Members)
		}
		if res == "" {
			return "No councils defined yet.", nil
		}
		return res, nil
	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}
