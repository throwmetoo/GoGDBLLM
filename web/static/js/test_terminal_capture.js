/**
 * Simple test script to verify terminal output capturing functionality
 * 
 * This script simulates both user commands and LLM-initiated commands
 * to ensure our "Last Cmd" button correctly captures all output.
 */

function testTerminalCapture() {
    // Only run this test if the terminal interface is available
    if (!window.AppTerminal) {
        console.error("AppTerminal is not available. Test cannot run.");
        return;
    }
    
    console.log("Starting terminal capture test...");
    
    // Simulate a user command
    console.log("1. Simulating user command...");
    window.AppTerminal.appendToTerminal("User command: help\n");
    window.AppTerminal.appendToTerminal("Here's the help output for the user command\n");
    
    // Check capture
    let output1 = window.AppTerminal.getLastCommandOutput();
    console.log("Output after user command:", output1.length, "characters");
    
    // Simulate LLM-initiated command
    console.log("2. Simulating LLM command...");
    window.AppTerminal.appendToTerminal("=== EXECUTING GDB COMMANDS ===\n");
    window.AppTerminal.appendToTerminal("Commands: info breakpoints\n");
    window.AppTerminal.appendToTerminal("=== END COMMAND LIST ===\n");
    window.AppTerminal.appendToTerminal("No breakpoints or watchpoints.\n");
    
    // Check capture again
    let output2 = window.AppTerminal.getLastCommandOutput();
    console.log("Output after LLM command:", output2.length, "characters");
    
    // Verify if both outputs were captured in the buffer
    if (output2.includes("User command") && output2.includes("EXECUTING GDB COMMANDS")) {
        console.log("✅ TEST PASSED: Both user and LLM commands were captured");
    } else {
        console.error("❌ TEST FAILED: Not all outputs were captured");
        console.log("Captured content:", output2);
    }
}

// Add a button to run the test
function addTestButton() {
    const button = document.createElement('button');
    button.textContent = 'Test Terminal Capture';
    button.style.position = 'fixed';
    button.style.bottom = '10px';
    button.style.left = '10px';
    button.style.zIndex = '9999';
    button.style.display = 'none'; // Hidden by default, enable in console for testing
    
    button.addEventListener('click', testTerminalCapture);
    document.body.appendChild(button);
    
    console.log("Terminal capture test button added. Enable with: document.querySelector('button').style.display = 'block'");
}

// Call this function to set up the test
window.addEventListener('load', addTestButton); 