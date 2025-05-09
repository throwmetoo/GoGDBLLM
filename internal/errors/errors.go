package errors

import (
	"errors"
	"fmt"
)

// Standard errors
var (
	ErrNotFound             = errors.New("resource not found")
	ErrBadRequest           = errors.New("bad request")
	ErrInternalServer       = errors.New("internal server error")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrTimeout              = errors.New("operation timed out")
	ErrInvalidConfiguration = errors.New("invalid configuration")
	ErrUnsupported          = errors.New("operation not supported")
)

// Domain-specific errors
var (
	ErrGDBNotRunning      = errors.New("GDB is not running")
	ErrGDBCommandFailed   = errors.New("GDB command failed")
	ErrFileUpload         = errors.New("file upload failed")
	ErrLLMAPICall         = errors.New("LLM API call failed")
	ErrInvalidLLMResponse = errors.New("invalid response from LLM")
)

// Wrap wraps an error with additional context
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Is reports whether any error in err's tree matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches target, and if one is found, sets
// target to that error value and returns true. Otherwise, it returns false.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Unwrap returns the underlying wrapped error.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// ErrorCode represents an HTTP status code for an error
type ErrorCode int

// Error codes
const (
	CodeBadRequest     ErrorCode = 400
	CodeUnauthorized   ErrorCode = 401
	CodeForbidden      ErrorCode = 403
	CodeNotFound       ErrorCode = 404
	CodeTimeout        ErrorCode = 408
	CodeInternal       ErrorCode = 500
	CodeNotImplemented ErrorCode = 501
)

// AppError represents an application error with a status code and user-friendly message
type AppError struct {
	Err      error     // Original error
	Code     ErrorCode // HTTP status code
	Message  string    // User-friendly message
	Op       string    // Operation where the error occurred
	LogLevel string    // Log level (info, warn, error)
}

// Error returns the error message
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Err.Error())
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(err error, code ErrorCode, message, op, logLevel string) *AppError {
	return &AppError{
		Err:      err,
		Code:     code,
		Message:  message,
		Op:       op,
		LogLevel: logLevel,
	}
}

// ErrorResponse represents an error response to the client
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code"`
}
