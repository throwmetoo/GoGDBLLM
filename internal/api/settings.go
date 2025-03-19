package api

import (
	"encoding/json"
	"net/http"

	"github.com/throwmetoo/GoGDBLLM/internal/api/response"
	"github.com/throwmetoo/GoGDBLLM/internal/config"
)

// handleSettings handles settings requests
func (h *Handler) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getSettings(w, r)
	case http.MethodPut:
		h.updateSettings(w, r)
	default:
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getSettings returns the current settings
func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	settings := h.config.LLMSettings
	response.JSON(w, http.StatusOK, settings)
}

// updateSettings updates the settings
func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var settings config.LLMSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		h.logger.Printf("Error parsing settings: %v", err)
		response.Error(w, "Invalid settings format", http.StatusBadRequest)
		return
	}

	// Validate settings
	if settings.Provider == "" {
		response.Error(w, "Provider is required", http.StatusBadRequest)
		return
	}

	if settings.Model == "" {
		response.Error(w, "Model is required", http.StatusBadRequest)
		return
	}

	// Update settings
	h.config.LLMSettings = settings

	// Save settings
	if err := h.config.Save(); err != nil {
		h.logger.Printf("Error saving settings: %v", err)
		response.Error(w, "Failed to save settings", http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Settings updated successfully",
	})
}

// handleSaveSettings handles requests to save settings
// Note: This is an alias for updateSettings and is kept for backward compatibility
func (h *Handler) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Delegate to updateSettings
	h.updateSettings(w, r)
}
