package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ChatMessage represents a message in the chat history
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the chat API
type ChatRequest struct {
	Message string        `json:"message"`
	History []ChatMessage `json:"history"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	Response string `json:"response"`
}

// AnthropicMessage represents a message for Anthropic API
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest represents a request to the Anthropic API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
}

// AnthropicResponse represents a response from the Anthropic API
type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// OpenAIMessage represents a message for OpenAI API
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest represents a request to the OpenAI API
type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

// OpenAIResponse represents a response from the OpenAI API
type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// HandleChat handles chat requests
func (s *Server) HandleChat(w http.ResponseWriter, r *http.Request) {
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
	settings := s.settingsManager.GetSettings()

	var response string
	var err error

	// Call the appropriate API based on the provider
	switch settings.Provider {
	case "anthropic":
		response, err = s.callAnthropicAPI(chatReq, settings)
	case "openai":
		response, err = s.callOpenAIAPI(chatReq, settings)
	case "openrouter":
		response, err = s.callOpenRouterAPI(chatReq, settings)
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
func (s *Server) callAnthropicAPI(chatReq ChatRequest, settings Settings) (string, error) {
	// Anthropic doesn't support a dedicated system message, so we'll include it in the first user message
	systemMessage := "You are an AI assistant that helps with programming and debugging. Provide clear explanations and code examples when needed."

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

	// Add the current message
	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: chatReq.Message,
	})

	// Create request
	apiReq := AnthropicRequest{
		Model:     settings.Model,
		Messages:  messages,
		MaxTokens: 2000,
	}

	// Convert to JSON
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Check for error
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	// Parse response
	var apiResp AnthropicResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", err
	}

	// Extract text from response
	if len(apiResp.Content) > 0 {
		return apiResp.Content[0].Text, nil
	}

	return "", fmt.Errorf("empty response from API")
}

// callOpenAIAPI calls the OpenAI API
func (s *Server) callOpenAIAPI(chatReq ChatRequest, settings Settings) (string, error) {
	// Convert chat history to OpenAI format
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: "You are an AI assistant that helps with programming and debugging. Provide clear explanations and code examples when needed.",
		},
	}

	// Add chat history
	for _, msg := range chatReq.History {
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
		Content: chatReq.Message,
	})

	// Create request
	apiReq := OpenAIRequest{
		Model:    settings.Model,
		Messages: messages,
	}

	// Convert to JSON
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Check for error
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	// Parse response
	var apiResp OpenAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", err
	}

	// Extract text from response
	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("empty response from API")
}

// callOpenRouterAPI calls the OpenRouter API
func (s *Server) callOpenRouterAPI(chatReq ChatRequest, settings Settings) (string, error) {
	// Convert chat history to OpenRouter format (similar to OpenAI)
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: "You are an AI assistant that helps with programming and debugging. Provide clear explanations and code examples when needed.",
		},
	}

	// Add chat history
	for _, msg := range chatReq.History {
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
		Content: chatReq.Message,
	})

	// Create request
	apiReq := OpenAIRequest{
		Model:    settings.Model,
		Messages: messages,
	}

	// Convert to JSON
	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("HTTP-Referer", "https://gogdbllm.app") // Replace with your actual domain
	req.Header.Set("X-Title", "GoGDBLLM")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Check for error
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	// Parse response
	var apiResp OpenAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", err
	}

	// Extract text from response
	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("empty response from API")
}

// ProcessChatRequest processes a chat request and returns a response
func (s *Server) ProcessChatRequest(chatReq ChatRequest) (string, error) {
	// Get settings
	settings := s.settingsManager.GetSettings()

	// Call the appropriate API based on the provider
	switch settings.Provider {
	case "openai":
		return s.callOpenAIAPI(chatReq, settings)
	case "anthropic":
		return s.callAnthropicAPI(chatReq, settings)
	case "openrouter":
		return s.callOpenRouterAPI(chatReq, settings)
	default:
		return "", fmt.Errorf("unsupported provider: %s", settings.Provider)
	}
}
