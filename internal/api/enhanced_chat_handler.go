package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
)

// EnhancedChatHandler provides advanced chat handling with retry logic, caching, and monitoring
type EnhancedChatHandler struct {
	settingsManager *settings.Manager
	loggerHolder    LoggerHolder
	gdbHandler      GDBCommandHandler

	// Enhanced features
	cache           *ResponseCache
	metrics         *MetricsCollector
	retryManager    *RetryManager
	circuitBreakers map[string]*CircuitBreaker

	// Configuration
	config *EnhancedConfig
	mutex  sync.RWMutex
}

// EnhancedConfig holds configuration for enhanced features
type EnhancedConfig struct {
	CacheEnabled            bool          `yaml:"cache_enabled"`
	CacheTTL                time.Duration `yaml:"cache_ttl"`
	CacheMaxSize            int           `yaml:"cache_max_size"`
	ContextEnabled          bool          `yaml:"context_enabled"`
	MaxTokens               int           `yaml:"max_tokens"`
	PriorityRecentMessages  int           `yaml:"priority_recent_messages"`
	RetryMaxAttempts        int           `yaml:"retry_max_attempts"`
	RetryBaseDelay          time.Duration `yaml:"retry_base_delay"`
	RetryMaxDelay           time.Duration `yaml:"retry_max_delay"`
	CircuitBreakerThreshold int           `yaml:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `yaml:"circuit_breaker_timeout"`
}

// DefaultEnhancedConfig returns default configuration
func DefaultEnhancedConfig() *EnhancedConfig {
	return &EnhancedConfig{
		CacheEnabled:            false,
		CacheTTL:                time.Hour,
		CacheMaxSize:            1000,
		ContextEnabled:          false,
		MaxTokens:               4000,
		PriorityRecentMessages:  10,
		RetryMaxAttempts:        3,
		RetryBaseDelay:          time.Second,
		RetryMaxDelay:           30 * time.Second,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30 * time.Second,
	}
}

// ResponseCache provides simple in-memory caching
type ResponseCache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	config  *EnhancedConfig
}

type CacheEntry struct {
	Response    string    `json:"response"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	AccessCount int       `json:"access_count"`
}

// MetricsCollector collects performance metrics
type MetricsCollector struct {
	providerMetrics map[string]*ProviderMetrics
	mutex           sync.RWMutex
}

type ProviderMetrics struct {
	RequestCount    int64         `json:"request_count"`
	ErrorCount      int64         `json:"error_count"`
	CacheHits       int64         `json:"cache_hits"`
	CacheMisses     int64         `json:"cache_misses"`
	RetryAttempts   int64         `json:"retry_attempts"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	TotalCost       float64       `json:"total_cost"`
}

// RetryManager handles retry logic
type RetryManager struct {
	config *EnhancedConfig
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	failureCount    int
	lastFailureTime time.Time
	state           CircuitBreakerState
	threshold       int
	timeout         time.Duration
	mutex           sync.Mutex
}

type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewEnhancedChatHandler creates a new enhanced chat handler
func NewEnhancedChatHandler(settingsManager *settings.Manager, loggerHolder LoggerHolder, gdbHandler GDBCommandHandler, config *EnhancedConfig) *EnhancedChatHandler {
	if config == nil {
		config = DefaultEnhancedConfig()
	}

	return &EnhancedChatHandler{
		settingsManager: settingsManager,
		loggerHolder:    loggerHolder,
		gdbHandler:      gdbHandler,
		cache:           NewResponseCache(config),
		metrics:         NewMetricsCollector(),
		retryManager:    NewRetryManager(config),
		circuitBreakers: make(map[string]*CircuitBreaker),
		config:          config,
	}
}

// HandleChat handles chat requests with enhanced features
func (h *EnhancedChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := h.generateRequestID()

	logger := h.loggerHolder.Get()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var chatReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		if logger != nil {
			logger.LogError(err, "Decoding chat request body")
		}
		return
	}

	// Get current settings
	settings := h.settingsManager.GetSettings()
	provider := settings.Provider

	// Record request metric
	h.metrics.RecordRequest(provider)

	// Log request
	if logger != nil {
		h.logRequestDetails(logger, &chatReq, requestID, provider)
	}

	// Check cache first
	if h.config.CacheEnabled {
		if cachedResponse := h.cache.Get(&chatReq, provider, settings.Model); cachedResponse != "" {
			h.metrics.RecordCacheHit(provider)
			if logger != nil {
				logger.LogTerminalOutput(fmt.Sprintf("=== CACHE HIT %s ===", requestID))
			}

			chatResp := ChatResponse{Response: cachedResponse}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(chatResp)
			return
		}
		h.metrics.RecordCacheMiss(provider)
	}

	// Process context if enabled
	if h.config.ContextEnabled {
		h.processContext(&chatReq, logger)
	}

	// Send request with retry and circuit breaker
	response, err := h.sendWithEnhancements(&chatReq, settings, logger, requestID)
	if err != nil {
		h.metrics.RecordError(provider)
		errorMsg := fmt.Sprintf("Error calling LLM API: %v", err)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		if logger != nil {
			logger.LogError(err, "Enhanced chat request failed")
		}
		return
	}

	// Record metrics
	responseTime := time.Since(start)
	h.metrics.RecordResponse(provider, responseTime)

	// Cache the response
	if h.config.CacheEnabled {
		h.cache.Set(&chatReq, provider, settings.Model, response)
	}

	// Log response
	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== ENHANCED RESPONSE %s ===\nTime: %v", requestID, responseTime))
	}

	// Process and return response
	h.processLLMResponse(w, response, &chatReq, settings, logger)
}

// sendWithEnhancements sends request with retry logic and circuit breaker
func (h *EnhancedChatHandler) sendWithEnhancements(chatReq *ChatRequest, settings settings.Settings, logger *logsession.SessionLogger, requestID string) (string, error) {
	provider := settings.Provider

	// Get or create circuit breaker for provider
	circuitBreaker := h.getCircuitBreaker(provider)

	// Check circuit breaker state
	if !circuitBreaker.CanExecute() {
		return "", fmt.Errorf("circuit breaker is open for provider %s", provider)
	}

	var lastErr error
	var response string

	// Retry logic
	for attempt := 0; attempt < h.config.RetryMaxAttempts; attempt++ {
		if attempt > 0 {
			h.metrics.RecordRetry(provider)
			delay := h.calculateRetryDelay(attempt)
			if logger != nil {
				logger.LogTerminalOutput(fmt.Sprintf("=== RETRY ATTEMPT %d ===\nDelay: %v", attempt, delay))
			}
			time.Sleep(delay)
		}

		// Execute request
		resp, err := h.executeRequest(chatReq, settings, logger)
		if err != nil {
			lastErr = err
			circuitBreaker.RecordFailure()

			// Check if error is retryable
			if !h.isRetryableError(err) {
				break
			}
			continue
		}

		// Success
		circuitBreaker.RecordSuccess()
		response = resp
		break
	}

	if response == "" && lastErr != nil {
		return "", lastErr
	}

	return response, nil
}

// executeRequest executes the actual API request
func (h *EnhancedChatHandler) executeRequest(chatReq *ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) (string, error) {
	switch settings.Provider {
	case "anthropic":
		return h.callAnthropicAPI(*chatReq, settings, logger)
	case "openai":
		return h.callOpenAIAPI(*chatReq, settings, logger)
	case "openrouter":
		return "", fmt.Errorf("OpenRouter is temporarily disabled")
	default:
		return "", fmt.Errorf("unsupported provider: %s", settings.Provider)
	}
}

// callAnthropicAPI calls the Anthropic API (simplified version)
func (h *EnhancedChatHandler) callAnthropicAPI(chatReq ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) (string, error) {
	systemMessage := `You are an AI assistant that helps with programming and debugging.

YOU MUST RESPOND IN VALID JSON FORMAT according to this structure:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."],
  "waitForOutput": true/false
}

Do not include any text outside the JSON structure. Your entire response must be a single JSON object.`

	// Context injection
	currentUserMessageContent := chatReq.Message
	if len(chatReq.SentContext) > 0 {
		contextPrefix := "\n\n--- Provided Context ---\n"
		for _, item := range chatReq.SentContext {
			contextPrefix += fmt.Sprintf("Type: %s\nDescription: %s\n", item.Type, item.Description)
			if item.Content != "" {
				contextPrefix += fmt.Sprintf("Content:\n```\n%s\n```\n", item.Content)
			}
			contextPrefix += "---\n"
		}
		currentUserMessageContent = contextPrefix + currentUserMessageContent
	}

	// Build messages array
	messages := []AnthropicMessage{}
	for _, msg := range chatReq.History {
		role := msg.Role
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		messages = append(messages, AnthropicMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	apiReq := AnthropicRequest{
		Model:     settings.Model,
		Messages:  messages,
		MaxTokens: 4096,
		System:    systemMessage,
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	var apiResp AnthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(apiResp.Content) > 0 && apiResp.Content[0].Type == "text" {
		return apiResp.Content[0].Text, nil
	}

	return "", fmt.Errorf("no content in Anthropic response")
}

// callOpenAIAPI calls the OpenAI API (simplified version)
func (h *EnhancedChatHandler) callOpenAIAPI(chatReq ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) (string, error) {
	systemMessage := `You are an AI assistant that helps with programming and debugging.

YOU MUST RESPOND IN VALID JSON FORMAT according to this structure:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."],
  "waitForOutput": true/false
}

Do not include any text outside the JSON structure. Your entire response must be a single JSON object.`

	// Context injection
	currentUserMessageContent := chatReq.Message
	if len(chatReq.SentContext) > 0 {
		contextPrefix := "\n\n--- Provided Context ---\n"
		for _, item := range chatReq.SentContext {
			contextPrefix += fmt.Sprintf("Type: %s\nDescription: %s\n", item.Type, item.Description)
			if item.Content != "" {
				contextPrefix += fmt.Sprintf("Content:\n```\n%s\n```\n", item.Content)
			}
			contextPrefix += "---\n"
		}
		currentUserMessageContent = contextPrefix + currentUserMessageContent
	}

	// Build messages array
	messages := []OpenAIMessage{
		{Role: "system", Content: systemMessage},
	}

	for _, msg := range chatReq.History {
		role := msg.Role
		if role == "user" {
			role = "user"
		} else {
			role = "assistant"
		}
		messages = append(messages, OpenAIMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	apiReq := OpenAIRequest{
		Model:    settings.Model,
		Messages: messages,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
	}

	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no content in OpenAI response")
}

// processContext handles context management and trimming
func (h *EnhancedChatHandler) processContext(chatReq *ChatRequest, logger *logsession.SessionLogger) {
	if !h.config.ContextEnabled {
		return
	}

	// Estimate token count
	tokenCount := h.estimateTokens(chatReq)

	if tokenCount <= h.config.MaxTokens {
		return
	}

	// Trim context
	if len(chatReq.History) > h.config.PriorityRecentMessages {
		// Keep only recent messages
		keepCount := h.config.PriorityRecentMessages
		if keepCount > len(chatReq.History) {
			keepCount = len(chatReq.History)
		}

		startIndex := len(chatReq.History) - keepCount
		chatReq.History = chatReq.History[startIndex:]

		if logger != nil {
			logger.LogTerminalOutput(fmt.Sprintf("=== CONTEXT TRIMMED ===\nKept %d recent messages", keepCount))
		}
	}
}

// processLLMResponse processes the LLM response and handles JSON validation
func (h *EnhancedChatHandler) processLLMResponse(w http.ResponseWriter, response string, chatReq *ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) {
	var llmResponse LLMResponse
	var responseText string
	var gdbOutput string

	// Debug log the incoming response
	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== PROCESSING LLM RESPONSE ===\nRaw response: %s", response))
	}

	// Try to extract JSON from the response (handles mixed content)
	extractedJSON, jsonFound := h.extractJSONFromResponse(response)

	var parseErr error
	if jsonFound {
		parseErr = json.Unmarshal([]byte(extractedJSON), &llmResponse)
		if logger != nil {
			logger.LogTerminalOutput(fmt.Sprintf("=== JSON EXTRACTED AND PARSED ===\nExtracted: %s\nText: %s\nGDB Commands: %v\nWaitForOutput: %v",
				extractedJSON, llmResponse.Text, llmResponse.GDBCommands, llmResponse.WaitForOutput))
		}
	} else {
		// Try parsing the entire response as JSON (fallback)
		parseErr = json.Unmarshal([]byte(response), &llmResponse)
		if logger != nil {
			if parseErr != nil {
				logger.LogTerminalOutput(fmt.Sprintf("=== NO JSON FOUND, PARSE ERROR ===\nError: %v", parseErr))
			} else {
				logger.LogTerminalOutput(fmt.Sprintf("=== FULL RESPONSE PARSED AS JSON ===\nText: %s\nGDB Commands: %v\nWaitForOutput: %v",
					llmResponse.Text, llmResponse.GDBCommands, llmResponse.WaitForOutput))
			}
		}
	}

	isValidJSON := parseErr == nil && strings.TrimSpace(llmResponse.Text) != ""

	if !isValidJSON {
		if logger != nil {
			logger.LogTerminalOutput("=== JSON VALIDATION FAILED ===\nAttempting to reformat...")
		}

		// Try to reformat response
		reformattedResponse, reformatErr := h.reformatResponse(response, chatReq, settings, logger)
		if reformatErr == nil {
			if err := json.Unmarshal([]byte(reformattedResponse), &llmResponse); err == nil && strings.TrimSpace(llmResponse.Text) != "" {
				responseText = llmResponse.Text
				if logger != nil {
					logger.LogTerminalOutput("=== REFORMAT SUCCESS ===\nUsing reformatted response")
				}
			} else {
				responseText = reformattedResponse
				if logger != nil {
					logger.LogTerminalOutput("=== REFORMAT PARTIAL SUCCESS ===\nUsing reformatted response as-is")
				}
			}
		} else {
			responseText = response
			if logger != nil {
				logger.LogTerminalOutput("=== REFORMAT FAILED ===\nUsing original response")
			}
		}
	} else {
		responseText = llmResponse.Text
		if logger != nil {
			logger.LogTerminalOutput("=== JSON VALIDATION SUCCESS ===\nExtracting text field")
		}
	}

	// Handle GDB commands if present
	if len(llmResponse.GDBCommands) > 0 && h.gdbHandler != nil && h.gdbHandler.IsRunning() {
		if logger != nil {
			cmdList := strings.Join(llmResponse.GDBCommands, ", ")
			logger.LogTerminalOutput(fmt.Sprintf("=== EXECUTING GDB COMMANDS ===\nCommands: %s", cmdList))
		}

		for _, cmd := range llmResponse.GDBCommands {
			if output, err := h.gdbHandler.ExecuteCommandWithOutput(cmd); err == nil {
				gdbOutput += output + "\n"
				if logger != nil {
					logger.LogTerminalOutput(fmt.Sprintf("=== GDB COMMAND OUTPUT ===\nCommand: %s\nOutput: %s", cmd, output))
				}
			}
		}

		// If waitForOutput is true and we have GDB output, send it back to LLM for analysis
		if llmResponse.WaitForOutput && gdbOutput != "" {
			if logger != nil {
				logger.LogTerminalOutput("=== SENDING GDB OUTPUT TO LLM FOR ANALYSIS ===")
			}

			// Create a follow-up context item with the GDB output
			gdbContext := ContextItem{
				Type:        "command_output",
				Description: "GDB Command Output",
				Content:     gdbOutput,
			}

			// Add to the original context for the follow-up request
			followupReq := *chatReq
			followupReq.SentContext = append(followupReq.SentContext, gdbContext)

			// Make a follow-up request to the LLM with the GDB output
			followupResponse, followupErr := h.executeRequest(&followupReq, settings, logger)
			if followupErr == nil {
				if logger != nil {
					logger.LogTerminalOutput(fmt.Sprintf("=== FOLLOW-UP RESPONSE RECEIVED ===\nRaw: %s", followupResponse))
				}

				// Try to extract JSON from the follow-up response
				followupJSON, followupJSONFound := h.extractJSONFromResponse(followupResponse)

				var followupLLM LLMResponse
				var parseErr error

				if followupJSONFound {
					parseErr = json.Unmarshal([]byte(followupJSON), &followupLLM)
					if logger != nil {
						logger.LogTerminalOutput(fmt.Sprintf("=== FOLLOW-UP JSON EXTRACTED ===\nExtracted: %s", followupJSON))
					}
				} else {
					parseErr = json.Unmarshal([]byte(followupResponse), &followupLLM)
				}

				if logger != nil {
					if parseErr != nil {
						logger.LogTerminalOutput(fmt.Sprintf("=== FOLLOW-UP JSON PARSE ERROR ===\nError: %v", parseErr))
					} else {
						logger.LogTerminalOutput(fmt.Sprintf("=== FOLLOW-UP JSON PARSE SUCCESS ===\nText: '%s'\nText Length: %d\nTrimmed Length: %d",
							followupLLM.Text, len(followupLLM.Text), len(strings.TrimSpace(followupLLM.Text))))
					}
				}

				if parseErr == nil && strings.TrimSpace(followupLLM.Text) != "" {
					// Use the text from the follow-up response
					responseText = followupLLM.Text
					if logger != nil {
						logger.LogTerminalOutput("=== USING FOLLOW-UP LLM RESPONSE ===")
					}
				} else {
					// Fallback to the whole follow-up response if JSON parsing fails
					responseText = followupResponse
					if logger != nil {
						logger.LogTerminalOutput("=== USING RAW FOLLOW-UP RESPONSE ===\nReason: JSON parse failed or empty text field")
					}
				}
			} else {
				if logger != nil {
					logger.LogError(followupErr, "Follow-up LLM request failed")
				}
				// Keep the original response text if follow-up fails
			}
		}

		// Note: GDB output is executed and sent via WebSocket to terminal interface
		// If waitForOutput=true, it's also sent back to LLM for analysis
	}

	// Debug log final response
	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== FINAL RESPONSE TEXT ===\n%s", responseText))
	}

	// Send response
	chatResp := ChatResponse{Response: responseText}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chatResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Encoding chat response")
		}
	}
}

// reformatResponse attempts to reformat an invalid JSON response
func (h *EnhancedChatHandler) reformatResponse(originalResponse string, chatReq *ChatRequest, settings settings.Settings, logger *logsession.SessionLogger) (string, error) {
	reformatReq := ChatRequest{
		Message: fmt.Sprintf(`ERROR: Your previous response was not in the required JSON format.

YOU MUST RESPOND WITH VALID JSON ONLY. No text outside the JSON object is allowed.

Please reformat your entire response using EXACTLY this JSON structure and nothing else:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."], 
  "waitForOutput": true/false
}

Original response to reformat:
%s`, originalResponse),
		History:     chatReq.History,
		SentContext: chatReq.SentContext,
	}

	if logger != nil {
		logger.LogTerminalOutput("=== REQUESTING JSON REFORMAT ===")
	}

	return h.executeRequest(&reformatReq, settings, logger)
}

// extractJSONFromResponse attempts to extract a JSON object from a mixed response
func (h *EnhancedChatHandler) extractJSONFromResponse(response string) (string, bool) {
	// Look for JSON object boundaries
	startIdx := strings.Index(response, "{")
	if startIdx == -1 {
		return "", false
	}

	// Find the matching closing brace
	braceCount := 0
	for i := startIdx; i < len(response); i++ {
		switch response[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				// Found complete JSON object
				jsonStr := response[startIdx : i+1]

				// Validate that this is actually valid JSON
				var temp interface{}
				if json.Unmarshal([]byte(jsonStr), &temp) == nil {
					return jsonStr, true
				}
			}
		}
	}

	return "", false
}

// Helper methods for the enhanced features

func (h *EnhancedChatHandler) generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (h *EnhancedChatHandler) logRequestDetails(logger *logsession.SessionLogger, chatReq *ChatRequest, requestID, provider string) {
	logMsg := fmt.Sprintf("=== ENHANCED REQUEST %s ===\nProvider: %s\nMessage: %s", requestID, provider, chatReq.Message)
	if len(chatReq.SentContext) > 0 {
		logMsg += fmt.Sprintf("\nContext Items: %d", len(chatReq.SentContext))
	}
	if len(chatReq.History) > 0 {
		logMsg += fmt.Sprintf("\nHistory Messages: %d", len(chatReq.History))
	}
	logger.LogTerminalOutput(logMsg)
}

func (h *EnhancedChatHandler) getCircuitBreaker(provider string) *CircuitBreaker {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if cb, exists := h.circuitBreakers[provider]; exists {
		return cb
	}

	cb := &CircuitBreaker{
		threshold: h.config.CircuitBreakerThreshold,
		timeout:   h.config.CircuitBreakerTimeout,
		state:     CircuitClosed,
	}
	h.circuitBreakers[provider] = cb
	return cb
}

func (h *EnhancedChatHandler) calculateRetryDelay(attempt int) time.Duration {
	delay := time.Duration(float64(h.config.RetryBaseDelay) * math.Pow(2, float64(attempt)))
	if delay > h.config.RetryMaxDelay {
		delay = h.config.RetryMaxDelay
	}
	return delay
}

func (h *EnhancedChatHandler) isRetryableError(err error) bool {
	errStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"timeout", "connection", "network", "service unavailable", "rate limit", "502", "503", "504",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}
	return false
}

func (h *EnhancedChatHandler) estimateTokens(chatReq *ChatRequest) int {
	tokens := len(chatReq.Message) / 4 // Rough approximation
	for _, msg := range chatReq.History {
		tokens += len(msg.Content) / 4
	}
	for _, ctx := range chatReq.SentContext {
		tokens += len(ctx.Content) / 4
	}
	return tokens
}

// GetMetrics returns current metrics
func (h *EnhancedChatHandler) GetMetrics() map[string]*ProviderMetrics {
	return h.metrics.GetAllMetrics()
}

// GetCacheStats returns cache statistics
func (h *EnhancedChatHandler) GetCacheStats() map[string]interface{} {
	return h.cache.GetStats()
}

// Additional implementation for cache, metrics, and circuit breaker methods would go here...
// (Abbreviated for space - the core implementation pattern is established)

// NewResponseCache creates a new response cache
func NewResponseCache(config *EnhancedConfig) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]*CacheEntry),
		config:  config,
	}
}

func (rc *ResponseCache) Get(req *ChatRequest, provider, model string) string {
	if !rc.config.CacheEnabled {
		return ""
	}

	key := rc.generateKey(req, provider, model)
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	if entry, exists := rc.entries[key]; exists && time.Now().Before(entry.ExpiresAt) {
		entry.AccessCount++
		return entry.Response
	}

	return ""
}

func (rc *ResponseCache) Set(req *ChatRequest, provider, model, response string) {
	if !rc.config.CacheEnabled {
		return
	}

	key := rc.generateKey(req, provider, model)
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	rc.entries[key] = &CacheEntry{
		Response:    response,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(rc.config.CacheTTL),
		AccessCount: 1,
	}
}

func (rc *ResponseCache) generateKey(req *ChatRequest, provider, model string) string {
	// Simple key generation - in production you'd want a more sophisticated approach
	data := fmt.Sprintf("%s:%s:%s", provider, model, req.Message)
	return fmt.Sprintf("%x", data)[:16]
}

func (rc *ResponseCache) GetStats() map[string]interface{} {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	return map[string]interface{}{
		"enabled":     rc.config.CacheEnabled,
		"entry_count": len(rc.entries),
		"max_size":    rc.config.CacheMaxSize,
		"ttl":         rc.config.CacheTTL.String(),
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		providerMetrics: make(map[string]*ProviderMetrics),
	}
}

func (mc *MetricsCollector) RecordRequest(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.providerMetrics[provider]; !exists {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}
	mc.providerMetrics[provider].RequestCount++
}

func (mc *MetricsCollector) RecordError(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.providerMetrics[provider]; !exists {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}
	mc.providerMetrics[provider].ErrorCount++
}

func (mc *MetricsCollector) RecordCacheHit(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.providerMetrics[provider]; !exists {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}
	mc.providerMetrics[provider].CacheHits++
}

func (mc *MetricsCollector) RecordCacheMiss(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.providerMetrics[provider]; !exists {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}
	mc.providerMetrics[provider].CacheMisses++
}

func (mc *MetricsCollector) RecordRetry(provider string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.providerMetrics[provider]; !exists {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}
	mc.providerMetrics[provider].RetryAttempts++
}

func (mc *MetricsCollector) RecordResponse(provider string, responseTime time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if _, exists := mc.providerMetrics[provider]; !exists {
		mc.providerMetrics[provider] = &ProviderMetrics{}
	}

	metrics := mc.providerMetrics[provider]
	// Simple running average
	if metrics.RequestCount > 0 {
		metrics.AvgResponseTime = time.Duration(
			(int64(metrics.AvgResponseTime) + int64(responseTime)) / 2,
		)
	} else {
		metrics.AvgResponseTime = responseTime
	}
}

func (mc *MetricsCollector) GetAllMetrics() map[string]*ProviderMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]*ProviderMetrics)
	for k, v := range mc.providerMetrics {
		// Create a copy to avoid data races
		copy := *v
		result[k] = &copy
	}
	return result
}

// NewRetryManager creates a new retry manager
func NewRetryManager(config *EnhancedConfig) *RetryManager {
	return &RetryManager{config: config}
}

// CircuitBreaker methods
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount = 0
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
	}
}
