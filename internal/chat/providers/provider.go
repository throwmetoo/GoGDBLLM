package providers

import (
	"context"
	"time"

	"github.com/yourusername/gogdbllm/internal/chat"
)

// Provider defines the interface that all LLM providers must implement
type Provider interface {
	// SendRequest sends a standardized request to the provider
	SendRequest(ctx context.Context, req *chat.StandardRequest) (*chat.StandardResponse, error)

	// ValidateConfig validates the provider configuration
	ValidateConfig(config *ProviderConfig) error

	// GetSupportedModels returns a list of supported models
	GetSupportedModels() []ModelInfo

	// GetName returns the provider name
	GetName() string

	// EstimateCost estimates the cost for a request (optional)
	EstimateCost(req *chat.StandardRequest) float64

	// GetHealthStatus checks if the provider is healthy
	GetHealthStatus(ctx context.Context) (*HealthStatus, error)
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Name         string                 `yaml:"name"`
	Type         string                 `yaml:"type"`
	Enabled      bool                   `yaml:"enabled"`
	APIKey       string                 `yaml:"api_key"`
	BaseURL      string                 `yaml:"base_url,omitempty"`
	DefaultModel string                 `yaml:"default_model"`
	Timeout      time.Duration          `yaml:"timeout"`
	MaxTokens    int                    `yaml:"max_tokens,omitempty"`
	Settings     map[string]interface{} `yaml:"settings,omitempty"`

	// Rate limiting
	RateLimit *RateLimitConfig `yaml:"rate_limit,omitempty"`

	// Cost settings
	CostPerToken *CostConfig `yaml:"cost_per_token,omitempty"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	TokensPerMinute   int `yaml:"tokens_per_minute"`
}

// CostConfig holds cost calculation configuration
type CostConfig struct {
	InputTokens  float64 `yaml:"input_tokens"`
	OutputTokens float64 `yaml:"output_tokens"`
}

// ModelInfo represents information about a supported model
type ModelInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	MaxTokens    int      `json:"max_tokens,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	CostTier     string   `json:"cost_tier,omitempty"`
}

// HealthStatus represents the health status of a provider
type HealthStatus struct {
	Healthy      bool          `json:"healthy"`
	ResponseTime time.Duration `json:"response_time"`
	LastCheck    time.Time     `json:"last_check"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// Registry manages all available providers
type Registry struct {
	providers map[string]Provider
	configs   map[string]*ProviderConfig
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
		configs:   make(map[string]*ProviderConfig),
	}
}

// Register registers a provider with the registry
func (r *Registry) Register(name string, provider Provider, config *ProviderConfig) error {
	if err := provider.ValidateConfig(config); err != nil {
		return err
	}

	r.providers[name] = provider
	r.configs[name] = config
	return nil
}

// GetProvider returns a provider by name
func (r *Registry) GetProvider(name string) (Provider, *ProviderConfig, bool) {
	provider, exists := r.providers[name]
	if !exists {
		return nil, nil, false
	}

	config := r.configs[name]
	return provider, config, true
}

// GetEnabledProviders returns all enabled providers
func (r *Registry) GetEnabledProviders() map[string]Provider {
	enabled := make(map[string]Provider)
	for name, provider := range r.providers {
		config := r.configs[name]
		if config != nil && config.Enabled {
			enabled[name] = provider
		}
	}
	return enabled
}

// GetProviderNames returns all registered provider names
func (r *Registry) GetProviderNames() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// GetProviderConfig returns the configuration for a provider
func (r *Registry) GetProviderConfig(name string) (*ProviderConfig, bool) {
	config, exists := r.configs[name]
	return config, exists
}

// UpdateProviderConfig updates the configuration for a provider
func (r *Registry) UpdateProviderConfig(name string, config *ProviderConfig) error {
	provider, exists := r.providers[name]
	if !exists {
		return &chat.ProviderError{
			Provider:  name,
			ErrorType: chat.ErrorTypeValidation,
			Message:   "provider not found",
			Retryable: false,
		}
	}

	if err := provider.ValidateConfig(config); err != nil {
		return err
	}

	r.configs[name] = config
	return nil
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	name   string
	config *ProviderConfig
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string, config *ProviderConfig) *BaseProvider {
	return &BaseProvider{
		name:   name,
		config: config,
	}
}

// GetName returns the provider name
func (bp *BaseProvider) GetName() string {
	return bp.name
}

// ValidateConfig provides basic validation for provider config
func (bp *BaseProvider) ValidateConfig(config *ProviderConfig) error {
	if config.Name == "" {
		return &chat.ProviderError{
			Provider:  bp.name,
			ErrorType: chat.ErrorTypeValidation,
			Message:   "provider name is required",
			Retryable: false,
		}
	}

	if config.APIKey == "" {
		return &chat.ProviderError{
			Provider:  bp.name,
			ErrorType: chat.ErrorTypeAuth,
			Message:   "API key is required",
			Retryable: false,
		}
	}

	return nil
}

// EstimateCost provides basic cost estimation
func (bp *BaseProvider) EstimateCost(req *chat.StandardRequest) float64 {
	if bp.config.CostPerToken == nil {
		return 0.0
	}

	// Rough token estimation for input
	inputTokens := bp.estimateInputTokens(req)

	// Estimate output tokens (assume average response)
	outputTokens := 500 // Default estimate
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		outputTokens = *req.MaxTokens / 2 // Assume half of max tokens
	}

	inputCost := float64(inputTokens) * bp.config.CostPerToken.InputTokens
	outputCost := float64(outputTokens) * bp.config.CostPerToken.OutputTokens

	return inputCost + outputCost
}

// estimateInputTokens estimates the number of input tokens
func (bp *BaseProvider) estimateInputTokens(req *chat.StandardRequest) int {
	tokens := 0

	// System prompt
	if req.SystemPrompt != "" {
		tokens += len(req.SystemPrompt) / 4 // Rough approximation
	}

	// Messages
	for _, msg := range req.Messages {
		tokens += len(msg.Content) / 4 // Rough approximation
	}

	return tokens
}

// GetHealthStatus provides basic health check
func (bp *BaseProvider) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()

	// Basic connectivity test would go here
	// For now, just return healthy

	return &HealthStatus{
		Healthy:      true,
		ResponseTime: time.Since(start),
		LastCheck:    time.Now(),
	}, nil
}
