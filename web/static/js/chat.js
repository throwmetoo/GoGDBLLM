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
    let savedPanelWidth = localStorage.getItem('chatPanelWidth') || '400px';
    
    // Set initial width from saved value
    chatPanel.style.width = savedPanelWidth;
    
    // Create and add resize handle
    const resizeHandle = document.createElement('div');
    resizeHandle.className = 'chat-resize-handle';
    chatPanel.appendChild(resizeHandle);
    
    // Resize functionality
    let isResizing = false;
    let startX, startWidth;
    
    // Mouse events for resizing
    resizeHandle.addEventListener('mousedown', startResizing);
    
    // Touch events for mobile support
    resizeHandle.addEventListener('touchstart', (e) => {
        const touch = e.touches[0];
        startResizing(touch);
    });
    
    function startResizing(e) {
        isResizing = true;
        startX = e.clientX;
        startWidth = parseInt(document.defaultView.getComputedStyle(chatPanel).width, 10);
        resizeHandle.classList.add('active');
        
        // Add event listeners based on input type
        if (e.type === 'touchstart') {
            document.addEventListener('touchmove', handleTouchMove);
            document.addEventListener('touchend', handleTouchEnd);
        } else {
            document.addEventListener('mousemove', handleMouseMove);
            document.addEventListener('mouseup', handleMouseUp);
        }
        
        // Prevent default to avoid selection and scrolling
        e.preventDefault();
    }
    
    function handleMouseMove(e) {
        if (!isResizing) return;
        const width = startWidth - (e.clientX - startX);
        chatPanel.style.width = `${Math.max(280, Math.min(window.innerWidth * 0.8, width))}px`;
    }
    
    function handleTouchMove(e) {
        if (!isResizing || !e.touches[0]) return;
        const touch = e.touches[0];
        const width = startWidth - (touch.clientX - startX);
        chatPanel.style.width = `${Math.max(280, Math.min(window.innerWidth * 0.8, width))}px`;
        e.preventDefault();
    }
    
    function handleMouseUp() {
        endResizing();
    }
    
    function handleTouchEnd() {
        endResizing();
    }
    
    function endResizing() {
        isResizing = false;
        resizeHandle.classList.remove('active');
        
        // Remove all event listeners
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
        document.removeEventListener('touchmove', handleTouchMove);
        document.removeEventListener('touchend', handleTouchEnd);
        
        // Save current width
        localStorage.setItem('chatPanelWidth', chatPanel.style.width);
        savedPanelWidth = chatPanel.style.width;
    }
    
    // Open/close chat panel
    openChatBtn.addEventListener('click', () => {
        // Restore saved width before opening
        chatPanel.style.width = savedPanelWidth;
        chatPanel.classList.add('open');
        
        // Update terminal context
        terminalOutput.textContent = terminal.textContent;
        
        // Focus input
        setTimeout(() => chatInput.focus(), 300);
    });
    
    closeChatBtn.addEventListener('click', () => {
        // Save width before closing
        if (chatPanel.style.width) {
            savedPanelWidth = chatPanel.style.width;
            localStorage.setItem('chatPanelWidth', savedPanelWidth);
        }
        chatPanel.classList.remove('open');
    });
    
    // Send chat message
    async function sendMessage() {
        const userQuery = chatInput.value.trim(); // Get user query first
        if (!userQuery) return;

        // Clear input AFTER getting the value
        chatInput.value = '';

        // Add user's query to UI immediately
        addMessageToUI('user', userQuery);

        // Determine the context: selected text or full terminal output
        const selection = window.getSelection();
        const selectedText = selection.toString().trim();
        let context = '';
        let contextDescription = '';

        // Check if selection exists and is within the terminal element
        if (selectedText && terminal.contains(selection.anchorNode) && terminal.contains(selection.focusNode)) {
            context = selectedText;
            contextDescription = "Here's the selected terminal output related to my question:";
        } else {
            // Fallback to full terminal content if no valid selection
            context = terminal.textContent; // Use the actual terminal element content
            contextDescription = "Here's the current terminal output for context:";
        }

        // Prepare full message with context for the LLM
        // Ensure context isn't excessively long (optional, add if needed)
        // const MAX_CONTEXT_LENGTH = 4000;
        // if (context.length > MAX_CONTEXT_LENGTH) {
        //     context = `... (trimmed) ...\\n${context.slice(-MAX_CONTEXT_LENGTH)}`;
        //     contextDescription += " (trimmed due to length)";
        // }

        const fullMessage = `Here's my question about the debugging session:\n\n${userQuery}\n\n${contextDescription}\n\`\`\`\n${context}\n\`\`\``;

        try {
            // Show thinking indicator
            const thinkingMsg = addThinkingMessage();

            // Create payload - history includes the user message we already added to UI
             chatHistory.push({ role: 'user', content: userQuery }); // Add user query to history *before* sending

            const payload = {
                message: fullMessage, // This now contains query + context (selected or full)
                history: chatHistory.map(msg => ({ // Send history *excluding* the current user query already in fullMessage
                    role: msg.role,
                    content: msg.content
                })).slice(0, -1) // Remove the last element which is the current user query
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
                 const errorText = await response.text(); // Read error response body
                 throw new Error(`API Error: ${response.status} ${response.statusText} - ${errorText}`);
            }

            // Parse response
            const data = await response.json();

            // Remove thinking message
            if (thinkingMsg && chatMessages.contains(thinkingMsg)) {
                chatMessages.removeChild(thinkingMsg);
            }


            // Add response to UI
            addMessageToUI('assistant', data.response);

            // Add assistant response to history
            chatHistory.push({ role: 'assistant', content: data.response });

            // Limit history length (apply after adding both user and assistant messages)
             const MAX_HISTORY_PAIRS = 10; // Store 10 pairs (user + assistant)
             if (chatHistory.length > MAX_HISTORY_PAIRS * 2) {
                 chatHistory = chatHistory.slice(-(MAX_HISTORY_PAIRS * 2));
             }
        } catch (error) {
            console.error('Chat error:', error);

            // Remove thinking message if it exists
            const thinkingMsg = document.querySelector('.message.thinking');
            if (thinkingMsg) {
                chatMessages.removeChild(thinkingMsg);
            }

            // Show error message in chat
             addMessageToUI('assistant', `Sorry, I encountered an error: ${error.message}. Please check the console for details.`);
             AppUtils.showNotification('Failed to get AI response', 'error');

             // Remove the user message that failed from history
             if (chatHistory.length > 0 && chatHistory[chatHistory.length - 1].role === 'user') {
                chatHistory.pop();
             }
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