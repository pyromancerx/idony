package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pyromancer/idony/internal/config"
	"github.com/pyromancer/idony/internal/mcp"
	"github.com/pyromancer/idony/internal/tools/base"
)

type MCPManager struct {
	clients []*mcp.Client
}

func NewMCPManager() *MCPManager {
	return &MCPManager{}
}

func (m *MCPManager) LoadFromConfig(conf *config.Config) ([]base.Tool, error) {
	settings := conf.AllSettings()
	serverConfigs := make(map[string]*struct {
		Command string
		Args    []string
	})

	// Group settings by server name
	for k, v := range settings {
		if strings.HasPrefix(k, "MCP_SERVER_") {
			parts := strings.Split(k, "_")
			if len(parts) < 4 {
				continue
			}
			// Format: MCP_SERVER_[NAME]_CMD or MCP_SERVER_[NAME]_ARGS
			serverName := strings.Join(parts[2:len(parts)-1], "_")
			suffix := parts[len(parts)-1]

			if _, ok := serverConfigs[serverName]; !ok {
				serverConfigs[serverName] = &struct {
					Command string
					Args    []string
				}{}
			}

			if suffix == "CMD" {
				serverConfigs[serverName].Command = v
			} else if suffix == "ARGS" {
				// Split by space, handle basic quoted strings if needed? 
				// For now, simple space split.
				serverConfigs[serverName].Args = strings.Fields(v)
			}
		}
	}

	var tools []base.Tool
	for name, cfg := range serverConfigs {
		if cfg.Command == "" {
			continue
		}

		client, err := mcp.NewClient(cfg.Command, cfg.Args)
		if err != nil {
			fmt.Printf("Failed to start MCP server %s: %v\n", name, err)
			continue
		}
		m.clients = append(m.clients, client)

		if err := client.Initialize(); err != nil {
			fmt.Printf("Failed to initialize MCP server %s: %v\n", name, err)
			continue
		}

		mcpTools, err := client.ListTools()
		if err != nil {
			fmt.Printf("Failed to list tools for %s: %v\n", name, err)
			continue
		}

		for _, mt := range mcpTools {
			tools = append(tools, &MCPToolWrapper{
				client: client,
				tool:   mt,
				source: name,
			})
		}
	}

	return tools, nil
}

func (m *MCPManager) LoadServers(configPath string) ([]base.Tool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config, no tools
		}
		return nil, err
	}

	var config map[string]struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var tools []base.Tool

	for name, cfg := range config {
		client, err := mcp.NewClient(cfg.Command, cfg.Args)
		if err != nil {
			fmt.Printf("Failed to start MCP server %s: %v\n", name, err)
			continue
		}
		m.clients = append(m.clients, client)

		if err := client.Initialize(); err != nil {
			fmt.Printf("Failed to initialize MCP server %s: %v\n", name, err)
			continue
		}

		mcpTools, err := client.ListTools()
		if err != nil {
			fmt.Printf("Failed to list tools for %s: %v\n", name, err)
			continue
		}

		for _, mt := range mcpTools {
			tools = append(tools, &MCPToolWrapper{
				client: client,
				tool:   mt,
				source: name,
			})
		}
	}

	return tools, nil
}

type MCPToolWrapper struct {
	client *mcp.Client
	tool   mcp.MCPTool
	source string
}

func (w *MCPToolWrapper) Name() string {
	return w.tool.Name
}

func (w *MCPToolWrapper) Description() string {
	return fmt.Sprintf("[%s] %s", w.source, w.tool.Description)
}

func (w *MCPToolWrapper) Execute(ctx context.Context, input string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("invalid JSON input for MCP tool: %w", err)
	}
	return w.client.CallTool(w.tool.Name, args)
}

func (w *MCPToolWrapper) Schema() map[string]interface{} {
	// Convert MCP inputSchema (JSON Schema) to our UI Schema
	// MCP schema is standard JSON Schema. Our UI schema is custom.
	// We'll do a best-effort conversion.
	
	fields := []map[string]interface{}{}
	if props, ok := w.tool.InputSchema["properties"].(map[string]interface{}); ok {
		for k, v := range props {
			def := v.(map[string]interface{})
			f := map[string]interface{}{
				"name": k,
				"label": k,
				"type": "string", // Default
			}
			if t, ok := def["type"].(string); ok {
				if t == "integer" || t == "number" {
					f["type"] = "string" // UI treats numbers as strings for input
					f["hint"] = "number"
				}
			}
			if desc, ok := def["description"].(string); ok {
				f["hint"] = desc
			}
			fields = append(fields, f)
		}
	}

	return map[string]interface{}{
		"title": w.tool.Name,
		"fields": fields,
	}
}
