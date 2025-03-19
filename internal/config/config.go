package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/viper"
)

const (
	defaultConfigFile    = ".gogdbllm_config.json"
	defaultUploadDir     = "uploads"
	defaultPort          = 8080
	defaultMaxUploadSize = 10 << 20 // 10 MB
)

// Config represents the application configuration
type Config struct {
	Port          int         `json:"port"`
	UploadDir     string      `json:"uploadDir"`
	GDBPath       string      `json:"gdbPath"`
	MaxUploadSize int64       `json:"maxUploadSize"`
	LLMSettings   LLMSettings `json:"llmSettings"`

	mu         sync.RWMutex
	configPath string
}

// LLMSettings represents the LLM configuration
type LLMSettings struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

// Load loads the configuration from various sources
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("port", defaultPort)
	v.SetDefault("uploadDir", defaultUploadDir)
	v.SetDefault("gdbPath", "gdb")
	v.SetDefault("maxUploadSize", defaultMaxUploadSize)
	v.SetDefault("llmSettings.provider", "anthropic")
	v.SetDefault("llmSettings.model", "claude-3-sonnet-20240229")
	v.SetDefault("llmSettings.apiKey", "")

	// Environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("GOGDBLLM")

	// Config file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, defaultConfigFile)

	// Check if config file exists
	if _, err := os.Stat(configPath); err == nil {
		// Read config file
		file, err := os.Open(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		if err := v.ReadConfig(file); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Create config
	config := &Config{
		Port:          v.GetInt("port"),
		UploadDir:     v.GetString("uploadDir"),
		GDBPath:       v.GetString("gdbPath"),
		MaxUploadSize: v.GetInt64("maxUploadSize"),
		LLMSettings: LLMSettings{
			Provider: v.GetString("llmSettings.provider"),
			Model:    v.GetString("llmSettings.model"),
			APIKey:   v.GetString("llmSettings.apiKey"),
		},
		configPath: configPath,
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(config.UploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	return config, nil
}

// Save saves the configuration to a file
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(c.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateLLMSettings updates the LLM settings
func (c *Config) UpdateLLMSettings(settings LLMSettings) error {
	c.mu.Lock()
	c.LLMSettings = settings
	c.mu.Unlock()

	return c.Save()
}

// GetLLMSettings returns the current LLM settings
func (c *Config) GetLLMSettings() LLMSettings {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LLMSettings
}
