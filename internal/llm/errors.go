package llm

import (
	"errors"
	"fmt"
)

var (
	// ErrUnsupportedProvider is returned when an unsupported provider is specified
	ErrUnsupportedProvider = errors.New("unsupported LLM provider")

	// ErrInvalidAPIKey is returned when an invalid API key is provided
	ErrInvalidAPIKey = errors.New("invalid API key")

	// ErrInvalidModel is returned when an invalid model is specified
	ErrInvalidModel = errors.New("invalid model")

	// ErrEmptyResponse is returned when the LLM returns an empty response
	ErrEmptyResponse = errors.New("empty response from LLM")

	// ErrRequestFailed is returned when the request to the LLM provider fails
	ErrRequestFailed = errors.New("request to LLM provider failed")
)

// APIError represents an error returned by an LLM API
type APIError struct {
	StatusCode int
	Message    string
	Provider   string
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("%s API error (status %d): %s", e.Provider, e.StatusCode, e.Message)
}

// NewAPIError creates a new APIError
func NewAPIError(provider string, statusCode int, message string) *APIError {
	return &APIError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    message,
	}
}
