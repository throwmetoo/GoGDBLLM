package di

import (
	"fmt"

	"github.com/yourusername/gogdbllm/internal/api"
	"github.com/yourusername/gogdbllm/internal/config"
	"github.com/yourusername/gogdbllm/internal/gdb"
	"github.com/yourusername/gogdbllm/internal/handlers"
	"github.com/yourusername/gogdbllm/internal/logger"
	"github.com/yourusername/gogdbllm/internal/websocket"
	"go.uber.org/dig"
)

// Container is the dependency injection container
type Container struct {
	container *dig.Container
}

// NewContainer creates a new dependency injection container
func NewContainer() *Container {
	return &Container{
		container: dig.New(),
	}
}

// Configure sets up the dependency injection container
func (c *Container) Configure(configPath string) error {
	// Provide config
	if err := c.container.Provide(func() (*config.Config, error) {
		return config.LoadConfig(configPath)
	}); err != nil {
		return fmt.Errorf("failed to provide config: %w", err)
	}

	// Initialize logger
	if err := c.container.Provide(func(cfg *config.Config) error {
		return logger.Init(cfg)
	}); err != nil {
		return fmt.Errorf("failed to provide logger initializer: %w", err)
	}

	// Provide WebSocket hub
	if err := c.container.Provide(websocket.NewHub); err != nil {
		return fmt.Errorf("failed to provide WebSocket hub: %w", err)
	}

	// Provide handlers
	if err := c.container.Provide(handlers.NewFileHandler); err != nil {
		return fmt.Errorf("failed to provide file handler: %w", err)
	}

	if err := c.container.Provide(handlers.NewGDBHandler); err != nil {
		return fmt.Errorf("failed to provide GDB handler: %w", err)
	}

	if err := c.container.Provide(handlers.NewSettingsHandler); err != nil {
		return fmt.Errorf("failed to provide settings handler: %w", err)
	}

	if err := c.container.Provide(api.NewChatHandler); err != nil {
		return fmt.Errorf("failed to provide chat handler: %w", err)
	}

	// Provide GDB service
	if err := c.container.Provide(gdb.NewService); err != nil {
		return fmt.Errorf("failed to provide GDB service: %w", err)
	}

	return nil
}

// Invoke calls a function with dependencies from the container
func (c *Container) Invoke(function interface{}) error {
	return c.container.Invoke(function)
}
