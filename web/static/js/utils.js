/**
 * utils.js - Shared utility functions
 */

// Show a notification message that fades out
function showNotification(message, type = 'info') {
    const notification = document.getElementById('notification');
    
    // Set message and type
    notification.textContent = message;
    notification.className = 'notification';
    notification.classList.add(type);
    notification.classList.add('show');
    
    // Automatically hide after 3 seconds
    setTimeout(() => {
        notification.classList.remove('show');
    }, 3000);
}

// Format code blocks in markdown content
function formatMarkdown(content) {
    // Simple markdown formatter - could be replaced with a library like marked.js
    // for more comprehensive formatting
    
    // Handle code blocks
    content = content.replace(/```(\w+)?\n([\s\S]*?)\n```/g, '<pre><code>$2</code></pre>');
    
    // Handle inline code
    content = content.replace(/`([^`]+)`/g, '<code>$1</code>');
    
    // Handle headers
    content = content.replace(/^### (.*$)/gm, '<h3>$1</h3>');
    content = content.replace(/^## (.*$)/gm, '<h2>$1</h2>');
    content = content.replace(/^# (.*$)/gm, '<h1>$1</h1>');
    
    // Handle paragraphs
    content = content.replace(/^\s*(\n)?(.+)/gm, function(m) {
        return /^<(\/)?(h1|h2|h3|pre|code)/.test(m) ? m : '<p>' + m + '</p>';
    });
    
    // Handle line breaks
    content = content.replace(/\n/g, '<br>');
    
    return content;
}

// Handle API requests with fetch
async function apiRequest(url, method = 'GET', data = null) {
    const options = {
        method,
        headers: {
            'Content-Type': 'application/json'
        }
    };
    
    if (data) {
        options.body = JSON.stringify(data);
    }
    
    try {
        const response = await fetch(url, options);
        
        if (!response.ok) {
            throw new Error(`API error: ${response.status} ${response.statusText}`);
        }
        
        return await response.json();
    } catch (error) {
        console.error('API request failed:', error);
        throw error;
    }
}

// Export utilities to global scope
window.AppUtils = {
    showNotification,
    formatMarkdown,
    apiRequest
}; 