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
	"path/filepath"
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
	}

	content, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create sub FS: %v", err)
	}

	fileServer := http.FileServer(http.FS(content))
	http.Handle("/", fileServer)
	http.HandleFunc("/upload", server.uploadHandler)
	http.HandleFunc("/ws", server.wsHandler)
	http.HandleFunc("/test-connection", server.testConnectionHandler)
	http.HandleFunc("/api/settings", server.settingsHandler)
	http.HandleFunc("/api/chat", server.HandleChat)
	http.HandleFunc("/save-settings", server.handleSaveSettings)
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is working"))
	})

	fmt.Println("Serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("Parse form error: %v", err), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to retrieve file from form", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmpDir := os.TempDir()
	tmpFilePath := filepath.Join(tmpDir, header.Filename)
	outFile, err := os.Create(tmpFilePath)
	if err != nil {
		http.Error(w, "Unable to create file on the server", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	err = os.Chmod(tmpFilePath, 0755)
	if err != nil {
		http.Error(w, "Unable to set file permissions", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "File uploaded and ready for execution")
}

func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// Create a command processor to manage the interactive session
	var cmd *exec.Cmd
	var stdin io.WriteCloser
	var isGDBRunning bool

	cleanup := func() {
		if stdin != nil {
			stdin.Close()
		}
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
		}
		isGDBRunning = false
	}
	defer cleanup()

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			command := string(rawMsg)
			if strings.HasPrefix(command, "/tmp/") {
				s.startGDBSession(command, conn, &cmd, &stdin, &isGDBRunning)
			} else if isGDBRunning {
				s.sendCommandToGDB(command, conn, stdin)
			} else {
				conn.WriteMessage(websocket.TextMessage, []byte("Error: GDB is not running. Please upload and execute a file first"))
			}
			continue
		}

		switch msg.Type {
		case "special":
			s.handleSpecialCommand(msg.Command, conn, cmd, stdin, &isGDBRunning)
		case "regular":
			if strings.HasPrefix(msg.Command, "/tmp/") {
				s.startGDBSession(msg.Command, conn, &cmd, &stdin, &isGDBRunning)
			} else if isGDBRunning {
				s.sendCommandToGDB(msg.Command, conn, stdin)
			} else {
				conn.WriteMessage(websocket.TextMessage, []byte("Error: GDB is not running. Please upload and execute a file first."))
			}
		default:
			conn.WriteMessage(websocket.TextMessage, []byte("Unknown message type"))
		}
	}
}

// Function to handle special commands like CTRL+C
func (s *Server) handleSpecialCommand(commandType string, conn *websocket.Conn, cmd *exec.Cmd, stdin io.WriteCloser, isGDBRunning *bool) {
	if !*isGDBRunning || cmd == nil || cmd.Process == nil {
		conn.WriteMessage(websocket.TextMessage, []byte("No running GDB process to control"))
		return
	}

	switch commandType {
	case "CTRL_C":
		// Send SIGINT to the process group
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			log.Printf("Error getting process group: %v", err)
			conn.WriteMessage(websocket.TextMessage, []byte("Error interrupting process"))
			return
		}
		if err := syscall.Kill(-pgid, syscall.SIGINT); err != nil {
			log.Printf("Error sending SIGINT: %v", err)
			conn.WriteMessage(websocket.TextMessage, []byte("Error interrupting process"))
		}
	case "CTRL_Z":
		// Send SIGTSTP to the process group
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err != nil {
			log.Printf("Error getting process group: %v", err)
			return
		}
		if err := syscall.Kill(-pgid, syscall.SIGTSTP); err != nil {
			log.Printf("Error sending SIGTSTP: %v", err)
		}
	case "CTRL_D":
		// Send EOF to the process
		conn.WriteMessage(websocket.TextMessage, []byte("Sending EOF to GDB"))
		// Implementation depends on your specific requirements
	case "ARROW_UP":
		// These would typically access command history
		// For GDB, we'd send the appropriate escape sequence
		s.sendCommandToGDB("\x1b[A", conn, stdin)
	case "ARROW_DOWN":
		s.sendCommandToGDB("\x1b[B", conn, stdin)
	default:
		conn.WriteMessage(websocket.TextMessage, []byte("Unknown special command: "+commandType))
	}
}

// Function to start a new GDB session
func (s *Server) startGDBSession(filePath string, conn *websocket.Conn, cmdPtr **exec.Cmd, stdinPtr *io.WriteCloser, isGDBRunning *bool) {
	// Clean up any existing process
	if *isGDBRunning && *cmdPtr != nil && (*cmdPtr).Process != nil {
		(*cmdPtr).Process.Kill()
		*isGDBRunning = false
	}

	// Create a new command that will have its own process group
	cmd := exec.Command("gdb", filePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Set process group ID for proper signal handling
	}

	// Get stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Error getting stdin pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	// Get stdout and stderr pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error getting stdout pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Error getting stderr pipe: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}

	// Start command
	if err := cmd.Start(); err != nil {
		log.Printf("Error starting GDB: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Failed to start GDB: %v", err)))
		return
	}

	*cmdPtr = cmd
	*stdinPtr = stdin
	*isGDBRunning = true

	// Read output in a goroutine
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			text := scanner.Text()
			err := conn.WriteMessage(websocket.TextMessage, []byte(text))
			if err != nil {
				log.Printf("Error writing to WebSocket: %v", err)
				return
			}
		}

		// Check if GDB exited
		if err := cmd.Wait(); err != nil {
			log.Printf("GDB exited with error: %v", err)
		} else {
			log.Println("GDB exited normally")
		}

		*isGDBRunning = false
	}()
}

// Function to send a command to GDB
func (s *Server) sendCommandToGDB(command string, conn *websocket.Conn, stdin io.WriteCloser) {
	// Send command to GDB's stdin
	_, err := fmt.Fprintln(stdin, command)
	if err != nil {
		log.Printf("Error writing to GDB stdin: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
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
