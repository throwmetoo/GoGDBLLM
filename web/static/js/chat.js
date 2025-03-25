/**
 * chat.js - Handles chat panel and communication with LLM APIs
 */

// Initialize chat panel
function initChatPanel() {
    const chatPanel = document.getElementById('chatPanel');
    const openChatBtn = document.getElementById('openChatBtn');
    const closeChatBtn = document.getElementById('closeChatBtn');
    const chatMessages = document.getElementById('chatMessages');
    const chatInput = document.getElementById('chatInput');
    const sendChatBtn = document.getElementById('sendChatBtn');
    const terminalOutput = document.getElementById('terminalOutput');
    const terminal = document.getElementById('terminal');
    
    // Chat state
    let chatHistory = [];
    
    // Open/close chat panel
    openChatBtn.addEventListener('click', () => {
        chatPanel.classList.add('open');
        
        // Update terminal context
        terminalOutput.textContent = terminal.textContent;
        
        // Focus input
        setTimeout(() => chatInput.focus(), 300);
    });
    
    closeChatBtn.addEventListener('click', () => {
        chatPanel.classList.remove('open');
    });
    
    // Send chat message
    async function sendMessage() {
        const message = chatInput.value.trim();
        if (!message) return;
        
        // Clear input
        chatInput.value = '';
        
        // Add message to UI
        addMessageToUI('user', message);
        
        // Get terminal context
        const context = terminal.textContent;
        
        // Prepare full message with context
        const fullMessage = `Here's my question about the debugging session:\n\n${message}\n\nHere's the current terminal output:\n\`\`\`\n${context}\n\`\`\``;
        
        try {
            // Show thinking indicator
            const thinkingMsg = addThinkingMessage();
            
            // Create payload
            const payload = {
                message: fullMessage,
                history: chatHistory.map(msg => ({
                    role: msg.role,
                    content: msg.content
                }))
            };
            
            // Send to server
            const response = await fetch('/api/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });
            
            // Handle errors
            if (!response.ok) {
                throw new Error(`API Error: ${response.status} ${response.statusText}`);
            }
            
            // Parse response
            const data = await response.json();
            
            // Remove thinking message
            chatMessages.removeChild(thinkingMsg);
            
            // Add response to UI
            addMessageToUI('assistant', data.response);
            
            // Add to history (use original message, not the full context one)
            chatHistory.push({ role: 'user', content: message });
            chatHistory.push({ role: 'assistant', content: data.response });
            
            // Limit history length
            if (chatHistory.length > 10) {
                chatHistory = chatHistory.slice(-10);
            }
        } catch (error) {
            console.error('Chat error:', error);
            
            // Remove thinking message if it exists
            const thinkingMsg = document.querySelector('.message.thinking');
            if (thinkingMsg) {
                chatMessages.removeChild(thinkingMsg);
            }
            
            // Show error message
            addMessageToUI('assistant', `Error: ${error.message}`);
            AppUtils.showNotification('Failed to send message', 'error');
        }
    }
    
    // Add message to UI
    function addMessageToUI(role, content) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${role}`;
        
        // Format content if from assistant
        if (role === 'assistant') {
            const formattedContent = AppUtils.formatMarkdown(content);
            messageDiv.innerHTML = formattedContent;
            messageDiv.classList.add('markdown');
        } else {
            messageDiv.textContent = content;
        }
        
        // Add to messages
        chatMessages.appendChild(messageDiv);
        
        // Scroll to bottom
        chatMessages.scrollTop = chatMessages.scrollHeight;
        
        return messageDiv;
    }
    
    // Add thinking message
    function addThinkingMessage() {
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message assistant thinking';
        messageDiv.textContent = 'Thinking...';
        
        // Add to messages
        chatMessages.appendChild(messageDiv);
        
        // Scroll to bottom
        chatMessages.scrollTop = chatMessages.scrollHeight;
        
        return messageDiv;
    }
    
    // Handle send button click
    sendChatBtn.addEventListener('click', sendMessage);
    
    // Handle enter key press
    chatInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });
    
    // Initial welcome message
    setTimeout(() => {
        addMessageToUI(
            'assistant',
            "Hello! I'm your AI debugging assistant. I can help you understand your code, debug issues, and explain GDB commands. How can I assist you today?"
        );
    }, 500);
    
    console.log('Chat panel initialized');
}

// Make chat interface available globally
window.AppChat = {
    initChatPanel
}; 