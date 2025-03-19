package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/throwmetoo/GoGDBLLM/internal/debugger"
)

// Add this variable as a field in the Manager struct
type Manager struct {
	clients      map[chan string]bool
	clientsMutex sync.Mutex
	logger       *log.Logger
	upgrader     websocket.Upgrader
	debuggerSvc  debugger.Service
	isGDBRunning bool // Add this field
}

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
				// Then replace the direct reference to isGDBRunning with m.isGDBRunning
				if !m.isGDBRunning {
					conn.WriteMessage(websocket.TextMessage, []byte("Error: GDB is not running. Please start the debugger first"))
					continue
				}

				// Send command to GDB using SendCommand method
				err := m.debuggerSvc.SendCommand(message.Command)
				if err != nil {
					m.logger.Printf("Error sending command to GDB: %v", err)
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

// NewManager creates a new websocket manager
func NewManager(logger *log.Logger) *Manager {
	return &Manager{
		clients:      make(map[chan string]bool),
		clientsMutex: sync.Mutex{},
		logger:       logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
		isGDBRunning: false,
	}
}

// RegisterOutputChannel registers an output channel from the debugger
func (m *Manager) RegisterOutputChannel(ch <-chan string) {
	go func() {
		for msg := range ch {
			m.clientsMutex.Lock()
			for client := range m.clients {
				select {
				case client <- msg:
					// Message sent successfully
				default:
					// Channel buffer is full, skip this message
					m.logger.Printf("Client channel buffer full, dropping message")
				}
			}
			m.clientsMutex.Unlock()
		}
	}()
}

// UnregisterOutputChannel unregisters an output channel
func (m *Manager) UnregisterOutputChannel(ch <-chan string) {
	// Nothing to do here, the channel should be closed by the owner
}

// Shutdown closes all client connections
func (m *Manager) Shutdown() {
	m.clientsMutex.Lock()
	defer m.clientsMutex.Unlock()

	// Close all client channels
	for client := range m.clients {
		close(client)
		delete(m.clients, client)
	}
}

// SetDebuggerService sets the debugger service
func (m *Manager) SetDebuggerService(svc debugger.Service) {
	m.debuggerSvc = svc
}

// SetGDBRunning sets the GDB running state
func (m *Manager) SetGDBRunning(running bool) {
	m.isGDBRunning = running
}
