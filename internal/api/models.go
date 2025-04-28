package api

// ChatMessage represents a message in the chat history
type ChatMessage struct {
	Role        string        `json:"role"`
	Content     string        `json:"content"`
	SentContext []ContextItem `json:"sent_context,omitempty"`
}

// ContextItem represents a piece of context sent to the LLM
type ContextItem struct {
	Type        string `json:"type"`              // e.g., "file", "code_snippet", "command_output", "message_history"
	Description string `json:"description"`       // e.g., file path, command executed, "Previous messages"
	Content     string `json:"content,omitempty"` // The actual content snippet (optional for brevity)
}

// ChatRequest represents a request to the chat API
type ChatRequest struct {
	Message     string        `json:"message"`
	History     []ChatMessage `json:"history"`
	SentContext []ContextItem `json:"sentContext,omitempty"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	Response string `json:"response"`
}

// LLMResponse represents a structured response from the LLM
type LLMResponse struct {
	Text          string   `json:"text"`          // Text to display to the user
	GDBCommands   []string `json:"gdbCommands"`   // Array of GDB commands to execute
	WaitForOutput bool     `json:"waitForOutput"` // Whether to wait for output before continuing
}

// --- LLM Provider Specific Structs ---

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
	System    string             `json:"system,omitempty"`
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
	Model          string          `json:"model"`
	Messages       []OpenAIMessage `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ResponseFormat specifies the format for OpenAI API responses
type ResponseFormat struct {
	Type string `json:"type"`
}

// OpenAIResponse represents a response from the OpenAI API
type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// OpenRouterMessage represents a message for OpenRouter API
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterRequest represents a request to the OpenRouter API
type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
}

// OpenRouterResponse represents a response from the OpenRouter API
type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
