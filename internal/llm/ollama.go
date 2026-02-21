package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Message represents a single message in the conversation history
type Message struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // Base64 encoded images
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
		HTTP:    &http.Client{Timeout: 120 * time.Second},
		Model:   model,
	}
}

// SetModel updates the model used for generations
func (c *OllamaClient) SetModel(model string) {
	c.Model = model
}

// ListModels retrieves the available models from the Ollama server
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var data struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var names []string
	for _, m := range data.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

// EncodeImage converts a local file to a base64 string
func EncodeImage(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
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

	maxRetries := 2
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			fmt.Printf("[OllamaClient]: Retry %d after error: %v\n", i, lastErr)
			time.Sleep(time.Second * time.Duration(i))
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/chat", bytes.NewBuffer(jsonData))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			if strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "timeout") {
				continue
			}
			return "", fmt.Errorf("failed to execute request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
			continue
		}

		var ollamaResp Response
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			lastErr = fmt.Errorf("failed to decode response: %w", err)
			continue
		}

		return ollamaResp.Message.Content, nil
	}

	return "", fmt.Errorf("failed after %d retries: %v", maxRetries, lastErr)
}
