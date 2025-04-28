package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"github.com/yourusername/gogdbllm/internal/gdb"
	"github.com/yourusername/gogdbllm/internal/utils"
	"github.com/yourusername/gogdbllm/internal/websocket"
)

// GDBRequest represents the expected JSON payload for starting GDB
type GDBRequest struct {
	Filename string `json:"filename"`
}

// GDBHandler handles GDB-related operations
type GDBHandler struct {
	gdbService   *gdb.GDBService
	hub          *websocket.Hub
	loggerHolder LoggerHolder // Use the interface type defined in file_handler (or move interface)
}

// NewGDBHandler creates a new GDB handler
func NewGDBHandler(hub *websocket.Hub, loggerHolder LoggerHolder) *GDBHandler { // Accept interface
	return &GDBHandler{
		gdbService:   gdb.NewGDBService(),
		hub:          hub,
		loggerHolder: loggerHolder,
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
	// Assuming uploadsDir is accessible or configured elsewhere. For simplicity, using relative path.
	uploadsDir := "uploads" // Define or get from config
	filePath := filepath.Join(uploadsDir, req.Filename)

	// Get current logger
	logger := h.loggerHolder.Get()

	// Start GDB
	if err := h.gdbService.StartGDB(filePath); err != nil {
		http.Error(w, "Failed to start GDB: "+err.Error(), http.StatusInternalServerError)
		if logger != nil {
			logger.LogError(err, "Starting GDB session for "+filePath)
		}
		return
	}

	log.Println("GDB session started for:", filePath)

	// Start a goroutine to receive messages from GDB and broadcast them
	go func() {
		outputChan := h.gdbService.GetOutputChannel()
		for outputBytes := range outputChan {
			rawOutputString := string(outputBytes)
			// Sanitize the string for logging
			sanitizedOutputString := utils.StripAnsiAndControlChars(rawOutputString)

			// Get current logger inside goroutine (it might change)
			currentLogger := h.loggerHolder.Get()
			if currentLogger != nil {
				// Log the sanitized string
				currentLogger.LogTerminalOutput(sanitizedOutputString)
			}
			// Broadcast the original bytes (which might contain ANSI codes for frontend)
			h.hub.Broadcast(outputBytes)
		}
		log.Println("GDB output channel closed for:", filePath)
	}()

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "GDB started successfully",
	})
}

// HandleCommand handles incoming GDB commands from WebSocket clients (received as string)
// Signature changed to satisfy the websocket.GDBHandler interface
func (h *GDBHandler) HandleCommand(cmd string) error { // Changed parameter to string, added error return
	// Get current logger
	logger := h.loggerHolder.Get()
	if err := h.gdbService.SendCommand(cmd); err != nil {
		log.Printf("Error sending command to GDB: %v", err)
		if logger != nil {
			logger.LogError(err, "Sending command to GDB: "+cmd)
		}
		return err // Return the error
	}
	return nil // Return nil on success
}

// IsRunning returns whether GDB is currently running
func (h *GDBHandler) IsRunning() bool {
	return h.gdbService.IsRunning()
}

// ExecuteCommandWithOutput runs a GDB command and returns its output
func (h *GDBHandler) ExecuteCommandWithOutput(cmd string) (string, error) {
	// Get current logger
	logger := h.loggerHolder.Get()

	// Default timeout of 2 seconds
	output, err := h.gdbService.ExecuteCommandWithOutput(cmd, 2)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "ExecuteCommandWithOutput for GDB: "+cmd)
		}
		return "", err
	}

	// Log that we executed the command
	if logger != nil {
		logger.LogTerminalOutput("(LLM-Capture) " + cmd)
	}

	return output, nil
}
