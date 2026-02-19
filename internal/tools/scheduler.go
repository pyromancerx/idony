package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type SchedulerStore interface {
	SaveTask(taskType, schedule, prompt string) error
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
	return `Schedules a task. Input must be a JSON object: {"type": "one-shot|recurring", "schedule": "RFC3339|cron", "prompt": "the prompt Idony should run"}`
}

func (s *ScheduleTool) Execute(ctx context.Context, input string) (string, error) {
	var task struct {
		Type     string `json:"type"`
		Schedule string `json:"schedule"`
		Prompt   string `json:"prompt"`
	}

	if err := json.Unmarshal([]byte(input), &task); err != nil {
		return "", fmt.Errorf("invalid input format: %w", err)
	}

	if task.Type != "one-shot" && task.Type != "recurring" {
		return "", fmt.Errorf("invalid task type: %s", task.Type)
	}

	if task.Type == "one-shot" {
		_, err := time.Parse(time.RFC3339, task.Schedule)
		if err != nil {
			return "", fmt.Errorf("invalid schedule format for one-shot (RFC3339 expected): %w", err)
		}
	}

	err := s.store.SaveTask(task.Type, task.Schedule, task.Prompt)
	if err != nil {
		return "", fmt.Errorf("failed to save task to db: %w", err)
	}

	return fmt.Sprintf("Successfully scheduled %s task for %s", task.Type, task.Schedule), nil
}
