package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	// "time" // No longer needed for initial session ID

	"github.com/gorilla/mux"
	"github.com/yourusername/gogdbllm/internal/api"
	"github.com/yourusername/gogdbllm/internal/handlers"
	"github.com/yourusername/gogdbllm/internal/logsession"
	"github.com/yourusername/gogdbllm/internal/settings"
	"github.com/yourusername/gogdbllm/internal/websocket"
)

// SharedLogger holds the active session logger instance, protected by a mutex.
type SharedLogger struct {
	mu     sync.Mutex
	logger *logsession.SessionLogger
}

// Get returns the current logger instance safely.
func (sl *SharedLogger) Get() *logsession.SessionLogger {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.logger
}

// Set sets a new logger instance, closing the old one if it exists.
func (sl *SharedLogger) Set(newLogger *logsession.SessionLogger) {
	sl.mu.Lock()
	oldLogger := sl.logger
	sl.logger = newLogger
	sl.mu.Unlock()

	if oldLogger != nil {
		oldLogger.Close() // Close the previous log file
	}
}

func main() {
	// Initialize settings manager
	settingsManager, err := settings.NewManager("")
	if err != nil {
		log.Fatalf("Failed to initialize settings manager: %v", err)
	}

	// --- Shared Logger Initialization ---
	// Initialize the shared logger holder (logger starts as nil)
	sharedLogger := &SharedLogger{}
	// No initial logger created here
	// defer sharedLogger.Close() // Closing is handled by Set
	// --- Shared Logger Initialization End ---

	// Create uploads directory if it doesn't exist
	uploadsDir := "uploads"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	// Initialize router
	router := mux.NewRouter()

	// Initialize websocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize API handlers (pass shared logger holder)
	fileHandler := handlers.NewFileHandler(uploadsDir, sharedLogger) // Pass sharedLogger
	gdbHandler := handlers.NewGDBHandler(hub, sharedLogger)          // Pass sharedLogger
	chatHandler := api.NewChatHandler(settingsManager, sharedLogger) // Pass sharedLogger
	settingsHandler := handlers.NewSettingsHandler(settingsManager)

	// Register API routes
	router.HandleFunc("/upload", fileHandler.HandleUpload).Methods("POST")
	router.HandleFunc("/ws", websocket.ServeWs(hub, gdbHandler))
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

	// Start server
	addr := ":8080"
	fmt.Printf("Server started on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
