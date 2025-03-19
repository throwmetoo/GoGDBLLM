package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/throwmetoo/gogdbllm/internal/api/response"
)

// StartDebuggerRequest represents a request to start the debugger
type StartDebuggerRequest struct {
	Filepath string `json:"filepath"`
}

// DebuggerCommandRequest represents a request to send a command to the debugger
type DebuggerCommandRequest struct {
	Command string `json:"command"`
}

// handleStartDebugger handles requests to start the debugger
func (h *Handler) handleStartDebugger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartDebuggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("Error parsing start debugger request: %v", err)
		response.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate filepath
	if req.Filepath == "" {
		response.Error(w, "Filepath is required", http.StatusBadRequest)
		return
	}

	// Check if file exists
	if _, err := os.Stat(req.Filepath); os.IsNotExist(err) {
		response.Error(w, "File does not exist", http.StatusBadRequest)
		return
	}

	// Start debugger
	if err := h.debuggerSvc.Start(); err != nil {
		h.logger.Printf("Error starting debugger: %v", err)
		response.Error(w, "Failed to start debugger", http.StatusInternalServerError)
		return
	}

	// Register the debugger output with the WebSocket manager
	h.wsManager.RegisterOutputChannel(h.debuggerSvc.OutputChannel())

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Debugger started successfully",
	})
}

// handleDebuggerCommand handles requests to send commands to the debugger
func (h *Handler) handleDebuggerCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DebuggerCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("Error parsing debugger command request: %v", err)
		response.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate command
	if req.Command == "" {
		response.Error(w, "Command is required", http.StatusBadRequest)
		return
	}

	// Send command to debugger
	if err := h.debuggerSvc.SendCommand(req.Command); err != nil {
		h.logger.Printf("Error sending command to debugger: %v", err)
		response.Error(w, "Failed to send command to debugger", http.StatusInternalServerError)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Command sent successfully",
	})
}

// handleStopDebugger handles requests to stop the debugger
func (h *Handler) handleStopDebugger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Stop the debugger
	if err := h.debuggerSvc.Stop(); err != nil {
		h.logger.Printf("Error stopping debugger: %v", err)
		response.Error(w, "Failed to stop debugger", http.StatusInternalServerError)
		return
	}

	// Unregister the debugger output from the WebSocket manager
	h.wsManager.UnregisterOutputChannel(h.debuggerSvc.OutputChannel())

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Debugger stopped successfully",
	})
}
