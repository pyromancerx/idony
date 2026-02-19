package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Message represents a single message in the conversation history
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents an Ollama generation request
type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Response represents a response from Ollama
type Response struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

// OllamaClient is a basic client for the Ollama HTTP API
type OllamaClient struct {
	BaseURL string
	HTTP    *http.Client
	Model   string
}

// NewOllamaClient creates a new instance of OllamaClient
func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		BaseURL: baseURL,
		HTTP:    &http.Client{},
		Model:   model,
	}
}

// SetModel updates the model used for generations
func (c *OllamaClient) SetModel(model string) {
	c.Model = model
}

// GenerateResponse sends a conversation history to Ollama and returns the assistant's response
func (c *OllamaClient) GenerateResponse(ctx context.Context, messages []Message) (string, error) {
	reqBody := Request{
		Model:    c.Model,
		Messages: messages,
		Stream:   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp Response
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}
