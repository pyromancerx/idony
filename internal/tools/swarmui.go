package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// SwarmUITool is a tool that interfaces with the SwarmUI CLI for image generation.
type SwarmUITool struct {
	path         string
	url          string
	defaultModel string
}

func NewSwarmUITool(path, url, model string) *SwarmUITool {
	return &SwarmUITool{
		path:         path,
		url:          url,
		defaultModel: model,
	}
}

func (s *SwarmUITool) Name() string {
	return "generate_image"
}

func (s *SwarmUITool) Description() string {
	return `Generates an image based on a text prompt using SwarmUI. 
Input must be a JSON object: {"prompt": "description of the image", "model": "optional_model_name", "resolution": "optional_resolution (e.g., 512x512)"}`
}

func (s *SwarmUITool) Execute(ctx context.Context, input string) (string, error) {
	var params struct {
		Prompt     string `json:"prompt"`
		Model      string `json:"model"`
		Resolution string `json:"resolution"`
	}

	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return "", fmt.Errorf("invalid input format, expected JSON: %w", err)
	}

	if params.Prompt == "" {
		return "", fmt.Errorf("prompt is required for image generation")
	}

	model := params.Model
	if model == "" {
		model = s.defaultModel
	}

	args := []string{"generate", "--prompt", params.Prompt, "--model", model, "--url", s.url}
	if params.Resolution != "" {
		args = append(args, "--resolution", params.Resolution)
	}

	// Create command to run swarmui CLI
	cmd := exec.CommandContext(ctx, s.path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error executing SwarmUI CLI: %v\nOutput: %s", err, string(output)), nil
	}

	return string(output), nil
}

func (s *SwarmUITool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Image Generation (SwarmUI)",
		"fields": []map[string]interface{}{
			{"name": "prompt", "label": "Image Prompt", "type": "longtext", "required": true},
			{"name": "model", "label": "Model Name", "type": "string", "hint": "v1-5-pruned-emaonly.safetensors"},
			{"name": "resolution", "label": "Resolution", "type": "string", "hint": "512x512"},
		},
	}
}
