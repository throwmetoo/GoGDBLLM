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

// OpenRouterClient implements the Client interface for OpenRouter
type OpenRouterClient struct {
	logger   *log.Logger
	settings config.LLMSettings
	client   *http.Client
}

// NewOpenRouterClient creates a new OpenRouter client
func NewOpenRouterClient(logger *log.Logger, settings config.LLMSettings) *OpenRouterClient {
	return &OpenRouterClient{
		logger:   logger,
		settings: settings,
		client:   &http.Client{},
	}
}

// ProcessRequest processes a chat request and returns a response
func (c *OpenRouterClient) ProcessRequest(ctx context.Context, req ChatRequest) (string, error) {
	// Convert chat history to OpenRouter format (similar to OpenAI)
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: "You are an AI assistant that helps with programming and debugging. Provide clear explanations and code examples when needed.",
		},
	}

	// Add chat history
	for _, msg := range req.History {
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		messages = append(messages, OpenAIMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add the current message
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: req.Message,
	})

	// Create request
	apiReq := OpenAIRequest{
		Model:    c.settings.Model,
		Messages: messages,
	}

	// Convert to JSON
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.settings.APIKey)
	httpReq.Header.Set("HTTP-Referer", "https://gogdbllm.app") // Replace with your actual domain
	httpReq.Header.Set("X-Title", "GoGDBLLM")
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
	var apiResp OpenAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract text from response
	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("empty response from API")
}

// TestConnection tests the connection to OpenRouter
func (c *OpenRouterClient) TestConnection(ctx context.Context, settings config.LLMSettings) error {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader([]byte(`{
		"model": "`+settings.Model+`",
		"messages": [{"role": "user", "content": "test"}]
	}`)))

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("HTTP-Referer", "https://gogdbllm.app")
	req.Header.Set("X-Title", "GoGDBLLM")

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
