package api

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/gogdbllm/internal/logsession"
)

// GDBExecutor handles execution of GDB commands
type GDBExecutor struct {
	gdbHandler GDBCommandHandler
	mutex      sync.Mutex
}

// GDBExecutionResult contains the results of GDB command execution
type GDBExecutionResult struct {
	Commands       []string
	Outputs        []string
	CombinedOutput string
	Errors         []error
	ExecutionTime  time.Duration
}

// NewGDBExecutor creates a new GDB executor
func NewGDBExecutor(gdbHandler GDBCommandHandler) *GDBExecutor {
	return &GDBExecutor{
		gdbHandler: gdbHandler,
	}
}

// ExecuteCommands executes a list of GDB commands synchronously
func (ge *GDBExecutor) ExecuteCommands(ctx context.Context, commands []string, logger *logsession.SessionLogger) (*GDBExecutionResult, error) {
	if len(commands) == 0 {
		return &GDBExecutionResult{}, nil
	}

	if ge.gdbHandler == nil {
		return nil, fmt.Errorf("GDB handler not available")
	}

	if !ge.gdbHandler.IsRunning() {
		return nil, fmt.Errorf("GDB is not running")
	}

	ge.mutex.Lock()
	defer ge.mutex.Unlock()

	startTime := time.Now()
	result := &GDBExecutionResult{
		Commands: commands,
		Outputs:  make([]string, len(commands)),
		Errors:   make([]error, len(commands)),
	}

	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== EXECUTING %d GDB COMMANDS ===", len(commands)))
	}

	var combinedOutput strings.Builder

	for i, cmd := range commands {
		if logger != nil {
			logger.LogTerminalOutput(fmt.Sprintf("Executing command %d/%d: %s", i+1, len(commands), cmd))
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute command with timeout
		output, err := ge.executeCommandWithTimeout(ctx, cmd, 30*time.Second)

		result.Outputs[i] = output
		result.Errors[i] = err

		if err != nil {
			if logger != nil {
				logger.LogTerminalOutput(fmt.Sprintf("Command failed: %v", err))
			}
			// Continue with other commands even if one fails
		} else {
			if logger != nil {
				logger.LogTerminalOutput(fmt.Sprintf("Command output (%d chars): %s", len(output), ge.truncateForLog(output, 200)))
			}

			if output != "" {
				if combinedOutput.Len() > 0 {
					combinedOutput.WriteString("\n")
				}
				combinedOutput.WriteString(output)
			}
		}
	}

	result.CombinedOutput = combinedOutput.String()
	result.ExecutionTime = time.Since(startTime)

	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== GDB EXECUTION COMPLETED ===\nTotal time: %v\nCombined output: %d chars",
			result.ExecutionTime, len(result.CombinedOutput)))
	}

	return result, nil
}

// executeCommandWithTimeout executes a single command with timeout
func (ge *GDBExecutor) executeCommandWithTimeout(ctx context.Context, cmd string, timeout time.Duration) (string, error) {
	// Create a context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Channel to receive result
	resultChan := make(chan struct {
		output string
		err    error
	}, 1)

	// Execute command in goroutine
	go func() {
		output, err := ge.gdbHandler.ExecuteCommandWithOutput(cmd)
		resultChan <- struct {
			output string
			err    error
		}{output, err}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result.output, result.err
	case <-cmdCtx.Done():
		return "", fmt.Errorf("command timed out after %v: %s", timeout, cmd)
	}
}

// truncateForLog truncates output for logging purposes
func (ge *GDBExecutor) truncateForLog(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	truncated := text[:maxLen]
	// Try to break at a line boundary
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxLen/2 {
		truncated = text[:lastNewline]
	}

	return truncated + "... [truncated]"
}

// GetExecutionSummary returns a summary of the execution result
func (ger *GDBExecutionResult) GetExecutionSummary() string {
	successCount := 0
	for _, err := range ger.Errors {
		if err == nil {
			successCount++
		}
	}

	return fmt.Sprintf("Executed %d commands (%d successful, %d failed) in %v",
		len(ger.Commands), successCount, len(ger.Commands)-successCount, ger.ExecutionTime)
}

// HasErrors returns true if any commands failed
func (ger *GDBExecutionResult) HasErrors() bool {
	for _, err := range ger.Errors {
		if err != nil {
			return true
		}
	}
	return false
}

// GetErrorSummary returns a summary of errors encountered
func (ger *GDBExecutionResult) GetErrorSummary() string {
	if !ger.HasErrors() {
		return "No errors"
	}

	var errorMsgs []string
	for i, err := range ger.Errors {
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("Command %d (%s): %v", i+1, ger.Commands[i], err))
		}
	}

	return strings.Join(errorMsgs, "; ")
}
