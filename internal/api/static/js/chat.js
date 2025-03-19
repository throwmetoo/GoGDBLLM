/**
 * ChatManager handles chat interactions with the LLM
 */
export class ChatManager {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.chatHistory = [];
        this.messagesElement = null;
        this.isWaitingForResponse = false;
        
        // Event callbacks
        this.onMessageCallbacks = [];
        this.onStatusChangeCallbacks = [];
    }
    
    /**
     * Initialize the chat manager
     * @param {HTMLElement} messagesElement - Element to display chat messages
     */
    init(messagesElement) {
        this.messagesElement = messagesElement;
    }
    
    /**
     * Send a message to the LLM
     * @param {string} message - Message to send
     */
    async sendMessage(message) {
        if (this.isWaitingForResponse) {
            throw new Error('Already waiting for a response');
        }
        
        try {
            // Add user message to history
            this.addMessage('user', message);
            
            // Update status
            this.isWaitingForResponse = true;
            this.notifyStatusChange();
            
            // Send message to server
            const response = await this.apiClient.sendChatMessage(message, this.chatHistory);
            
            // Add assistant response to history
            this.addMessage('assistant', response.response);
            
            return response.response;
        } catch (error) {
            console.error('Failed to send message:', error);
            throw error;
        } finally {
            this.isWaitingForResponse = false;
            this.notifyStatusChange();
        }
    }
    
    /**
     * Handle chat response from WebSocket
     * @param {string} response - Chat response
     */
    handleResponse(response) {
        if (this.isWaitingForResponse) {
            this.addMessage('assistant', response);
            this.isWaitingForResponse = false;
            this.notifyStatusChange();
        }
    }
    
    /**
     * Add a message to the chat history
     * @param {string} role - Message role ('user' or 'assistant')
     * @param {string} content - Message content
     */
    addMessage(role, content) {
        const message = { role, content };
        this.chatHistory.push(message);
        
        // Display message in UI
        this.displayMessage(message);
        
        // Notify listeners
        this.onMessageCallbacks.forEach(callback => callback(message));
    }
    
    /**
     * Display a message in the chat UI
     * @param {Object} message - Message object
     */
    displayMessage(message) {
        if (!this.messagesElement) return;
        
        const messageElement = document.createElement('div');
        messageElement.className = `chat-message ${message.role}-message`;
        
        const contentElement = document.createElement('div');
        contentElement.className = 'message-content';
        
        // Convert markdown to HTML (simple version)
        const formattedContent = this.formatMessageContent(message.content);
        contentElement.innerHTML = formattedContent;
        
        messageElement.appendChild(contentElement);
        this.messagesElement.appendChild(messageElement);
        
        // Scroll to bottom
        this.messagesElement.scrollTop = this.messagesElement.scrollHeight;
    }
    
    /**
     * Format message content (simple markdown to HTML conversion)
     * @param {string} content - Message content
     * @returns {string} - Formatted HTML
     */
    formatMessageContent(content) {
        // Replace code blocks
        content = content.replace(/```(\w*)([\s\S]*?)```/g, (match, language, code) => {
            return `<pre><code class="language-${language}">${this.escapeHtml(code.trim())}</code></pre>`;
        });
        
        // Replace inline code
        content = content.replace(/`([^`]+)`/g, '<code>$1</code>');
        
        // Replace bold text
        content = content.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
        
        // Replace italic text
        content = content.replace(/\*([^*]+)\*/g, '<em>$1</em>');
        
        // Replace newlines with <br>
        content = content.replace(/\n/g, '<br>');
        
        return content;
    }
    
    /**
     * Escape HTML special characters
     * @param {string} html - HTML string
     * @returns {string} - Escaped HTML
     */
    escapeHtml(html) {
        const div = document.createElement('div');
        div.textContent = html;
        return div.innerHTML;
    }
    
    /**
     * Clear the chat history
     */
    clearHistory() {
        this.chatHistory = [];
        
        if (this.messagesElement) {
            this.messagesElement.innerHTML = '';
        }
    }
    
    /**
     * Register a callback for new messages
     * @param {Function} callback - Callback function
     */
    onMessage(callback) {
        this.onMessageCallbacks.push(callback);
    }
    
    /**
     * Register a callback for status changes
     * @param {Function} callback - Callback function
     */
    onStatusChange(callback) {
        this.onStatusChangeCallbacks.push(callback);
    }
    
    /**
     * Notify all status change listeners
     */
    notifyStatusChange() {
        const status = {
            isWaitingForResponse: this.isWaitingForResponse
        };
        
        this.onStatusChangeCallbacks.forEach(callback => callback(status));
    }
    
    /**
     * Check if waiting for a response
     * @returns {boolean} - True if waiting
     */
    isWaiting() {
        return this.isWaitingForResponse;
    }
    
    /**
     * Get the chat history
     * @returns {Array} - Chat history
     */
    getHistory() {
        return [...this.chatHistory];
    }
} 