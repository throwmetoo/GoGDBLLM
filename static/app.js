// DOM elements
const terminalDiv = document.getElementById("terminal");
const commandInput = document.getElementById("commandInput");
const fileInput = document.getElementById("fileInput");
const uploadBtn = document.getElementById("uploadBtn");
const executeBtn = document.getElementById("executeBtn");
const dropZone = document.getElementById("dropZone");
const openChatBtn = document.getElementById("openChatBtn");
const closeChatBtn = document.getElementById("closeChatBtn");
const chatPanel = document.getElementById("chatPanel");
const terminalOutput = document.getElementById("terminalOutput");

const CTRL_C = '\x03';  // Control-C character
const CTRL_D = '\x04';  // Control-D character
const CTRL_Z = '\x1A';  // Control-Z character

// Application state
let socket = null;
let uploadedFileName = "";
let terminalConnected = false;

// Add these at the top with your other constants
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

// Add to your application state
let currentSettings = {
    provider: 'anthropic',
    model: '',
    apiKey: ''
};

// Define appendToTerminal early and attach to window
window.appendToTerminal = function (text) {
  terminalDiv.textContent += text + "\n";
  terminalDiv.scrollTop = terminalDiv.scrollHeight;

  // Update chat panel if it's open
  if (chatPanel.classList.contains("open")) {
    terminalOutput.textContent = terminalDiv.textContent;
  }
};

// Initialize the application
document.addEventListener("DOMContentLoaded", () => {
  // Set initial terminal message
  window.appendToTerminal(
    "File Runner Terminal\nUpload an executable file to run it.",
  );

  // Set up event listeners
  setupEventListeners();
  setupChatPanel();

  // Add this to your existing initialization code
  const commandInput = document.getElementById('commandInput');
  
  // Prevent password manager detection
  commandInput.setAttribute('type', 'text');
  commandInput.setAttribute('autocomplete', 'off');
  
  // Force type="text" if it gets changed
  const observer = new MutationObserver(function(mutations) {
      mutations.forEach(function(mutation) {
          if (mutation.type === "attributes" && mutation.attributeName === "type") {
              commandInput.setAttribute('type', 'text');
          }
      });
  });
  
  observer.observe(commandInput, {
      attributes: true
  });
  
  // Prevent password manager popup on focus
  commandInput.addEventListener('focus', function(e) {
      setTimeout(() => {
          if (commandInput.type !== 'text') {
              commandInput.setAttribute('type', 'text');
          }
      }, 100);
  });

  // Make sure execute button is properly set up
  setupExecuteButton();

  // Set up auto-execute checkbox to toggle execute button visibility
  setupAutoExecuteCheckbox();
});

// Set up all event listeners
function setupEventListeners() {
  // Navigation
  document.querySelectorAll("#navbar a").forEach((link) => {
    link.addEventListener("click", (e) => {
      e.preventDefault();
      const sectionId =
        e.target.getAttribute("href").substring(1) ||
        e.target.textContent.toLowerCase() + "Section";
      showSection(sectionId);
    });
  });

  // Command input
  commandInput.addEventListener("keydown", handleCommand);

  // Upload button
  if (uploadBtn) {
    uploadBtn.addEventListener('click', function() {
      fileInput.click();
    });
  }

  // File input change
  if (fileInput) {
    fileInput.addEventListener('change', handleFileUpload);
  }

  // Execute button
  if (executeBtn) {
    executeBtn.addEventListener('click', executeUploadedFile);
  }

  // File drop handling
  dropZone.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropZone.style.backgroundColor = "#f0f0f0";
  });

  dropZone.addEventListener("dragleave", () => {
    dropZone.style.backgroundColor = "";
  });

  dropZone.addEventListener("drop", (e) => {
    e.preventDefault();
    dropZone.style.backgroundColor = "";

    if (e.dataTransfer.files.length) {
      fileInput.files = e.dataTransfer.files;
      handleFileUpload();
    }
  });

  // Prevent the default browser behavior for the entire terminal section
  document
    .getElementById("terminalSection")
    .addEventListener("keydown", (e) => {
      // Allow only when focused on the command input
      if (e.target !== commandInput) {
        commandInput.focus();
      }
    });

  // Add settings related listeners
  document.getElementById('saveSettings').addEventListener('click', saveSettings);
  
  // Initialize settings
  loadSettings().then(() => {
    // Check upload button state after settings are loaded
    updateUploadButtonState();
  });

  document.getElementById('testConnection').addEventListener('click', testConnection);

  // Add notification close button listener
  document.querySelector('.notification-close').addEventListener('click', hideNotification);
  
  // Add provider change listener to update button state
  document.getElementById('provider').addEventListener('change', updateModelOptions);
  
  // Add save settings success handler
  document.getElementById('saveSettings').addEventListener('click', () => {
    // Update upload button state after settings are saved
    setTimeout(updateUploadButtonState, 500);
  });
}

// Set up chat panel functionality
function setupChatPanel() {
  // Open chat panel
  openChatBtn.addEventListener("click", () => {
    chatPanel.classList.add("open");
    terminalOutput.textContent = terminalDiv.textContent;
  });

  // Close chat panel
  closeChatBtn.addEventListener("click", () => {
    chatPanel.classList.remove("open");
  });

  // Add resize functionality
  setupChatResize();
}

// Show the specified section and hide others
function showSection(sectionId) {
  // Default to upload section if sectionId is not specified
  sectionId = sectionId || "uploadSection";

  // Hide all sections
  document.getElementById("uploadSection").classList.remove("active");
  document.getElementById("terminalSection").classList.remove("active");
  document.getElementById("settingsSection").classList.remove("active");

  // Show selected section
  document.getElementById(sectionId).classList.add("active");

  // Focus on command input when terminal is shown
  if (sectionId === "terminalSection") {
    commandInput.focus();

    // Ensure WebSocket is connected
    if (!terminalConnected) {
      connectWebSocket();
    }
  }
}

// Function to load settings
async function loadSettings() {
    try {
        console.log("Loading settings...");
        const response = await fetch('/api/settings');
        if (!response.ok) {
            console.error("Failed to load settings:", response.status, response.statusText);
            throw new Error(`Failed to load settings: ${response.status} ${response.statusText}`);
        }
        
        const settings = await response.json();
        console.log("Loaded settings:", settings);
        
        currentSettings = {
            provider: settings.provider || 'anthropic',
            model: settings.model || MODEL_OPTIONS.anthropic[0].id,
            apiKey: settings.apiKey || ''
        };
        
        // Update UI
        document.getElementById('provider').value = currentSettings.provider;
        document.getElementById('apiKey').value = currentSettings.apiKey;
        updateModelOptions();
        
        // Make sure to set the model after updating options
        setTimeout(() => {
            const modelSelect = document.getElementById('model');
            if (modelSelect.querySelector(`option[value="${currentSettings.model}"]`)) {
                modelSelect.value = currentSettings.model;
            } else {
                // If the model doesn't exist in the options, select the first one
                currentSettings.model = modelSelect.options[0].value;
            }
            
            // Update the model info in the chat panel
            updateCurrentModelDisplay();
        }, 100);
        
        return settings.apiKey && settings.apiKey.trim() !== '';
    } catch (error) {
        console.error('Error loading settings:', error);
        // Fall back to defaults
        currentSettings = {
            provider: 'anthropic',
            model: MODEL_OPTIONS.anthropic[0].id,
            apiKey: ''
        };
        
        document.getElementById('provider').value = currentSettings.provider;
        updateModelOptions();
        return false;
    }
}

// Function to update the current model display in the chat panel
function updateCurrentModelDisplay() {
    const currentModelElement = document.getElementById('currentModel');
    if (currentModelElement) {
        const modelOptions = MODEL_OPTIONS[currentSettings.provider] || [];
        const modelInfo = modelOptions.find(m => m.id === currentSettings.model);
        currentModelElement.textContent = modelInfo ? modelInfo.name : currentSettings.model;
    }
}

// Function to validate settings
async function validateSettings() {
    try {
        console.log("Validating settings...");
        const response = await fetch('/api/settings');
        if (!response.ok) {
            console.error("Failed to validate settings:", response.status, response.statusText);
            return false;
        }
        
        const settings = await response.json();
        console.log("Validated settings:", settings);
        
        // Update current settings
        currentSettings = {
            provider: settings.provider || 'anthropic',
            model: settings.model || MODEL_OPTIONS.anthropic[0].id,
            apiKey: settings.apiKey || ''
        };
        
        // Update the model info in the chat panel
        updateCurrentModelDisplay();
        
        return settings.apiKey && settings.apiKey.trim() !== '';
    } catch (error) {
        console.error('Error validating settings:', error);
        return false;
    }
}

// Function to handle file upload
function handleFileUpload() {
    // Get the file from the input
    const file = fileInput.files[0];
    if (!file) {
        showNotification('Please select a file first', 'error');
        return;
    }

    const formData = new FormData();
    formData.append("file", file);

    // Update UI to show upload in progress
    const uploadStatus = document.getElementById("uploadStatus");
    const executeBtn = document.getElementById("executeBtn");
    const filePath = document.getElementById("filePath");
    uploadStatus.textContent = "Uploading...";
    uploadStatus.style.color = "#ffcc00";

    fetch("/upload", {
        method: "POST",
        body: formData,
    })
        .then(response => {
            // First check if the response is ok
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            // Then try to parse the JSON
            return response.json().catch(e => {
                throw new Error('Failed to parse server response as JSON');
            });
        })
        .then(data => {
            if (data.success) {
                uploadStatus.textContent = `File uploaded: ${data.filename}`;
                uploadStatus.style.color = "#00cc00";
                
                // Store the uploaded file path and display it
                uploadedFileName = data.filename;
                currentFilePath = data.filepath;
                filePath.textContent = data.filepath;
                
                // Auto-execute if checkbox is checked
                const autoExecuteCheckbox = document.getElementById("autoExecute");
                if (autoExecuteCheckbox && autoExecuteCheckbox.checked) {
                    executeUploadedFile();
                    executeBtn.style.display = "none";
                } else {
                    executeBtn.style.display = "block";
                }
            } else {
                uploadStatus.textContent = `Upload failed: ${data.error}`;
                uploadStatus.style.color = "#ff0000";
                filePath.textContent = "";
                executeBtn.style.display = "none";
            }
        })
        .catch(error => {
            uploadStatus.textContent = `Upload error: ${error.message}`;
            uploadStatus.style.color = "#ff0000";
            filePath.textContent = "";
            executeBtn.style.display = "none";
            console.error('Upload error:', error);
        });
}

// Function to execute the uploaded file
function executeUploadedFile() {
  if (!currentFilePath) {
    showNotification('Please upload a file first', 'error');
    return;
  }
  
  // Show terminal section
  showSection('terminalSection');
  
  // First ensure GDB is started
  fetch("/start-gdb", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ filepath: currentFilePath })
  })
  .then(response => response.json())
  .then(data => {
    if (data.success) {
      appendToTerminal(`GDB started successfully. Loading ${uploadedFileName}...`);
      // Use the correct path from the server response
      sendCommand(`file ${currentFilePath}`);
    } else {
      appendToTerminal(`Error starting GDB: ${data.error}`);
    }
  })
  .catch(error => {
    appendToTerminal(`Error starting GDB: ${error.message}`);
    console.error('GDB start error:', error);
  });
}

// Make sure the upload button is not disabled by default
document.addEventListener('DOMContentLoaded', function() {
  const uploadBtn = document.getElementById('uploadBtn');
  if (uploadBtn) {
    uploadBtn.disabled = false;
    uploadBtn.classList.remove("disabled-btn");
  }
});

// Handle command input
function handleCommand(event) {
    const commandInput = document.getElementById('commandInput');
    const command = commandInput.value.trim();
    
    if (event.key === 'Enter' && !event.shiftKey && command) {
        event.preventDefault();
        sendCommand(command);
        
        // Clear input with a slight delay to ensure it happens after processing
        setTimeout(() => {
            commandInput.value = '';
            // Force input field refresh
            commandInput.focus();
        }, 10);
    }
}

function sendCommand(command) {
    if (!window.socket || window.socket.readyState !== WebSocket.OPEN) {
        appendToTerminal("Error: Not connected to server. Trying to reconnect...");
        connectWebSocket();
        // Queue the command to be sent after connection
        setTimeout(() => sendCommand(command), 1000);
        return;
    }
    
    appendToTerminal(`> ${command}`);
    window.socket.send(command);
}

// Connect to WebSocket server
function connectWebSocket() {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${wsProtocol}//${window.location.host}/ws`;
  
  appendToTerminal("Connecting to server...");
  
  const socket = new WebSocket(wsUrl);
  
  socket.onopen = function() {
    terminalConnected = true;
    appendToTerminal("Connected to server.");
  };
  
  socket.onmessage = function(event) {
    appendToTerminal(event.data);
  };
  
  socket.onclose = function() {
    terminalConnected = false;
    appendToTerminal("Disconnected from server.");
  };
  
  socket.onerror = function() {
    appendToTerminal("WebSocket error occurred.");
  };
  
  window.socket = socket;
}

// Add these new functions
function updateModelOptions() {
    const provider = document.getElementById('provider').value;
    const modelSelect = document.getElementById('model');
    modelSelect.innerHTML = '';
    
    MODEL_OPTIONS[provider].forEach(model => {
        const option = document.createElement('option');
        option.value = model.id;
        option.textContent = model.name;
        modelSelect.appendChild(option);
    });
}

// Update the saveSettings function to properly collect the current API key
async function saveSettings() {
  try {
    // Get the current values from the form
    const provider = document.getElementById('provider').value;
    const model = document.getElementById('model').value;
    const apiKey = document.getElementById('apiKey').value;
    
    // Update the currentSettings object
    currentSettings = {
      provider: provider,
      model: model,
      apiKey: apiKey
    };
    
    console.log("Saving settings:", currentSettings);
    
    const response = await fetch('/save-settings', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(currentSettings)
    });
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error("Error saving settings:", response.status, errorText);
      throw new Error(errorText || `HTTP error ${response.status}`);
    }
    
    showNotification('Settings saved successfully', 'success');
    return true;
  } catch (error) {
    console.error("Exception saving settings:", error);
    showNotification(`Failed to save settings: ${error.message}`, 'error');
    return false;
  }
}

// Also update the testConnection function to use the form values directly
async function testConnection() {
    const provider = document.getElementById('provider').value;
    const model = document.getElementById('model').value;
    const apiKey = document.getElementById('apiKey').value;
    
    if (!apiKey) {
        showNotification('Error: API key is required', 'error');
        return;
    }
    
    const testBtn = document.getElementById('testConnection');
    const saveBtn = document.getElementById('saveSettings');
    testBtn.disabled = true;
    saveBtn.disabled = true;
    
    try {
        window.appendToTerminal('Testing connection...');
        
        const response = await fetch('/test-connection', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                provider,
                model,
                apiKey
            })
        });
        
        testBtn.disabled = false;
        saveBtn.disabled = false;
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText);
        }
        
        // Update currentSettings after successful test
        currentSettings = {
            provider,
            model,
            apiKey
        };
        
        window.appendToTerminal('Connection successful! Model is available.');
        showNotification('Connection test successful! The API key and model are valid.', 'success');
        
    } catch (error) {
        testBtn.disabled = false;
        saveBtn.disabled = false;
        window.appendToTerminal(`Connection failed: ${error.message}`);
        showNotification(`Connection test failed: ${error.message}`, 'error');
    }
}

// Add these functions for notification handling
function showNotification(message, type = 'success') {
    const notification = document.getElementById('notification');
    const notificationMessage = document.getElementById('notificationMessage');
    
    // Set message and type
    notificationMessage.textContent = message;
    notification.className = `notification ${type}`;
    
    // Show notification
    notification.style.display = 'block';
    
    // Hide after 5 seconds
    const timeout = setTimeout(() => {
        hideNotification();
    }, 5000);
    
    // Store timeout in the DOM element
    notification.dataset.timeout = timeout;
}

function hideNotification() {
    const notification = document.getElementById('notification');
    
    // Clear any existing timeout
    if (notification.dataset.timeout) {
        clearTimeout(Number(notification.dataset.timeout));
    }
    
    // Add slide out animation
    notification.style.animation = 'slideOut 0.3s ease-out';
    
    // Hide after animation
    setTimeout(() => {
        notification.style.display = 'none';
        notification.style.animation = '';
    }, 300);
}

// Add this function to check and update upload button state
async function updateUploadButtonState() {
  try {
    const response = await fetch('/settings');
    if (!response.ok) {
      uploadBtn.disabled = true;
      return;
    }
    
    const settings = await response.json();
    if (!settings.apiKey) {
      uploadBtn.disabled = true;
      uploadBtn.title = "Configure AI settings first";
      
      // Add a visual indicator
      uploadBtn.classList.add("disabled-btn");
      
      // Add a message near the upload button
      const uploadMessage = document.createElement('div');
      uploadMessage.id = 'uploadMessage';
      uploadMessage.className = 'upload-message';
      uploadMessage.textContent = 'Please configure AI settings before uploading';
      
      // Insert the message after the upload button
      if (!document.getElementById('uploadMessage')) {
        uploadBtn.parentNode.insertBefore(uploadMessage, uploadBtn.nextSibling);
      }
    } else {
      uploadBtn.disabled = false;
      uploadBtn.title = "Upload file";
      uploadBtn.classList.remove("disabled-btn");
      
      // Remove the message if it exists
      const uploadMessage = document.getElementById('uploadMessage');
      if (uploadMessage) {
        uploadMessage.remove();
      }
    }
  } catch (error) {
    console.error('Error checking settings:', error);
    uploadBtn.disabled = true;
  }
}

// Chat functionality
let chatHistory = [];
let isChatOpen = false;

// Function to send a chat message
async function sendChatMessage() {
    const messageInput = document.getElementById('chatInput');
    const message = messageInput.value.trim();
    
    if (!message) return;
    
    // Validate settings first
    const valid = await validateSettings();
    if (!valid) {
        showNotification('Please configure your API key in settings first', 'error');
        toggleChat(); // Close chat panel
        showSection('settingsSection'); // Show settings section
        return;
    }
    
    // Clear input
    messageInput.value = '';
    
    // Add user message to chat
    addMessageToChat('user', message);
    
    // Show loading indicator
    const loadingIndicator = document.createElement('div');
    loadingIndicator.className = 'loading-indicator';
    loadingIndicator.innerHTML = 'AI is thinking<span></span><span></span><span></span>';
    document.getElementById('chatMessages').appendChild(loadingIndicator);
    
    try {
        console.log("Sending chat message:", message);
        console.log("Using settings:", currentSettings);
        
        // Get terminal output for context
        const terminalOutput = document.getElementById('terminal').innerText;
        
        // Prepare request
        const chatRequest = {
            message: message,
            history: chatHistory,
            terminalOutput: terminalOutput
        };
        
        // Send request to server
        const response = await fetch('/api/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(chatRequest)
        });
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `HTTP error ${response.status}`);
        }
        
        // Parse response
        const data = await response.json();
        
        // Remove loading indicator
        document.querySelector('.loading-indicator').remove();
        
        // Add assistant message to chat
        addMessageToChat('assistant', data.response);
        
    } catch (error) {
        console.error("Error sending chat message:", error);
        
        // Remove loading indicator
        const loadingIndicator = document.querySelector('.loading-indicator');
        if (loadingIndicator) {
            loadingIndicator.remove();
        }
        
        // Add error message to chat
        addMessageToChat('assistant', `Error: ${error.message}. Please check your API settings and try again.`);
    }
}

// Function to add a message to the chat
function addMessageToChat(role, content) {
  // Create message element
  const messageElement = document.createElement('div');
  messageElement.className = `${role}-message`;
  
  // Format content with markdown
  const formattedContent = formatMarkdown(content);
  messageElement.innerHTML = formattedContent;
  
  // Add to chat container
  document.getElementById('chatMessages').appendChild(messageElement);
  
  // Scroll to bottom
  scrollChatToBottom();
  
  // Add to history
  chatHistory.push({ role, content });
  
  // Limit history length to prevent excessive token usage
  if (chatHistory.length > 20) {
    chatHistory = chatHistory.slice(chatHistory.length - 20);
  }
}

// Function to scroll chat to bottom
function scrollChatToBottom() {
  const chatMessages = document.getElementById('chatMessages');
  chatMessages.scrollTop = chatMessages.scrollHeight;
}

// Function to format markdown
function formatMarkdown(text) {
  // Code blocks
  text = text.replace(/```(\w*)([\s\S]*?)```/g, function(match, language, code) {
    return `<pre><code class="language-${language}">${escapeHtml(code.trim())}</code></pre>`;
  });
  
  // Inline code
  text = text.replace(/`([^`]+)`/g, '<code>$1</code>');
  
  // Bold
  text = text.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  
  // Italic
  text = text.replace(/\*([^*]+)\*/g, '<em>$1</em>');
  
  // Headers
  text = text.replace(/^### (.*$)/gm, '<h3>$1</h3>');
  text = text.replace(/^## (.*$)/gm, '<h2>$1</h2>');
  text = text.replace(/^# (.*$)/gm, '<h1>$1</h1>');
  
  // Lists
  text = text.replace(/^\s*\- (.*$)/gm, '<li>$1</li>');
  text = text.replace(/<\/li>\n<li>/g, '</li><li>');
  text = text.replace(/(<li>.*<\/li>)/g, '<ul>$1</ul>');
  
  // Line breaks
  text = text.replace(/\n/g, '<br>');
  
  return text;
}

// Helper function to escape HTML
function escapeHtml(unsafe) {
  return unsafe
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

// Function to toggle chat panel
function toggleChat() {
  const chatPanel = document.getElementById('chatPanel');
  const openChatBtn = document.getElementById('openChatBtn');
  
  if (isChatOpen) {
    chatPanel.style.right = '-400px';
    openChatBtn.innerHTML = '💬';
  } else {
    chatPanel.style.right = '0';
    openChatBtn.innerHTML = '✕';
    
    // Focus on input
    document.getElementById('chatInput').focus();
    
    // Add terminal output to chat if it's the first message
    if (chatHistory.length === 0) {
      const terminalOutput = document.getElementById('terminal').innerText;
      if (terminalOutput.trim()) {
        addMessageToChat('system', 'Terminal Output:\n```\n' + terminalOutput + '\n```');
      }
    }
    
    // Scroll to bottom when opening
    setTimeout(scrollChatToBottom, 100);
  }
  
  isChatOpen = !isChatOpen;
}

// Add this to your setupEventListeners function
function setupChatEventListeners() {
  // Open chat button
  const openChatBtn = document.getElementById('openChatBtn');
  if (openChatBtn) {
    openChatBtn.addEventListener('click', toggleChat);
  }
  
  // Send button
  const sendBtn = document.getElementById('sendChatBtn');
  if (sendBtn) {
    sendBtn.addEventListener('click', sendChatMessage);
  }
  
  // Input enter key
  const chatInput = document.getElementById('chatInput');
  if (chatInput) {
    chatInput.addEventListener('keypress', function(e) {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendChatMessage();
      }
    });
  }
}

// Make sure to call this in your DOMContentLoaded event
document.addEventListener('DOMContentLoaded', function() {
  setupDragAndDrop();
  setupEventListeners();
  setupChatEventListeners();
  
  // Load settings
  loadSettings().then(valid => {
    if (!valid) {
      console.log("No valid API key found, showing notification");
      showNotification('Please configure your AI provider settings first', 'warning');
    } else {
      console.log("Valid API key found");
    }
  });
});

// Set up drag and drop functionality
function setupDragAndDrop() {
    const dropZone = document.getElementById('dropZone');
    const fileInput = document.getElementById('fileInput');
    
    if (!dropZone || !fileInput) return;
    
    // Prevent default drag behaviors
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });
    
    // Highlight drop zone when item is dragged over it
    ['dragenter', 'dragover'].forEach(eventName => {
        dropZone.addEventListener(eventName, highlight, false);
    });
    
    ['dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, unhighlight, false);
    });
    
    // Handle dropped files
    dropZone.addEventListener('drop', handleDrop, false);
    
    // Handle file input change
    fileInput.addEventListener('change', handleFileInputChange, false);
    
    function preventDefaults(e) {
        e.preventDefault();
        e.stopPropagation();
    }
    
    function highlight() {
        dropZone.classList.add('highlight');
    }
    
    function unhighlight() {
        dropZone.classList.remove('highlight');
    }
    
    function handleDrop(e) {
        const dt = e.dataTransfer;
        const files = dt.files;
        
        if (files.length) {
            fileInput.files = files;
            handleFileUpload();
        }
    }
    
    function handleFileInputChange() {
        // The upload button will be clicked manually
    }
}

// Add resize functionality to chat panel
function setupChatResize() {
  const chatPanel = document.getElementById('chatPanel');
  let isResizing = false;
  let startX;
  let startWidth;

  // Handle mouse down on the resize handle
  chatPanel.addEventListener('mousedown', (e) => {
    // Only trigger on the left 4px of the panel
    if (e.clientX > chatPanel.getBoundingClientRect().left + 4) return;

    isResizing = true;
    startX = e.clientX;
    startWidth = parseInt(getComputedStyle(chatPanel).width, 10);
    
    chatPanel.classList.add('resizing');
  });

  // Handle mouse move for resizing
  document.addEventListener('mousemove', (e) => {
    if (!isResizing) return;

    const width = startWidth - (e.clientX - startX);
    
    // Limit minimum and maximum width
    if (width >= 300 && width <= window.innerWidth - 100) {
      chatPanel.style.width = `${width}px`;
    }
  });

  // Handle mouse up to stop resizing
  document.addEventListener('mouseup', () => {
    if (!isResizing) return;
    
    isResizing = false;
    chatPanel.classList.remove('resizing');
  });
}

// Make sure execute button is properly set up
function setupExecuteButton() {
  const executeBtn = document.getElementById("executeBtn");
  if (executeBtn) {
    executeBtn.addEventListener("click", executeUploadedFile);
    
    // Set initial visibility based on auto-execute checkbox
    const autoExecuteCheckbox = document.getElementById("autoExecute");
    if (autoExecuteCheckbox) {
      executeBtn.style.display = autoExecuteCheckbox.checked ? "none" : "block";
    } else {
      executeBtn.style.display = "block";
    }
  }
}

// Set up auto-execute checkbox to toggle execute button visibility
function setupAutoExecuteCheckbox() {
  const autoExecuteCheckbox = document.getElementById("autoExecute");
  const executeBtn = document.getElementById("executeBtn");
  
  if (autoExecuteCheckbox && executeBtn) {
    autoExecuteCheckbox.addEventListener("change", function() {
      executeBtn.style.display = this.checked ? "none" : "block";
    });
  }
}
