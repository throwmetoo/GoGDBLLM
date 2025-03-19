/**
 * UIManager handles all UI interactions
 */
export class UIManager {
    constructor(settingsManager, debuggerManager, chatManager, wsManager) {
        this.settingsManager = settingsManager;
        this.debuggerManager = debuggerManager;
        this.chatManager = chatManager;
        this.wsManager = wsManager;
        
        // UI elements
        this.elements = {
            // Debugger elements
            debuggerOutput: null,
            debuggerInput: null,
            sendCommandBtn: null,
            startDebuggerBtn: null,
            stopDebuggerBtn: null,
            
            // Chat elements
            chatMessages: null,
            chatInput: null,
            sendChatBtn: null,
            
            // Modal elements
            settingsModal: null,
            uploadModal: null,
            
            // Forms
            settingsForm: null,
            uploadForm: null,
            
            // Settings elements
            providerSelect: null,
            modelSelect: null,
            apiKeyInput: null,
            testConnectionBtn: null,
            
            // Upload elements
            fileUpload: null,
        };
    }
    
    /**
     * Initialize the UI
     */
    initUI() {
        this.findElements();
        this.initManagers();
        this.setupEventListeners();
        this.updateModelOptions();
    }
    
    /**
     * Find all UI elements
     */
    findElements() {
        // Debugger elements
        this.elements.debuggerOutput = document.getElementById('debugger-output');
        this.elements.debuggerInput = document.getElementById('debugger-input');
        this.elements.sendCommandBtn = document.getElementById('send-command-btn');
        this.elements.startDebuggerBtn = document.getElementById('start-debugger-btn');
        this.elements.stopDebuggerBtn = document.getElementById('stop-debugger-btn');
        
        // Chat elements
        this.elements.chatMessages = document.getElementById('chat-messages');
        this.elements.chatInput = document.getElementById('chat-input');
        this.elements.sendChatBtn = document.getElementById('send-chat-btn');
        
        // Modal elements
        this.elements.settingsModal = document.getElementById('settings-modal');
        this.elements.uploadModal = document.getElementById('upload-modal');
        
        // Forms
        this.elements.settingsForm = document.getElementById('settings-form');
        this.elements.uploadForm = document.getElementById('upload-form');
        
        // Settings elements
        this.elements.providerSelect = document.getElementById('provider');
        this.elements.modelSelect = document.getElementById('model');
        this.elements.apiKeyInput = document.getElementById('api-key');
        this.elements.testConnectionBtn = document.getElementById('test-connection-btn');
        
        // Upload elements
        this.elements.fileUpload = document.getElementById('file-upload');
        
        // Modal buttons
        this.elements.settingsBtn = document.getElementById('settings-btn');
        this.elements.uploadBtn = document.getElementById('upload-btn');
        this.elements.closeModalBtns = document.querySelectorAll('.close-modal');
    }
    
    /**
     * Initialize managers with UI elements
     */
    initManagers() {
        this.debuggerManager.init(this.elements.debuggerOutput);
        this.chatManager.init(this.elements.chatMessages);
    }
    
    /**
     * Set up event listeners
     */
    setupEventListeners() {
        // Debugger events
        this.elements.sendCommandBtn.addEventListener('click', () => this.sendDebuggerCommand());
        this.elements.debuggerInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') {
                this.sendDebuggerCommand();
            } else if (e.key === 'ArrowUp') {
                this.elements.debuggerInput.value = this.debuggerManager.getPreviousCommand();
            } else if (e.key === 'ArrowDown') {
                this.elements.debuggerInput.value = this.debuggerManager.getNextCommand();
            }
        });
        this.elements.startDebuggerBtn.addEventListener('click', () => this.showUploadModal());
        this.elements.stopDebuggerBtn.addEventListener('click', () => this.stopDebugger());
        
        // Chat events
        this.elements.sendChatBtn.addEventListener('click', () => this.sendChatMessage());
        this.elements.chatInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendChatMessage();
            }
        });
        
        // Modal events
        this.elements.settingsBtn.addEventListener('click', () => this.showSettingsModal());
        this.elements.uploadBtn.addEventListener('click', () => this.showUploadModal());
        this.elements.closeModalBtns.forEach(btn => {
            btn.addEventListener('click', () => this.closeAllModals());
        });
        
        // Settings events
        this.elements.providerSelect.addEventListener('change', () => this.updateModelOptions());
        this.elements.testConnectionBtn.addEventListener('click', () => this.testConnection());
        this.elements.settingsForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.saveSettings();
        });
        
        // Upload events
        this.elements.uploadForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.uploadFile();
        });
        
        // Debugger status change
        this.debuggerManager.onStatusChange((status) => {
            this.updateDebuggerUI(status);
        });
        
        // Chat status change
        this.chatManager.onStatusChange((status) => {
            this.updateChatUI(status);
        });
        
        // Click outside modal to close
        window.addEventListener('click', (e) => {
            if (e.target === this.elements.settingsModal) {
                this.closeAllModals();
            }
            if (e.target === this.elements.uploadModal) {
                this.closeAllModals();
            }
        });
    }
    
    /**
     * Update the debugger UI based on status
     * @param {Object} status - Debugger status
     */
    updateDebuggerUI(status) {
        const isRunning = status.isRunning;
        
        this.elements.debuggerInput.disabled = !isRunning;
        this.elements.sendCommandBtn.disabled = !isRunning;
        this.elements.startDebuggerBtn.disabled = isRunning;
        this.elements.stopDebuggerBtn.disabled = !isRunning;
        
        if (isRunning) {
            this.elements.startDebuggerBtn.textContent = `GDB (${status.target.split('/').pop()})`;
        } else {
            this.elements.startDebuggerBtn.textContent = 'Start GDB';
        }
    }
    
    /**
     * Update the chat UI based on status
     * @param {Object} status - Chat status
     */
    updateChatUI(status) {
        const isWaiting = status.isWaitingForResponse;
        
        this.elements.sendChatBtn.disabled = isWaiting;
        this.elements.chatInput.disabled = isWaiting;
        
        if (isWaiting) {
            this.elements.sendChatBtn.textContent = 'Thinking...';
        } else {
            this.elements.sendChatBtn.textContent = 'Send';
        }
    }
    
    /**
     * Update connection status in UI
     * @param {boolean} connected - Whether connected to WebSocket
     */
    updateConnectionStatus(connected) {
        if (connected) {
            document.body.classList.remove('disconnected');
        } else {
            document.body.classList.add('disconnected');
            this.showNotification('Connection lost. Reconnecting...', 'warning');
        }
    }
    
    /**
     * Send a debugger command
     */
    sendDebuggerCommand() {
        const command = this.elements.debuggerInput.value.trim();
        if (!command) return;
        
        this.elements.debuggerInput.value = '';
        
        this.debuggerManager.sendCommand(command)
            .catch(error => {
                this.showNotification(error.message, 'error');
            });
    }
    
    /**
     * Stop the debugger
     */
    stopDebugger() {
        this.debuggerManager.sendCommand('quit')
            .catch(error => {
                this.showNotification(error.message, 'error');
            });
    }
    
    /**
     * Send a chat message
     */
    sendChatMessage() {
        const message = this.elements.chatInput.value.trim();
        if (!message) return;
        
        this.chatManager.sendMessage(message)
            .then(() => {
                this.elements.chatInput.value = '';
            })
            .catch(error => {
                this.showNotification(error.message, 'error');
            });
    }
    
    /**
     * Show the settings modal
     */
    showSettingsModal() {
        const settings = this.settingsManager.getSettings();
        
        this.elements.providerSelect.value = settings.provider;
        this.updateModelOptions();
        this.elements.modelSelect.value = settings.model;
        this.elements.apiKeyInput.value = settings.apiKey;
        
        this.elements.settingsModal.style.display = 'flex';
    }
    
    /**
     * Show the upload modal
     */
    showUploadModal() {
        this.elements.uploadModal.style.display = 'flex';
    }
    
    /**
     * Close all modals
     */
    closeAllModals() {
        this.elements.settingsModal.style.display = 'none';
        this.elements.uploadModal.style.display = 'none';
    }
    
    /**
     * Update model options based on selected provider
     */
    updateModelOptions() {
        const provider = this.elements.providerSelect.value;
        const modelOptions = this.settingsManager.getModelOptions(provider);
        
        // Clear existing options
        this.elements.modelSelect.innerHTML = '';
        
        // Add new options
        modelOptions.forEach(option => {
            const optionElement = document.createElement('option');
            optionElement.value = option.value;
            optionElement.textContent = option.label;
            this.elements.modelSelect.appendChild(optionElement);
        });
    }
    
    /**
     * Save settings
     */
    saveSettings() {
        const settings = {
            provider: this.elements.providerSelect.value,
            model: this.elements.modelSelect.value,
            apiKey: this.elements.apiKeyInput.value,
        };
        
        this.settingsManager.saveSettings(settings)
            .then(() => {
                this.showNotification('Settings saved successfully', 'success');
                this.closeAllModals();
            })
            .catch(error => {
                this.showNotification(`Failed to save settings: ${error.message}`, 'error');
            });
    }
    
    /**
     * Test connection with current settings
     */
    testConnection() {
        const settings = {
            provider: this.elements.providerSelect.value,
            model: this.elements.modelSelect.value,
            apiKey: this.elements.apiKeyInput.value,
        };
        
        this.elements.testConnectionBtn.disabled = true;
        this.elements.testConnectionBtn.textContent = 'Testing...';
        
        this.settingsManager.testConnection(settings)
            .then(() => {
                this.showNotification('Connection test successful', 'success');
            })
            .catch(error => {
                this.showNotification(`Connection test failed: ${error.message}`, 'error');
            })
            .finally(() => {
                this.elements.testConnectionBtn.disabled = false;
                this.elements.testConnectionBtn.textContent = 'Test Connection';
            });
    }
    
    /**
     * Upload a file
     */
    uploadFile() {
        const file = this.elements.fileUpload.files[0];
        if (!file) {
            this.showNotification('Please select a file', 'error');
            return;
        }
        
        this.apiClient.uploadFile(file)
            .then(response => {
                this.showNotification('File uploaded successfully', 'success');
                this.closeAllModals();
                
                // Start debugger with uploaded file
                return this.debuggerManager.start(response.filepath);
            })
            .then(() => {
                this.showNotification('Debugger started successfully', 'success');
            })
            .catch(error => {
                this.showNotification(`Upload failed: ${error.message}`, 'error');
            });
    }
    
    /**
     * Show a notification
     * @param {string} message - Notification message
     * @param {string} type - Notification type (success, error, warning, info)
     */
    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        
        // Add to document
        document.body.appendChild(notification);
        
        // Show notification
        setTimeout(() => {
            notification.classList.add('show');
        }, 10);
        
        // Remove after delay
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => {
                notification.remove();
            }, 300);
        }, 5000);
    }
} 