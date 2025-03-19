/**
 * Utility functions for the application
 */

/**
 * Format a timestamp as a readable string
 * @param {Date} date - Date object
 * @returns {string} - Formatted timestamp
 */
export function formatTimestamp(date) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

/**
 * Escape HTML special characters
 * @param {string} html - HTML string
 * @returns {string} - Escaped HTML
 */
export function escapeHtml(html) {
    const div = document.createElement('div');
    div.textContent = html;
    return div.innerHTML;
}

/**
 * Truncate a string to the specified length
 * @param {string} str - String to truncate
 * @param {number} maxLength - Maximum length
 * @returns {string} - Truncated string
 */
export function truncateString(str, maxLength) {
    if (str.length <= maxLength) {
        return str;
    }
    return str.substring(0, maxLength - 3) + '...';
}

/**
 * Debounce a function
 * @param {Function} func - Function to debounce
 * @param {number} wait - Wait time in milliseconds
 * @returns {Function} - Debounced function
 */
export function debounce(func, wait) {
    let timeout;
    return function(...args) {
        const context = this;
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(context, args), wait);
    };
}

/**
 * Format code with syntax highlighting (simple version)
 * @param {string} code - Code to format
 * @param {string} language - Programming language
 * @returns {string} - Formatted HTML
 */
export function formatCode(code, language) {
    // This is a simple implementation
    // In a real application, you might want to use a library like highlight.js
    
    // Escape HTML
    code = escapeHtml(code);
    
    // Add basic syntax highlighting based on language
    if (language === 'go' || language === 'golang') {
        // Highlight Go keywords
        const keywords = ['func', 'return', 'var', 'const', 'package', 'import', 'type', 'struct', 'interface', 'map', 'chan', 'go', 'defer', 'if', 'else', 'for', 'range', 'switch', 'case', 'default', 'break', 'continue'];
        keywords.forEach(keyword => {
            const regex = new RegExp(`\\b${keyword}\\b`, 'g');
            code = code.replace(regex, `<span class="keyword">${keyword}</span>`);
        });
        
        // Highlight strings
        code = code.replace(/(["'`])(?:(?=(\\?))\2.)*?\1/g, '<span class="string">$&</span>');
        
        // Highlight comments
        code = code.replace(/\/\/.*$/gm, '<span class="comment">$&</span>');
    }
    
    return `<pre><code class="language-${language}">${code}</code></pre>`;
}

/**
 * Parse GDB output to extract useful information
 * @param {string} output - GDB output
 * @returns {Object} - Parsed information
 */
export function parseGdbOutput(output) {
    const result = {
        breakpoints: [],
        currentFile: null,
        currentLine: null,
        variables: [],
        error: null,
    };
    
    // Extract current file and line
    const fileLineMatch = output.match(/at ([^:]+):(\d+)/);
    if (fileLineMatch) {
        result.currentFile = fileLineMatch[1];
        result.currentLine = parseInt(fileLineMatch[2], 10);
    }
    
    // Extract breakpoints
    const breakpointMatches = output.matchAll(/Breakpoint (\d+) at (0x[0-9a-f]+): file ([^,]+), line (\d+)/g);
    for (const match of breakpointMatches) {
        result.breakpoints.push({
            id: parseInt(match[1], 10),
            address: match[2],
            file: match[3],
            line: parseInt(match[4], 10),
        });
    }
    
    // Extract variables (from print or display commands)
    const varMatches = output.matchAll(/\$\d+ = (.+)/g);
    for (const match of varMatches) {
        result.variables.push(match[1]);
    }
    
    // Check for errors
    if (output.includes('No such file or directory') || 
        output.includes('No symbol table') ||
        output.includes('Cannot access memory')) {
        result.error = output.split('\n').find(line => 
            line.includes('No such file') || 
            line.includes('No symbol table') ||
            line.includes('Cannot access memory')
        );
    }
    
    return result;
} 