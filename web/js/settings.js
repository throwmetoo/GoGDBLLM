/**
 * SettingsManager handles application settings
 */
export class SettingsManager {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.settings = {
            provider: 'anthropic',
            model: 'claude-3-sonnet-20240229',
            apiKey: '',
        };
        
        // Model options for each provider
        this.modelOptions = {
            anthropic: [
                { value: 'claude-3-opus-20240229', label: 'Claude 3 Opus' },
                { value: 'claude-3-sonnet-20240229', label: 'Claude 3 Sonnet' },
                { value: 'claude-3-haiku-20240307', label: 'Claude 3 Haiku' },
                { value: 'claude-2.1', label: 'Claude 2.1' },
                { value: 'claude-2.0', label: 'Claude 2.0' },
            ],
            openai: [
                { value: 'gpt-4o', label: 'GPT-4o' },
                { value: 'gpt-4-turbo', label: 'GPT-4 Turbo' },
                { value: 'gpt-4', label: 'GPT-4' },
                { value: 'gpt-3.5-turbo', label: 'GPT-3.5 Turbo' },
            ],
            openrouter: [
                { value: 'anthropic/claude-3-opus', label: 'Claude 3 Opus' },
                { value: 'anthropic/claude-3-sonnet', label: 'Claude 3 Sonnet' },
                { value: 'anthropic/claude-3-haiku', label: 'Claude 3 Haiku' },
                { value: 'openai/gpt-4o', label: 'GPT-4o' },
                { value: 'openai/gpt-4-turbo', label: 'GPT-4 Turbo' },
                { value: 'google/gemini-1.5-pro', label: 'Gemini 1.5 Pro' },
                { value: 'meta-llama/llama-3-70b-instruct', label: 'Llama 3 70B' },
            ],
        };
    }
    
    /**
     * Load settings from the server
     */
    async loadSettings() {
        try {
            const response = await this.apiClient.getSettings();
            this.settings = response.settings;
            return this.settings;
        } catch (error) {
            console.error('Failed to load settings:', error);
            throw error;
        }
    }
    
    /**
     * Save settings to the server
     * @param {Object} newSettings - New settings to save
     */
    async saveSettings(newSettings) {
        try {
            await this.apiClient.saveSettings(newSettings);
            this.settings = newSettings;
            return this.settings;
        } catch (error) {
            console.error('Failed to save settings:', error);
            throw error;
        }
    }
    
    /**
     * Test connection with the provided settings
     * @param {Object} settings - Settings to test
     */
    async testConnection(settings) {
        try {
            return await this.apiClient.testConnection(settings);
        } catch (error) {
            console.error('Connection test failed:', error);
            throw error;
        }
    }
    
    /**
     * Get current settings
     * @returns {Object} - Current settings
     */
    getSettings() {
        return { ...this.settings };
    }
    
    /**
     * Get model options for a provider
     * @param {string} provider - Provider name
     * @returns {Array} - Model options
     */
    getModelOptions(provider) {
        return this.modelOptions[provider] || [];
    }
} 