package tools

import (
	"context"
	"fmt"
	"os/exec"
)

// GeminiCoder is a tool that interfaces with the gemini-cli for coding tasks.
type GeminiCoder struct{}

func (g *GeminiCoder) Name() string {
	return "gemini_coder"
}

func (g *GeminiCoder) Description() string {
	return "Executes coding tasks using the Gemini CLI. Input should be a clear description of the code change or creation needed."
}

func (g *GeminiCoder) Execute(ctx context.Context, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("coding task input cannot be empty")
	}

	// Execute gemini-cli
	// We assume 'gemini' is in the PATH.
	cmd := exec.CommandContext(ctx, "gemini", input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing Gemini CLI: %v\nOutput: %s", err, string(output)), nil
	}

	return string(output), nil
}
