package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
	"github.com/pyromancer/idony/internal/config"
	"github.com/pyromancer/idony/internal/db"
)

// Refreshable interface defines things that can pick up config changes.
type Refreshable interface {
	SetModel(string)
	SetBaseURL(string)
}

// ConfigUpdateTool allows Idony to update its own config.txt
type ConfigUpdateTool struct {
	conf       *config.Config
	configPath string
	agent      Refreshable
}

func NewConfigUpdateTool(conf *config.Config, path string, a Refreshable) *ConfigUpdateTool {
	return &ConfigUpdateTool{conf: conf, configPath: path, agent: a}
}

func (c *ConfigUpdateTool) Name() string {
	return "update_config"
}

func (c *ConfigUpdateTool) Description() string {
	return "Updates the configuration file and in-memory settings. Input should be the key and new value, e.g., 'MODEL=llama3.1'."
}

func (c *ConfigUpdateTool) Execute(ctx context.Context, input string) (string, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid input format, expected KEY=VALUE")
	}

	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])

	// Update in-memory
	if c.conf != nil {
		c.conf.Set(key, val)
	}

	// Refresh agent if it's a key the agent cares about
	if c.agent != nil {
		if key == "MODEL" {
			c.agent.SetModel(val)
		} else if key == "OLLAMA_URL" {
			c.agent.SetBaseURL(val)
		}
	}

	// Update file
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

	return fmt.Sprintf("Successfully updated config and refreshed agent: %s=%s", key, val), nil
}

func (c *ConfigUpdateTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Update Configuration",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Setting (KEY=VALUE)", "type": "string", "hint": "MODEL=llama3.1", "required": true},
		},
	}
}

// ReloadConfigTool allows Idony to reload its config.txt from disk
type ReloadConfigTool struct {
	conf       *config.Config
	configPath string
	agent      Refreshable
}

func NewReloadConfigTool(conf *config.Config, path string, a Refreshable) *ReloadConfigTool {
	return &ReloadConfigTool{conf: conf, configPath: path, agent: a}
}

func (c *ReloadConfigTool) Name() string {
	return "reload_config"
}

func (c *ReloadConfigTool) Description() string {
	return "Reloads the configuration file from disk into memory."
}

func (c *ReloadConfigTool) Execute(ctx context.Context, input string) (string, error) {
	err := c.conf.Reload(c.configPath)
	if err != nil {
		return "", fmt.Errorf("failed to reload config: %v", err)
	}

	// Refresh agent from new config
	if c.agent != nil {
		c.agent.SetModel(c.conf.GetWithDefault("MODEL", "llama3.1"))
		c.agent.SetBaseURL(c.conf.GetWithDefault("OLLAMA_URL", "http://localhost:11434"))
	}

	return "Successfully reloaded configuration and refreshed agent from " + c.configPath, nil
}

func (c *ReloadConfigTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title":  "Reload Config from Disk",
		"fields": []map[string]interface{}{},
	}
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
	return "Updates the bot's personality or persona based on user instructions. Input should be the new detailed personality description."
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

func (p *PersonalityTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"title": "Update Personality",
		"fields": []map[string]interface{}{
			{"name": "input", "label": "Personality Description", "type": "longtext", "required": true},
		},
	}
}
