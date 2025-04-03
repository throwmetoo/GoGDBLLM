/**
 * chat.js - Handles chat panel and communication with LLM APIs
 */

// Helper function to remove existing custom context menus
function removeCustomContextMenu() {
    const existingMenu = document.getElementById('terminalContextMenu');
    if (existingMenu) {
        existingMenu.remove();
    }
}

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
    // Placeholder for the context preview area - add this div in your HTML near the chat input
    const contextPreviewArea = document.getElementById('contextPreviewArea'); // Example ID: <div id="contextPreviewArea"></div>
    
    // Chat state
    let chatHistory = [];
    let savedPanelWidth = localStorage.getItem('chatPanelWidth') || '400px';
    let stagedContext = null; // Variable to hold context from right-click selection
    
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
    
    // Function to update and show the context preview
    function showContextPreview(text) {
        if (!contextPreviewArea) return; // Need the HTML element
        const previewText = text.length > 100 ? text.substring(0, 97) + '...' : text;
        contextPreviewArea.innerHTML = `
            <span class="preview-label">Context:</span>
            <code class="preview-text">${previewText.replace(/</g, '&lt;').replace(/>/g, '&gt;')}</code>
            <button id="clearContextBtn" class="clear-context-btn" title="Clear selected context">✖</button>
        `;
        contextPreviewArea.style.display = 'block';

        // Add event listener to the new clear button
        document.getElementById('clearContextBtn').addEventListener('click', () => {
            clearStagedContext();
        });
    }

    // Function to clear the staged context and hide the preview
    function clearStagedContext() {
        stagedContext = null;
        if (contextPreviewArea) {
            contextPreviewArea.style.display = 'none';
            contextPreviewArea.innerHTML = '';
        }
        // Also clear the main context display area (now named "Selected Context")
        if (terminalOutput) {
             terminalOutput.textContent = ''; // Clear the renamed "Selected Context" area
        }
        // Optionally, update chat input placeholder if needed
        chatInput.placeholder = "Type your message...";
    }
    
    // Add context menu listener to the terminal
    terminal.addEventListener('contextmenu', (event) => {
        const selection = window.getSelection();
        const selectedText = selection.toString().trim();

        removeCustomContextMenu(); // Remove any previous menu

        if (selectedText) {
            // Check if selection is within the terminal
            let isInTerminal = false;
            try {
                if (selection.rangeCount > 0) { // Check rangeCount
                    const range = selection.getRangeAt(0);
                    if (range.commonAncestorContainer && terminal.contains(range.commonAncestorContainer)) {
                        isInTerminal = true;
                    }
                }
            } catch (e) { console.warn("Error checking selection range:", e); } // Log error

            if (isInTerminal) {
                event.preventDefault(); // Prevent default browser menu only if we show ours

                const menu = document.createElement('div');
                menu.id = 'terminalContextMenu';
                menu.className = 'custom-context-menu'; // Add CSS for styling
                menu.style.position = 'absolute';

                // --- Calculate position above selection --- 
                const rangeRect = selection.getRangeAt(0).getBoundingClientRect();
                const menuHeightEstimate = 30; // Estimate menu height to position above
                let menuTop = window.scrollY + rangeRect.top - menuHeightEstimate;
                let menuLeft = window.scrollX + rangeRect.left;
                
                // Adjust if menu goes off-screen top
                if (menuTop < window.scrollY) {
                    menuTop = window.scrollY + rangeRect.bottom + 5; // Position below instead
                }
                
                // Basic adjustment if menu goes off-screen left/right (can be improved)
                menuLeft = Math.max(5, menuLeft); // Keep some padding from left edge
                // Consider menu width if adjusting right edge
                
                menu.style.left = `${menuLeft}px`;
                menu.style.top = `${menuTop}px`;
                // --- End position calculation ---

                const menuItem = document.createElement('div');
                menuItem.className = 'context-menu-item';
                menuItem.textContent = 'Use Selection in Chat';
                menuItem.onclick = () => {
                    stagedContext = selectedText;
                    showContextPreview(stagedContext);
                    if (terminalOutput) {
                        terminalOutput.textContent = selectedText;
                        console.log("Immediately updated terminalOutput with selected text.");
                    } else {
                        console.warn("terminalOutput element not found for immediate update.");
                    }
                    removeCustomContextMenu();
                    chatInput.placeholder = "Ask about selected context...";
                };

                menu.appendChild(menuItem);
                document.body.appendChild(menu); // Append to body to avoid clipping

                // Add listener to close menu if clicked outside
                setTimeout(() => { // Timeout to prevent immediate closing
                    document.addEventListener('click', handleClickOutsideMenu, { capture: true, once: true });
                }, 0);
            }
        }
    });

    // Function to close context menu when clicking outside
    function handleClickOutsideMenu(event) {
        const menu = document.getElementById('terminalContextMenu');
        if (menu && !menu.contains(event.target)) {
            removeCustomContextMenu();
        }
    }
    
    // Send chat message
    async function sendMessage() {
        const userQuery = chatInput.value.trim();
        if (!userQuery) return;

        chatInput.value = '';
        addMessageToUI('user', userQuery);

        let context = '';
        let contextDescription = '';

        // Use staged context if available
        if (stagedContext) {
            context = stagedContext;
            contextDescription = "Here's the selected terminal output related to my question:";
        } else {
            // Fallback to full terminal content
            context = terminal.textContent || "";
            contextDescription = "Here's the current terminal output for context:";
        }

        // Prepare full message
        const MAX_CONTEXT_LENGTH = 4000;
        if (context.length > MAX_CONTEXT_LENGTH) {
            context = `... (trimmed) ...\n${context.slice(-MAX_CONTEXT_LENGTH)}`;
            contextDescription += " (trimmed due to length)";
        }
        const fullMessage = `Here's my question about the debugging session:\n\n${userQuery}\n\n${contextDescription}\n\`\`\`\n${context.trim()}\n\`\`\``;

        // Add user query to local history *before* sending
        chatHistory.push({ role: 'user', content: userQuery });

        const thinkingMsg = addThinkingMessage();

        try {
            const historyForPayload = chatHistory.slice(0, -1).map(msg => ({ role: msg.role, content: msg.content }));
            const payload = { message: fullMessage, history: historyForPayload };

            const response = await fetch('/api/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(payload)
            });

            if (thinkingMsg && chatMessages.contains(thinkingMsg)) {
                chatMessages.removeChild(thinkingMsg);
            }

            if (!response.ok) {
                let errorDetails = '';
                try { errorDetails = await response.text(); } catch (e) { }
                throw new Error(`API Error: ${response.status} ${response.statusText}. ${errorDetails}`);
            }

            const data = await response.json();
            if (!data || !data.response) {
                throw new Error("Received empty or invalid response from server.");
            }

            addMessageToUI('assistant', data.response);
            chatHistory.push({ role: 'assistant', content: data.response });

            // --- Debugging Log --- 
            console.log("Attempting to update Selected Context display.");
            console.log("Context variable:", context ? `"${context.trim()}"` : '(empty or null)');
            console.log("terminalOutput element:", terminalOutput);
            // --- End Debugging Log --- 
            
            // Update the "Selected Context" display area with what was actually sent
            if (terminalOutput) {
                console.log("Setting terminalOutput.textContent"); // Log entry into block
                terminalOutput.textContent = context.trim(); // Use the context variable
            } else {
                 console.warn("Could not find terminalOutput element to update!");
            }
            
            // Clear staged context ONLY on successful send
            clearStagedContext();

            const MAX_HISTORY_PAIRS = 10;
            if (chatHistory.length > MAX_HISTORY_PAIRS * 2) {
                chatHistory = chatHistory.slice(-(MAX_HISTORY_PAIRS * 2));
            }

        } catch (error) {
            console.error('Chat error:', error);
            if (thinkingMsg && chatMessages.contains(thinkingMsg)) {
                chatMessages.removeChild(thinkingMsg);
            }
            addMessageToUI('assistant', `Sorry, I encountered an error: ${error.message || 'Unable to get response.'}`);

            // Don't clear context on error, user might want to retry
            // clearStagedContext();

            // Remove the user message that led to the error from local history
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