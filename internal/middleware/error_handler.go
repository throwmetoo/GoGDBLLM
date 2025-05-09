package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/yourusername/gogdbllm/internal/errors"
	"github.com/yourusername/gogdbllm/internal/logger"
)

// ErrorHandlerMiddleware wraps http handlers with consistent error handling
func ErrorHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a response writer wrapper to capture status code
		rwWrapper := newResponseWriterWrapper(w)

		// Call the next handler
		next.ServeHTTP(rwWrapper, r)

		// Handle errors based on status code
		if rwWrapper.statusCode >= 400 {
			logErrorResponse(r, rwWrapper.statusCode)
		}
	})
}

// WithErrorHandling wraps a handler function with error handling
func WithErrorHandling(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set JSON content type
		w.Header().Set("Content-Type", "application/json")

		// Call the handler and handle any returned errors
		if err := handler(w, r); err != nil {
			handleError(w, r, err)
		}
	}
}

// handleError processes errors and returns appropriate responses
func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *errors.AppError
	var statusCode int
	var message string

	// Check if it's an application error
	if errors.As(err, &appErr) {
		statusCode = int(appErr.Code)
		message = appErr.Message

		// Log the error with appropriate level
		logAppError(r, appErr)
	} else {
		// For standard errors, try to determine the type
		switch {
		case errors.Is(err, errors.ErrNotFound):
			statusCode = 404
			message = "Resource not found"
		case errors.Is(err, errors.ErrBadRequest):
			statusCode = 400
			message = "Bad request"
		case errors.Is(err, errors.ErrUnauthorized):
			statusCode = 401
			message = "Unauthorized"
		case errors.Is(err, errors.ErrForbidden):
			statusCode = 403
			message = "Forbidden"
		case errors.Is(err, errors.ErrTimeout):
			statusCode = 408
			message = "Request timeout"
		default:
			// Default to internal server error
			statusCode = 500
			message = "Internal server error"
		}

		// Log the error
		logger.Log.Error().
			Err(err).
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Int("status", statusCode).
			Msg("Request error")
	}

	// Send JSON response
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errors.ErrorResponse{
		Success: false,
		Error:   message,
		Code:    statusCode,
	})
}

// logAppError logs an application error with the appropriate level
func logAppError(r *http.Request, appErr *errors.AppError) {
	// Create a log event with the appropriate level
	var event *zerolog.Event

	switch appErr.LogLevel {
	case "debug":
		event = logger.Log.Debug()
	case "info":
		event = logger.Log.Info()
	case "warn":
		event = logger.Log.Warn()
	default:
		event = logger.Log.Error()
	}

	// Add error details and log
	event.
		Err(appErr.Err).
		Str("operation", appErr.Op).
		Str("path", r.URL.Path).
		Str("method", r.Method).
		Int("status", int(appErr.Code)).
		Msg(appErr.Message)
}

// logErrorResponse logs error responses based on status code
func logErrorResponse(r *http.Request, statusCode int) {
	if statusCode >= 500 {
		logger.Log.Error().
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Int("status", statusCode).
			Msg("Server error response")
	} else {
		logger.Log.Info().
			Str("path", r.URL.Path).
			Str("method", r.Method).
			Int("status", statusCode).
			Msg("Client error response")
	}
}

// responseWriterWrapper wraps a http.ResponseWriter to capture the status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// newResponseWriterWrapper creates a new response writer wrapper
func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{w, http.StatusOK}
}

// WriteHeader captures the status code and passes it to the wrapped writer
func (rww *responseWriterWrapper) WriteHeader(code int) {
	rww.statusCode = code
	rww.ResponseWriter.WriteHeader(code)
}
