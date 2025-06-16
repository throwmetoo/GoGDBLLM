package context

import (
	"fmt"
	"strings"

	"github.com/yourusername/gogdbllm/internal/api"
	"github.com/yourusername/gogdbllm/internal/chat"
)

// Config holds context management configuration
type Config struct {
	Enabled                bool `yaml:"enabled"`
	MaxTokens              int  `yaml:"max_tokens"`
	PriorityRecentMessages int  `yaml:"priority_recent_messages"`
	CompressionThreshold   int  `yaml:"compression_threshold"`
	PreserveSystemContext  bool `yaml:"preserve_system_context"`
}

// DefaultConfig returns default context management configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:                false,
		MaxTokens:              4000,
		PriorityRecentMessages: 10,
		CompressionThreshold:   100,
		PreserveSystemContext:  true,
	}
}

// Manager handles context management and trimming
type Manager struct {
	config *Config
}

// New creates a new context manager
func New(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	return &Manager{
		config: config,
	}
}

// ProcessRequest processes a chat request and manages context
func (cm *Manager) ProcessRequest(request *chat.ChatRequest) (*chat.ChatRequest, bool, error) {
	if !cm.config.Enabled {
		return request, false, nil
	}

	// Estimate token count for the request
	tokenCount := cm.estimateTokens(request)

	// If within limits, return as-is
	if tokenCount <= cm.config.MaxTokens {
		return request, false, nil
	}

	// Create a copy of the request for modification
	processedRequest := *request
	processedRequest.History = make([]api.ChatMessage, len(request.History))
	copy(processedRequest.History, request.History)

	// Trim context to fit within limits
	trimmed, err := cm.trimContext(&processedRequest)
	if err != nil {
		return request, false, err
	}

	return &processedRequest, trimmed, nil
}

// trimContext trims the context to fit within token limits
func (cm *Manager) trimContext(request *chat.ChatRequest) (bool, error) {
	var trimmed bool

	// Step 1: Compress old messages if above threshold
	if len(request.History) > cm.config.CompressionThreshold {
		compressed, err := cm.compressOldMessages(request)
		if err != nil {
			return false, err
		}
		trimmed = trimmed || compressed
	}

	// Step 2: Remove old messages if still over limit
	currentTokens := cm.estimateTokens(request)
	if currentTokens > cm.config.MaxTokens {
		removed := cm.removeOldMessages(request)
		trimmed = trimmed || removed
	}

	// Step 3: Trim sent context if still over limit
	currentTokens = cm.estimateTokens(request)
	if currentTokens > cm.config.MaxTokens {
		contextTrimmed := cm.trimSentContext(request)
		trimmed = trimmed || contextTrimmed
	}

	return trimmed, nil
}

// compressOldMessages compresses older messages in the history
func (cm *Manager) compressOldMessages(request *chat.ChatRequest) (bool, error) {
	if len(request.History) <= cm.config.PriorityRecentMessages {
		return false, nil
	}

	// Identify messages to compress (all except recent priority messages)
	compressCount := len(request.History) - cm.config.PriorityRecentMessages
	if compressCount <= 0 {
		return false, nil
	}

	// Group messages for compression
	var messagesToCompress []api.ChatMessage
	for i := 0; i < compressCount; i++ {
		messagesToCompress = append(messagesToCompress, request.History[i])
	}

	// Create compressed summary
	summary, err := cm.createSummary(messagesToCompress)
	if err != nil {
		return false, err
	}

	// Replace compressed messages with summary
	compressedMessage := api.ChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("[CONVERSATION SUMMARY: %s]", summary),
	}

	// Rebuild history with compressed message + recent messages
	newHistory := []api.ChatMessage{compressedMessage}
	newHistory = append(newHistory, request.History[compressCount:]...)
	request.History = newHistory

	return true, nil
}

// removeOldMessages removes old messages from history
func (cm *Manager) removeOldMessages(request *chat.ChatRequest) bool {
	if len(request.History) <= cm.config.PriorityRecentMessages {
		return false
	}

	// Keep only recent priority messages
	keepCount := cm.config.PriorityRecentMessages
	if keepCount > len(request.History) {
		keepCount = len(request.History)
	}

	startIndex := len(request.History) - keepCount
	request.History = request.History[startIndex:]

	return true
}

// trimSentContext trims the sent context if needed
func (cm *Manager) trimSentContext(request *chat.ChatRequest) bool {
	if len(request.SentContext) == 0 {
		return false
	}

	// Sort context by importance (keep file content, trim larger items first)
	prioritized := cm.prioritizeContext(request.SentContext)

	// Calculate how much we need to trim
	currentTokens := cm.estimateTokens(request)
	targetTokens := cm.config.MaxTokens
	tokensToSave := currentTokens - targetTokens

	var newContext []api.ContextItem
	tokensSaved := 0

	for _, item := range prioritized {
		itemTokens := cm.estimateContextItemTokens(item)

		if tokensSaved < tokensToSave && itemTokens > 500 { // Only trim large items
			// Create truncated version
			truncated := item
			truncated.Content = cm.truncateContent(item.Content, 200)
			newContext = append(newContext, truncated)
			tokensSaved += itemTokens - cm.estimateContextItemTokens(truncated)
		} else {
			newContext = append(newContext, item)
		}

		if tokensSaved >= tokensToSave {
			break
		}
	}

	if len(newContext) < len(request.SentContext) || tokensSaved > 0 {
		request.SentContext = newContext
		return true
	}

	return false
}

// prioritizeContext sorts context items by importance
func (cm *Manager) prioritizeContext(context []api.ContextItem) []api.ContextItem {
	// Create a copy to avoid modifying original
	prioritized := make([]api.ContextItem, len(context))
	copy(prioritized, context)

	// Simple priority: files first, then smaller items
	// In a real implementation, you might want more sophisticated prioritization
	return prioritized
}

// createSummary creates a summary of multiple messages
func (cm *Manager) createSummary(messages []api.ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Previous conversation with %d messages. ", len(messages)))

	// Extract key topics and themes
	userMessages := 0
	assistantMessages := 0

	for _, msg := range messages {
		if msg.Role == "user" {
			userMessages++
		} else if msg.Role == "assistant" {
			assistantMessages++
		}
	}

	summary.WriteString(fmt.Sprintf("User asked %d questions, assistant provided %d responses. ",
		userMessages, assistantMessages))

	// Add summary of last few important messages
	if len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		summary.WriteString(fmt.Sprintf("Last topic: %s",
			cm.extractTopic(lastMessage.Content)))
	}

	return summary.String(), nil
}

// extractTopic extracts a topic from message content
func (cm *Manager) extractTopic(content string) string {
	// Simple topic extraction - take first 50 characters
	if len(content) > 50 {
		return content[:50] + "..."
	}
	return content
}

// truncateContent truncates content to a specified character limit
func (cm *Manager) truncateContent(content string, limit int) string {
	if len(content) <= limit {
		return content
	}

	truncated := content[:limit]

	// Try to break at word boundary
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > limit/2 {
		truncated = content[:lastSpace]
	}

	return truncated + "... [truncated]"
}

// estimateTokens estimates the total token count for a request
func (cm *Manager) estimateTokens(request *chat.ChatRequest) int {
	tokens := 0

	// Estimate tokens for message
	tokens += cm.estimateTextTokens(request.Message)

	// Estimate tokens for history
	for _, msg := range request.History {
		tokens += cm.estimateTextTokens(msg.Content)
	}

	// Estimate tokens for sent context
	for _, ctx := range request.SentContext {
		tokens += cm.estimateContextItemTokens(ctx)
	}

	return tokens
}

// estimateTextTokens estimates token count for text (rough approximation)
func (cm *Manager) estimateTextTokens(text string) int {
	// Rough approximation: 1 token ≈ 4 characters for English text
	return len(text) / 4
}

// estimateContextItemTokens estimates tokens for a context item
func (cm *Manager) estimateContextItemTokens(item api.ContextItem) int {
	tokens := cm.estimateTextTokens(item.Description)
	tokens += cm.estimateTextTokens(item.Content)
	return tokens
}

// GetStats returns context management statistics
func (cm *Manager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"enabled":                  cm.config.Enabled,
		"max_tokens":               cm.config.MaxTokens,
		"priority_recent_messages": cm.config.PriorityRecentMessages,
		"compression_threshold":    cm.config.CompressionThreshold,
	}
}

// IsEnabled returns whether context management is enabled
func (cm *Manager) IsEnabled() bool {
	return cm.config.Enabled
}
