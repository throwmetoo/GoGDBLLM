package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/throwmetoo/GoGDBLLM/internal/config"
)

// AnthropicClient implements the Client interface for Anthropic
type AnthropicClient struct {
	logger   *log.Logger
	settings config.LLMSettings
	client   *http.Client
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(logger *log.Logger, settings config.LLMSettings) *AnthropicClient {
	return &AnthropicClient{
		logger:   logger,
		settings: settings,
		client:   &http.Client{},
	}
}

// ProcessRequest processes a chat request and returns a response
func (c *AnthropicClient) ProcessRequest(ctx context.Context, req ChatRequest) (string, error) {
	// Convert chat history to Anthropic format
	messages := []AnthropicMessage{}

	// Add chat history
	for _, msg := range req.History {
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		messages = append(messages, AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add the current message
	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: req.Message,
	})

	// Create request
	apiReq := AnthropicRequest{
		Model:     c.settings.Model,
		MaxTokens: 4000,
		Messages:  messages,
	}

	// Convert to JSON
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.settings.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	// Parse response
	var apiResp AnthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract text from response
	if len(apiResp.Content) > 0 {
		return apiResp.Content[0].Text, nil
	}

	return "", fmt.Errorf("empty response from API")
}

// TestConnection tests the connection to Anthropic
func (c *AnthropicClient) TestConnection(ctx context.Context, settings config.LLMSettings) error {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader([]byte(`{
		"model": "`+settings.Model+`",
		"max_tokens": 1,
		"messages": [{"role": "user", "content": "test"}]
	}`)))

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", body)
	}

	return nil
}
