package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

// Embed all files in the "static" folder into the Go binary.
//
//go:embed static
var staticFiles embed.FS

type ConnectionTestRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

// Define a struct to parse incoming WebSocket messages
type WebSocketMessage struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server represents the HTTP server
type Server struct {
	settingsManager *SettingsManager
	terminalBuffer  bytes.Buffer
	bufferMutex     sync.Mutex
	gdbCmd          *exec.Cmd
	gdbStdin        io.WriteCloser
	gdbStdout       io.ReadCloser
	clients         map[chan string]bool
	clientsMutex    sync.Mutex
}

func main() {
	// Initialize settings manager
	var err error
	settingsManager, err := NewSettingsManager("")
	if err != nil {
		log.Fatalf("Failed to initialize settings manager: %v", err)
	}

	// Create server instance
	server := &Server{
		settingsManager: settingsManager,
		clients:         make(map[chan string]bool),
	}

	content, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create sub FS: %v", err)
	}

	fileServer := http.FileServer(http.FS(content))
	http.Handle("/", fileServer)
	http.HandleFunc("/upload", server.handleUpload)
	http.HandleFunc("/ws", server.wsHandler)
	http.HandleFunc("/test-connection", server.testConnectionHandler)
	http.HandleFunc("/api/settings", server.settingsHandler)
	http.HandleFunc("/api/chat", server.HandleChat)
	http.HandleFunc("/save-settings", server.handleSaveSettings)
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is working"))
	})
	http.HandleFunc("/start-gdb", server.handleStartGDB)

	fmt.Println("Serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	// Set JSON content type header
	w.Header().Set("Content-Type", "application/json")

	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to parse form: " + err.Error(),
		})
		return
	}

	// Get the file from form data
	file, header, err := r.FormFile("file")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to get file: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll("uploads", 0755); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to create uploads directory: " + err.Error(),
		})
		return
	}

	// Create the file
	filepath := path.Join("uploads", header.Filename)
	dst, err := os.Create(filepath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to create file: " + err.Error(),
		})
		return
	}
	defer dst.Close()

	// Copy the uploaded file
	if _, err := io.Copy(dst, file); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to save file: " + err.Error(),
		})
		return
	}

	// Make the file executable
	if err := os.Chmod(filepath, 0755); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to make file executable: " + err.Error(),
		})
		return
	}

	// Return success response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"filename": header.Filename,
		"filepath": filepath,
	})
}

func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Create a channel for this client
	messageChan := make(chan string, 100)
	s.clientsMutex.Lock()
	s.clients[messageChan] = true
	s.clientsMutex.Unlock()
	defer func() {
		s.clientsMutex.Lock()
		delete(s.clients, messageChan)
		s.clientsMutex.Unlock()
	}()

	// Create a command processor to manage the interactive session
	var isGDBRunning bool

	// Start a goroutine to send messages to the WebSocket
	go func() {
		for msg := range messageChan {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		}
	}()

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		command := string(rawMsg)
		if strings.HasPrefix(command, "file ") {
			if s.gdbCmd == nil || s.gdbStdin == nil {
				conn.WriteMessage(websocket.TextMessage, []byte("Error: GDB is not running. Please upload and execute a file first"))
				continue
			}
			isGDBRunning = true
		}

		if isGDBRunning {
			// Send command to GDB's stdin
			_, err := fmt.Fprintln(s.gdbStdin, command)
			if err != nil {
				log.Printf("Error writing to GDB stdin: %v", err)
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
			}
		} else {
			conn.WriteMessage(websocket.TextMessage, []byte("Error: GDB is not running. Please upload and execute a file first"))
		}
	}
}

func (s *Server) testConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectionTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Test the connection based on the provider
	var err error
	switch req.Provider {
	case "anthropic":
		err = testAnthropicConnection(req.APIKey, req.Model)
	case "openai":
		err = testOpenAIConnection(req.APIKey, req.Model)
	case "openrouter":
		err = testOpenRouterConnection(req.APIKey, req.Model)
	default:
		http.Error(w, "Invalid provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

func testAnthropicConnection(apiKey string, model string) error {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", strings.NewReader(`{
		"model": "`+model+`",
		"max_tokens": 1,
		"messages": [{"role": "user", "content": "test"}]
	}`))

	if err != nil {
		return err
	}

	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", body)
	}

	return nil
}

func testOpenAIConnection(apiKey, model string) error {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", strings.NewReader(`{
		"model": "`+model+`",
		"max_tokens": 1,
		"messages": [{"role": "user", "content": "test"}]
	}`))

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", body)
	}

	return nil
}

func testOpenRouterConnection(apiKey, model string) error {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", strings.NewReader(`{
		"model": "`+model+`",
		"max_tokens": 1,
		"messages": [{"role": "user", "content": "test"}]
	}`))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "localhost:8080")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", body)
	}

	return nil
}

// settingsHandler handles requests for settings
func (s *Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	// Get settings
	settings := s.settingsManager.GetSettings()

	// Return settings as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

// handleSaveSettings handles the request to save settings
func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	// Parse the settings from the request
	var settings Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		log.Printf("Error parsing settings: %v", err)
		http.Error(w, "Invalid settings format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Save the settings using the settings manager
	if err := s.settingsManager.SaveSettings(settings); err != nil {
		log.Printf("Error saving settings: %v", err)
		http.Error(w, "Failed to save settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Settings saved successfully"))
}

func (s *Server) handleStartGDB(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse the request body
	var req struct {
		Filepath string `json:"filepath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to parse request: " + err.Error(),
		})
		return
	}

	// Validate the filepath
	if req.Filepath == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "No filepath provided",
		})
		return
	}

	// Check if the file exists
	if _, err := os.Stat(req.Filepath); os.IsNotExist(err) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "File does not exist: " + req.Filepath,
		})
		return
	}

	// Start GDB (you'll need to implement this based on your existing GDB handling code)
	if err := s.startGDB(); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to start GDB: " + err.Error(),
		})
		return
	}

	// Return success
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (s *Server) startGDB() error {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	// Kill any existing GDB process
	if s.gdbCmd != nil && s.gdbCmd.Process != nil {
		s.gdbCmd.Process.Kill()
		s.gdbCmd.Wait() // Clean up the process
	}

	// Start a new GDB process
	cmd := exec.Command("gdb", "-q")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Set process group ID for proper signal handling
	}

	// Set up pipes for stdin/stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start GDB: %w", err)
	}

	// Store the command and pipes
	s.gdbCmd = cmd
	s.gdbStdin = stdin
	s.gdbStdout = stdout

	// Start a goroutine to read from stdout and stderr
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			text := scanner.Text()
			s.broadcastToClients(text)
		}
	}()

	return nil
}

func (s *Server) broadcastToClients(message string) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	for client := range s.clients {
		// Try to send, but don't block if it fails
		select {
		case client <- message:
			// Message sent successfully
		default:
			// Client buffer is full or disconnected, remove it
			close(client)
			delete(s.clients, client)
		}
	}
}

func NewServer() *Server {
	return &Server{
		clients: make(map[chan string]bool),
	}
}
