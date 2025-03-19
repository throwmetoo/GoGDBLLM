package api

import (
	"encoding/json"
	"net/http"

	"github.com/throwmetoo/gogdbllm/internal/api/response"
	"github.com/throwmetoo/gogdbllm/internal/config"
)

// TestConnectionRequest represents a request to test an LLM connection
type TestConnectionRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

// handleTestConnection handles requests to test an LLM connection
func (h *Handler) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("Error parsing test connection request: %v", err)
		response.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Provider == "" {
		response.Error(w, "Provider is required", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		response.Error(w, "Model is required", http.StatusBadRequest)
		return
	}

	if req.APIKey == "" {
		response.Error(w, "API key is required", http.StatusBadRequest)
		return
	}

	// Create temporary settings for testing
	testSettings := config.LLMSettings{
		Provider: req.Provider,
		Model:    req.Model,
		APIKey:   req.APIKey,
	}

	// Test connection
	if err := h.llmClient.TestConnection(r.Context(), testSettings); err != nil {
		h.logger.Printf("Connection test failed: %v", err)
		response.Error(w, "Connection test failed", http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Connection test successful",
	})
}
