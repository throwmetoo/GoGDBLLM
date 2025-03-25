package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/yourusername/gogdbllm/internal/api"
	"github.com/yourusername/gogdbllm/internal/settings"
)

// ConnectionTestRequest represents a request to test API connection
type ConnectionTestRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

// SettingsHandler handles settings-related operations
type SettingsHandler struct {
	settingsManager *settings.Manager
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsManager *settings.Manager) *SettingsHandler {
	return &SettingsHandler{
		settingsManager: settingsManager,
	}
}

// GetSettings handles requests to get the current settings
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	settings := h.settingsManager.GetSettings()

	// Don't expose the API key
	settings.APIKey = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// SaveSettings handles requests to save settings
func (h *SettingsHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newSettings settings.Settings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current settings to use the existing API key if not provided
	currentSettings := h.settingsManager.GetSettings()
	if newSettings.APIKey == "" {
		newSettings.APIKey = currentSettings.APIKey
	}

	// Update settings
	h.settingsManager.UpdateSettings(newSettings)

	// Save to disk
	if err := h.settingsManager.Save(); err != nil {
		http.Error(w, "Failed to save settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"status": "Settings saved successfully",
		},
	})
}

// TestConnection handles requests to test API connection
func (h *SettingsHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectionTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Test the connection
	testSettings := settings.Settings{
		Provider: req.Provider,
		Model:    req.Model,
		APIKey:   req.APIKey,
	}

	success, message := api.TestConnection(testSettings)

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: success,
		Data: map[string]string{
			"message": message,
		},
	})
}
