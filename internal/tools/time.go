package tools

import (
	"context"
	"time"
)

// TimeTool is a simple tool that returns the current system time.
type TimeTool struct{}

func (t *TimeTool) Name() string {
	return "get_time"
}

func (t *TimeTool) Description() string {
	return "Returns the current system time. Input is ignored."
}

func (t *TimeTool) Execute(ctx context.Context, input string) (string, error) {
	return time.Now().Format(time.RFC1123), nil
}

func (t *TimeTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title":  "Get Time",
		"fields": []map[string]interface{}{},
	}
}
