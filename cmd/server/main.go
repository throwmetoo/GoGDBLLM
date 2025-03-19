package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/throwmetoo/gogdbllm/internal/api"
	"github.com/throwmetoo/gogdbllm/internal/config"
	"github.com/throwmetoo/gogdbllm/internal/debugger"
	"github.com/throwmetoo/gogdbllm/internal/llm"
	"github.com/throwmetoo/gogdbllm/internal/websocket"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "[GoGDBLLM] ", log.LstdFlags)
	logger.Println("Starting GoGDBLLM server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		logger.Fatalf("Failed to create upload directory: %v", err)
	}

	// Initialize components
	debuggerSvc := debugger.NewService(logger, cfg.GDBPath)
	llmClient := llm.NewClient(cfg.LLMSettings, logger)
	wsManager := websocket.NewManager(logger)

	// Create API handlers
	apiHandler := api.NewHandler(
		logger,
		cfg,
		debuggerSvc,
		llmClient,
		wsManager,
	)

	// Set up HTTP server
	mux := http.NewServeMux()

	// Register API routes
	mux.Handle("/api/v1/upload", apiHandler.UploadHandler())
	mux.Handle("/api/v1/settings", apiHandler.SettingsHandler())
	mux.Handle("/api/v1/chat", apiHandler.ChatHandler())
	mux.Handle("/api/v1/debugger/start", apiHandler.StartDebuggerHandler())
	mux.Handle("/api/v1/debugger/command", apiHandler.DebuggerCommandHandler())
	mux.Handle("/api/v1/test-connection", apiHandler.TestConnectionHandler())
	mux.Handle("/api/v1/debugger/stop", apiHandler.DebuggerStopHandler())
	mux.Handle("/ws", wsManager.Handler())

	// Serve static files
	mux.Handle("/", http.FileServer(http.FS(api.StaticFiles)))

	// Configure the HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		logger.Printf("Server listening on http://localhost:%d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Println("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	// Clean up resources
	debuggerSvc.Shutdown()
	wsManager.Shutdown()

	logger.Println("Server stopped gracefully")
}
