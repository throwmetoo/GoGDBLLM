package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/gogdbllm/internal/chat"
)

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	*BaseProvider
	client *http.Client
}

// AnthropicRequest represents a request to the Anthropic API
type AnthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
}

// AnthropicMessage represents a message for Anthropic API
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents a response from the Anthropic API
type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
	Model      string `json:"model,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config *ProviderConfig) *AnthropicProvider {
	timeout := 30 * time.Second
	if config.Timeout > 0 {
		timeout = config.Timeout
	}

	client := &http.Client{
		Timeout: timeout,
	}

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider("anthropic", config),
		client:       client,
	}
}

// SendRequest sends a request to the Anthropic API
func (ap *AnthropicProvider) SendRequest(ctx context.Context, req *chat.StandardRequest) (*chat.StandardResponse, error) {
	start := time.Now()

	// Convert to Anthropic format
	anthropicReq, err := ap.convertRequest(req)
	if err != nil {
		return nil, &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeValidation,
			Message:   fmt.Sprintf("failed to convert request: %v", err),
			Retryable: false,
		}
	}

	// Marshal request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeInternal,
			Message:   fmt.Sprintf("failed to marshal request: %v", err),
			Retryable: false,
		}
	}

	// Create HTTP request
	baseURL := "https://api.anthropic.com/v1/messages"
	if ap.config.BaseURL != "" {
		baseURL = ap.config.BaseURL + "/v1/messages"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeInternal,
			Message:   fmt.Sprintf("failed to create HTTP request: %v", err),
			Retryable: false,
		}
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", ap.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := ap.client.Do(httpReq)
	if err != nil {
		return nil, &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeNetwork,
			Message:   fmt.Sprintf("failed to send request: %v", err),
			Retryable: true,
		}
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeNetwork,
			Message:   fmt.Sprintf("failed to read response: %v", err),
			Retryable: true,
		}
	}

	// Handle HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, ap.handleHTTPError(resp.StatusCode, respBody)
	}

	// Parse response
	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeInternal,
			Message:   fmt.Sprintf("failed to parse response: %v", err),
			Retryable: false,
		}
	}

	// Convert response
	return ap.convertResponse(&anthropicResp, req.RequestID, time.Since(start), string(respBody))
}

// convertRequest converts a standard request to Anthropic format
func (ap *AnthropicProvider) convertRequest(req *chat.StandardRequest) (*AnthropicRequest, error) {
	messages := make([]AnthropicMessage, len(req.Messages))
	for i, msg := range req.Messages {
		// Skip system messages as they go in the system field
		if msg.Role == "system" {
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}

		messages[i] = AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	maxTokens := 4096
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		maxTokens = *req.MaxTokens
	}

	return &AnthropicRequest{
		Model:     req.Model,
		Messages:  messages,
		MaxTokens: maxTokens,
		System:    req.SystemPrompt,
	}, nil
}

// convertResponse converts an Anthropic response to standard format
func (ap *AnthropicProvider) convertResponse(resp *AnthropicResponse, requestID string, responseTime time.Duration, rawResp string) (*chat.StandardResponse, error) {
	if len(resp.Content) == 0 {
		return nil, &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeInternal,
			Message:   "no content in response",
			Retryable: false,
		}
	}

	content := resp.Content[0].Text
	tokensUsed := 0
	if resp.Usage != nil {
		tokensUsed = resp.Usage.InputTokens + resp.Usage.OutputTokens
	}

	metadata := &chat.ProviderMetadata{
		RawResponse:  rawResp,
		FinishReason: resp.StopReason,
		ResponseTime: responseTime,
	}

	if resp.Usage != nil {
		metadata.PromptTokens = resp.Usage.InputTokens
		metadata.ResponseTokens = resp.Usage.OutputTokens
	}

	return &chat.StandardResponse{
		Content:    content,
		TokensUsed: tokensUsed,
		Model:      resp.Model,
		Provider:   ap.GetName(),
		RequestID:  requestID,
		Metadata:   metadata,
	}, nil
}

// handleHTTPError handles HTTP errors and converts them to provider errors
func (ap *AnthropicProvider) handleHTTPError(statusCode int, body []byte) error {
	message := string(body)

	var errorType string
	var retryable bool

	switch statusCode {
	case http.StatusUnauthorized:
		errorType = chat.ErrorTypeAuth
		retryable = false
	case http.StatusTooManyRequests:
		errorType = chat.ErrorTypeRateLimit
		retryable = true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		errorType = chat.ErrorTypeNetwork
		retryable = true
	default:
		errorType = chat.ErrorTypeInternal
		retryable = false
	}

	return &chat.ProviderError{
		Provider:  ap.GetName(),
		ErrorType: errorType,
		Message:   fmt.Sprintf("HTTP %d: %s", statusCode, message),
		Code:      statusCode,
		Retryable: retryable,
	}
}

// GetSupportedModels returns supported Anthropic models
func (ap *AnthropicProvider) GetSupportedModels() []ModelInfo {
	return []ModelInfo{
		{
			ID:           "claude-3-5-sonnet-20240620",
			Name:         "Claude 3.5 Sonnet",
			Description:  "Most intelligent model, ideal for complex reasoning",
			MaxTokens:    200000,
			Capabilities: []string{"text", "analysis", "coding", "reasoning"},
			CostTier:     "premium",
		},
		{
			ID:           "claude-3-opus-20240229",
			Name:         "Claude 3 Opus",
			Description:  "Most powerful model for highly complex tasks",
			MaxTokens:    200000,
			Capabilities: []string{"text", "analysis", "coding", "reasoning", "research"},
			CostTier:     "premium",
		},
		{
			ID:           "claude-3-sonnet-20240229",
			Name:         "Claude 3 Sonnet",
			Description:  "Balanced performance and speed",
			MaxTokens:    200000,
			Capabilities: []string{"text", "analysis", "coding"},
			CostTier:     "standard",
		},
		{
			ID:           "claude-3-haiku-20240307",
			Name:         "Claude 3 Haiku",
			Description:  "Fastest model for simple tasks",
			MaxTokens:    200000,
			Capabilities: []string{"text", "simple-analysis"},
			CostTier:     "economy",
		},
	}
}

// ValidateConfig validates Anthropic-specific configuration
func (ap *AnthropicProvider) ValidateConfig(config *ProviderConfig) error {
	if err := ap.BaseProvider.ValidateConfig(config); err != nil {
		return err
	}

	// Validate model
	supportedModels := ap.GetSupportedModels()
	modelValid := false
	for _, model := range supportedModels {
		if model.ID == config.DefaultModel {
			modelValid = true
			break
		}
	}

	if !modelValid {
		return &chat.ProviderError{
			Provider:  ap.GetName(),
			ErrorType: chat.ErrorTypeValidation,
			Message:   fmt.Sprintf("unsupported model: %s", config.DefaultModel),
			Retryable: false,
		}
	}

	return nil
}

// GetHealthStatus checks the health of the Anthropic API
func (ap *AnthropicProvider) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()

	// Create a minimal test request
	testReq := &chat.StandardRequest{
		Model: ap.config.DefaultModel,
		Messages: []chat.StandardMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: &[]int{10}[0],
		RequestID: "health-check",
	}

	// Try to send the request
	_, err := ap.SendRequest(ctx, testReq)

	responseTime := time.Since(start)

	if err != nil {
		return &HealthStatus{
			Healthy:      false,
			ResponseTime: responseTime,
			LastCheck:    time.Now(),
			ErrorMessage: err.Error(),
		}, nil
	}

	return &HealthStatus{
		Healthy:      true,
		ResponseTime: responseTime,
		LastCheck:    time.Now(),
	}, nil
}
