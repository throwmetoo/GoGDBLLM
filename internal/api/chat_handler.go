package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
)

// Define LoggerHolder interface locally (or move to a shared place)
type LoggerHolder interface {
	Set(newLogger *logsession.SessionLogger)
	Get() *logsession.SessionLogger
}

// ChatHandler handles chat-related operations
type ChatHandler struct {
	settingsManager *settings.Manager
	loggerHolder    LoggerHolder // Use interface type
}

// NewChatHandler creates a new chat handler
func NewChatHandler(settingsManager *settings.Manager, loggerHolder LoggerHolder) *ChatHandler { // Accept interface
	return &ChatHandler{
		settingsManager: settingsManager,
		loggerHolder:    loggerHolder,
	}
}

// HandleChat handles chat requests
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	// Get current logger first
	logger := h.loggerHolder.Get()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var chatReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		if logger != nil {
			logger.LogError(err, "Decoding chat request body")
		}
		return
	}

	// --- Log user input ---
	if logger != nil {
		// Convert []api.ContextItem to []logsession.ContextItem
		logContext := make([]logsession.ContextItem, len(chatReq.SentContext))
		for i, apiItem := range chatReq.SentContext {
			logContext[i] = logsession.ContextItem{
				Type:        apiItem.Type,
				Description: apiItem.Description,
				Content:     apiItem.Content,
			}
		}
		logger.LogUserChat(logContext, chatReq.Message)
	}
	// --- End log user input ---

	// Get current settings
	settings := h.settingsManager.GetSettings()

	var response string
	var err error
	var provider string

	// Call the appropriate API based on the provider
	switch settings.Provider {
	case "anthropic":
		provider = "Anthropic"
		response, err = h.callAnthropicAPI(chatReq, settings)
	case "openai":
		provider = "OpenAI"
		response, err = h.callOpenAIAPI(chatReq, settings)
	case "openrouter":
		provider = "OpenRouter"
		response, err = h.callOpenRouterAPI(chatReq, settings)
	default:
		err = fmt.Errorf("unsupported provider: %s", settings.Provider)
		http.Error(w, err.Error(), http.StatusBadRequest)
		if logger != nil {
			logger.LogError(err, "Checking provider in HandleChat")
		}
		return
	}

	if err != nil {
		errorMsg := fmt.Sprintf("Error calling %s API: %v", provider, err)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		if logger != nil {
			logger.LogError(err, "Calling "+provider+" API")
		}
		return
	}

	// Send response
	chatResp := ChatResponse{
		Response: response,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chatResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Encoding/Sending chat response")
		}
	}
}

// Helper function to get logger safely within API call methods
func (h *ChatHandler) getLogger() *logsession.SessionLogger {
	return h.loggerHolder.Get()
}

// callAnthropicAPI calls the Anthropic API
func (h *ChatHandler) callAnthropicAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	logger := h.getLogger()
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

	if logger != nil {
		logger.LogLLMRequestData("Anthropic", apiReq.Model, currentUserMessageContent)
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Marshalling Anthropic request")
		}
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Creating Anthropic HTTP request")
		}
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Sending Anthropic request")
		}
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Reading Anthropic response body")
		}
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
		if logger != nil {
			logger.LogError(err, "Anthropic API returned non-OK status")
		}
		return "", err
	}

	// Parse response
	var apiResp AnthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Unmarshalling Anthropic response")
		}
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract content
	var content string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	if logger != nil {
		logger.LogLLMResponse(content)
	}

	return content, nil
}

// callOpenAIAPI calls the OpenAI API
func (h *ChatHandler) callOpenAIAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	logger := h.getLogger()
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

	if logger != nil {
		logger.LogLLMRequestData("OpenAI", apiReq.Model, currentUserMessageContent)
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Marshalling OpenAI request")
		}
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Creating OpenAI HTTP request")
		}
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Sending OpenAI request")
		}
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Reading OpenAI response body")
		}
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
		if logger != nil {
			logger.LogError(err, "OpenAI API returned non-OK status")
		}
		return "", err
	}

	// Parse response
	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Unmarshalling OpenAI response")
		}
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract content
	if len(apiResp.Choices) > 0 {
		responseContent := apiResp.Choices[0].Message.Content
		if logger != nil {
			logger.LogLLMResponse(responseContent)
		}
		return responseContent, nil
	}

	err = fmt.Errorf("no content in OpenAI response")
	if logger != nil {
		logger.LogError(err, "Extracting content from OpenAI response")
	}
	return "", err
}

// callOpenRouterAPI calls the OpenRouter API
func (h *ChatHandler) callOpenRouterAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	logger := h.getLogger()
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

	if logger != nil {
		logger.LogLLMRequestData("OpenRouter", apiReq.Model, currentUserMessageContent)
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Marshalling OpenRouter request")
		}
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	openRouterURL := "https://openrouter.ai/api/v1/chat/completions"
	req, err := http.NewRequest("POST", openRouterURL, bytes.NewBuffer(reqBody))
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Creating OpenRouter HTTP request")
		}
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Sending OpenRouter request")
		}
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Reading OpenRouter response body")
		}
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
		if logger != nil {
			logger.LogError(err, "OpenRouter API returned non-OK status")
		}
		return "", err
	}

	var apiResp OpenRouterResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Unmarshalling OpenRouter response")
		}
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(apiResp.Choices) > 0 {
		responseContent := apiResp.Choices[0].Message.Content
		if logger != nil {
			logger.LogLLMResponse(responseContent)
		}
		return responseContent, nil
	}

	err = fmt.Errorf("no content in OpenRouter response")
	if logger != nil {
		logger.LogError(err, "Extracting content from OpenRouter response")
	}
	return "", err
}
