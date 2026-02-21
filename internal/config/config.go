package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Config holds all application settings in a modular map.
type Config struct {
	settings map[string]string
	mu       sync.RWMutex
}

// LoadConfig reads a simple KEY=VALUE text file into a dynamic map.
func LoadConfig(filePath string) (*Config, error) {
	conf := &Config{
		settings: make(map[string]string),
	}

	err := conf.Reload(filePath)
	return conf, err
}

// Reload re-reads the configuration file and updates the in-memory settings.
func (c *Config) Reload(filePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set hardcoded defaults here if necessary, or let tools handle their own defaults
	c.settings["MODEL"] = "llama3.1"
	c.settings["OLLAMA_URL"] = "http://localhost:11434"
	c.settings["SWARMUI_PATH"] = "/home/pyromancer/swarmconnector/swarmui"
	c.settings["SWARMUI_URL"] = "http://localhost:7801"
	c.settings["SWARMUI_DEFAULT_MODEL"] = "v1-5-pruned-emaonly.safetensors"

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		c.settings[key] = val
	}

	return scanner.Err()
}

// Get returns the value for a key, or an empty string if not found.
func (c *Config) Get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.settings[key]
}

// GetWithDefault returns the value for a key, or the provided default if not found.
func (c *Config) GetWithDefault(key, defaultValue string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if val, ok := c.settings[key]; ok {
		return val
	}
	return defaultValue
}

// AllSettings returns a copy of all current settings.
func (c *Config) AllSettings() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	copy := make(map[string]string)
	for k, v := range c.settings {
		copy[k] = v
	}
	return copy
}

// Set updates a setting in memory.
func (c *Config) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.settings[key] = value
}

// SaveToFile persists the current in-memory settings back to the config file.
func (c *Config) SaveToFile(filePath string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var lines []string
	for k, v := range c.settings {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}
	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}
