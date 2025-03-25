/**
 * terminal.js - Handles terminal interface and WebSocket communication
 */

// Initialize terminal section
function initTerminalSection() {
    const terminal = document.getElementById('terminal');
    const commandInput = document.getElementById('commandInput');
    const commandPrompt = document.getElementById('commandPrompt');
    const terminalOutput = document.getElementById('terminalOutput');
    
    let socket = null;
    let commandHistory = [];
    let historyIndex = -1;
    let terminalConnected = false;
    
    // Special control characters
    const CTRL_C = '\x03';  // Control-C character
    const CTRL_D = '\x04';  // Control-D character
    
    // Initialize WebSocket connection
    function connectWebSocket() {
        if (socket) {
            socket.close();
        }
        
        // Create WebSocket connection
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        socket = new WebSocket(wsUrl);
        
        // Connection opened
        socket.addEventListener('open', (event) => {
            terminalConnected = true;
            appendToTerminal('Terminal connected');
            console.log('WebSocket connection established');
        });
        
        // Listen for messages from server
        socket.addEventListener('message', (event) => {
            appendToTerminal(event.data);
        });
        
        // Connection closed
        socket.addEventListener('close', (event) => {
            terminalConnected = false;
            appendToTerminal('\nTerminal disconnected');
            console.log('WebSocket connection closed');
            
            // Try to reconnect after 3 seconds
            setTimeout(() => {
                if (!terminalConnected) {
                    connectWebSocket();
                }
            }, 3000);
        });
        
        // Connection error
        socket.addEventListener('error', (error) => {
            console.error('WebSocket error:', error);
            terminalConnected = false;
        });
    }
    
    // Append text to terminal and terminal output in chat panel
    function appendToTerminal(text) {
        terminal.textContent += text + '\n';
        terminal.scrollTop = terminal.scrollHeight;
        
        // Update chat panel if it's open
        if (document.getElementById('chatPanel').classList.contains('open')) {
            terminalOutput.textContent = terminal.textContent;
        }
    }
    
    // Send command to server
    function sendCommand(command) {
        if (!terminalConnected) {
            appendToTerminal('Terminal not connected. Reconnecting...');
            connectWebSocket();
            return;
        }
        
        // Add command to history
        if (command.trim() !== '') {
            commandHistory.unshift(command);
            historyIndex = -1;
            
            // Limit history size
            if (commandHistory.length > 100) {
                commandHistory.pop();
            }
        }
        
        // Send command to server
        try {
            socket.send(JSON.stringify({
                type: 'command',
                command: command
            }));
        } catch (error) {
            console.error('Error sending command:', error);
            appendToTerminal(`Error sending command: ${error.message}`);
        }
    }
    
    // Handle command input
    commandInput.addEventListener('keydown', (e) => {
        // Enter key to send command
        if (e.key === 'Enter') {
            e.preventDefault();
            
            const command = commandInput.value;
            appendToTerminal(`${commandPrompt.textContent} ${command}`);
            sendCommand(command);
            commandInput.value = '';
        }
        
        // Up arrow to navigate command history
        else if (e.key === 'ArrowUp') {
            e.preventDefault();
            
            if (commandHistory.length > 0 && historyIndex < commandHistory.length - 1) {
                historyIndex++;
                commandInput.value = commandHistory[historyIndex];
            }
        }
        
        // Down arrow to navigate command history
        else if (e.key === 'ArrowDown') {
            e.preventDefault();
            
            if (historyIndex > 0) {
                historyIndex--;
                commandInput.value = commandHistory[historyIndex];
            } else if (historyIndex === 0) {
                historyIndex = -1;
                commandInput.value = '';
            }
        }
        
        // Ctrl+C to send interrupt
        else if (e.key === 'c' && e.ctrlKey) {
            e.preventDefault();
            sendCommand(CTRL_C);
            appendToTerminal('^C');
        }
        
        // Ctrl+D to send EOF
        else if (e.key === 'd' && e.ctrlKey) {
            e.preventDefault();
            sendCommand(CTRL_D);
            appendToTerminal('^D');
        }
    });
    
    // Prevent password manager detection
    commandInput.setAttribute('autocomplete', 'off');
    
    // Force type="text" if it gets changed
    const observer = new MutationObserver((mutations) => {
        mutations.forEach((mutation) => {
            if (mutation.type === "attributes" && mutation.attributeName === "type") {
                commandInput.setAttribute('type', 'text');
            }
        });
    });
    
    observer.observe(commandInput, {
        attributes: true
    });
    
    // Connect WebSocket
    connectWebSocket();
    
    // Initial terminal message
    appendToTerminal('GDB Terminal\nUse the terminal to debug your program.');
    
    console.log('Terminal section initialized');
    
    // Return public interface
    return {
        appendToTerminal,
        sendCommand
    };
}

// Make terminal interface available globally
const TerminalInterface = initTerminalSection();
window.AppTerminal = TerminalInterface; 