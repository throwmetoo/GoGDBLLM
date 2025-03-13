package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

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
		filePath = filepath.Join(homeDir, ".fileopener_settings.json")
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
func (sm *SettingsManager) UpdateSettings(settings Settings) error {
	sm.mutex.Lock()
	sm.settings = settings
	sm.mutex.Unlock()
	return sm.Save()
}
