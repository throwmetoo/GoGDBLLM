package logsession

import (
	"sync"
)

// LoggerHolderImpl provides thread-safe access to a shared SessionLogger instance
type LoggerHolderImpl struct {
	logger *SessionLogger
	mutex  sync.RWMutex
}

// NewLoggerHolder creates a new LoggerHolder instance
func NewLoggerHolder() *LoggerHolderImpl {
	return &LoggerHolderImpl{}
}

// Set sets a new logger, replacing any existing one
func (h *LoggerHolderImpl) Set(newLogger *SessionLogger) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Close the old logger if it exists
	if h.logger != nil {
		h.logger.Close()
	}

	h.logger = newLogger
}

// Get retrieves the current logger (may be nil if not set)
func (h *LoggerHolderImpl) Get() *SessionLogger {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.logger
}
