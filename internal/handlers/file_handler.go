package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// FileHandler handles file upload operations
type FileHandler struct {
	uploadDir string
}

// NewFileHandler creates a new file handler
func NewFileHandler(uploadDir string) *FileHandler {
	return &FileHandler{
		uploadDir: uploadDir,
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
	// Set JSON content type header
	w.Header().Set("Content-Type", "application/json")

	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Failed to parse form: " + err.Error(),
		})
		return
	}

	// Get the file from form data
	file, header, err := r.FormFile("file")
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Failed to get file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(h.uploadDir, 0755); err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Failed to create uploads directory: " + err.Error(),
		})
		return
	}

	// Create the file
	filePath := filepath.Join(h.uploadDir, header.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Failed to create file: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	// Copy the uploaded file
	if _, err := io.Copy(dst, file); err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Failed to save file: " + err.Error(),
		})
		return
	}

	// Make the file executable
	if err := os.Chmod(filePath, 0755); err != nil {
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "Failed to make file executable: " + err.Error(),
		})
		return
	}

	// Return success
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Data: map[string]string{
			"filename": header.Filename,
			"path":     filePath,
		},
	})
}
