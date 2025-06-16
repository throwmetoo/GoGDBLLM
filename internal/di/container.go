package di

import (
	"fmt"

	"github.com/yourusername/gogdbllm/internal/api"
	"github.com/yourusername/gogdbllm/internal/config"
	"github.com/yourusername/gogdbllm/internal/gdb"
	"github.com/yourusername/gogdbllm/internal/handlers"
	"github.com/yourusername/gogdbllm/internal/logger"
	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
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
	// Initialize logger - call directly instead of providing a function
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger directly
	if err := logger.Init(cfg); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Provide config
	if err := c.container.Provide(func() *config.Config {
		return cfg
	}); err != nil {
		return fmt.Errorf("failed to provide config: %w", err)
	}

	// Provide LoggerHolder - a shared instance for all handlers
	if err := c.container.Provide(func() handlers.LoggerHolder {
		return logsession.NewLoggerHolder()
	}); err != nil {
		return fmt.Errorf("failed to provide logger holder: %w", err)
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

	// Provide simple chat handler (clean architecture)
	if err := c.container.Provide(func(
		settingsManager *settings.Manager,
		loggerHolder api.LoggerHolder,
		gdbHandler api.GDBCommandHandler,
	) *api.SimpleChatHandler {
		return api.NewSimpleChatHandler(settingsManager, loggerHolder, gdbHandler)
	}); err != nil {
		return fmt.Errorf("failed to provide simple chat handler: %w", err)
	}

	// Provide GDB service
	if err := c.container.Provide(gdb.NewGDBService); err != nil {
		return fmt.Errorf("failed to provide GDB service: %w", err)
	}

	// Provide settings manager
	if err := c.container.Provide(func() (*settings.Manager, error) {
		return settings.NewManager("")
	}); err != nil {
		return fmt.Errorf("failed to provide settings manager: %w", err)
	}

	// Provide LoggerHolder for API package
	if err := c.container.Provide(func(holder handlers.LoggerHolder) api.LoggerHolder {
		return holder // Use the same LoggerHolder instance
	}); err != nil {
		return fmt.Errorf("failed to provide API logger holder: %w", err)
	}

	// Provide GDBCommandHandler for API package
	if err := c.container.Provide(func(handler *handlers.GDBHandler) api.GDBCommandHandler {
		return handler // Use GDBHandler as GDBCommandHandler
	}); err != nil {
		return fmt.Errorf("failed to provide API GDB command handler: %w", err)
	}

	return nil
}

// Invoke calls a function with dependencies from the container
func (c *Container) Invoke(function interface{}) error {
	return c.container.Invoke(function)
}
