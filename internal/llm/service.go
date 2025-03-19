package llm

import (
	"context"
	"log"
	"sync"

	"github.com/throwmetoo/GoGDBLLM/internal/config"
)

// Service provides a high-level interface for LLM operations
type Service struct {
	logger     *log.Logger
	config     *config.Config
	factory    *Factory
	clientLock sync.RWMutex
	client     Client
}

// NewService creates a new LLM service
func NewService(logger *log.Logger, cfg *config.Config) *Service {
	factory := NewFactory(logger)

	// Create initial client based on config
	client, err := factory.CreateClient(cfg.LLMSettings)
	if err != nil {
		logger.Printf("Warning: Failed to create initial LLM client: %v", err)
	}

	return &Service{
		logger:  logger,
		config:  cfg,
		factory: factory,
		client:  client,
	}
}

// ProcessRequest processes a chat request
func (s *Service) ProcessRequest(ctx context.Context, req ChatRequest) (string, error) {
	s.clientLock.RLock()
	client := s.client
	settings := s.config.GetLLMSettings()
	s.clientLock.RUnlock()

	// If client is nil or settings have changed, create a new client
	if client == nil ||
		settings.Provider != s.config.LLMSettings.Provider ||
		settings.Model != s.config.LLMSettings.Model ||
		settings.APIKey != s.config.LLMSettings.APIKey {

		s.clientLock.Lock()
		var err error
		s.client, err = s.factory.CreateClient(settings)
		client = s.client
		s.clientLock.Unlock()

		if err != nil {
			return "", err
		}
	}

	return client.ProcessRequest(ctx, req)
}

// TestConnection tests the connection to an LLM provider
func (s *Service) TestConnection(ctx context.Context, settings config.LLMSettings) error {
	client, err := s.factory.CreateClient(settings)
	if err != nil {
		return err
	}

	return client.TestConnection(ctx, settings)
}

// UpdateSettings updates the LLM settings and creates a new client
func (s *Service) UpdateSettings(settings config.LLMSettings) error {
	// Update config
	if err := s.config.UpdateLLMSettings(settings); err != nil {
		return err
	}

	// Create new client
	s.clientLock.Lock()
	defer s.clientLock.Unlock()

	client, err := s.factory.CreateClient(settings)
	if err != nil {
		return err
	}

	s.client = client
	return nil
}
