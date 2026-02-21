package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pyromancer/idony/internal/db"
)

type PlannerStore interface {
	SaveProject(p db.Project) error
	GetProjects() ([]db.Project, error)
	SaveTask(t db.Task) error
	GetTasks(projectID string) ([]db.Task, error)
}

// PlannerTool allows Idony to manage project plans.
type PlannerTool struct {
	store PlannerStore
}

func NewPlannerTool(s PlannerStore) *PlannerTool {
	return &PlannerTool{store: s}
}

func (p *PlannerTool) Name() string {
	return "planner"
}

func (p *PlannerTool) Description() string {
	return `Manages project plans. Actions: "create_project", "add_task", "list_projects", "list_tasks".
JSON Input: {"action": "create_project|add_task|list_projects|list_tasks", "project_id": "uuid", "parent_id": "optional_task_id", "name": "project name", "title": "task title", "description": "details"}`
}

func (p *PlannerTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action      string `json:"action"`
		ProjectID   string `json:"project_id"`
		ParentID    string `json:"parent_id"`
		Name        string `json:"name"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	switch req.Action {
	case "create_project":
		id := uuid.New().String()[:8]
		proj := db.Project{
			ID:          id,
			Name:        req.Name,
			Description: req.Description,
			Status:      "planning",
		}
		if err := p.store.SaveProject(proj); err != nil {
			return "", err
		}
		return fmt.Sprintf("Project created with ID: %s", id), nil

	case "add_task":
		if req.ProjectID == "" {
			return "", fmt.Errorf("project_id is required")
		}
		id := uuid.New().String()[:8]
		task := db.Task{
			ID:          id,
			ProjectID:   req.ProjectID,
			ParentID:    req.ParentID,
			Title:       req.Title,
			Description: req.Description,
			Status:      "pending",
		}
		if err := p.store.SaveTask(task); err != nil {
			return "", err
		}
		return fmt.Sprintf("Task added with ID: %s", id), nil

	case "list_projects":
		projs, err := p.store.GetProjects()
		if err != nil {
			return "", err
		}
		var res string
		for _, pr := range projs {
			res += fmt.Sprintf("- [%s] %s: %s (%s)\n", pr.ID, pr.Name, pr.Description, pr.Status)
		}
		if res == "" { return "No projects found.", nil }
		return res, nil

	case "list_tasks":
		if req.ProjectID == "" {
			return "", fmt.Errorf("project_id is required")
		}
		tasks, err := p.store.GetTasks(req.ProjectID)
		if err != nil {
			return "", err
		}
		var res string
		for _, t := range tasks {
			prefix := ""
			if t.ParentID != "" { prefix = "  └─ " }
			res += fmt.Sprintf("%s[%s] %s: %s\n", prefix, t.ID, t.Title, t.Status)
		}
		if res == "" { return "No tasks found for this project.", nil }
		return res, nil

	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}

func (p *PlannerTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Project Planner",
		"actions": []map[string]interface{}{
			{
				"name":  "create_project",
				"label": "New Project",
				"fields": []map[string]interface{}{
					{"name": "name", "label": "Project Name", "type": "string", "required": true},
					{"name": "description", "label": "Description", "type": "longtext"},
				},
			},
			{
				"name":  "add_task",
				"label": "Add Task",
				"fields": []map[string]interface{}{
					{"name": "project_id", "label": "Project ID", "type": "string", "required": true},
					{"name": "parent_id", "label": "Parent Task ID (Optional)", "type": "string"},
					{"name": "title", "label": "Task Title", "type": "string", "required": true},
					{"name": "description", "label": "Task Details", "type": "longtext"},
				},
			},
			{
				"name":  "list_projects",
				"label": "List All Projects",
				"fields": []map[string]interface{}{},
			},
			{
				"name":  "list_tasks",
				"label": "List Project Tasks",
				"fields": []map[string]interface{}{
					{"name": "project_id", "label": "Project ID", "type": "string", "required": true},
				},
			},
		},
	}
}
