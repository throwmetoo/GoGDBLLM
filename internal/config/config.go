package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	LLM     LLMConfig     `mapstructure:"llm"`
	GDB     GDBConfig     `mapstructure:"gdb"`
	Logs    LogConfig     `mapstructure:"logs"`
	Uploads UploadsConfig `mapstructure:"uploads"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// LLMConfig holds configuration for LLM providers
type LLMConfig struct {
	DefaultProvider string `mapstructure:"default_provider"`
	DefaultModel    string `mapstructure:"default_model"`
	APIKey          string `mapstructure:"api_key"`
}

// GDBConfig holds GDB-related configuration
type GDBConfig struct {
	Path         string `mapstructure:"path"`
	Timeout      int    `mapstructure:"timeout"`
	MaxProcesses int    `mapstructure:"max_processes"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Directory  string `mapstructure:"directory"`
	JSONFormat bool   `mapstructure:"json_format"`
}

// UploadsConfig holds file upload configuration
type UploadsConfig struct {
	Directory   string `mapstructure:"directory"`
	MaxFileSize int64  `mapstructure:"max_file_size"` // in bytes
}

// LoadConfig loads configuration from files and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default configuration
	setDefaults(v)

	// If a config path is specified, try to use it
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Otherwise, look for config in default locations
		v.SetConfigName("config")   // name of config file (without extension)
		v.SetConfigType("yaml")     // type of config file
		v.AddConfigPath(".")        // look for config in working directory
		v.AddConfigPath("./config") // look for config in config/ directory
		homeDir, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(filepath.Join(homeDir, ".gogdbllm")) // look in user's home directory
		}
	}

	// Read environment variables
	v.SetEnvPrefix("GOGDBLLM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Try to read config file, but don't fail if it doesn't exist
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults and environment variables
	}

	// Unmarshal config into struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)

	// LLM defaults
	v.SetDefault("llm.default_provider", "anthropic")
	v.SetDefault("llm.default_model", "claude-3-sonnet-20240229")

	// GDB defaults
	v.SetDefault("gdb.path", "gdb")
	v.SetDefault("gdb.timeout", 2)
	v.SetDefault("gdb.max_processes", 5)

	// Logs defaults
	v.SetDefault("logs.level", "info")
	v.SetDefault("logs.directory", "./logs")
	v.SetDefault("logs.json_format", true)

	// Uploads defaults
	v.SetDefault("uploads.directory", "./uploads")
	v.SetDefault("uploads.max_file_size", 10*1024*1024) // 10MB
}

// WriteDefaultConfig writes a default configuration file
func WriteDefaultConfig(configPath string) error {
	v := viper.New()

	// Set default configuration
	setDefaults(v)

	// Write the config file
	return v.WriteConfigAs(configPath)
}
