package debugger

import (
	"errors"
	"fmt"
)

var (
	// ErrNotRunning is returned when trying to interact with a debugger that is not running
	ErrNotRunning = errors.New("debugger is not running")

	// ErrAlreadyRunning is returned when trying to start a debugger that is already running
	ErrAlreadyRunning = errors.New("debugger is already running")

	// ErrInvalidCommand is returned when an invalid command is sent to the debugger
	ErrInvalidCommand = errors.New("invalid debugger command")

	// ErrProcessFailed is returned when the debugger process fails to start or crashes
	ErrProcessFailed = errors.New("debugger process failed")
)

// CommandError represents an error that occurred while executing a debugger command
type CommandError struct {
	Command string
	Err     error
}

// Error implements the error interface
func (e *CommandError) Error() string {
	return fmt.Sprintf("command '%s' failed: %v", e.Command, e.Err)
}

// Unwrap returns the underlying error
func (e *CommandError) Unwrap() error {
	return e.Err
}

// NewCommandError creates a new CommandError
func NewCommandError(command string, err error) *CommandError {
	return &CommandError{
		Command: command,
		Err:     err,
	}
}
