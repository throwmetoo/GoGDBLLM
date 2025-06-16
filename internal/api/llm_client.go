package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
)

// LLMClient handles communication with LLM providers
type LLMClient struct {
	settingsManager *settings.Manager
	httpClient      *http.Client
}

// NewLLMClient creates a new LLM client
func NewLLMClient(settingsManager *settings.Manager) *LLMClient {
	return &LLMClient{
		settingsManager: settingsManager,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SendRequest sends a request to the configured LLM provider
func (lc *LLMClient) SendRequest(ctx context.Context, req *ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) (string, error) {
	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== LLM REQUEST ===\nProvider: %s\nModel: %s\nMessage length: %d\nContext items: %d",
			settings.Provider, settings.Model, len(req.Message), len(req.SentContext)))
	}

	var response string
	var err error

	switch settings.Provider {
	case "anthropic":
		response, err = lc.sendAnthropicRequest(ctx, req, settings, logger)
	case "openai":
		response, err = lc.sendOpenAIRequest(ctx, req, settings, logger)
	default:
		return "", fmt.Errorf("unsupported provider: %s", settings.Provider)
	}

	if err != nil {
		if logger != nil {
			logger.LogTerminalOutput(fmt.Sprintf("=== LLM REQUEST FAILED ===\nError: %v", err))
		}
		return "", err
	}

	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== LLM RESPONSE RECEIVED ===\nLength: %d chars", len(response)))
	}

	return response, nil
}

// sendAnthropicRequest sends a request to Anthropic API
func (lc *LLMClient) sendAnthropicRequest(ctx context.Context, req *ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) (string, error) {
	systemMessage := `You are an AI assistant that helps with programming and debugging.

YOU MUST RESPOND IN VALID JSON FORMAT according to this structure:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."],
  "waitForOutput": true/false
}

Do not include any text outside the JSON structure. Your entire response must be a single JSON object.`

	// Build user message with context
	userMessage := req.Message
	if len(req.SentContext) > 0 {
		contextPrefix := "\n\n--- Provided Context ---\n"
		for _, item := range req.SentContext {
			contextPrefix += fmt.Sprintf("Type: %s\nDescription: %s\n", item.Type, item.Description)
			if item.Content != "" {
				contextPrefix += fmt.Sprintf("Content:\n```\n%s\n```\n", item.Content)
			}
			contextPrefix += "---\n"
		}
		userMessage = contextPrefix + userMessage
	}

	// Build messages array
	messages := []AnthropicMessage{}
	for _, msg := range req.History {
		messages = append(messages, AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Create request
	apiReq := AnthropicRequest{
		Model:     settings.Model,
		Messages:  messages,
		MaxTokens: 4096,
		System:    systemMessage,
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create Anthropic HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", settings.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := lc.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("Anthropic API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, respBody)
	}

	var apiResp AnthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	if len(apiResp.Content) > 0 {
		return apiResp.Content[0].Text, nil
	}

	return "", fmt.Errorf("no content in Anthropic response")
}

// sendOpenAIRequest sends a request to OpenAI API
func (lc *LLMClient) sendOpenAIRequest(ctx context.Context, req *ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) (string, error) {
	systemMessage := `You are an AI assistant that helps with programming and debugging.

YOU MUST RESPOND IN VALID JSON FORMAT according to this structure:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."],
  "waitForOutput": true/false
}

Do not include any text outside the JSON structure. Your entire response must be a single JSON object.`

	// Build user message with context
	userMessage := req.Message
	if len(req.SentContext) > 0 {
		contextPrefix := "\n\n--- Provided Context ---\n"
		for _, item := range req.SentContext {
			contextPrefix += fmt.Sprintf("Type: %s\nDescription: %s\n", item.Type, item.Description)
			if item.Content != "" {
				contextPrefix += fmt.Sprintf("Content:\n```\n%s\n```\n", item.Content)
			}
			contextPrefix += "---\n"
		}
		userMessage = contextPrefix + userMessage
	}

	// Build messages array
	messages := []OpenAIMessage{
		{Role: "system", Content: systemMessage},
	}
	for _, msg := range req.History {
		messages = append(messages, OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: userMessage,
	})

	// Create request
	apiReq := OpenAIRequest{
		Model:    settings.Model,
		Messages: messages,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+settings.APIKey)

	resp, err := lc.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("OpenAI API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenAI response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, respBody)
	}

	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no content in OpenAI response")
}
