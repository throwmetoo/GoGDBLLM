package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections (you might want to restrict this in production)
	},
}

// Handler handles WebSocket connections
type Handler struct {
	hub    *Hub
	logger *log.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(logger *log.Logger) *Handler {
	hub := NewHub(logger)
	go hub.Run()

	return &Handler{
		hub:    hub,
		logger: logger,
	}
}

// ServeHTTP handles WebSocket requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Printf("Error upgrading connection: %v", err)
		return
	}

	client := NewClient(h.hub, conn, h.logger)
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

// Broadcast sends a message to all connected clients
func (h *Handler) Broadcast(message []byte) {
	h.hub.Broadcast(message)
}

// ClientCount returns the number of connected clients
func (h *Handler) ClientCount() int {
	return h.hub.ClientCount()
}
