package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Minimal MCP Client implementation

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	id     int
	mu     sync.Mutex
}

func NewClient(command string, args []string) (*Client, error) {
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
		id:     1,
	}, nil
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Call(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	id := c.id
	c.id++
	c.mu.Unlock()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Write request
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	// Read response (assume one line per response for simple stdio MCP, 
	// though spec allows headers. We assume simple Line-delimited JSON-RPC for now as per some MCP implementations, 
	// but official MCP uses JSON-RPC over stdio which might be robust. 
	// We'll read until we find a matching ID.)
	
	for c.stdout.Scan() {
		line := c.stdout.Bytes()
		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue // Skip logs or invalid json
		}
		if resp.ID == id {
			if resp.Error != nil {
				return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
			}
			return resp.Result, nil
		}
	}

	return nil, fmt.Errorf("connection closed")
}

func (c *Client) Initialize() error {
	_, err := c.Call("initialize", map[string]interface{}{
		"protocolVersion": "0.1.0",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "Idony",
			"version": "1.0.0",
		},
	})
	return err
}

type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

func (c *Client) ListTools() ([]MCPTool, error) {
	res, err := c.Call("tools/list", nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Tools []MCPTool `json:"tools"`
	}
	if err := json.Unmarshal(res, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func (c *Client) CallTool(name string, args map[string]interface{}) (string, error) {
	res, err := c.Call("tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}
	
	// MCP returns { content: [{type: "text", text: "..."}] }
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(res, &result); err != nil {
		return string(res), nil // Fallback
	}
	
	var sb string
	for _, c := range result.Content {
		if c.Type == "text" {
			sb += c.Text
		}
	}
	return sb, nil
}
