package gdb

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// GDBService manages the interaction with the GDB process
type GDBService struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	outputChan  chan string
	mutex       sync.Mutex
	processLock sync.Mutex
	isRunning   bool
	// Add new fields for capturing command output
	lastOutput     []string
	outputLock     sync.Mutex
	captureEnabled bool
}

// NewGDBService creates a new GDB service
func NewGDBService() *GDBService {
	return &GDBService{
		outputChan:     make(chan string, 100),
		isRunning:      false,
		lastOutput:     make([]string, 0),
		captureEnabled: false,
	}
}

// StartGDB starts a new GDB process for the specified file
func (g *GDBService) StartGDB(filePath string) error {
	g.processLock.Lock()
	defer g.processLock.Unlock()

	// Stop any existing GDB process
	if g.isRunning {
		g.StopGDB()
	}

	// Create a new GDB command
	g.cmd = exec.Command("gdb", filePath)

	// Set up stdin and stdout
	var err error
	g.stdin, err = g.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	g.stdout, err = g.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Start reading from stdout
	go g.readOutput()

	// Start the command
	if err := g.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start GDB: %v", err)
	}

	g.isRunning = true
	return nil
}

// StartOutputCapture begins capturing output
func (g *GDBService) StartOutputCapture() {
	g.outputLock.Lock()
	defer g.outputLock.Unlock()
	g.lastOutput = make([]string, 0)
	g.captureEnabled = true
}

// StopOutputCapture stops capturing output and returns the captured content
func (g *GDBService) StopOutputCapture() string {
	g.outputLock.Lock()
	defer g.outputLock.Unlock()
	g.captureEnabled = false
	output := strings.Join(g.lastOutput, "\n")
	g.lastOutput = make([]string, 0)
	return output
}

// ExecuteCommandWithOutput executes a GDB command and captures its output
func (g *GDBService) ExecuteCommandWithOutput(command string, timeoutSeconds int) (string, error) {
	if !g.isRunning {
		return "", fmt.Errorf("GDB is not running")
	}

	// Start capturing output
	g.StartOutputCapture()

	// Send the command
	if err := g.SendCommand(command); err != nil {
		g.StopOutputCapture() // Make sure to stop capture even on error
		return "", err
	}

	// Wait a bit for the command to complete and output to be captured
	// This is a simple approach - a more robust solution would use a
	// mechanism to detect when the command has completed
	time.Sleep(time.Duration(timeoutSeconds) * time.Second)

	// Stop capturing and get the output
	output := g.StopOutputCapture()
	return output, nil
}

// StopGDB stops the GDB process
func (g *GDBService) StopGDB() error {
	g.processLock.Lock()
	defer g.processLock.Unlock()

	if !g.isRunning {
		return nil
	}

	// Send SIGTERM to process group
	if g.cmd.Process != nil {
		pgid, err := syscall.Getpgid(g.cmd.Process.Pid)
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGTERM)
		}

		// Try to kill the process directly if still running
		g.cmd.Process.Kill()
	}

	g.isRunning = false
	return nil
}

// SendCommand sends a command to GDB
func (g *GDBService) SendCommand(command string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.isRunning {
		return fmt.Errorf("GDB is not running")
	}

	_, err := fmt.Fprintln(g.stdin, command)
	return err
}

// GetOutputChannel returns the channel for GDB output
func (g *GDBService) GetOutputChannel() <-chan string {
	return g.outputChan
}

// IsRunning returns whether GDB is currently running
func (g *GDBService) IsRunning() bool {
	g.processLock.Lock()
	defer g.processLock.Unlock()
	return g.isRunning
}

// readOutput reads the output from GDB and sends it to the output channel
func (g *GDBService) readOutput() {
	scanner := bufio.NewScanner(g.stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if output capture is enabled
		g.outputLock.Lock()
		if g.captureEnabled {
			g.lastOutput = append(g.lastOutput, line)
		}
		g.outputLock.Unlock()

		g.outputChan <- line
	}

	// Process has exited
	g.processLock.Lock()
	g.isRunning = false
	g.processLock.Unlock()

	// Output a message that GDB has exited
	g.outputChan <- "\n[GDB has exited]"

	// Try to send an EOF signal to any waiting goroutines
	if g.stdin != nil {
		g.stdin.Close()
	}

	// Wait for the process to clean up
	if g.cmd.Process != nil {
		g.cmd.Wait()
	}
}
