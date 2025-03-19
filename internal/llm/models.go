package llm

// ChatRequest represents a request to the chat API
type ChatRequest struct {
	Message string        `json:"message"`
	History []ChatMessage `json:"history"`
}

// ChatMessage represents a message in the chat history
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	Response string `json:"response"`
}

// AnthropicMessage represents a message in the Anthropic API format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest represents a request to the Anthropic API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
}

// AnthropicContent represents content in the Anthropic API response
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicResponse represents a response from the Anthropic API
type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
}

// OpenAIMessage represents a message in the OpenAI API format
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIRequest represents a request to the OpenAI API
type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

// OpenAIChoice represents a choice in the OpenAI API response
type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

// OpenAIResponse represents a response from the OpenAI API
type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
}
