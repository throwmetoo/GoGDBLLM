package api

import (
	"encoding/json"
	"net/http"

	"github.com/throwmetoo/gogdbllm/internal/api/response"
	"github.com/throwmetoo/gogdbllm/internal/llm"
)

// handleChat handles chat requests
func (h *Handler) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var chatReq llm.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		h.logger.Printf("Error parsing chat request: %v", err)
		response.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate request
	if chatReq.Message == "" {
		response.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Process chat request
	resp, err := h.llmClient.ProcessRequest(r.Context(), chatReq)
	if err != nil {
		h.logger.Printf("Error processing chat request: %v", err)
		response.Error(w, "Failed to process chat request", http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"response": resp,
	})
}
