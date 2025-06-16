package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/gogdbllm/internal/chat"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxAttempts       int           `yaml:"max_attempts"`
	BaseDelay         time.Duration `yaml:"base_delay"`
	MaxDelay          time.Duration `yaml:"max_delay"`
	Jitter            bool          `yaml:"jitter"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:       3,
		BaseDelay:         time.Second,
		MaxDelay:          30 * time.Second,
		Jitter:            true,
		BackoffMultiplier: 2.0,
	}
}

// RetryManager handles retry logic with exponential backoff
type RetryManager struct {
	config         *RetryConfig
	circuitBreaker *CircuitBreaker
}

// NewRetryManager creates a new retry manager
func NewRetryManager(config *RetryConfig, circuitBreaker *CircuitBreaker) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &RetryManager{
		config:         config,
		circuitBreaker: circuitBreaker,
	}
}

// Execute executes a function with retry logic
func (rm *RetryManager) Execute(ctx context.Context, fn func() error) error {
	if rm.circuitBreaker != nil {
		return rm.circuitBreaker.Call(func() error {
			return rm.executeWithRetry(ctx, fn)
		})
	}

	return rm.executeWithRetry(ctx, fn)
}

// executeWithRetry performs the actual retry logic
func (rm *RetryManager) executeWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < rm.config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !rm.shouldRetry(err, attempt) {
			break
		}

		// Calculate delay for next attempt
		if attempt < rm.config.MaxAttempts-1 {
			delay := rm.calculateDelay(attempt)

			// Wait with context cancellation support
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return lastErr
}

// shouldRetry determines if we should retry based on the error and attempt number
func (rm *RetryManager) shouldRetry(err error, attempt int) bool {
	// Don't retry if we've reached max attempts
	if attempt >= rm.config.MaxAttempts-1 {
		return false
	}

	// Check if it's a provider error with retry information
	if providerErr, ok := err.(*chat.ProviderError); ok {
		return providerErr.Retryable
	}

	// Check for specific HTTP status codes that should be retried
	if httpErr, ok := err.(*HTTPError); ok {
		return rm.isRetryableHTTPStatus(httpErr.StatusCode)
	}

	// Check for network-related errors (strings contain)
	errStr := strings.ToLower(err.Error())
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"timeout",
		"network is unreachable",
		"no such host",
		"temporary failure",
		"service unavailable",
	}

	for _, networkErr := range networkErrors {
		if strings.Contains(errStr, networkErr) {
			return true
		}
	}

	return false
}

// isRetryableHTTPStatus checks if an HTTP status code is retryable
func (rm *RetryManager) isRetryableHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// calculateDelay calculates the delay for the next retry attempt
func (rm *RetryManager) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * (multiplier ^ attempt)
	delay := float64(rm.config.BaseDelay) * math.Pow(rm.config.BackoffMultiplier, float64(attempt))

	// Cap the delay at maxDelay
	if delay > float64(rm.config.MaxDelay) {
		delay = float64(rm.config.MaxDelay)
	}

	// Add jitter if enabled
	if rm.config.Jitter {
		// Add random jitter up to 25% of the delay
		jitter := delay * 0.25 * rand.Float64()
		delay += jitter
	}

	return time.Duration(delay)
}

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

// RetryableError wraps an error to indicate it's retryable
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error) *RetryableError {
	return &RetryableError{Err: err}
}

// IsRetryableError checks if an error is marked as retryable
func IsRetryableError(err error) bool {
	_, ok := err.(*RetryableError)
	return ok
}
