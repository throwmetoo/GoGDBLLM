package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"github.com/yourusername/gogdbllm/internal/gdb"
	"github.com/yourusername/gogdbllm/internal/websocket"
)

// GDBRequest represents a request to start GDB
type GDBRequest struct {
	Filename string `json:"filename"`
}

// GDBHandler handles GDB-related operations
type GDBHandler struct {
	gdbService *gdb.GDBService
	hub        *websocket.Hub
}

// NewGDBHandler creates a new GDB handler
func NewGDBHandler(hub *websocket.Hub) *GDBHandler {
	return &GDBHandler{
		gdbService: gdb.NewGDBService(),
		hub:        hub,
	}
}

// HandleStartGDB handles requests to start GDB
func (h *GDBHandler) HandleStartGDB(w http.ResponseWriter, r *http.Request) {
	var req GDBRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Construct the full path to the executable
	filePath := filepath.Join("uploads", req.Filename)

	// Start GDB
	if err := h.gdbService.StartGDB(filePath); err != nil {
		http.Error(w, "Failed to start GDB: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Start a goroutine to receive messages from GDB and broadcast them
	go func() {
		outputChan := h.gdbService.GetOutputChannel()
		for output := range outputChan {
			h.hub.Broadcast(output)
		}
	}()

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"status": "GDB started successfully",
		},
	})
}

// HandleCommand handles a command sent to GDB
func (h *GDBHandler) HandleCommand(cmd string) error {
	if err := h.gdbService.SendCommand(cmd); err != nil {
		log.Printf("Error sending command to GDB: %v", err)
		return err
	}
	return nil
}
