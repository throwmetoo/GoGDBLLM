package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Level represents the severity level of a log message
type Level int

const (
	// DEBUG level for detailed troubleshooting information
	DEBUG Level = iota
	// INFO level for general operational information
	INFO
	// WARN level for potentially harmful situations
	WARN
	// ERROR level for error events that might still allow the application to continue
	ERROR
	// FATAL level for severe error events that will lead the application to abort
	FATAL
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger is a custom logger with support for different log levels
type Logger struct {
	logger *log.Logger
	level  Level
}

// NewLogger creates a new logger with the specified output and level
func NewLogger(out io.Writer, prefix string, level Level) *Logger {
	return &Logger{
		logger: log.New(out, prefix, log.LstdFlags),
		level:  level,
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// log logs a message at the specified level
func (l *Logger) log(level Level, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Format the message
	msg := fmt.Sprintf(format, v...)

	// Log with level and caller information
	l.logger.Printf("[%s] %s:%d: %s", level.String(), filepath.Base(file), line, msg)

	// If fatal, exit the program
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(FATAL, format, v...)
}
