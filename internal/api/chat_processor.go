package api

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
)

// ChatProcessor handles the complete chat processing pipeline
type ChatProcessor struct {
	settingsManager *settings.Manager
	loggerHolder    LoggerHolder
	gdbHandler      GDBCommandHandler
	responseParser  *ResponseParser
	gdbExecutor     *GDBExecutor
	llmClient       *LLMClient
}

// ProcessingResult contains the final result of chat processing
type ProcessingResult struct {
	FinalText     string
	ExecutedCmds  []string
	GDBOutput     string
	Error         error
	ProcessingLog []string
}

// ProcessingContext holds context for a single chat processing session
type ProcessingContext struct {
	RequestID     string
	OriginalReq   *ChatRequest
	Settings      settings.Settings
	Logger        *logsession.SessionLogger
	ProcessingLog []string
}

// NewChatProcessor creates a new chat processor
func NewChatProcessor(
	settingsManager *settings.Manager,
	loggerHolder LoggerHolder,
	gdbHandler GDBCommandHandler,
) *ChatProcessor {
	return &ChatProcessor{
		settingsManager: settingsManager,
		loggerHolder:    loggerHolder,
		gdbHandler:      gdbHandler,
		responseParser:  NewResponseParser(),
		gdbExecutor:     NewGDBExecutor(gdbHandler),
		llmClient:       NewLLMClient(settingsManager),
	}
}

// ProcessChat handles the complete chat processing pipeline
func (cp *ChatProcessor) ProcessChat(ctx context.Context, req *ChatRequest) (*ProcessingResult, error) {
	// Initialize processing context
	procCtx := &ProcessingContext{
		RequestID:     cp.generateRequestID(),
		OriginalReq:   req,
		Settings:      cp.settingsManager.GetSettings(),
		Logger:        cp.loggerHolder.Get(),
		ProcessingLog: []string{},
	}

	cp.logStep(procCtx, fmt.Sprintf("Starting chat processing - RequestID: %s", procCtx.RequestID))

	// Step 1: Get initial LLM response
	initialResponse, err := cp.llmClient.SendRequest(ctx, req, procCtx.Settings, procCtx.Logger)
	if err != nil {
		return &ProcessingResult{Error: fmt.Errorf("initial LLM request failed: %w", err)}, nil
	}

	cp.logStep(procCtx, fmt.Sprintf("Received initial LLM response: %d chars", len(initialResponse)))

	// Step 2: Parse the response
	parsedResponse, err := cp.responseParser.ParseResponse(initialResponse, procCtx.Logger)
	if err != nil {
		return &ProcessingResult{Error: fmt.Errorf("response parsing failed: %w", err)}, nil
	}

	cp.logStep(procCtx, fmt.Sprintf("Parsed response - Text: %d chars, Commands: %d, WaitForOutput: %v",
		len(parsedResponse.Text), len(parsedResponse.GDBCommands), parsedResponse.WaitForOutput))

	// Step 3: Execute GDB commands if present
	result := &ProcessingResult{
		FinalText:     parsedResponse.Text,
		ExecutedCmds:  parsedResponse.GDBCommands,
		ProcessingLog: procCtx.ProcessingLog,
	}

	if len(parsedResponse.GDBCommands) > 0 && cp.gdbHandler != nil && cp.gdbHandler.IsRunning() {
		gdbResult, err := cp.gdbExecutor.ExecuteCommands(ctx, parsedResponse.GDBCommands, procCtx.Logger)
		if err != nil {
			cp.logStep(procCtx, fmt.Sprintf("GDB execution failed: %v", err))
			// Don't fail the whole request, just log the error
		} else {
			result.GDBOutput = gdbResult.CombinedOutput
			cp.logStep(procCtx, fmt.Sprintf("GDB commands executed - Output: %d chars", len(gdbResult.CombinedOutput)))

			// Step 4: Send follow-up request if waitForOutput is true
			if parsedResponse.WaitForOutput && gdbResult.CombinedOutput != "" {
				followupText, err := cp.processFollowup(ctx, procCtx, gdbResult.CombinedOutput)
				if err != nil {
					cp.logStep(procCtx, fmt.Sprintf("Follow-up processing failed: %v", err))
					// Keep original text if follow-up fails
				} else {
					result.FinalText = followupText
					cp.logStep(procCtx, fmt.Sprintf("Using follow-up response: %d chars", len(followupText)))
				}
			}
		}
	} else if len(parsedResponse.GDBCommands) > 0 {
		cp.logStep(procCtx, "GDB commands present but GDB is not running")
		result.FinalText += "\n\n(Note: GDB is not running, cannot execute commands)"
	}

	cp.logStep(procCtx, "Chat processing completed successfully")
	result.ProcessingLog = procCtx.ProcessingLog
	return result, nil
}

// processFollowup handles the follow-up request with GDB output
func (cp *ChatProcessor) processFollowup(ctx context.Context, procCtx *ProcessingContext, gdbOutput string) (string, error) {
	cp.logStep(procCtx, "Processing follow-up request with GDB output")

	// Create follow-up request with GDB output as context
	followupReq := *procCtx.OriginalReq
	followupReq.SentContext = append(followupReq.SentContext, ContextItem{
		Type:        "command_output",
		Description: "GDB Command Output",
		Content:     gdbOutput,
	})

	// Send follow-up request
	followupResponse, err := cp.llmClient.SendRequest(ctx, &followupReq, procCtx.Settings, procCtx.Logger)
	if err != nil {
		return "", fmt.Errorf("follow-up LLM request failed: %w", err)
	}

	cp.logStep(procCtx, fmt.Sprintf("Received follow-up response: %d chars", len(followupResponse)))

	// Parse follow-up response
	parsedFollowup, err := cp.responseParser.ParseResponse(followupResponse, procCtx.Logger)
	if err != nil {
		cp.logStep(procCtx, fmt.Sprintf("Follow-up parsing failed, using raw response: %v", err))
		return followupResponse, nil // Use raw response if parsing fails
	}

	return parsedFollowup.Text, nil
}

// logStep adds a step to the processing log
func (cp *ChatProcessor) logStep(ctx *ProcessingContext, message string) {
	timestamp := time.Now().Format("15:04:05.000")
	logMessage := fmt.Sprintf("[%s] %s", timestamp, message)
	ctx.ProcessingLog = append(ctx.ProcessingLog, logMessage)

	if ctx.Logger != nil {
		ctx.Logger.LogTerminalOutput(logMessage)
	}
}

// generateRequestID generates a unique request ID
func (cp *ChatProcessor) generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
