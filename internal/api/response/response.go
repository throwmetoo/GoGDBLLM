package response

import (
	"encoding/json"
	"net/http"
)

// Error sends an error response
func Error(w http.ResponseWriter, message string, statusCode int) {
	JSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// JSON sends a JSON response
func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
