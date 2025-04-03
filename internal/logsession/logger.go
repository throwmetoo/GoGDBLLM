package logsession

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ContextItem represents a piece of context sent to the LLM (defined locally)
type ContextItem struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Content     string `json:"content,omitempty"`
}

const logDir = "./logs"

// SessionLogger handles writing session logs to a file in JSON Lines format.
type SessionLogger struct {
	file      *os.File
	encoder   *json.Encoder
	mutex     sync.Mutex
	sessionID string
}

// NewSessionLogger creates a new logger for a session.
func NewSessionLogger(sessionID string) (*SessionLogger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory '%s': %w", logDir, err)
	}

	logFileName := filepath.Join(logDir, fmt.Sprintf("%s.log", sessionID))
	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file '%s': %w", logFileName, err)
	}

	logger := &SessionLogger{
		file:      file,
		encoder:   json.NewEncoder(file), // Use a JSON encoder
		sessionID: sessionID,
	}

	// No header needed for JSON Lines
	log.Printf("Session log started (JSON Lines): %s", logFileName) // Log to console

	return logger, nil
}

// LogEvent creates a structured log entry and writes it as a JSON line.
func (l *SessionLogger) LogEvent(level string, eventType string, message string, details map[string]interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	entry := map[string]interface{}{
		"timestamp":  time.Now().Format(time.RFC3339Nano),
		"level":      level,
		"session.id": l.sessionID,
		"event.type": eventType,
		"message":    message,
	}

	// Merge details into the entry
	for k, v := range details {
		entry[k] = v
	}

	if err := l.encoder.Encode(entry); err != nil {
		// Fallback to console logging if file write fails
		log.Printf("ERROR writing JSON log entry to %s: %v | Entry: %+v", l.file.Name(), err, entry)
	}
}

// LogUserChat logs a user chat message and its context.
func (l *SessionLogger) LogUserChat(context []ContextItem, message string) {
	details := map[string]interface{}{
		"user.message": message,
	}
	if len(context) > 0 {
		details["user.context"] = context // Log context as a JSON array
	}
	l.LogEvent("INFO", "user.input", "User submitted chat message", details)
}

// LogLLMRequestData logs the data being sent to the LLM.
func (l *SessionLogger) LogLLMRequestData(provider, model, fullMessage string) {
	l.LogEvent("INFO", "llm.request", "Sending request to LLM", map[string]interface{}{
		"llm.provider":        provider,
		"llm.model":           model,
		"llm.request.message": fullMessage, // Contains user query + injected context
	})
}

// LogLLMResponse logs the response received from the LLM.
func (l *SessionLogger) LogLLMResponse(response string) {
	l.LogEvent("INFO", "llm.response", "Received response from LLM", map[string]interface{}{
		"llm.response.body": response,
	})
}

// LogTerminalOutput logs output from the terminal/GDB.
func (l *SessionLogger) LogTerminalOutput(output string) {
	l.LogEvent("INFO", "gdb.output", "Received output from GDB", map[string]interface{}{
		"gdb.output": output,
	})
}

// LogError logs an error that occurred.
func (l *SessionLogger) LogError(err error, contextMsg string) {
	if err == nil {
		return
	}
	l.LogEvent("ERROR", "error", contextMsg, map[string]interface{}{
		"error.message": err.Error(),
		// Optionally add stack trace here if possible
	})
}

// Close closes the log file.
func (l *SessionLogger) Close() {
	if l.file != nil {
		log.Printf("Closing session log: %s", l.file.Name())
		l.file.Close()
	}
}
