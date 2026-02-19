package config

import (
	"bufio"
	"os"
	"strings"
)

// Config holds all application settings in a modular map.
type Config struct {
	settings map[string]string
}

// LoadConfig reads a simple KEY=VALUE text file into a dynamic map.
func LoadConfig(filePath string) (*Config, error) {
	conf := &Config{
		settings: make(map[string]string),
	}

	// Set hardcoded defaults here if necessary, or let tools handle their own defaults
	conf.settings["MODEL"] = "llama3.1"
	conf.settings["OLLAMA_URL"] = "http://localhost:11434"
	conf.settings["SWARMUI_PATH"] = "/home/pyromancer/swarmconnector/swarmui"
	conf.settings["SWARMUI_URL"] = "http://localhost:7801"
	conf.settings["SWARMUI_DEFAULT_MODEL"] = "v1-5-pruned-emaonly.safetensors"

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return conf, nil
		}
		return nil, err
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
		conf.settings[key] = val
	}

	return conf, scanner.Err()
}

// Get returns the value for a key, or an empty string if not found.
func (c *Config) Get(key string) string {
	return c.settings[key]
}

// GetWithDefault returns the value for a key, or the provided default if not found.
func (c *Config) GetWithDefault(key, defaultValue string) string {
	if val, ok := c.settings[key]; ok {
		return val
	}
	return defaultValue
}

// AllSettings returns a copy of all current settings.
func (c *Config) AllSettings() map[string]string {
	copy := make(map[string]string)
	for k, v := range c.settings {
		copy[k] = v
	}
	return copy
}
