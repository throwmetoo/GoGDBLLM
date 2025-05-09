package gdb

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/gogdbllm/internal/config"
	"github.com/yourusername/gogdbllm/internal/errors"
)

// TestGDBServiceBasics tests the basic functionality of the GDB service
func TestGDBServiceBasics(t *testing.T) {
	// Skip this test if GDB is not installed
	if _, err := os.Stat("/usr/bin/gdb"); os.IsNotExist(err) {
		t.Skip("GDB not installed, skipping test")
	}

	// Create a simple test program
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "test.c")
	execFile := filepath.Join(tempDir, "test")

	// Write a simple C program
	source := `
	#include <stdio.h>
	
	int main() {
		int x = 5;
		printf("Hello, world! x = %d\\n", x);
		return 0;
	}
	`

	err := ioutil.WriteFile(sourceFile, []byte(source), 0644)
	assert.NoError(t, err)

	// Compile the program
	err = exec.Command("gcc", "-g", "-o", execFile, sourceFile).Run()
	if err != nil {
		t.Skip("Failed to compile test program, skipping test:", err)
	}

	// Create config
	cfg := &config.Config{
		GDB: config.GDBConfig{
			Path:    "gdb",
			Timeout: 2,
		},
	}

	// Create GDB service
	gdbService := NewGDBService(cfg)
	assert.NotNil(t, gdbService)
	assert.False(t, gdbService.IsRunning())

	// Test error when GDB is not running
	_, err = gdbService.ExecuteCommandWithOutput("info breakpoints", 1)
	assert.Equal(t, errors.ErrGDBNotRunning, err)

	// Start GDB
	err = gdbService.StartGDB(execFile)
	assert.NoError(t, err)
	assert.True(t, gdbService.IsRunning())

	// Give GDB time to initialize
	time.Sleep(1 * time.Second)

	// Test executing a command
	output, err := gdbService.ExecuteCommandWithOutput("info breakpoints", 1)
	assert.NoError(t, err)
	assert.Contains(t, output, "No breakpoints")

	// Test stopping GDB
	err = gdbService.StopGDB()
	assert.NoError(t, err)
	assert.False(t, gdbService.IsRunning())
}

// TestGDBOutputCapture tests the output capture functionality
func TestGDBOutputCapture(t *testing.T) {
	// Skip this test if GDB is not installed
	if _, err := os.Stat("/usr/bin/gdb"); os.IsNotExist(err) {
		t.Skip("GDB not installed, skipping test")
	}

	// Create config
	cfg := &config.Config{
		GDB: config.GDBConfig{
			Path:    "gdb",
			Timeout: 2,
		},
	}

	// Create GDB service
	gdbService := NewGDBService(cfg)

	// Test capture when not running
	gdbService.StartOutputCapture()
	assert.Equal(t, "", gdbService.StopOutputCapture())

	// Create a sample output manually
	gdbService.outputLock.Lock()
	gdbService.captureEnabled = true
	gdbService.lastOutput = []string{"Line 1", "Line 2", "Line 3"}
	gdbService.outputLock.Unlock()

	// Test capturing output
	output := gdbService.StopOutputCapture()
	assert.Equal(t, "Line 1\nLine 2\nLine 3", output)
	assert.False(t, gdbService.captureEnabled)
	assert.Empty(t, gdbService.lastOutput)
}

// Test mocking would be implemented here in a real-world scenario
// For this example, we'll use skippable integration tests
