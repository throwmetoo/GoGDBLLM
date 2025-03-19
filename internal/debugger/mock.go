package debugger

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// MockService implements the Service interface for testing
type MockService struct {
	logger     *log.Logger
	outputChan chan string
	mu         sync.Mutex
	isRunning  bool
	commands   []string
}

// NewMockService creates a new mock debugger service for testing
func NewMockService(logger *log.Logger) Service {
	return &MockService{
		logger:     logger,
		outputChan: make(chan string, 100),
		commands:   make([]string, 0),
	}
}

// Start starts the mock debugger
func (m *MockService) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return nil
	}

	m.isRunning = true
	m.outputChan <- "(gdb) Mock GDB started"
	return nil
}

// Stop stops the mock debugger
func (m *MockService) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	m.isRunning = false
	m.outputChan <- "(gdb) Mock GDB stopped"
	return nil
}

// SendCommand sends a command to the mock debugger
func (m *MockService) SendCommand(command string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("mock GDB is not running")
	}

	m.commands = append(m.commands, command)

	// Generate mock responses based on the command
	switch {
	case strings.HasPrefix(command, "file "):
		filename := strings.TrimPrefix(command, "file ")
		m.outputChan <- fmt.Sprintf("Reading symbols from %s...", filename)
		m.outputChan <- "Reading symbols from /lib/x86_64-linux-gnu/libc.so.6..."
		m.outputChan <- "(gdb) "
	case command == "list":
		m.outputChan <- "1\t#include <stdio.h>"
		m.outputChan <- "2\t"
		m.outputChan <- "3\tint main() {"
		m.outputChan <- "4\t    printf(\"Hello, world!\\n\");"
		m.outputChan <- "5\t    return 0;"
		m.outputChan <- "6\t}"
		m.outputChan <- "(gdb) "
	case command == "break main":
		m.outputChan <- "Breakpoint 1 at 0x1149: file main.c, line 4."
		m.outputChan <- "(gdb) "
	case command == "run":
		m.outputChan <- "Starting program: /tmp/example"
		m.outputChan <- "Breakpoint 1, main () at main.c:4"
		m.outputChan <- "4\t    printf(\"Hello, world!\\n\");"
		m.outputChan <- "(gdb) "
	case command == "next" || command == "n":
		m.outputChan <- "5\t    return 0;"
		m.outputChan <- "(gdb) "
	case command == "continue" || command == "c":
		m.outputChan <- "Continuing."
		m.outputChan <- "Hello, world!"
		m.outputChan <- "[Inferior 1 (process 12345) exited normally]"
		m.outputChan <- "(gdb) "
	case command == "quit" || command == "q":
		m.outputChan <- "Quitting..."
		m.isRunning = false
	default:
		m.outputChan <- fmt.Sprintf("Unknown command: %s", command)
		m.outputChan <- "(gdb) "
	}

	return nil
}

// OutputChannel returns the channel for mock debugger output
func (m *MockService) OutputChannel() <-chan string {
	return m.outputChan
}

// Shutdown cleans up resources
func (m *MockService) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		m.isRunning = false
	}

	close(m.outputChan)
	return nil
}

// GetCommands returns the list of commands sent to the mock debugger
func (m *MockService) GetCommands() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]string{}, m.commands...)
}
