package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/yourusername/gogdbllm/internal/config"
)

var (
	// Log is the global logger instance
	Log zerolog.Logger
)

// Init initializes the logger based on configuration
func Init(cfg *config.Config) error {
	// Set up zerolog
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Set global log level
	level, err := zerolog.ParseLevel(cfg.Logs.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(cfg.Logs.Directory, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create application log file
	appLogFile, err := os.OpenFile(
		filepath.Join(cfg.Logs.Directory, "application.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to open application log file: %w", err)
	}

	// Configure writers - we'll log to both stdout and file
	var writer io.Writer

	// Console writer for development mode
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	// Use both console and file
	writer = io.MultiWriter(consoleWriter, appLogFile)

	// Create logger
	Log = zerolog.New(writer).With().Timestamp().Caller().Logger()

	Log.Info().
		Str("log_level", level.String()).
		Bool("json_format", cfg.Logs.JSONFormat).
		Msg("Logger initialized")

	return nil
}

// Shutdown gracefully shuts down the logger
func Shutdown() {
	// Nothing to do for zerolog shutdown, but this gives us a hook if we need it later
}

// NewSessionLogger creates a logger for a specific debugging session
func NewSessionLogger(sessionID string, cfg *config.Config) (zerolog.Logger, error) {
	// Create session log file
	sessionLogFile, err := os.OpenFile(
		filepath.Join(cfg.Logs.Directory, fmt.Sprintf("%s.log", sessionID)),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf("failed to create session log file: %w", err)
	}

	// Get the base logger in one of two formats based on config
	var sessionLogger zerolog.Logger
	if cfg.Logs.JSONFormat {
		sessionLogger = zerolog.New(sessionLogFile).With().
			Timestamp().
			Str("session_id", sessionID).
			Logger()
	} else {
		sessionWriter := zerolog.ConsoleWriter{
			Out:        sessionLogFile,
			TimeFormat: time.RFC3339,
			NoColor:    true, // No colors in log files
		}
		sessionLogger = zerolog.New(sessionWriter).With().
			Timestamp().
			Str("session_id", sessionID).
			Logger()
	}

	return sessionLogger, nil
}
