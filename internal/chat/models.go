package chat

import (
	"time"

	"github.com/yourusername/gogdbllm/internal/api"
)

// ChatRequest represents an internal chat request
type ChatRequest struct {
	Message     string            `json:"message"`
	History     []api.ChatMessage `json:"history"`
	SentContext []api.ContextItem `json:"sentContext,omitempty"`
	SessionID   string            `json:"sessionId,omitempty"`
	UserID      string            `json:"userId,omitempty"`
	RequestID   string            `json:"requestId"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ChatResponse represents an internal chat response
type ChatResponse struct {
	Response    string            `json:"response"`
	LLMResponse *api.LLMResponse  `json:"llmResponse,omitempty"`
	Metadata    *ResponseMetadata `json:"metadata,omitempty"`
	RequestID   string            `json:"requestId"`
	Timestamp   time.Time         `json:"timestamp"`
	FromCache   bool              `json:"fromCache"`
}

// ResponseMetadata contains additional information about the response
type ResponseMetadata struct {
	Provider       string        `json:"provider"`
	Model          string        `json:"model"`
	TokensUsed     int           `json:"tokensUsed,omitempty"`
	EstimatedCost  float64       `json:"estimatedCost,omitempty"`
	ResponseTime   time.Duration `json:"responseTime"`
	RetryAttempts  int           `json:"retryAttempts"`
	CacheHit       bool          `json:"cacheHit"`
	ContextLength  int           `json:"contextLength"`
	ContextTrimmed bool          `json:"contextTrimmed"`
}

// StandardRequest represents a standardized request to any provider
type StandardRequest struct {
	Model          string            `json:"model"`
	Messages       []StandardMessage `json:"messages"`
	MaxTokens      *int              `json:"maxTokens,omitempty"`
	Temperature    *float64          `json:"temperature,omitempty"`
	SystemPrompt   string            `json:"systemPrompt,omitempty"`
	ResponseFormat *ResponseFormat   `json:"responseFormat,omitempty"`
	RequestID      string            `json:"requestId"`
}

// StandardMessage represents a standardized message
type StandardMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat specifies the desired response format
type ResponseFormat struct {
	Type   string `json:"type"`
	Schema string `json:"schema,omitempty"`
}

// StandardResponse represents a standardized response from any provider
type StandardResponse struct {
	Content    string            `json:"content"`
	TokensUsed int               `json:"tokensUsed,omitempty"`
	Model      string            `json:"model"`
	Provider   string            `json:"provider"`
	RequestID  string            `json:"requestId"`
	Metadata   *ProviderMetadata `json:"metadata,omitempty"`
}

// ProviderMetadata contains provider-specific metadata
type ProviderMetadata struct {
	RawResponse    string        `json:"rawResponse,omitempty"`
	FinishReason   string        `json:"finishReason,omitempty"`
	PromptTokens   int           `json:"promptTokens,omitempty"`
	ResponseTokens int           `json:"responseTokens,omitempty"`
	ResponseTime   time.Duration `json:"responseTime"`
}

// ProviderError represents an error from a provider
type ProviderError struct {
	Provider  string `json:"provider"`
	ErrorType string `json:"errorType"`
	Message   string `json:"message"`
	Code      int    `json:"code,omitempty"`
	Retryable bool   `json:"retryable"`
}

func (e *ProviderError) Error() string {
	return e.Message
}

// ErrorType constants
const (
	ErrorTypeRateLimit   = "rate_limit"
	ErrorTypeInvalidJSON = "invalid_json"
	ErrorTypeNetwork     = "network"
	ErrorTypeAuth        = "authentication"
	ErrorTypeQuota       = "quota_exceeded"
	ErrorTypeModel       = "model_error"
	ErrorTypeValidation  = "validation"
	ErrorTypeTimeout     = "timeout"
	ErrorTypeInternal    = "internal"
)

// CacheKey represents a cache key for requests
type CacheKey struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Hash     string `json:"hash"`
}

// CacheEntry represents a cached response
type CacheEntry struct {
	Response     *ChatResponse `json:"response"`
	CreatedAt    time.Time     `json:"createdAt"`
	ExpiresAt    time.Time     `json:"expiresAt"`
	AccessCount  int           `json:"accessCount"`
	LastAccessed time.Time     `json:"lastAccessed"`
}

// Metrics represents various metrics for monitoring
type Metrics struct {
	RequestCount        int64         `json:"requestCount"`
	ResponseTime        time.Duration `json:"responseTime"`
	ErrorCount          int64         `json:"errorCount"`
	CacheHits           int64         `json:"cacheHits"`
	CacheMisses         int64         `json:"cacheMisses"`
	TokensUsed          int64         `json:"tokensUsed"`
	EstimatedCost       float64       `json:"estimatedCost"`
	RetryAttempts       int64         `json:"retryAttempts"`
	CircuitBreakerTrips int64         `json:"circuitBreakerTrips"`
	ContextTrimCount    int64         `json:"contextTrimCount"`
}

// ProviderMetrics represents metrics for a specific provider
type ProviderMetrics struct {
	Provider    string    `json:"provider"`
	Metrics     *Metrics  `json:"metrics"`
	LastUpdated time.Time `json:"lastUpdated"`
}
