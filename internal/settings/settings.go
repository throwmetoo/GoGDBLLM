package settings

import (
	"encoding/json"
	"errors"
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

// Manager handles loading and saving settings
type Manager struct {
	filePath string
	settings Settings
	mutex    sync.RWMutex
}

// NewManager creates a new settings manager
func NewManager(filePath string) (*Manager, error) {
	// If no path is provided, use the default
	if filePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		filePath = filepath.Join(homeDir, settingsFile)
	}

	manager := &Manager{
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
func (m *Manager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Try to read from the file path
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		// If file doesn't exist, use default settings
		if os.IsNotExist(err) {
			m.settings = Settings{
				Provider: "anthropic",
				Model:    "claude-3-sonnet-20240229",
				APIKey:   "",
			}
			return os.ErrNotExist
		}
		return err
	}

	// Unmarshal the data
	if err := json.Unmarshal(data, &m.settings); err != nil {
		return err
	}

	return nil
}

// Save settings to file
func (m *Manager) Save() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	data, err := json.MarshalIndent(m.settings, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0600)
}

// GetSettings returns the current settings
func (m *Manager) GetSettings() Settings {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.settings
}

// UpdateSettings updates the current settings
func (m *Manager) UpdateSettings(newSettings Settings) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.settings = newSettings
}
