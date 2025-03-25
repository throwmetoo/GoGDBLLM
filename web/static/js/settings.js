/**
 * settings.js - Handles application settings
 */

// Initialize settings section
function initSettingsSection() {
    const providerSelect = document.getElementById('providerSelect');
    const modelSelect = document.getElementById('modelSelect');
    const apiKeyInput = document.getElementById('apiKeyInput');
    const testConnectionBtn = document.getElementById('testConnectionBtn');
    const saveSettingsBtn = document.getElementById('saveSettingsBtn');
    const connectionStatus = document.getElementById('connectionStatus');
    const currentModelElement = document.getElementById('currentModel');
    
    // Model options for each provider
    const MODEL_OPTIONS = {
        anthropic: [
            { id: "claude-3-5-sonnet-20240620", name: "Claude 3.5 Sonnet" },
            { id: "claude-3-7-sonnet-20250219", name: "Claude 3.7 Sonnet" },
            { id: "claude-3-opus-20240229", name: "Claude 3 Opus" },
            { id: "claude-3-sonnet-20240229", name: "Claude 3 Sonnet" },
            { id: "claude-3-haiku-20240307", name: "Claude 3 Haiku" },
            { id: "claude-2.1", name: "Claude 2.1" },
            { id: "claude-2.0", name: "Claude 2.0" },
            { id: "claude-instant-1.2", name: "Claude Instant 1.2" }
        ],
        openai: [
            { id: "gpt-4o", name: "GPT-4o" },
            { id: "gpt-4-turbo", name: "GPT-4 Turbo" },
            { id: "gpt-4", name: "GPT-4" },
            { id: "gpt-3.5-turbo", name: "GPT-3.5 Turbo" }
        ],
        openrouter: [
            { id: "anthropic/claude-3-5-sonnet", name: "Claude 3.5 Sonnet" },
            { id: "anthropic/claude-3-7-sonnet", name: "Claude 3.7 Sonnet" },
            { id: "anthropic/claude-3-opus", name: "Claude 3 Opus" },
            { id: "anthropic/claude-3-sonnet", name: "Claude 3 Sonnet" },
            { id: "anthropic/claude-3-haiku", name: "Claude 3 Haiku" },
            { id: "openai/gpt-4o", name: "GPT-4o" },
            { id: "openai/gpt-4-turbo", name: "GPT-4 Turbo" },
            { id: "google/gemini-1.5-pro", name: "Gemini 1.5 Pro" },
            { id: "google/gemini-1.5-flash", name: "Gemini 1.5 Flash" },
            { id: "meta-llama/llama-3-70b-instruct", name: "Llama 3 70B" }
        ]
    };
    
    // Track current settings
    let currentSettings = {
        provider: 'anthropic',
        model: MODEL_OPTIONS.anthropic[0].id,
        apiKey: ''
    };
    
    // Update model select options based on provider
    function updateModelOptions(provider) {
        // Clear existing options
        modelSelect.innerHTML = '';
        
        // Add new options
        MODEL_OPTIONS[provider].forEach(model => {
            const option = document.createElement('option');
            option.value = model.id;
            option.textContent = model.name;
            modelSelect.appendChild(option);
        });
    }
    
    // Load settings from server
    async function loadSettings() {
        try {
            const response = await fetch('/api/settings');
            if (!response.ok) {
                throw new Error(`Failed to load settings: ${response.statusText}`);
            }
            
            const settings = await response.json();
            
            // Update current settings
            currentSettings = {
                provider: settings.provider || 'anthropic',
                model: settings.model || '',
                apiKey: settings.apiKey || ''
            };
            
            // Update UI
            providerSelect.value = currentSettings.provider;
            updateModelOptions(currentSettings.provider);
            
            // Set model value if it exists in options
            if (currentSettings.model) {
                // Check if the model exists in the options
                const modelExists = Array.from(modelSelect.options).some(
                    option => option.value === currentSettings.model
                );
                
                if (modelExists) {
                    modelSelect.value = currentSettings.model;
                } else {
                    // Use the first model if the current one doesn't exist
                    modelSelect.selectedIndex = 0;
                    currentSettings.model = modelSelect.value;
                }
            }
            
            // Update model info in chat panel
            updateModelInfo();
            
            console.log('Settings loaded:', currentSettings);
        } catch (error) {
            console.error('Error loading settings:', error);
            AppUtils.showNotification('Failed to load settings', 'error');
        }
    }
    
    // Save settings to server
    async function saveSettings() {
        try {
            const settings = {
                provider: providerSelect.value,
                model: modelSelect.value,
                apiKey: apiKeyInput.value.trim()
            };
            
            // Only include the API key if it's not empty
            const dataToSend = {
                provider: settings.provider,
                model: settings.model,
                apiKey: settings.apiKey === '' ? undefined : settings.apiKey
            };
            
            const response = await fetch('/save-settings', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(dataToSend)
            });
            
            if (!response.ok) {
                throw new Error(`Failed to save settings: ${response.statusText}`);
            }
            
            const result = await response.json();
            
            if (result.success) {
                // Update current settings
                currentSettings = {
                    provider: settings.provider,
                    model: settings.model,
                    apiKey: settings.apiKey
                };
                
                // Update model info in chat panel
                updateModelInfo();
                
                // Show success message
                connectionStatus.textContent = 'Settings saved successfully';
                connectionStatus.className = 'status-message success';
                AppUtils.showNotification('Settings saved successfully', 'success');
            } else {
                throw new Error(result.error || 'Failed to save settings');
            }
        } catch (error) {
            console.error('Error saving settings:', error);
            connectionStatus.textContent = `Error: ${error.message}`;
            connectionStatus.className = 'status-message error';
            AppUtils.showNotification('Failed to save settings', 'error');
        }
    }
    
    // Test connection to LLM API
    async function testConnection() {
        try {
            connectionStatus.textContent = 'Testing connection...';
            connectionStatus.className = 'status-message';
            
            const settings = {
                provider: providerSelect.value,
                model: modelSelect.value,
                apiKey: apiKeyInput.value.trim()
            };
            
            // Validate API key
            if (!settings.apiKey) {
                connectionStatus.textContent = 'API key is required';
                connectionStatus.className = 'status-message error';
                return;
            }
            
            const response = await fetch('/test-connection', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(settings)
            });
            
            const result = await response.json();
            
            if (result.success) {
                connectionStatus.textContent = result.data.message;
                connectionStatus.className = 'status-message success';
            } else {
                connectionStatus.textContent = `Connection failed: ${result.data.message}`;
                connectionStatus.className = 'status-message error';
            }
        } catch (error) {
            console.error('Error testing connection:', error);
            connectionStatus.textContent = `Error: ${error.message}`;
            connectionStatus.className = 'status-message error';
        }
    }
    
    // Update model info in chat panel
    function updateModelInfo() {
        // Find the model display name
        const modelOptions = MODEL_OPTIONS[currentSettings.provider] || [];
        const model = modelOptions.find(m => m.id === currentSettings.model) || { name: currentSettings.model };
        
        // Update UI
        const providerName = currentSettings.provider.charAt(0).toUpperCase() + currentSettings.provider.slice(1);
        currentModelElement.textContent = `${providerName}: ${model.name}`;
    }
    
    // Set up event listeners
    providerSelect.addEventListener('change', () => {
        const provider = providerSelect.value;
        updateModelOptions(provider);
        
        // Use the first model as the default
        modelSelect.selectedIndex = 0;
    });
    
    testConnectionBtn.addEventListener('click', testConnection);
    saveSettingsBtn.addEventListener('click', saveSettings);
    
    // Initialize UI
    updateModelOptions(currentSettings.provider);
    
    // Load settings from server
    loadSettings();
    
    console.log('Settings section initialized');
    
    // Return public interface
    return {
        getCurrentSettings: () => ({ ...currentSettings })
    };
}

// Make settings interface available globally
const SettingsInterface = initSettingsSection();
window.AppSettings = SettingsInterface; 