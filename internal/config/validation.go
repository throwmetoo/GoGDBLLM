package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	ErrInvalidProvider = errors.New("invalid provider")
	ErrInvalidModel    = errors.New("invalid model")
	ErrEmptyAPIKey     = errors.New("API key cannot be empty")
	ErrInvalidGDBPath  = errors.New("invalid GDB path")
)

// ValidateLLMSettings validates the LLM settings
func ValidateLLMSettings(settings LLMSettings) error {
	// Validate provider
	switch settings.Provider {
	case "anthropic", "openai", "openrouter":
		// Valid providers
	default:
		return fmt.Errorf("%w: %s", ErrInvalidProvider, settings.Provider)
	}

	// Validate model (basic check)
	if settings.Model == "" {
		return fmt.Errorf("%w", ErrInvalidModel)
	}

	// Validate API key
	if settings.APIKey == "" {
		return fmt.Errorf("%w", ErrEmptyAPIKey)
	}

	return nil
}

// ValidateConfig validates the configuration
func (c *Config) Validate() error {
	// Validate LLM settings
	if err := ValidateLLMSettings(c.LLMSettings); err != nil {
		return err
	}

	// Validate GDB path
	if c.GDBPath != "" {
		if _, err := os.Stat(c.GDBPath); err != nil {
			// If GDB path is not a file, check if it's in PATH
			_, err := exec.LookPath(c.GDBPath)
			if err != nil {
				return fmt.Errorf("%w: %s", ErrInvalidGDBPath, c.GDBPath)
			}
		}
	}

	return nil
}
