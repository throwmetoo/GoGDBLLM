package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
)

// SimpleChatHandler provides a clean, maintainable chat interface
type SimpleChatHandler struct {
	processor *ChatProcessor
}

// NewSimpleChatHandler creates a new simple chat handler
func NewSimpleChatHandler(
	settingsManager *settings.Manager,
	loggerHolder LoggerHolder,
	gdbHandler GDBCommandHandler,
) *SimpleChatHandler {
	return &SimpleChatHandler{
		processor: NewChatProcessor(settingsManager, loggerHolder, gdbHandler),
	}
}

// HandleChat handles incoming chat requests with the new architecture
func (sch *SimpleChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var chatReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Log user input
	logger := sch.processor.loggerHolder.Get()
	if logger != nil {
		logContext := make([]logsession.ContextItem, len(chatReq.SentContext))
		for i, apiItem := range chatReq.SentContext {
			logContext[i] = logsession.ContextItem{
				Type:        apiItem.Type,
				Description: apiItem.Description,
				Content:     apiItem.Content,
			}
		}
		logger.LogUserChat(logContext, chatReq.Message)
	}

	// Process the chat request using the new architecture
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second) // Extended timeout for GDB operations
	defer cancel()

	result, err := sch.processor.ProcessChat(ctx, &chatReq)
	if err != nil {
		http.Error(w, "Chat processing failed", http.StatusInternalServerError)
		if logger != nil {
			logger.LogError(err, "Chat processing failed")
		}
		return
	}

	// Handle processing errors (non-fatal)
	if result.Error != nil {
		if logger != nil {
			logger.LogError(result.Error, "Chat processing encountered errors")
		}
		// Continue with partial results
	}

	// Log the final response
	if logger != nil {
		logger.LogLLMResponse(result.FinalText)
	}

	// Send response
	chatResp := ChatResponse{Response: result.FinalText}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chatResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Encoding chat response")
		}
	}
}
