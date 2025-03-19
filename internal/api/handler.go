package api

import (
	"embed"
	"log"
	"net/http"

	"github.com/throwmetoo/GoGDBLLM/internal/config"
	"github.com/throwmetoo/GoGDBLLM/internal/debugger"
	"github.com/throwmetoo/GoGDBLLM/internal/llm"
	"github.com/throwmetoo/GoGDBLLM/internal/websocket"
)

//go:embed static/index.html static/js/* static/css/*
var StaticFiles embed.FS

// Handler manages all API endpoints
type Handler struct {
	logger      *log.Logger
	config      *config.Config
	debuggerSvc debugger.Service
	llmClient   llm.Client
	wsManager   *websocket.Manager
}

// NewHandler creates a new API handler
func NewHandler(
	logger *log.Logger,
	cfg *config.Config,
	debuggerSvc debugger.Service,
	llmClient llm.Client,
	wsManager *websocket.Manager,
) *Handler {
	return &Handler{
		logger:      logger,
		config:      cfg,
		debuggerSvc: debuggerSvc,
		llmClient:   llmClient,
		wsManager:   wsManager,
	}
}

// UploadHandler returns a handler for file uploads
func (h *Handler) UploadHandler() http.HandlerFunc {
	return h.handleUpload
}

// SettingsHandler returns a handler for settings management
func (h *Handler) SettingsHandler() http.HandlerFunc {
	return h.handleSettings
}

// ChatHandler returns a handler for chat requests
func (h *Handler) ChatHandler() http.HandlerFunc {
	return h.handleChat
}

// StartDebuggerHandler returns a handler for starting the debugger
func (h *Handler) StartDebuggerHandler() http.HandlerFunc {
	return h.handleStartDebugger
}

// DebuggerCommandHandler returns a handler for sending commands to the debugger
func (h *Handler) DebuggerCommandHandler() http.HandlerFunc {
	return h.handleDebuggerCommand
}

// TestConnectionHandler returns a handler for testing LLM connections
func (h *Handler) TestConnectionHandler() http.HandlerFunc {
	return h.handleTestConnection
}

// DebuggerStopHandler returns a handler for stopping the debugger
func (h *Handler) DebuggerStopHandler() http.HandlerFunc {
	return h.handleStopDebugger
}
