package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pyromancer/idony/internal/db"
)

type SchedulerStore interface {
	SaveScheduledTask(taskType, schedule, prompt, targetType, targetName string) error
	LoadScheduledTasks() ([]db.ScheduledTask, error)
	DeleteTask(id int) error
}

// ScheduleTool allows Idony to schedule future prompts.
type ScheduleTool struct {
	store SchedulerStore
}

func NewScheduleTool(store SchedulerStore) *ScheduleTool {
	return &ScheduleTool{store: store}
}

func (s *ScheduleTool) Name() string {
	return "schedule_task"
}

func (s *ScheduleTool) Description() string {
	return `Schedules tasks. Actions: add, list, delete.
Input: {"action": "add|list|delete", "type": "one-shot|recurring", "schedule": "...", "prompt": "...", "id": "123"}`
}

func (s *ScheduleTool) Execute(ctx context.Context, input string) (string, error) {
	var req struct {
		Action     string `json:"action"`
		Type       string `json:"type"`
		Schedule   string `json:"schedule"`
		Prompt     string `json:"prompt"`
		TargetType string `json:"target_type"`
		TargetName string `json:"target_name"`
		ID         string `json:"id"`
	}

	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	if req.Action == "" {
		req.Action = "add" // Default behavior
	}

	switch req.Action {
	case "add":
		if req.Type != "one-shot" && req.Type != "recurring" {
			return "", fmt.Errorf("invalid task type: %s", req.Type)
		}
		if req.Type == "one-shot" {
			_, err := time.Parse(time.RFC3339, req.Schedule)
			if err != nil {
				return "", fmt.Errorf("invalid schedule format for one-shot (RFC3339 expected): %w", err)
			}
		}
		if req.TargetType == "" { req.TargetType = "main" }

		err := s.store.SaveScheduledTask(req.Type, req.Schedule, req.Prompt, req.TargetType, req.TargetName)
		if err != nil {
			return "", fmt.Errorf("failed to save task: %w", err)
		}
		return fmt.Sprintf("Scheduled %s task: %s", req.Type, req.Prompt), nil

	case "list":
		tasks, err := s.store.LoadScheduledTasks()
		if err != nil {
			return "", err
		}
		if len(tasks) == 0 {
			return "No scheduled tasks.", nil
		}
		var sb strings.Builder
		sb.WriteString("Scheduled Tasks:\n")
		for _, t := range tasks {
			sb.WriteString(fmt.Sprintf("[%d] %s | %s | %s\n", t.ID, t.Type, t.Schedule, t.Prompt))
		}
		return sb.String(), nil

	case "delete":
		id, err := strconv.Atoi(req.ID)
		if err != nil {
			return "", fmt.Errorf("invalid ID: %s", req.ID)
		}
		if err := s.store.DeleteTask(id); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted task %d", id), nil

	default:
		return "", fmt.Errorf("invalid action: %s", req.Action)
	}
}

func (s *ScheduleTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Task Scheduler",
		"actions": []map[string]interface{}{
			{
				"name": "add",
				"label": "Schedule Task",
				"fields": []map[string]interface{}{
					{"name": "type", "label": "Type", "type": "choice", "options": []string{"one-shot", "recurring"}},
					{"name": "schedule", "label": "Schedule", "type": "string", "hint": "RFC3339 or Cron"},
					{"name": "prompt", "label": "Prompt", "type": "string"},
					{"name": "target_type", "label": "Target", "type": "choice", "options": []string{"main", "subagent", "council"}},
					{"name": "target_name", "label": "Target Name", "type": "string"},
				},
			},
			{
				"name": "list",
				"label": "List Tasks",
				"fields": []map[string]interface{}{},
			},
			{
				"name": "delete",
				"label": "Delete Task",
				"fields": []map[string]interface{}{
					{"name": "id", "label": "Task ID", "type": "string"},
				},
			},
		},
	}
}
