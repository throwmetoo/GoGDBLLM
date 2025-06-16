package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/yourusername/gogdbllm/internal/api"
	"github.com/yourusername/gogdbllm/internal/config"
	"github.com/yourusername/gogdbllm/internal/di"
	"github.com/yourusername/gogdbllm/internal/handlers"
	"github.com/yourusername/gogdbllm/internal/websocket"
)

var diContainer *di.Container

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	genConfig := flag.String("gen-config", "", "Generate default configuration file at specified path and exit")
	flag.Parse()

	// Generate config file if requested
	if *genConfig != "" {
		if err := config.WriteDefaultConfig(*genConfig); err != nil {
			log.Fatalf("Failed to generate configuration file: %v", err)
		}
		fmt.Printf("Default configuration written to %s\n", *genConfig)
		return
	}

	// Create DI container
	diContainer = di.NewContainer()
	if err := diContainer.Configure(*configPath); err != nil {
		log.Fatalf("Failed to configure container: %v", err)
	}

	// Run application with DI container
	if err := diContainer.Invoke(run); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}

// run is the main application function that gets invoked with dependencies
func run(cfg *config.Config) error {
	// Create uploads directory if it doesn't exist
	uploadsDir := cfg.Uploads.Directory
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create uploads directory: %v", err)
	}

	// Initialize router
	router := mux.NewRouter()

	// Setup routes and handlers using dependency injection
	if err := setupRoutes(router); err != nil {
		return fmt.Errorf("failed to setup routes: %v", err)
	}

	// Configure and start the HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		fmt.Printf("Server started on http://localhost%s\n", addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for interrupt/terminate signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-shutdown:
		fmt.Println("\nShutting down gracefully...")

		// Create a context with a timeout to tell the server how long to wait
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt to gracefully shutdown the server
		if err := server.Shutdown(ctx); err != nil {
			// Force shutdown if graceful shutdown fails
			server.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

// setupRoutes configures all the routes for the application
func setupRoutes(router *mux.Router) error {
	// This will be automatically invoked by the DI container with all required dependencies
	return diContainer.Invoke(func(
		fileHandler *handlers.FileHandler,
		gdbHandler *handlers.GDBHandler,
		settingsHandler *handlers.SettingsHandler,
		chatHandler *api.SimpleChatHandler,
		wsHub *websocket.Hub,
	) {
		// Register API routes
		router.HandleFunc("/upload", fileHandler.HandleUpload).Methods("POST")
		router.HandleFunc("/ws", websocket.ServeWs(wsHub, gdbHandler))
		router.HandleFunc("/start-gdb", gdbHandler.HandleStartGDB).Methods("POST")
		router.HandleFunc("/api/chat", chatHandler.HandleChat).Methods("POST")
		router.HandleFunc("/api/settings", settingsHandler.GetSettings).Methods("GET")
		router.HandleFunc("/save-settings", settingsHandler.SaveSettings).Methods("POST")
		router.HandleFunc("/test-connection", settingsHandler.TestConnection).Methods("POST")

		// Serve static files
		fs := http.FileServer(http.Dir("./web/static"))
		router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

		// Serve index page
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join("web/templates", "index.html"))
		})

		// Health check endpoint
		router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Server is working"))
		})

		// Start WebSocket hub
		go wsHub.Run()
	})
}
