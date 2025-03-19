package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// Handler returns an http.HandlerFunc for WebSocket connections
func (m *Manager) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := m.upgrader.Upgrade(w, r, nil)
		if err != nil {
			m.logger.Printf("Error upgrading connection: %v", err)
			return
		}
		defer conn.Close()

		// Create a channel for this client
		messageChan := make(chan string, 100)

		// Add client to the map
		m.clientsMutex.Lock()
		m.clients[messageChan] = true
		m.clientsMutex.Unlock()

		// Ensure proper cleanup when connection closes
		defer func() {
			// First lock the mutex to prevent race conditions
			m.clientsMutex.Lock()

			// Remove client from map before closing channel
			delete(m.clients, messageChan)

			// Now it's safe to close the channel
			close(messageChan)

			m.clientsMutex.Unlock()

			m.logger.Println("WebSocket connection closed")
		}()

		for {
			_, rawMsg, err := conn.ReadMessage()
			if err != nil {
				m.logger.Printf("Error reading message: %v", err)
				break
			}

			// Parse message as JSON instead of plain string
			var message struct {
				Type    string                 `json:"type"`
				Command string                 `json:"command"`
				Data    map[string]interface{} `json:"data,omitempty"`
			}

			if err := json.Unmarshal(rawMsg, &message); err != nil {
				m.logger.Printf("Error parsing message: %v", err)
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Invalid message format: %v", err)))
				continue
			}

			// Process message based on type
			switch message.Type {
			case "debugger_command":
				if !isGDBRunning {
					conn.WriteMessage(websocket.TextMessage, []byte("Error: GDB is not running. Please start the debugger first"))
					continue
				}

				// Send command to GDB's stdin
				_, err := fmt.Fprintln(m.debuggerSvc.GetStdin(), message.Command)
				if err != nil {
					m.logger.Printf("Error writing to GDB stdin: %v", err)
					conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
				}

			case "ping":
				// Respond to ping with pong
				conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))

			default:
				m.logger.Printf("Unknown message type: %s", message.Type)
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Unknown message type: %s", message.Type)))
			}
		}
	}
}
