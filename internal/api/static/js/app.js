import { SettingsManager } from './settings.js';
import { DebuggerManager } from './debugger.js';
import { ChatManager } from './chat.js';
import { UIManager } from './ui.js';
import { WebSocketManager } from './websocket.js';
import { ApiClient } from './api.js';

class App {
    constructor() {
        this.apiClient = new ApiClient();
        this.wsManager = new WebSocketManager();
        this.settingsManager = new SettingsManager(this.apiClient);
        this.debuggerManager = new DebuggerManager(this.apiClient, this.wsManager);
        this.chatManager = new ChatManager(this.apiClient);
        this.uiManager = new UIManager(
            this.settingsManager,
            this.debuggerManager,
            this.chatManager,
            this.wsManager
        );
        
        this.init();
    }
    
    async init() {
        try {
            // Initialize WebSocket connection
            this.wsManager.connect();
            
            // Load settings
            await this.settingsManager.loadSettings();
            
            // Initialize UI
            this.uiManager.initUI();
            
            // Setup event listeners
            this.setupEventListeners();
            
            console.log('Application initialized successfully');
        } catch (error) {
            console.error('Failed to initialize application:', error);
            this.uiManager.showNotification('Failed to initialize application', 'error');
        }
    }
    
    setupEventListeners() {
        // WebSocket message handling
        this.wsManager.onMessage((message) => {
            switch (message.type) {
                case 'debugger_output':
                    this.debuggerManager.handleOutput(message.content);
                    break;
                case 'debugger_status':
                    this.debuggerManager.handleStatusUpdate(message.data);
                    break;
                case 'chat_response':
                    this.chatManager.handleResponse(message.content);
                    break;
                case 'error':
                    this.uiManager.showNotification(message.content, 'error');
                    break;
                case 'info':
                    this.uiManager.showNotification(message.content, 'info');
                    break;
                default:
                    console.warn('Unknown message type:', message.type);
            }
        });
        
        // WebSocket connection events
        this.wsManager.onOpen(() => {
            console.log('WebSocket connection established');
            this.uiManager.updateConnectionStatus(true);
        });
        
        this.wsManager.onClose(() => {
            console.log('WebSocket connection closed');
            this.uiManager.updateConnectionStatus(false);
            
            // Try to reconnect after a delay
            setTimeout(() => {
                this.wsManager.connect();
            }, 3000);
        });
        
        this.wsManager.onError((error) => {
            console.error('WebSocket error:', error);
            this.uiManager.showNotification('WebSocket connection error', 'error');
        });
    }
}

// Initialize the application when the DOM is fully loaded
document.addEventListener('DOMContentLoaded', () => {
    new App();
}); 