/**
 * ApiClient handles all HTTP API requests to the backend
 */
export class ApiClient {
    constructor() {
        this.baseUrl = window.location.origin;
    }
    
    /**
     * Send a request to the API
     * @param {string} endpoint - API endpoint
     * @param {string} method - HTTP method
     * @param {Object} data - Request data
     * @returns {Promise<any>} - Response data
     */
    async request(endpoint, method = 'GET', data = null) {
        const url = `${this.baseUrl}/api/v1/${endpoint}`;
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
            },
        };
        
        if (data) {
            options.body = JSON.stringify(data);
        }
        
        try {
            const response = await fetch(url, options);
            
            // Parse JSON response
            const responseData = await response.json();
            
            // Check if the request was successful
            if (!response.ok) {
                throw new Error(responseData.error || 'Unknown error occurred');
            }
            
            return responseData;
        } catch (error) {
            console.error(`API request failed (${method} ${endpoint}):`, error);
            throw error;
        }
    }
    
    /**
     * Upload a file to the server
     * @param {File} file - File to upload
     * @returns {Promise<any>} - Response data
     */
    async uploadFile(file) {
        const url = `${this.baseUrl}/api/v1/upload`;
        const formData = new FormData();
        formData.append('file', file);
        
        try {
            const response = await fetch(url, {
                method: 'POST',
                body: formData,
            });
            
            const responseData = await response.json();
            
            if (!response.ok) {
                throw new Error(responseData.error || 'File upload failed');
            }
            
            return responseData;
        } catch (error) {
            console.error('File upload failed:', error);
            throw error;
        }
    }
    
    // Settings API methods
    async getSettings() {
        return this.request('settings', 'GET');
    }
    
    async saveSettings(settings) {
        return this.request('settings', 'POST', settings);
    }
    
    async testConnection(settings) {
        return this.request('test-connection', 'POST', settings);
    }
    
    // Debugger API methods
    async startDebugger(filepath) {
        return this.request('debugger/start', 'POST', { filepath });
    }
    
    async sendDebuggerCommand(command) {
        return this.request('debugger/command', 'POST', { command });
    }
    
    // Chat API methods
    async sendChatMessage(message, history) {
        return this.request('chat', 'POST', { message, history });
    }
} 