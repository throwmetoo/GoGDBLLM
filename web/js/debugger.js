/**
 * DebuggerManager handles interaction with the debugger
 */
export class DebuggerManager {
    constructor(apiClient, wsManager) {
        this.apiClient = apiClient;
        this.wsManager = wsManager;
        this.isRunning = false;
        this.outputElement = null;
        this.commandHistory = [];
        this.historyIndex = -1;
        this.currentTarget = '';
        this.maxHistorySize = 100;
        
        // Event callbacks
        this.onStatusChangeCallbacks = [];
        this.onOutputCallbacks = [];
    }
    
    /**
     * Initialize the debugger manager
     * @param {HTMLElement} outputElement - Element to display debugger output
     */
    init(outputElement) {
        this.outputElement = outputElement;
    }
    
    /**
     * Start the debugger with the specified binary
     * @param {string} filepath - Path to the binary file
     */
    async start(filepath) {
        try {
            const response = await this.apiClient.startDebugger(filepath);
            this.isRunning = true;
            this.currentTarget = filepath;
            this.notifyStatusChange();
            return response;
        } catch (error) {
            console.error('Failed to start debugger:', error);
            throw error;
        }
    }
    
    /**
     * Send a command to the debugger
     * @param {string} command - Command to send
     */
    async sendCommand(command) {
        if (!this.isRunning) {
            throw new Error('Debugger is not running');
        }
        
        try {
            // Add command to history with size limit
            this.commandHistory.push(command);
            if (this.commandHistory.length > this.maxHistorySize) {
                this.commandHistory.shift(); // Remove oldest command
            }
            this.historyIndex = this.commandHistory.length;
            
            // Display command in output
            this.appendOutput(`(gdb) ${command}\n`, 'gdb-command');
            
            // Send command to server
            await this.apiClient.sendDebuggerCommand(command);
        } catch (error) {
            console.error('Failed to send command:', error);
            this.appendOutput(`Error: ${error.message}\n`, 'gdb-error');
            throw error;
        }
    }
    
    /**
     * Handle debugger output from WebSocket
     * @param {string} output - Debugger output
     */
    handleOutput(output) {
        this.appendOutput(output);
        
        // Notify listeners
        this.onOutputCallbacks.forEach(callback => callback(output));
    }
    
    /**
     * Handle debugger status update from WebSocket
     * @param {Object} statusData - Status data
     */
    handleStatusUpdate(statusData) {
        this.isRunning = statusData.running;
        this.currentTarget = statusData.target || '';
        
        this.notifyStatusChange();
    }
    
    /**
     * Append output to the debugger output element
     * @param {string} text - Text to append
     * @param {string} className - Optional CSS class for styling
     */
    appendOutput(text, className = '') {
        if (!this.outputElement) return;
        
        const span = document.createElement('span');
        span.textContent = text;
        
        if (className) {
            span.className = className;
        }
        
        this.outputElement.appendChild(span);
        
        // Scroll to bottom
        this.outputElement.scrollTop = this.outputElement.scrollHeight;
    }
    
    /**
     * Clear the debugger output
     */
    clearOutput() {
        if (this.outputElement) {
            this.outputElement.innerHTML = '';
        }
    }
    
    /**
     * Register a callback for status changes
     * @param {Function} callback - Callback function
     */
    onStatusChange(callback) {
        this.onStatusChangeCallbacks.push(callback);
    }
    
    /**
     * Register a callback for output
     * @param {Function} callback - Callback function
     */
    onOutput(callback) {
        this.onOutputCallbacks.push(callback);
    }
    
    /**
     * Notify all status change listeners
     */
    notifyStatusChange() {
        const status = {
            isRunning: this.isRunning,
            target: this.currentTarget
        };
        
        this.onStatusChangeCallbacks.forEach(callback => callback(status));
    }
    
    /**
     * Get previous command from history
     * @returns {string} - Previous command
     */
    getPreviousCommand() {
        if (this.historyIndex > 0) {
            this.historyIndex--;
            return this.commandHistory[this.historyIndex];
        }
        return '';
    }
    
    /**
     * Get next command from history
     * @returns {string} - Next command
     */
    getNextCommand() {
        if (this.historyIndex < this.commandHistory.length - 1) {
            this.historyIndex++;
            return this.commandHistory[this.historyIndex];
        }
        return '';
    }
    
    /**
     * Check if the debugger is running
     * @returns {boolean} - True if running
     */
    isDebuggerRunning() {
        return this.isRunning;
    }
    
    /**
     * Get the current target
     * @returns {string} - Current target
     */
    getCurrentTarget() {
        return this.currentTarget;
    }
} 