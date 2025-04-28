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
    const selectLastCmdOutputBtn = document.getElementById('selectLastCmdOutputBtn'); // Added: Get new button
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
        chatInput.placeholder = "Ask a question about your debugging session...";
    }
    
    // Added: Event listener for the new button
    selectLastCmdOutputBtn.addEventListener('click', () => {
        if (window.AppTerminal && typeof window.AppTerminal.getLastCommandOutput === 'function') {
            const lastOutput = window.AppTerminal.getLastCommandOutput();
            if (lastOutput && lastOutput.trim() !== '') {
                stagedContext = lastOutput.trim();
                // Update the existing preview area for now
                if (terminalOutput) {
                    terminalOutput.textContent = stagedContext;
                }
                // Optionally use the dedicated preview area if it exists
                // showContextPreview(stagedContext);
                chatInput.placeholder = "Ask about last command output...";
            } else {
                clearStagedContext(); // Clear if there's no output
                AppUtils.showNotification('No output captured since last command.', 'info');
            }
        } else {
            console.error('TerminalInterface or getLastCommandOutput not available.');
            AppUtils.showNotification('Error accessing terminal output.', 'error');
        }
    });
    
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
    
    // Process LLM response - attempt to parse JSON if present
    function processLLMResponse(responseText) {
        // Try to parse response as JSON
        try {
            // Ensure we're working with a string
            if (typeof responseText !== 'string') {
                console.warn('processLLMResponse received non-string input:', responseText);
                return responseText;
            }
            
            // Check if it looks like JSON (starts with { and ends with })
            const trimmed = responseText.trim();
            if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
                const jsonData = JSON.parse(trimmed);
                
                // Check if this is our expected format with a text field
                if (jsonData && typeof jsonData.text === 'string') {
                    console.log('Successfully parsed LLM response as JSON:', jsonData);
                    
                    // Extract GDB commands if present
                    if (jsonData.gdbCommands && Array.isArray(jsonData.gdbCommands) && jsonData.gdbCommands.length > 0) {
                        console.log('JSON contains GDB commands:', jsonData.gdbCommands);
                    }
                    
                    return jsonData.text;
                }
            }
        } catch (e) {
            console.warn('Failed to parse response as JSON:', e);
        }
        
        // If not JSON or parsing failed, return the original text
        return responseText;
    }

    // Send chat message
    async function sendMessage() {
        const userQuery = chatInput.value.trim();
        if (!userQuery) return;

        // Create the user message object *before* clearing input and context
        const userMessage = {
            role: 'user',
            content: userQuery,
            // Include stagedContext if it exists
            sentContext: stagedContext ? [{
                type: 'selection',
                description: 'Selected Text Snippet',
                content: stagedContext
            }] : []
        };

        chatInput.value = '';
        addMessageToUI(userMessage.role, userMessage.content, userMessage.sentContext);
        clearStagedContext(); // Clear context after sending

        addThinkingMessage();

        // Prepare history, excluding the just-added user message's context for the API call
        const historyForAPI = chatHistory.map(msg => ({
            role: msg.role,
            content: msg.content // Only send role and content
        }));

        try {
            const response = await fetch('/api/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    message: userQuery, // Send the raw user message text
                    history: historyForAPI,
                    // Include the sentContext for the current message if it exists
                    // Backend needs to be updated to handle this field.
                    sentContext: userMessage.sentContext && userMessage.sentContext.length > 0 ? userMessage.sentContext : undefined
                }),
            });

            document.getElementById('thinkingMessage')?.remove();

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(`HTTP error! status: ${response.status}, ${errorText}`);
            }

            const data = await response.json();
            console.log('Raw LLM response:', data.response);

            // Process the response to extract text from JSON if needed
            const llmResponseText = processLLMResponse(data.response);
            console.log('Processed LLM response:', llmResponseText);

            const assistantMessage = {
                role: 'assistant',
                content: data.response, // Store original response (with JSON) in history
            };
            
            // Display the processed text to the user
            addMessageToUI(assistantMessage.role, llmResponseText);

            // Add both user and assistant messages to history
            chatHistory.push(userMessage);
            chatHistory.push(assistantMessage);

        } catch (error) {
            console.error('Error sending message:', error);
            addMessageToUI('error', `Error: ${error.message}`);
            document.getElementById('thinkingMessage')?.remove();
        }
    }
    
    // Add message to UI
    function addMessageToUI(role, content, sentContext = null) {
        const messageElement = document.createElement('div');
        // Ensure both 'message' and 'chat-message' classes are added for style compatibility
        messageElement.classList.add('message', 'chat-message', role);

        const textElement = document.createElement('div');
        textElement.classList.add('message-content');
        
        // Process content if it's from the assistant AND isn't already processed
        // (check if it has a property called processedContent)
        let displayContent = content;
        
        // Only process content in addMessageToUI if it hasn't been processed already
        // by sendMessage function (which would pass the already processed text)
        if (role === 'assistant' && typeof content === 'string' && 
            content.trim().startsWith('{') && content.trim().endsWith('}')) {
            displayContent = processLLMResponse(content);
        }
        
        // Basic escaping, replace with Markdown rendering if available
        const escapedContent = displayContent.replace(/</g, "&lt;").replace(/>/g, "&gt;");
        textElement.innerHTML = escapedContent;
        messageElement.appendChild(textElement);

        // --- Context Display Logic ---
        if (role === 'user' && sentContext && sentContext.length > 0) {
            const toggle = document.createElement('span');
            toggle.classList.add('context-toggle');
            toggle.innerHTML = '&#x25BC; Context'; // Down arrow
            toggle.title = 'Show/Hide Sent Context';
            // Append toggle *after* textElement but *inside* messageElement
            messageElement.appendChild(toggle);

            const contextDetails = document.createElement('div');
            contextDetails.classList.add('context-details');
            contextDetails.style.display = 'none'; // Initially hidden

            sentContext.forEach(item => {
                const itemElement = document.createElement('div');
                itemElement.classList.add('context-item');
                itemElement.innerHTML = `
                    <strong>${item.type.replace(/</g, "&lt;").replace(/>/g, "&gt;")}:</strong> ${item.description.replace(/</g, "&lt;").replace(/>/g, "&gt;")}
                    ${item.content ? `<pre><code>${item.content.replace(/</g, "&lt;").replace(/>/g, "&gt;")}</code></pre>` : ''}
                `; // Added escaping for type
                contextDetails.appendChild(itemElement);
            });

            // Append details *after* toggle but *inside* messageElement
            messageElement.appendChild(contextDetails);

            // Toggle functionality
            toggle.addEventListener('click', (e) => {
                e.stopPropagation(); // Prevent potential clicks on the message bubble itself
                const isHidden = contextDetails.style.display === 'none';
                contextDetails.style.display = isHidden ? 'block' : 'none';
                toggle.innerHTML = isHidden ? '&#x25B2; Context' : '&#x25BC; Context'; // Up/Down arrow
            });
        }
        // --- End Context Display Logic ---

        chatMessages.appendChild(messageElement);
        chatMessages.scrollTop = chatMessages.scrollHeight;

        // Add message to internal history if it's not an error/thinking message
        if (role !== 'error' && role !== 'thinking') {
            // Note: We push to chatHistory in sendMessage *after* successful API call now
            // to ensure we store the user message with its context correctly.
        }
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