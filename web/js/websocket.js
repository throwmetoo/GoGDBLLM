/**
 * WebSocketManager handles WebSocket connections
 */
export class WebSocketManager {
    constructor() {
        this.socket = null;
        this.isConnected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000; // Start with 1 second
        this.pingInterval = null;
        this.pingTimeout = null;
        
        // Event callbacks
        this.messageCallbacks = [];
        this.openCallbacks = [];
        this.closeCallbacks = [];
        this.errorCallbacks = [];
    }
    
    /**
     * Connect to the WebSocket server
     */
    connect() {
        if (this.socket && (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)) {
            return;
        }
        
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.socket = new WebSocket(wsUrl);
        
        this.socket.onopen = () => {
            console.log('WebSocket connection established');
            this.isConnected = true;
            this.reconnectAttempts = 0;
            this.reconnectDelay = 1000;
            
            // Start ping/pong for connection health check
            this.startPingPong();
            
            // Notify listeners
            this.openCallbacks.forEach(callback => callback());
        };
        
        this.socket.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                
                // Notify listeners
                this.messageCallbacks.forEach(callback => callback(message));
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };
        
        this.socket.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            this.isConnected = false;
            
            // Clear ping/pong interval
            this.clearPingPong();
            
            // Notify listeners
            this.closeCallbacks.forEach(callback => callback(event));
            
            // Attempt to reconnect
            this.attemptReconnect();
        };
        
        this.socket.onerror = (error) => {
            console.error('WebSocket error:', error);
            
            // Notify listeners
            this.errorCallbacks.forEach(callback => callback(error));
        };
    }
    
    /**
     * Attempt to reconnect to the WebSocket server
     */
    attemptReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.log('Maximum reconnect attempts reached');
            return;
        }
        
        this.reconnectAttempts++;
        
        // Exponential backoff
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
        console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
        
        setTimeout(() => {
            this.connect();
        }, delay);
    }
    
    /**
     * Send a message to the WebSocket server
     * @param {Object} message - Message to send
     */
    send(message) {
        if (!this.isConnected) {
            throw new Error('WebSocket is not connected');
        }
        
        try {
            this.socket.send(JSON.stringify(message));
        } catch (error) {
            console.error('Error sending WebSocket message:', error);
            // Notify any error listeners
            this.errorCallbacks.forEach(callback => callback(error));
            throw error;
        }
    }
    
    /**
     * Close the WebSocket connection
     */
    close() {
        if (this.socket) {
            this.socket.close();
        }
    }
    
    /**
     * Register a callback for WebSocket messages
     * @param {Function} callback - Callback function
     */
    onMessage(callback) {
        this.messageCallbacks.push(callback);
    }
    
    /**
     * Register a callback for WebSocket open events
     * @param {Function} callback - Callback function
     */
    onOpen(callback) {
        this.openCallbacks.push(callback);
    }
    
    /**
     * Register a callback for WebSocket close events
     * @param {Function} callback - Callback function
     */
    onClose(callback) {
        this.closeCallbacks.push(callback);
    }
    
    /**
     * Register a callback for WebSocket error events
     * @param {Function} callback - Callback function
     */
    onError(callback) {
        this.errorCallbacks.push(callback);
    }
    
    /**
     * Check if the WebSocket is connected
     * @returns {boolean} - True if connected
     */
    isWebSocketConnected() {
        return this.isConnected;
    }
    
    /**
     * Start ping/pong mechanism for connection health check
     */
    startPingPong() {
        // Clear any existing intervals
        this.clearPingPong();
        
        // Send ping every 30 seconds
        this.pingInterval = setInterval(() => {
            if (this.isConnected) {
                // Set a timeout to detect if pong is not received
                this.pingTimeout = setTimeout(() => {
                    console.warn('No pong received, closing connection');
                    this.socket.close();
                }, 5000); // Wait 5 seconds for pong
                
                // Send ping
                try {
                    this.send({ type: 'ping' });
                } catch (error) {
                    console.error('Error sending ping:', error);
                    this.clearPingPong();
                    this.socket.close();
                }
            }
        }, 30000); // 30 seconds
    }
    
    /**
     * Clear ping/pong timers
     */
    clearPingPong() {
        if (this.pingInterval) {
            clearInterval(this.pingInterval);
            this.pingInterval = null;
        }
        if (this.pingTimeout) {
            clearTimeout(this.pingTimeout);
            this.pingTimeout = null;
        }
    }
    
    /**
     * Handle pong message from server
     */
    handlePong() {
        // Clear the timeout since we received a pong
        if (this.pingTimeout) {
            clearTimeout(this.pingTimeout);
            this.pingTimeout = null;
        }
    }
} 