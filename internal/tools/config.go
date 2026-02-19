package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"github.com/pyromancer/idony/internal/db"
)

// ConfigUpdateTool allows Idony to update its own config.txt
type ConfigUpdateTool struct {
	configPath string
}

func NewConfigUpdateTool(path string) *ConfigUpdateTool {
	return &ConfigUpdateTool{configPath: path}
}

func (c *ConfigUpdateTool) Name() string {
	return "update_config"
}

func (c *ConfigUpdateTool) Description() string {
	return "Updates the configuration file. Input should be the key and new value, e.g., 'MODEL=llama3.1'."
}

func (c *ConfigUpdateTool) Execute(ctx context.Context, input string) (string, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid input format, expected KEY=VALUE")
	}

	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])

	content, err := os.ReadFile(c.configPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, val)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, val))
	}

	err = os.WriteFile(c.configPath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully updated config: %s=%s", key, val), nil
}

// PersonalityTool allows Idony to update its own personality in the DB
type PersonalityTool struct {
	store *db.Store
}

func NewPersonalityTool(store *db.Store) *PersonalityTool {
	return &PersonalityTool{store: store}
}

func (p *PersonalityTool) Name() string {
	return "update_personality"
}

func (p *PersonalityTool) Description() string {
	return "Updates the bot's personality or persona based on user instructions. Input should be the new detailed persona description."
}

func (p *PersonalityTool) Execute(ctx context.Context, input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("personality description cannot be empty")
	}

	err := p.store.SetSetting("personality", input)
	if err != nil {
		return "", err
	}

	return "Personality updated successfully.", nil
}
