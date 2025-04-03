package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yourusername/gogdbllm/internal/settings"
)

// ChatHandler handles chat-related operations
type ChatHandler struct {
	settingsManager *settings.Manager
}

// NewChatHandler creates a new chat handler
func NewChatHandler(settingsManager *settings.Manager) *ChatHandler {
	return &ChatHandler{
		settingsManager: settingsManager,
	}
}

// HandleChat handles chat requests
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var chatReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current settings
	settings := h.settingsManager.GetSettings()

	var response string
	var err error

	// Call the appropriate API based on the provider
	switch settings.Provider {
	case "anthropic":
		response, err = h.callAnthropicAPI(chatReq, settings)
	case "openai":
		response, err = h.callOpenAIAPI(chatReq, settings)
	case "openrouter":
		response, err = h.callOpenRouterAPI(chatReq, settings)
	default:
		http.Error(w, "Unsupported provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Error calling LLM API: %v", err), http.StatusInternalServerError)
		return
	}

	// Send response
	chatResp := ChatResponse{
		Response: response,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResp)
}

// callAnthropicAPI calls the Anthropic API
func (h *ChatHandler) callAnthropicAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	// Anthropic doesn't support a dedicated system message, so we'll include it in the first user message
	systemMessage := "You are an AI assistant that helps with programming and debugging. Provide clear explanations and code examples when needed."

	// --- Context Injection Start ---
	currentUserMessageContent := chatReq.Message
	if len(chatReq.SentContext) > 0 {
		contextPrefix := "\n\n--- Provided Context ---\n"
		for _, item := range chatReq.SentContext {
			contextPrefix += fmt.Sprintf("Type: %s\nDescription: %s\n", item.Type, item.Description)
			if item.Content != "" {
				contextPrefix += fmt.Sprintf("Content:\n```\n%s\n```\n", item.Content)
			}
			contextPrefix += "---\n"
		}
		currentUserMessageContent = contextPrefix + currentUserMessageContent
	}
	// --- Context Injection End ---

	// Build the messages array
	messages := []AnthropicMessage{}

	// Add chat history
	for i, msg := range chatReq.History {
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
			// If this is the first user message, prepend the system message
			if i == 0 {
				msg.Content = systemMessage + "\n\n" + msg.Content
			}
		}
		messages = append(messages, AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add the current message (with context potentially prepended)
	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	// Create request
	apiReq := AnthropicRequest{
		Model:     settings.Model,
		Messages:  messages,
		MaxTokens: 4096,
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	// Parse response
	var apiResp AnthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract content
	var content string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return content, nil
}

// callOpenAIAPI calls the OpenAI API
func (h *ChatHandler) callOpenAIAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	// --- Context Injection Start ---
	currentUserMessageContent := chatReq.Message
	if len(chatReq.SentContext) > 0 {
		contextPrefix := "\n\n--- Provided Context ---\n"
		for _, item := range chatReq.SentContext {
			contextPrefix += fmt.Sprintf("Type: %s\nDescription: %s\n", item.Type, item.Description)
			if item.Content != "" {
				contextPrefix += fmt.Sprintf("Content:\n```\n%s\n```\n", item.Content)
			}
			contextPrefix += "---\n"
		}
		currentUserMessageContent = contextPrefix + currentUserMessageContent
	}
	// --- Context Injection End ---

	// Build the messages array
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: "You are an AI assistant that helps with programming and debugging. Provide clear explanations and code examples when needed.",
		},
	}

	// Add chat history
	for _, msg := range chatReq.History {
		role := msg.Role
		if role == "user" {
			role = "user"
		} else {
			role = "assistant"
		}
		messages = append(messages, OpenAIMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add the current message (with context potentially prepended)
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	// Create request
	apiReq := OpenAIRequest{
		Model:    settings.Model,
		Messages: messages,
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	// Parse response
	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract content
	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no content in response")
}

// callOpenRouterAPI calls the OpenRouter API
func (h *ChatHandler) callOpenRouterAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	// --- Context Injection Start ---
	currentUserMessageContent := chatReq.Message
	if len(chatReq.SentContext) > 0 {
		contextPrefix := "\n\n--- Provided Context ---\n"
		for _, item := range chatReq.SentContext {
			contextPrefix += fmt.Sprintf("Type: %s\nDescription: %s\n", item.Type, item.Description)
			if item.Content != "" {
				contextPrefix += fmt.Sprintf("Content:\n```\n%s\n```\n", item.Content)
			}
			contextPrefix += "---\n"
		}
		currentUserMessageContent = contextPrefix + currentUserMessageContent
	}
	// --- Context Injection End ---

	// Build the messages array
	messages := []OpenRouterMessage{
		{
			Role:    "system",
			Content: "You are an AI assistant that helps with programming and debugging. Provide clear explanations and code examples when needed.",
		},
	}

	// Add chat history
	for _, msg := range chatReq.History {
		role := msg.Role
		if role == "user" {
			role = "user"
		} else {
			role = "assistant"
		}
		messages = append(messages, OpenRouterMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add the current message (with context potentially prepended)
	messages = append(messages, OpenRouterMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	// Create request
	apiReq := OpenRouterRequest{
		Model:    settings.Model,
		Messages: messages,
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/yourusername/gogdbllm")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	// Parse response
	var apiResp OpenRouterResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract content
	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no content in response")
}
