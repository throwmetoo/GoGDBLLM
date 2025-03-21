package llm

import (
	"context"
	"fmt"
	"log"

	"github.com/throwmetoo/GoGDBLLM/internal/config"
)

// Client defines the interface for LLM clients
type Client interface {
	// ProcessRequest processes a chat request and returns a response
	ProcessRequest(ctx context.Context, req ChatRequest) (string, error)

	// TestConnection tests the connection to the LLM provider
	TestConnection(ctx context.Context, settings config.LLMSettings) error
}

// Factory creates LLM clients based on provider
type Factory struct {
	logger *log.Logger
}

// NewFactory creates a new LLM client factory
func NewFactory(logger *log.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateClient creates a new LLM client based on the provider
func (f *Factory) CreateClient(settings config.LLMSettings) (Client, error) {
	switch settings.Provider {
	case "anthropic":
		return NewAnthropicClient(f.logger, settings), nil
	case "openai":
		return NewOpenAIClient(f.logger, settings), nil
	case "openrouter":
		return NewOpenRouterClient(f.logger, settings), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", settings.Provider)
	}
}

// NewClient creates a new LLM client based on the provided settings
// This is a convenience function for direct client creation
func NewClient(settings config.LLMSettings, logger *log.Logger) Client {
	factory := NewFactory(logger)
	client, err := factory.CreateClient(settings)
	if err != nil {
		logger.Printf("Warning: Failed to create LLM client: %v", err)
		return nil
	}
	return client
}
