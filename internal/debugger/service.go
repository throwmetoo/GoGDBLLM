package debugger

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Service defines the interface for the debugger service
type Service interface {
	Start() error
	Stop() error
	SendCommand(command string) error
	OutputChannel() <-chan string
	Shutdown() error
}

// GDBService implements the Service interface for GDB
type GDBService struct {
	logger        *log.Logger
	gdbPath       string
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	outputChan    chan string
	mu            sync.Mutex
	isRunning     bool
	currentTarget string
}

// NewService creates a new debugger service
func NewService(logger *log.Logger, gdbPath string) Service {
	return &GDBService{
		logger:     logger,
		gdbPath:    gdbPath,
		outputChan: make(chan string, 100),
	}
}

// Start starts the GDB process
func (g *GDBService) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// If already running, stop it first
	if g.isRunning {
		g.stopProcess()
	}

	// Start a new GDB process
	g.logger.Println("Starting GDB process...")
	cmd := exec.Command(g.gdbPath, "-q")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Set process group ID for proper signal handling
	}

	// Set up pipes for stdin/stdout/stderr
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
	g.cmd = cmd
	g.stdin = stdin
	g.stdout = stdout
	g.stderr = stderr
	g.isRunning = true

	// Start a goroutine to read from stdout and stderr
	go g.readOutput(io.MultiReader(stdout, stderr))

	return nil
}

// Stop stops the GDB process and cleans up resources
func (g *GDBService) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.isRunning {
		return nil // Already stopped
	}

	// Send quit command to GDB
	if g.stdin != nil {
		fmt.Fprintln(g.stdin, "quit")
		fmt.Fprintln(g.stdin, "y")
	}

	// Give GDB a chance to exit gracefully
	done := make(chan error, 1)
	go func() {
		done <- g.cmd.Wait()
	}()

	// Wait for process to exit or force kill after timeout
	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(3 * time.Second):
		// Force kill if it doesn't exit
		g.logger.Println("GDB didn't exit gracefully, forcing termination")
		if err := g.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill GDB process: %w", err)
		}
	}

	// Clean up resources
	if g.stdout != nil {
		g.stdout.Close()
	}
	if g.stderr != nil {
		g.stderr.Close()
	}

	// Reset state
	g.cmd = nil
	g.stdin = nil
	g.stdout = nil
	g.stderr = nil
	g.isRunning = false

	// Close output channel
	close(g.outputChan)
	g.outputChan = make(chan string, 100)

	g.logger.Println("Debugger stopped successfully")
	return nil
}

// Add this method to your GDBService struct
func (g *GDBService) stopProcess() error {
	if g.cmd == nil || g.cmd.Process == nil {
		return nil // Already stopped
	}

	// Send quit command to GDB
	if g.stdin != nil {
		fmt.Fprintln(g.stdin, "quit")
		fmt.Fprintln(g.stdin, "y")
	}

	// Give GDB a chance to exit gracefully
	done := make(chan error, 1)
	go func() {
		done <- g.cmd.Wait()
	}()

	// Wait for process to exit or force kill after timeout
	select {
	case <-done:
		// Process exited gracefully
	case <-time.After(3 * time.Second):
		// Force kill if it doesn't exit
		g.logger.Println("GDB didn't exit gracefully, forcing termination")
		if err := g.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill GDB process: %w", err)
		}
	}

	return nil
}

// SendCommand sends a command to GDB
func (g *GDBService) SendCommand(command string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.isRunning {
		return fmt.Errorf("GDB is not running")
	}

	// Check if this is a file command to load a target
	if len(command) > 5 && command[:5] == "file " {
		g.currentTarget = command[5:]
	}

	_, err := fmt.Fprintln(g.stdin, command)
	if err != nil {
		return fmt.Errorf("failed to send command to GDB: %w", err)
	}

	return nil
}

// OutputChannel returns the channel for GDB output
func (g *GDBService) OutputChannel() <-chan string {
	return g.outputChan
}

// Shutdown cleans up resources
func (g *GDBService) Shutdown() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.isRunning {
		g.stopProcess()
	}

	close(g.outputChan)
	return nil
}

// readOutput reads from the given reader and sends the output to the output channel
func (g *GDBService) readOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		g.logger.Printf("GDB output: %s", text)

		// Send to output channel, but don't block if it's full
		select {
		case g.outputChan <- text:
			// Sent successfully
		default:
			g.logger.Println("Output channel full, dropping message")
		}
	}

	if err := scanner.Err(); err != nil {
		g.logger.Printf("Error reading GDB output: %v", err)
	}
}
