package websocket

import (
	"encoding/json"
	"fmt"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	// MessageTypeDebuggerOutput is sent when the debugger produces output
	MessageTypeDebuggerOutput MessageType = "debugger_output"

	// MessageTypeDebuggerStatus is sent when the debugger status changes
	MessageTypeDebuggerStatus MessageType = "debugger_status"

	// MessageTypeChatResponse is sent when a chat response is received
	MessageTypeChatResponse MessageType = "chat_response"

	// MessageTypeError is sent when an error occurs
	MessageTypeError MessageType = "error"

	// MessageTypeInfo is sent for informational messages
	MessageTypeInfo MessageType = "info"
)

// Message represents a WebSocket message
type Message struct {
	Type    MessageType     `json:"type"`
	Content string          `json:"content,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// NewMessage creates a new message with the given type and content
func NewMessage(msgType MessageType, content string) *Message {
	return &Message{
		Type:    msgType,
		Content: content,
	}
}

// NewDataMessage creates a new message with the given type and data
func NewDataMessage(msgType MessageType, data interface{}) (*Message, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return &Message{
		Type: msgType,
		Data: dataBytes,
	}, nil
}

// Encode encodes the message to JSON
func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DebuggerStatusData represents the data for a debugger status message
type DebuggerStatusData struct {
	Running bool   `json:"running"`
	Status  string `json:"status"`
	Target  string `json:"target,omitempty"`
}

// NewDebuggerStatusMessage creates a new debugger status message
func NewDebuggerStatusMessage(running bool, status, target string) (*Message, error) {
	data := DebuggerStatusData{
		Running: running,
		Status:  status,
		Target:  target,
	}

	return NewDataMessage(MessageTypeDebuggerStatus, data)
}

// NewDebuggerOutputMessage creates a new debugger output message
func NewDebuggerOutputMessage(output string) *Message {
	return NewMessage(MessageTypeDebuggerOutput, output)
}

// NewErrorMessage creates a new error message
func NewErrorMessage(errorMsg string) *Message {
	return NewMessage(MessageTypeError, errorMsg)
}

// NewInfoMessage creates a new info message
func NewInfoMessage(infoMsg string) *Message {
	return NewMessage(MessageTypeInfo, infoMsg)
}

// NewChatResponseMessage creates a new chat response message
func NewChatResponseMessage(response string) *Message {
	return NewMessage(MessageTypeChatResponse, response)
}
