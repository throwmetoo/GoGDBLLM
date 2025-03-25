package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/gogdbllm/internal/settings"
)

// TestConnection tests the connection to the specified API
func TestConnection(settings settings.Settings) (bool, string) {
	// Set a timeout for the test
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	switch settings.Provider {
	case "anthropic":
		return testAnthropicConnection(client, settings)
	case "openai":
		return testOpenAIConnection(client, settings)
	case "openrouter":
		return testOpenRouterConnection(client, settings)
	default:
		return false, fmt.Sprintf("Unsupported provider: %s", settings.Provider)
	}
}

// testAnthropicConnection tests the connection to the Anthropic API
func testAnthropicConnection(client *http.Client, settings settings.Settings) (bool, string) {
	// Create a minimal request
	apiReq := AnthropicRequest{
		Model: settings.Model,
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: "Hello! This is a connection test.",
			},
		},
		MaxTokens: 10,
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("Failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	return true, "Connection to Anthropic API successful"
}

// testOpenAIConnection tests the connection to the OpenAI API
func testOpenAIConnection(client *http.Client, settings settings.Settings) (bool, string) {
	// Create a minimal request
	apiReq := OpenAIRequest{
		Model: settings.Model,
		Messages: []OpenAIMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Hello! This is a connection test.",
			},
		},
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("Failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	return true, "Connection to OpenAI API successful"
}

// testOpenRouterConnection tests the connection to the OpenRouter API
func testOpenRouterConnection(client *http.Client, settings settings.Settings) (bool, string) {
	// Create a minimal request
	apiReq := OpenRouterRequest{
		Model: settings.Model,
		Messages: []OpenRouterMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Hello! This is a connection test.",
			},
		},
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return false, fmt.Sprintf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/yourusername/gogdbllm")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Connection failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("Failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	return true, "Connection to OpenRouter API successful"
}
