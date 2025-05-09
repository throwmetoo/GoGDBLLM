package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Test with default configuration
	t.Run("Default configuration", func(t *testing.T) {
		cfg, err := LoadConfig("")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "anthropic", cfg.LLM.DefaultProvider)
	})

	// Test with file configuration
	t.Run("From config file", func(t *testing.T) {
		// Create a temporary config file
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		// Write test config content
		configContent := `
server:
  port: 9090
  read_timeout: 60s
  write_timeout: 60s

llm:
  default_provider: "openai"
  default_model: "gpt-4"
  api_key: "test-key"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		assert.NoError(t, err)

		// Load the config
		cfg, err := LoadConfig(configPath)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify the values from the file
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, 60*time.Second, cfg.Server.ReadTimeout)
		assert.Equal(t, 60*time.Second, cfg.Server.WriteTimeout)
		assert.Equal(t, "openai", cfg.LLM.DefaultProvider)
		assert.Equal(t, "gpt-4", cfg.LLM.DefaultModel)
		assert.Equal(t, "test-key", cfg.LLM.APIKey)
	})

	// Test with environment variables
	t.Run("From environment variables", func(t *testing.T) {
		// Set environment variables
		os.Setenv("GOGDBLLM_SERVER_PORT", "7070")
		os.Setenv("GOGDBLLM_LLM_DEFAULT_PROVIDER", "openrouter")
		defer func() {
			os.Unsetenv("GOGDBLLM_SERVER_PORT")
			os.Unsetenv("GOGDBLLM_LLM_DEFAULT_PROVIDER")
		}()

		// Load the config
		cfg, err := LoadConfig("")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify the values from environment variables
		assert.Equal(t, 7070, cfg.Server.Port)
		assert.Equal(t, "openrouter", cfg.LLM.DefaultProvider)
	})
}

func TestWriteDefaultConfig(t *testing.T) {
	// Create a temporary file path
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Write default config
	err := WriteDefaultConfig(configPath)
	assert.NoError(t, err)

	// Check if file exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Load the config from the file to verify it contains default values
	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
}
