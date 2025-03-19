package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/throwmetoo/gogdbllm/internal/api/response"
)

// handleUpload handles file uploads
func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set max upload size
	r.Body = http.MaxBytesReader(w, r.Body, h.config.MaxUploadSize)

	// Parse the multipart form
	if err := r.ParseMultipartForm(h.config.MaxUploadSize); err != nil {
		h.logger.Printf("Failed to parse form: %v", err)
		response.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	// Get the file from form data
	file, header, err := r.FormFile("file")
	if err != nil {
		h.logger.Printf("Failed to get file: %v", err)
		response.Error(w, fmt.Sprintf("Failed to get file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type if needed
	// TODO: Add file type validation

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(h.config.UploadDir, 0755); err != nil {
		h.logger.Printf("Failed to create uploads directory: %v", err)
		response.Error(w, fmt.Sprintf("Failed to create uploads directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Create a unique filename to prevent overwriting
	filename := header.Filename
	filepath := filepath.Join(h.config.UploadDir, filename)

	// Create the file
	dst, err := os.Create(filepath)
	if err != nil {
		h.logger.Printf("Failed to create file: %v", err)
		response.Error(w, fmt.Sprintf("Failed to create file: %v", err), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file
	if _, err := io.Copy(dst, file); err != nil {
		h.logger.Printf("Failed to save file: %v", err)
		response.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	// Make the file executable
	if err := os.Chmod(filepath, 0755); err != nil {
		h.logger.Printf("Failed to make file executable: %v", err)
		response.Error(w, fmt.Sprintf("Failed to make file executable: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"filename": filename,
		"filepath": filepath,
	})
}
