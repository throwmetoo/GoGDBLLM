package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const settingsFile = ".gogdbllm_settings.json"

// Settings represents the application settings
type Settings struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

// SettingsManager handles loading and saving settings
type SettingsManager struct {
	filePath string
	settings Settings
	mutex    sync.RWMutex
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager(filePath string) (*SettingsManager, error) {
	// If no path is provided, use the default
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		filePath = filepath.Join(homeDir, settingsFile)
	}

	manager := &SettingsManager{
		filePath: filePath,
		settings: Settings{
			Provider: "anthropic",                // Default provider
			Model:    "claude-3-sonnet-20240229", // Default model
			APIKey:   "",
		},
	}

	// Try to load existing settings
	err := manager.Load()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return manager, nil
}

// Load settings from file
func (sm *SettingsManager) Load() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &sm.settings)
}

// Save settings to file
func (sm *SettingsManager) Save() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(sm.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(sm.filePath, data, 0600) // 0600 for read/write by owner only
}

// GetSettings returns a copy of the current settings
func (sm *SettingsManager) GetSettings() Settings {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.settings
}

// UpdateSettings updates the settings
func (sm *SettingsManager) UpdateSettings(settings Settings) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.settings = settings
}

// SaveSettings saves the settings to a file
func (sm *SettingsManager) SaveSettings(settings Settings) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Update the settings in memory
	sm.settings = settings

	// Marshal the settings to JSON
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write the settings to the file in the current directory
	if err := os.WriteFile(settingsFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

// LoadSettings loads the settings from a file
func (sm *SettingsManager) LoadSettings() (Settings, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Try to read from current directory first
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		// If not found in current directory, try home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return Settings{}, fmt.Errorf("failed to get home directory: %w", err)
		}

		settingsPath := filepath.Join(homeDir, settingsFile)
		data, err = os.ReadFile(settingsPath)
		if err != nil {
			// If not found anywhere, return default settings
			return Settings{
				Provider: "openai",
				Model:    "gpt-3.5-turbo",
				APIKey:   "",
			}, nil
		}
	}

	// Unmarshal the settings
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	// Update the settings in memory
	sm.settings = settings

	return settings, nil
}
