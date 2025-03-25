/**
 * app.js - Main entry point for the GoGDBLLM application
 */

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    console.log('Initializing GoGDBLLM application...');
    
    // Initialize navigation
    if (window.AppNav && typeof window.AppNav.initNavigation === 'function') {
        window.AppNav.initNavigation();
    } else {
        // Fallback navigation initialization
        initBasicNavigation();
    }
    
    // Initialize upload section
    if (window.AppUpload && typeof window.AppUpload.initUploadSection === 'function') {
        window.AppUpload.initUploadSection();
    }
    
    // Terminal section is initialized in terminal.js due to global interface requirements
    
    // Initialize settings section
    if (window.AppSettings === undefined) {
        // Settings module not loaded, initialize here
        console.warn('Settings module not loaded, using inline initialization');
        initBasicSettings();
    }
    
    // Initialize chat panel
    if (window.AppChat && typeof window.AppChat.initChatPanel === 'function') {
        window.AppChat.initChatPanel();
    }
    
    console.log('Application initialization complete');
    
    // Show welcome notification
    if (window.AppUtils && typeof window.AppUtils.showNotification === 'function') {
        window.AppUtils.showNotification('Application initialized successfully', 'success');
    }
});

// Basic navigation initialization as fallback
function initBasicNavigation() {
    const navButtons = document.querySelectorAll('.nav-btn');
    const sections = document.querySelectorAll('.section');

    navButtons.forEach(button => {
        button.addEventListener('click', () => {
            const targetSection = button.getAttribute('data-section');
            
            // Update active button
            navButtons.forEach(btn => btn.classList.remove('active'));
            button.classList.add('active');
            
            // Update active section
            sections.forEach(section => {
                section.classList.remove('active');
                if (section.id === targetSection) {
                    section.classList.add('active');
                }
            });
        });
    });
    
    console.log('Basic navigation initialized');
}

// Basic settings initialization as fallback
function initBasicSettings() {
    const providerSelect = document.getElementById('providerSelect');
    const modelSelect = document.getElementById('modelSelect');
    const apiKeyInput = document.getElementById('apiKeyInput');
    const testConnectionBtn = document.getElementById('testConnectionBtn');
    const saveSettingsBtn = document.getElementById('saveSettingsBtn');
    const connectionStatus = document.getElementById('connectionStatus');
    
    // Basic model options
    const modelOptions = {
        anthropic: ['claude-3-sonnet-20240229', 'claude-3-haiku-20240307'],
        openai: ['gpt-4-turbo', 'gpt-3.5-turbo'],
        openrouter: ['anthropic/claude-3-sonnet', 'openai/gpt-4-turbo']
    };
    
    function updateModelOptions() {
        const provider = providerSelect.value;
        
        // Clear existing options
        modelSelect.innerHTML = '';
        
        // Add new options
        (modelOptions[provider] || []).forEach(model => {
            const option = document.createElement('option');
            option.value = model;
            option.textContent = model;
            modelSelect.appendChild(option);
        });
    }
    
    // Set up provider change event
    providerSelect.addEventListener('change', updateModelOptions);
    
    // Initialize model options
    updateModelOptions();
    
    console.log('Basic settings initialized');
}

// Basic notification function as fallback
if (typeof window.AppUtils === 'undefined') {
    window.AppUtils = {
        showNotification: function(message, type) {
            const notification = document.getElementById('notification');
            if (!notification) return;
            
            notification.textContent = message;
            notification.className = 'notification';
            if (type) notification.classList.add(type);
            notification.classList.add('show');
            
            setTimeout(() => {
                notification.classList.remove('show');
            }, 3000);
        },
        formatMarkdown: function(text) {
            return text;
        }
    };
} 