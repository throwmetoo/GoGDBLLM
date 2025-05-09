package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/gogdbllm/internal/config"
	"github.com/yourusername/gogdbllm/internal/logsession" // Import logsession
)

// Define SharedLogger interface locally for dependency inversion (optional but good practice)
type LoggerHolder interface {
	Set(newLogger *logsession.SessionLogger)
	Get() *logsession.SessionLogger
}

// FileHandler handles file uploads
type FileHandler struct {
	uploadsDir   string
	loggerHolder LoggerHolder // Use the interface type
}

// NewFileHandler creates a new file handler
func NewFileHandler(cfg *config.Config, loggerHolder LoggerHolder) *FileHandler { // Use config
	return &FileHandler{
		uploadsDir:   cfg.Uploads.Directory,
		loggerHolder: loggerHolder,
	}
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HandleUpload handles file upload requests
func (h *FileHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// Always set JSON content type first
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Method not allowed"})
		return
	}

	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max file size
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Unable to parse form: " + err.Error()})
		return
	}

	// Get the file from the form data
	file, handler, err := r.FormFile("executable")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Unable to get file from form: " + err.Error()})
		return
	}
	defer file.Close()

	// Sanitize filename
	sanitizedFilename := sanitizeFilename(handler.Filename)
	if sanitizedFilename == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Invalid filename"})
		return
	}

	// Create the uploads directory if it doesn't exist
	if err := os.MkdirAll(h.uploadsDir, 0755); err != nil {
		log.Printf("Error creating uploads directory: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Unable to create uploads directory"})
		return
	}

	// Create the destination file path
	dstPath := filepath.Join(h.uploadsDir, sanitizedFilename)

	// Create the destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Printf("Error creating destination file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Unable to create the file for writing"})
		return
	}
	defer dst.Close()

	// Copy the uploaded file data to the destination file
	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("Error copying uploaded file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{Success: false, Error: "Unable to save file"})
		return
	}

	// --- Start New Log Session ---
	uploadTime := time.Now().Format("20060102_150405")
	sessionID := fmt.Sprintf("%s_%s", uploadTime, sanitizedFilename)

	newLogger, err := logsession.NewSessionLogger(sessionID)
	if err != nil {
		// Log to console, but don't fail the upload entirely
		log.Printf("CRITICAL: Failed to create new session logger for %s: %v", sessionID, err)
		// Respond with success=false but indicate the underlying issue
		w.WriteHeader(http.StatusInternalServerError) // Use 500, as logging is critical
		json.NewEncoder(w).Encode(Response{Success: false, Error: "File uploaded but failed to start logging session"})
		return
	} else {
		h.loggerHolder.Set(newLogger) // Set the new logger, implicitly closes the old one
		log.Printf("Started new log session: %s", sessionID)
	}
	// --- End New Log Session ---

	// Send success response (use Response struct for consistency)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"message":  "File uploaded successfully",
			"filename": sanitizedFilename,
		},
	})

	log.Printf("File uploaded successfully: %s", sanitizedFilename)
}

// sanitizeFilename removes potentially unsafe characters from a filename.
func sanitizeFilename(filename string) string {
	// Basic sanitization: replace slashes and dots (except the last one for extension)
	name := strings.ReplaceAll(filename, "..", "") // Avoid directory traversal
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	// Allow alphanumeric, underscores, hyphens, and a single dot for extension
	// This is a simplified example; more robust sanitization might be needed
	// depending on security requirements.
	// A better approach might be a whitelist of allowed characters.
	return name
}
