package base

import "context"

// Tool defines the interface that all bot capabilities must implement.
type Tool interface {
	// Name returns the unique identifier for the tool.
	Name() string
	// Description returns a clear explanation of what the tool does and its expected input.
	Description() string
	// Execute performs the tool's action and returns the result as a string.
	Execute(ctx context.Context, input string) (string, error)
}
