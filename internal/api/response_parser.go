package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourusername/gogdbllm/internal/logsession"
)

// ResponseParser handles parsing of LLM responses
type ResponseParser struct{}

// ParsedResponse contains the parsed components of an LLM response
type ParsedResponse struct {
	Text          string   `json:"text"`
	GDBCommands   []string `json:"gdbCommands"`
	WaitForOutput bool     `json:"waitForOutput"`
	RawResponse   string   `json:"rawResponse"`
	ParseMethod   string   `json:"parseMethod"`
}

// NewResponseParser creates a new response parser
func NewResponseParser() *ResponseParser {
	return &ResponseParser{}
}

// ParseResponse attempts to parse an LLM response, handling various formats
func (rp *ResponseParser) ParseResponse(response string, logger *logsession.SessionLogger) (*ParsedResponse, error) {
	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== PARSING RESPONSE ===\nLength: %d chars", len(response)))
	}

	// Try different parsing strategies in order of preference

	// Strategy 1: Try full response as JSON
	if parsed, err := rp.tryParseFullJSON(response, logger); err == nil {
		return parsed, nil
	}

	// Strategy 2: Try extracting JSON from mixed content
	if parsed, err := rp.tryExtractJSON(response, logger); err == nil {
		return parsed, nil
	}

	// Strategy 3: Try reformatting and parsing
	if parsed, err := rp.tryReformatAndParse(response, logger); err == nil {
		return parsed, nil
	}

	// Strategy 4: Fallback to text-only response
	if logger != nil {
		logger.LogTerminalOutput("=== USING FALLBACK TEXT RESPONSE ===")
	}

	return &ParsedResponse{
		Text:          response,
		GDBCommands:   []string{},
		WaitForOutput: false,
		RawResponse:   response,
		ParseMethod:   "fallback_text",
	}, nil
}

// tryParseFullJSON attempts to parse the entire response as JSON
func (rp *ResponseParser) tryParseFullJSON(response string, logger *logsession.SessionLogger) (*ParsedResponse, error) {
	var llmResp LLMResponse
	err := json.Unmarshal([]byte(response), &llmResp)

	if err != nil {
		if logger != nil {
			logger.LogTerminalOutput(fmt.Sprintf("=== FULL JSON PARSE FAILED ===\nError: %v", err))
		}
		return nil, err
	}

	if strings.TrimSpace(llmResp.Text) == "" {
		if logger != nil {
			logger.LogTerminalOutput("=== FULL JSON PARSE: EMPTY TEXT FIELD ===")
		}
		return nil, fmt.Errorf("empty text field")
	}

	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== FULL JSON PARSE SUCCESS ===\nText: %d chars, Commands: %d",
			len(llmResp.Text), len(llmResp.GDBCommands)))
	}

	return &ParsedResponse{
		Text:          llmResp.Text,
		GDBCommands:   llmResp.GDBCommands,
		WaitForOutput: llmResp.WaitForOutput,
		RawResponse:   response,
		ParseMethod:   "full_json",
	}, nil
}

// tryExtractJSON attempts to extract JSON from mixed content
func (rp *ResponseParser) tryExtractJSON(response string, logger *logsession.SessionLogger) (*ParsedResponse, error) {
	jsonStr, found := rp.extractJSONFromResponse(response)
	if !found {
		if logger != nil {
			logger.LogTerminalOutput("=== JSON EXTRACTION: NO JSON FOUND ===")
		}
		return nil, fmt.Errorf("no JSON found in response")
	}

	var llmResp LLMResponse
	err := json.Unmarshal([]byte(jsonStr), &llmResp)
	if err != nil {
		if logger != nil {
			logger.LogTerminalOutput(fmt.Sprintf("=== EXTRACTED JSON PARSE FAILED ===\nError: %v", err))
		}
		return nil, err
	}

	if strings.TrimSpace(llmResp.Text) == "" {
		if logger != nil {
			logger.LogTerminalOutput("=== EXTRACTED JSON: EMPTY TEXT FIELD ===")
		}
		return nil, fmt.Errorf("empty text field in extracted JSON")
	}

	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== JSON EXTRACTION SUCCESS ===\nExtracted: %d chars -> Text: %d chars, Commands: %d",
			len(jsonStr), len(llmResp.Text), len(llmResp.GDBCommands)))
	}

	return &ParsedResponse{
		Text:          llmResp.Text,
		GDBCommands:   llmResp.GDBCommands,
		WaitForOutput: llmResp.WaitForOutput,
		RawResponse:   response,
		ParseMethod:   "extracted_json",
	}, nil
}

// tryReformatAndParse attempts to clean up and reparse the response
func (rp *ResponseParser) tryReformatAndParse(response string, logger *logsession.SessionLogger) (*ParsedResponse, error) {
	// Simple cleanup strategies
	cleaned := strings.TrimSpace(response)

	// Remove common prefixes/suffixes
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	}
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}

	cleaned = strings.TrimSpace(cleaned)

	var llmResp LLMResponse
	err := json.Unmarshal([]byte(cleaned), &llmResp)
	if err != nil {
		if logger != nil {
			logger.LogTerminalOutput(fmt.Sprintf("=== REFORMAT PARSE FAILED ===\nError: %v", err))
		}
		return nil, err
	}

	if strings.TrimSpace(llmResp.Text) == "" {
		if logger != nil {
			logger.LogTerminalOutput("=== REFORMAT PARSE: EMPTY TEXT FIELD ===")
		}
		return nil, fmt.Errorf("empty text field after reformat")
	}

	if logger != nil {
		logger.LogTerminalOutput(fmt.Sprintf("=== REFORMAT PARSE SUCCESS ===\nText: %d chars, Commands: %d",
			len(llmResp.Text), len(llmResp.GDBCommands)))
	}

	return &ParsedResponse{
		Text:          llmResp.Text,
		GDBCommands:   llmResp.GDBCommands,
		WaitForOutput: llmResp.WaitForOutput,
		RawResponse:   response,
		ParseMethod:   "reformatted",
	}, nil
}

// extractJSONFromResponse extracts the first valid JSON object from a response
func (rp *ResponseParser) extractJSONFromResponse(response string) (string, bool) {
	startIdx := strings.Index(response, "{")
	if startIdx == -1 {
		return "", false
	}

	braceCount := 0
	for i := startIdx; i < len(response); i++ {
		switch response[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				jsonStr := response[startIdx : i+1]

				// Validate JSON
				var temp interface{}
				if json.Unmarshal([]byte(jsonStr), &temp) == nil {
					return jsonStr, true
				}
			}
		}
	}

	return "", false
}
