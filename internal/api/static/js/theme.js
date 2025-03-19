/**
 * ThemeManager handles application theming
 */
export class ThemeManager {
    constructor() {
        this.currentTheme = 'light';
        this.themes = {
            light: {
                '--primary-color': '#4a86e8',
                '--secondary-color': '#6c757d',
                '--background-color': '#f8f9fa',
                '--panel-bg-color': '#ffffff',
                '--border-color': '#dee2e6',
                '--text-color': '#212529',
                '--debugger-bg': '#1e1e1e',
                '--debugger-text': '#d4d4d4',
                '--debugger-prompt': '#569cd6',
                '--debugger-command': '#ce9178',
                '--debugger-output': '#b5cea8',
                '--debugger-error': '#f44747',
            },
            dark: {
                '--primary-color': '#4a86e8',
                '--secondary-color': '#adb5bd',
                '--background-color': '#212529',
                '--panel-bg-color': '#343a40',
                '--border-color': '#495057',
                '--text-color': '#f8f9fa',
                '--debugger-bg': '#1e1e1e',
                '--debugger-text': '#d4d4d4',
                '--debugger-prompt': '#569cd6',
                '--debugger-command': '#ce9178',
                '--debugger-output': '#b5cea8',
                '--debugger-error': '#f44747',
            }
        };
        
        this.init();
    }
    
    /**
     * Initialize the theme manager
     */
    init() {
        // Check for saved theme preference
        const savedTheme = localStorage.getItem('theme');
        if (savedTheme && this.themes[savedTheme]) {
            this.currentTheme = savedTheme;
        } else {
            // Check for system preference
            if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
                this.currentTheme = 'dark';
            }
        }
        
        // Apply the theme
        this.applyTheme(this.currentTheme);
        
        // Listen for system theme changes
        if (window.matchMedia) {
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', e => {
                if (!localStorage.getItem('theme')) {
                    this.applyTheme(e.matches ? 'dark' : 'light');
                }
            });
        }
    }
    
    /**
     * Apply a theme
     * @param {string} themeName - Theme name
     */
    applyTheme(themeName) {
        if (!this.themes[themeName]) {
            console.error(`Theme "${themeName}" not found`);
            return;
        }
        
        this.currentTheme = themeName;
        
        // Save preference
        localStorage.setItem('theme', themeName);
        
        // Apply CSS variables
        const theme = this.themes[themeName];
        for (const [property, value] of Object.entries(theme)) {
            document.documentElement.style.setProperty(property, value);
        }
        
        // Update body class
        document.body.classList.remove('theme-light', 'theme-dark');
        document.body.classList.add(`theme-${themeName}`);
    }
    
    /**
     * Toggle between light and dark themes
     */
    toggleTheme() {
        const newTheme = this.currentTheme === 'light' ? 'dark' : 'light';
        this.applyTheme(newTheme);
    }
    
    /**
     * Get the current theme
     * @returns {string} - Current theme name
     */
    getTheme() {
        return this.currentTheme;
    }
} 