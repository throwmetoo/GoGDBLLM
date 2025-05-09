package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
)

// Define LoggerHolder interface locally (or move to a shared place)
type LoggerHolder interface {
	Set(newLogger *logsession.SessionLogger)
	Get() *logsession.SessionLogger
}

// GDBCommandHandler interface for handling GDB commands
type GDBCommandHandler interface {
	HandleCommand(cmd string) error
	IsRunning() bool
	ExecuteCommandWithOutput(cmd string) (string, error)
}

// ChatHandler handles chat-related operations
type ChatHandler struct {
	settingsManager *settings.Manager
	loggerHolder    LoggerHolder      // Use interface type
	gdbHandler      GDBCommandHandler // Add GDB handler
}

// NewChatHandler creates a new chat handler
func NewChatHandler(settingsManager *settings.Manager, loggerHolder LoggerHolder, gdbHandler GDBCommandHandler) *ChatHandler {
	return &ChatHandler{
		settingsManager: settingsManager,
		loggerHolder:    loggerHolder,
		gdbHandler:      gdbHandler,
	}
}

// HandleChat handles chat requests
func (h *ChatHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	// Get current logger first
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

	// --- Log user input ---
	if logger != nil {
		// Convert []api.ContextItem to []logsession.ContextItem
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
	// --- End log user input ---

	// Process the request and get the initial LLM response
	response, err := h.processLLMRequest(chatReq)
	if err != nil {
		errorMsg := fmt.Sprintf("Error calling LLM API: %v", err)
		http.Error(w, errorMsg, http.StatusInternalServerError)
		if logger != nil {
			logger.LogError(err, "Calling LLM API")
		}
		return
	}

	// We don't need to log the raw response here since it's already logged in processLLMRequest

	// Parse the response as LLMResponse
	var llmResponse LLMResponse
	var responseText string
	var gdbOutput string

	// Check if response is a valid JSON and contains required fields
	parseErr := json.Unmarshal([]byte(response), &llmResponse)
	isValidJSON := parseErr == nil && llmResponse.Text != ""

	// Check if there's text outside JSON structure
	trimmedResp := strings.TrimSpace(response)
	hasExtraText := false
	if isValidJSON {
		// Verify that the response is only JSON (no text before or after)
		if !strings.HasPrefix(trimmedResp, "{") || !strings.HasSuffix(trimmedResp, "}") {
			hasExtraText = true
		}
	}

	// If the response is not valid JSON, has extra text, or doesn't have required fields
	if !isValidJSON || hasExtraText {
		if logger != nil {
			if parseErr != nil {
				logger.LogError(parseErr, "Parsing LLM response as JSON")
			} else {
				logger.LogTerminalOutput("=== JSON FORMAT ERROR ===\nLLM response contains text outside JSON structure or missing required fields\n=== END ERROR ===")
			}
		}

		// Create a follow-up request asking the LLM to reformat its response
		reformatReq := ChatRequest{
			Message: `ERROR: Your previous response was not in the required JSON format, or contained text outside the JSON structure. 

YOU MUST RESPOND WITH VALID JSON ONLY. No text outside the JSON object is allowed.

Please reformat your entire response using EXACTLY this JSON structure and nothing else:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."], 
  "waitForOutput": true/false
}

If you don't need to run GDB commands, just include an empty array for gdbCommands and set waitForOutput to false.

Original response to reformat:
` + response,
			History:     chatReq.History,
			SentContext: chatReq.SentContext,
		}

		// Log that we're requesting a reformatted response
		if logger != nil && h.gdbHandler != nil && h.gdbHandler.IsRunning() {
			logger.LogTerminalOutput("=== REQUESTING JSON REFORMAT ===")
		}

		// Send the reformatting request to the LLM
		reformattedResponse, reformatErr := h.processLLMRequest(reformatReq)
		if reformatErr != nil {
			// If reformatting fails, just use the original response as plain text
			responseText = response
			if logger != nil {
				logger.LogError(reformatErr, "Failed to get reformatted response from LLM")
			}
		} else {
			// Don't need to log here, already logged in processLLMRequest

			// Try to parse the reformatted response
			if err := json.Unmarshal([]byte(reformattedResponse), &llmResponse); err != nil {
				// If still not valid JSON, just use as plain text
				responseText = reformattedResponse
				if logger != nil {
					logger.LogError(err, "Parsing reformatted LLM response as JSON")
				}
			} else {
				// Successfully reformatted to JSON
				responseText = llmResponse.Text
				if responseText == "" {
					responseText = "Sorry, I couldn't format my response correctly. Please try again."
				}
			}
		}
	} else {
		// Valid JSON in the entire response
		responseText = llmResponse.Text
	}

	// Handle cases where the LLM response might be cut off
	// Check for responses that start with text but don't end with proper punctuation
	if !strings.HasPrefix(responseText, "{") && len(responseText) > 0 {
		lastChar := responseText[len(responseText)-1]
		// If the response doesn't end with common sentence-ending punctuation
		if lastChar != '.' && lastChar != '!' && lastChar != '?' && lastChar != ':' &&
			!strings.HasSuffix(responseText, "```") {
			// Add an ellipsis to indicate the response was cut off
			responseText += " [Response may be incomplete...]"
		}
	}

	if len(llmResponse.GDBCommands) > 0 {
		// Log GDB commands found in response
		if logger != nil && h.gdbHandler != nil && h.gdbHandler.IsRunning() {
			cmdList := strings.Join(llmResponse.GDBCommands, ", ")
			logger.LogTerminalOutput(fmt.Sprintf("=== EXECUTING GDB COMMANDS ===\nCommands: %s\nWaitForOutput: %t\n=== END COMMAND LIST ===",
				cmdList, llmResponse.WaitForOutput))
		}

		// Process GDB commands if GDB is running
		if h.gdbHandler != nil && h.gdbHandler.IsRunning() {
			var commandOutputs []string

			for i, cmd := range llmResponse.GDBCommands {
				// For commands where we need to capture output
				if llmResponse.WaitForOutput || i == len(llmResponse.GDBCommands)-1 {
					// Use ExecuteCommandWithOutput to get the output
					output, err := h.gdbHandler.ExecuteCommandWithOutput(cmd)
					if err != nil {
						if logger != nil {
							logger.LogError(err, "Executing GDB command with output: "+cmd)
						}
					} else {
						commandOutputs = append(commandOutputs, "Command: "+cmd+"\nOutput:\n"+output)
					}
				} else {
					// For commands where we don't need to capture output
					if err := h.gdbHandler.HandleCommand(cmd); err != nil {
						if logger != nil {
							logger.LogError(err, "Executing GDB command from LLM: "+cmd)
						}
					}

					// Log the command execution
					if logger != nil {
						logger.LogTerminalOutput("(LLM) " + cmd)
					}
				}
			}

			// If we need to wait for output and send it back to LLM
			if llmResponse.WaitForOutput && len(commandOutputs) > 0 {
				// Combine all command outputs
				gdbOutput = strings.Join(commandOutputs, "\n\n")

				// Create a follow-up context item with the GDB output
				gdbContext := ContextItem{
					Type:        "command_output",
					Description: "GDB Command Output",
					Content:     gdbOutput,
				}

				// Add to original context
				chatReq.SentContext = append(chatReq.SentContext, gdbContext)

				// Log that we're sending GDB output back to LLM
				if logger != nil && h.gdbHandler != nil && h.gdbHandler.IsRunning() {
					logger.LogTerminalOutput(fmt.Sprintf("=== GDB OUTPUT FOR LLM ===\n%s\n=== END GDB OUTPUT ===", gdbOutput))
				}

				// Make a follow-up request to the LLM with the output
				followupResponse, followupErr := h.processLLMRequest(chatReq)
				if followupErr == nil {
					// Don't need to log here, already logged in processLLMRequest

					// Try to parse as JSON
					var followupLLM LLMResponse
					if json.Unmarshal([]byte(followupResponse), &followupLLM) == nil {
						// If valid JSON, extract just the text portion
						responseText = followupLLM.Text
						if responseText == "" {
							// Fallback if text field is empty
							responseText = followupResponse
						}
					} else {
						// Otherwise use the whole response
						responseText = followupResponse
					}
				}
			}
		} else {
			// GDB is not running, just send the text response
			responseText = llmResponse.Text + "\n\n(Note: GDB is not running, cannot execute commands)"
		}
	}

	// Final check for potentially truncated non-JSON responses
	if !strings.HasPrefix(responseText, "{") && len(responseText) > 0 {
		lastChar := responseText[len(responseText)-1]
		// If the response doesn't end with common sentence-ending punctuation
		if lastChar != '.' && lastChar != '!' && lastChar != '?' && lastChar != ':' &&
			!strings.HasSuffix(responseText, "```") &&
			!strings.HasSuffix(responseText, "may be incomplete...") {
			// Add an ellipsis to indicate the response was cut off
			responseText += " [Response may be incomplete...]"
			if logger != nil {
				logger.LogTerminalOutput("WARNING: Final response appears to be cut off, adding indicator")
			}
		}
	}

	// Send response to the user (only the text part)
	chatResp := ChatResponse{
		Response: responseText,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chatResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Encoding/Sending chat response")
		}
	}
}

// processLLMRequest handles the actual API call to the LLM provider
func (h *ChatHandler) processLLMRequest(chatReq ChatRequest) (string, error) {
	settings := h.settingsManager.GetSettings()
	logger := h.getLogger()

	// Log the request being sent to the LLM in the terminal if GDB is running
	if h.gdbHandler != nil && h.gdbHandler.IsRunning() && logger != nil {
		// Log basic request info
		logMsg := fmt.Sprintf("=== LLM REQUEST (%s/%s) ===\nUser Message: %s\n",
			settings.Provider, settings.Model, chatReq.Message)

		// Log context items if any
		if len(chatReq.SentContext) > 0 {
			logMsg += "\nContext Items:\n"
			for i, ctx := range chatReq.SentContext {
				logMsg += fmt.Sprintf("[%d] Type: %s, Description: %s\n",
					i+1, ctx.Type, ctx.Description)
				if len(ctx.Content) > 200 {
					// Truncate long content
					logMsg += fmt.Sprintf("Content: %s...(truncated)\n", ctx.Content[:200])
				} else if ctx.Content != "" {
					logMsg += fmt.Sprintf("Content: %s\n", ctx.Content)
				}
			}
		}

		// Log message history summary if any
		if len(chatReq.History) > 0 {
			logMsg += fmt.Sprintf("\nHistory: %d previous messages\n", len(chatReq.History))
		}

		logMsg += "=== END REQUEST ===\n"
		logger.LogTerminalOutput(logMsg)
	}

	var response string
	var err error
	var provider string

	// Call the appropriate API based on the provider
	switch settings.Provider {
	case "anthropic":
		provider = "Anthropic"
		response, err = h.callAnthropicAPI(chatReq, settings)
	case "openai":
		provider = "OpenAI"
		response, err = h.callOpenAIAPI(chatReq, settings)
	case "openrouter":
		// Temporarily disabled
		err = fmt.Errorf("OpenRouter is temporarily disabled. Please use Anthropic or OpenAI for JSON mode support")
		if logger != nil {
			logger.LogError(err, "OpenRouter is temporarily disabled")
		}
	default:
		err = fmt.Errorf("unsupported provider: %s", settings.Provider)
		if logger != nil {
			logger.LogError(err, "Checking provider in processLLMRequest")
		}
		return "", err
	}

	if err != nil {
		if logger != nil {
			logger.LogError(err, "Calling "+provider+" API")
		}
		return "", err
	}

	// Log the response received from the LLM in the terminal if GDB is running
	if h.gdbHandler != nil && h.gdbHandler.IsRunning() && logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== LLM RESPONSE (%s) ===\n%s\n=== END RESPONSE ===\n",
			provider, response))
	}

	return response, nil
}

// Helper function to get logger safely within API call methods
func (h *ChatHandler) getLogger() *logsession.SessionLogger {
	return h.loggerHolder.Get()
}

// callAnthropicAPI calls the Anthropic API
func (h *ChatHandler) callAnthropicAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	logger := h.getLogger()

	// Define system message for proper JSON formatting
	systemMessage := `You are an AI assistant that helps with programming and debugging.

YOU MUST RESPOND IN VALID JSON FORMAT according to this structure:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."],
  "waitForOutput": true/false
}

Do not include any text outside the JSON structure. Your entire response must be a single JSON object.`

	// --- Context Injection Start ---
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
	// --- Context Injection End ---

	// Build the messages array (don't include system in messages for Anthropic)
	messages := []AnthropicMessage{}

	// Add chat history
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

	// Add the current message (with context potentially prepended)
	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	// Create request with system message
	apiReq := AnthropicRequest{
		Model:     settings.Model,
		Messages:  messages,
		MaxTokens: 4096,
		System:    systemMessage, // Use system field for enforcing JSON
	}

	if logger != nil {
		logger.LogLLMRequestData("Anthropic", apiReq.Model, currentUserMessageContent)
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Marshalling Anthropic request")
		}
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Creating Anthropic HTTP request")
		}
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", settings.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Sending Anthropic request")
		}
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Reading Anthropic response body")
		}
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
		if logger != nil {
			logger.LogError(err, "Anthropic API returned non-OK status")
		}
		return "", err
	}

	// Parse response
	var apiResp AnthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Unmarshalling Anthropic response")
		}
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract content
	if len(apiResp.Content) > 0 && apiResp.Content[0].Type == "text" {
		responseContent := apiResp.Content[0].Text
		if logger != nil {
			logger.LogLLMResponse(responseContent)
		}

		// Check if the response might be cut off (for non-JSON responses)
		if !strings.HasPrefix(responseContent, "{") && len(responseContent) > 0 {
			lastChar := responseContent[len(responseContent)-1]
			// If the response doesn't end with common sentence-ending punctuation
			if lastChar != '.' && lastChar != '!' && lastChar != '?' && lastChar != ':' &&
				!strings.HasSuffix(responseContent, "```") {
				// Add an ellipsis to indicate the response was cut off
				responseContent += " [Response may be incomplete...]"
				if logger != nil {
					logger.LogTerminalOutput("WARNING: LLM response appears to be cut off, adding indicator")
				}
			}
		}

		return responseContent, nil
	}

	err = fmt.Errorf("no content in Anthropic response")
	if logger != nil {
		logger.LogError(err, "Extracting content from Anthropic response")
	}
	return "", err
}

// callOpenAIAPI calls the OpenAI API
func (h *ChatHandler) callOpenAIAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	logger := h.getLogger()
	// System message for OpenAI
	systemMessage := `You are an AI assistant that helps with programming and debugging.

YOU MUST RESPOND IN VALID JSON FORMAT according to this structure:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."],
  "waitForOutput": true/false
}

Do not include any text outside the JSON structure. Your entire response must be a single JSON object.`

	// --- Context Injection Start ---
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
	// --- Context Injection End ---

	// Build the messages array
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: systemMessage,
		},
	}

	// Add chat history
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

	// Add the current message (with context potentially prepended)
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	// Create request with JSON mode enforced
	apiReq := OpenAIRequest{
		Model:    settings.Model,
		Messages: messages,
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	}

	if logger != nil {
		logger.LogLLMRequestData("OpenAI", apiReq.Model, currentUserMessageContent)
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Marshalling OpenAI request")
		}
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Creating OpenAI HTTP request")
		}
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Sending OpenAI request")
		}
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Reading OpenAI response body")
		}
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
		if logger != nil {
			logger.LogError(err, "OpenAI API returned non-OK status")
		}
		return "", err
	}

	// Parse response
	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Unmarshalling OpenAI response")
		}
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Extract content
	if len(apiResp.Choices) > 0 {
		responseContent := apiResp.Choices[0].Message.Content
		if logger != nil {
			logger.LogLLMResponse(responseContent)
		}

		// Check if the response might be cut off (for non-JSON responses)
		if !strings.HasPrefix(responseContent, "{") && len(responseContent) > 0 {
			lastChar := responseContent[len(responseContent)-1]
			// If the response doesn't end with common sentence-ending punctuation
			if lastChar != '.' && lastChar != '!' && lastChar != '?' && lastChar != ':' &&
				!strings.HasSuffix(responseContent, "```") {
				// Add an ellipsis to indicate the response was cut off
				responseContent += " [Response may be incomplete...]"
				if logger != nil {
					logger.LogTerminalOutput("WARNING: LLM response appears to be cut off, adding indicator")
				}
			}
		}

		return responseContent, nil
	}

	err = fmt.Errorf("no content in OpenAI response")
	if logger != nil {
		logger.LogError(err, "Extracting content from OpenAI response")
	}
	return "", err
}

// callOpenRouterAPI calls the OpenRouter API
func (h *ChatHandler) callOpenRouterAPI(chatReq ChatRequest, settings settings.Settings) (string, error) {
	logger := h.getLogger()
	// System message for OpenRouter
	systemMessage := `You are an AI assistant that helps with programming and debugging.

⚠️ CRITICAL INSTRUCTION ⚠️: YOU MUST RESPOND WITH VALID JSON ONLY. Your entire response must be a single JSON object without any text before or after it.

REQUIRED JSON FORMAT:
{
  "text": "Your explanation or message to the user",
  "gdbCommands": ["command1", "command2", "..."],
  "waitForOutput": true/false
}

Failure to format your response as proper JSON will result in an error and your response will not be processed correctly.

DO NOT include any explanatory text, greeting, or other content outside the JSON structure.
DO NOT use markdown formatting or code blocks around the JSON.
ONLY return the JSON object itself.

- text: Your message that will be shown to the user (required)
- gdbCommands: Array of GDB commands to execute (can be empty array [])
- waitForOutput: If true, the output from your GDB commands will be automatically captured and sent back to you for analysis without user intervention; if false, execute all commands in sequence

COMMAND FEEDBACK LOOP: When waitForOutput is true, the system will:
1. Execute your GDB commands
2. Capture the output
3. Automatically send the output back to you
4. You must respond with another properly formatted JSON object

EXAMPLES OF CORRECT RESPONSES:

To run "info registers" and automatically get the output back:
{
  "text": "Let me check the current register values for you.",
  "gdbCommands": ["info registers"],
  "waitForOutput": true
}

To set a breakpoint and continue execution without waiting:
{
  "text": "I'll set a breakpoint at main() and start execution.",
  "gdbCommands": ["break main", "run"],
  "waitForOutput": false
}

To provide information without running GDB commands:
{
  "text": "The segmentation fault occurs when the program tries to access memory that it doesn't have permission to access.",
  "gdbCommands": [],
  "waitForOutput": false
}

REMINDER: EVERY response must be ONLY this JSON format with no text outside the JSON.`

	// --- Context Injection Start ---
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
	// --- Context Injection End ---

	// Build the messages array
	messages := []OpenRouterMessage{
		{
			Role:    "system",
			Content: systemMessage,
		},
	}

	// Add chat history
	for _, msg := range chatReq.History {
		role := msg.Role
		if role == "user" {
			role = "user"
		} else {
			role = "assistant"
		}
		messages = append(messages, OpenRouterMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Add the current message (with context potentially prepended)
	messages = append(messages, OpenRouterMessage{
		Role:    "user",
		Content: currentUserMessageContent,
	})

	// Create request
	apiReq := OpenRouterRequest{
		Model:    settings.Model,
		Messages: messages,
	}

	if logger != nil {
		logger.LogLLMRequestData("OpenRouter", apiReq.Model, currentUserMessageContent)
	}

	reqBody, err := json.Marshal(apiReq)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Marshalling OpenRouter request")
		}
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	openRouterURL := "https://openrouter.ai/api/v1/chat/completions"
	req, err := http.NewRequest("POST", openRouterURL, bytes.NewBuffer(reqBody))
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Creating OpenRouter HTTP request")
		}
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Sending OpenRouter request")
		}
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if logger != nil {
			logger.LogError(err, "Reading OpenRouter response body")
		}
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("API error (status %d): %s", resp.StatusCode, respBody)
		if logger != nil {
			logger.LogError(err, "OpenRouter API returned non-OK status")
		}
		return "", err
	}

	var apiResp OpenRouterResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if logger != nil {
			logger.LogError(err, "Unmarshalling OpenRouter response")
		}
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if len(apiResp.Choices) > 0 {
		responseContent := apiResp.Choices[0].Message.Content
		if logger != nil {
			logger.LogLLMResponse(responseContent)
		}
		return responseContent, nil
	}

	err = fmt.Errorf("no content in OpenRouter response")
	if logger != nil {
		logger.LogError(err, "Extracting content from OpenRouter response")
	}
	return "", err
}
