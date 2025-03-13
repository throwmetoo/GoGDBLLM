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
  commandInput.addEventListener("keydown", handleCommandInput);

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

// Add this function to check if settings are valid
async function validateSettings() {
  try {
    // First check if we have settings stored
    if (!currentSettings.apiKey) {
      showNotification('Please configure your AI provider settings first', 'error');
      showSection('settingsSection');
      return false;
    }
    
    // Test the connection to verify settings are valid
    const testBtn = document.getElementById('testConnection');
    const saveBtn = document.getElementById('saveSettings');
    testBtn.disabled = true;
    saveBtn.disabled = true;
    
    window.appendToTerminal('Validating AI provider connection...');
    
    const response = await fetch('/test-connection', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        provider: currentSettings.provider,
        model: currentSettings.model,
        apiKey: currentSettings.apiKey
      })
    });
    
    testBtn.disabled = false;
    saveBtn.disabled = false;
    
    if (!response.ok) {
      const errorText = await response.text();
      showNotification(`Invalid AI settings: ${errorText}`, 'error');
      showSection('settingsSection');
      return false;
    }
    
    return true;
  } catch (error) {
    showNotification(`Error validating settings: ${error.message}`, 'error');
    showSection('settingsSection');
    return false;
  }
}

// Function to handle file upload
async function handleFileUpload() {
  // Check if a file is selected
  if (!fileInput.files.length) {
    showNotification('Please select a file first', 'error');
    return;
  }

  const file = fileInput.files[0];
  const formData = new FormData();
  formData.append('file', file);

  // Show loading state
  uploadBtn.disabled = true;
  uploadBtn.textContent = 'Uploading...';
  
  try {
    console.log("Uploading file:", file.name);
    const response = await fetch('/upload', {
      method: 'POST',
      body: formData
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(errorText || `HTTP error ${response.status}`);
    }

    // Get the response text
    const responseText = await response.text();
    
    // Update UI
    uploadedFileName = file.name;
    uploadBtn.textContent = `Uploaded: ${file.name}`;
    executeBtn.disabled = false;
    
    // Show success message
    showNotification('File uploaded successfully', 'success');
    appendToTerminal(responseText);
    
    // Auto-execute if checkbox is checked
    if (document.getElementById('autoExecute').checked) {
      executeUploadedFile();
    }
  } catch (error) {
    console.error("Error uploading file:", error);
    showNotification(`Upload failed: ${error.message}`, 'error');
    appendToTerminal(`Error: ${error.message}`);
  } finally {
    // Reset button state
    uploadBtn.disabled = false;
    uploadBtn.textContent = 'Upload';
  }
}

// Function to execute the uploaded file
function executeUploadedFile() {
  if (!uploadedFileName) {
    showNotification('Please upload a file first', 'error');
    return;
  }
  
  // Show terminal section
  showSection('terminalSection');
  
  // Execute the file in GDB
  appendToTerminal(`Executing ${uploadedFileName} in GDB...`);
  sendCommand(`/tmp/${uploadedFileName}`);
}

// Make sure the upload button is not disabled by default
document.addEventListener('DOMContentLoaded', function() {
  const uploadBtn = document.getElementById('uploadBtn');
  if (uploadBtn) {
    uploadBtn.disabled = false;
    uploadBtn.classList.remove("disabled-btn");
  }
});

// Handle command input from the terminal
function handleCommandInput(event) {
  if (event.ctrlKey) {
      switch (event.key.toLowerCase()) {
        case 'c':
          event.preventDefault();
          sendSpecialCommand('CTRL_C');
          window.appendToTerminal('^C');
          return;
        case 'd':
          event.preventDefault();
          sendSpecialCommand('CTRL_D');
          window.appendToTerminal('^D');
          return;
        case 'z':
          event.preventDefault();
          sendSpecialCommand('CTRL_Z');
          window.appendToTerminal('^Z');
          return;
      }
    }
  
  // Handle arrow keys
  if (event.key === 'ArrowUp') {
    event.preventDefault();
    sendSpecialCommand('ARROW_UP');
    return;
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    sendSpecialCommand('ARROW_DOWN');
    return;
  }
  
  if (event.key === "Enter") {
    const command = commandInput.value.trim();
    if (command) {
      if (!terminalConnected) {
        window.appendToTerminal("Connecting to server...");
        connectWebSocket(() => {
          sendCommand(command);
        });
      } else {
        sendCommand(command);
      }
      commandInput.value = "";
    }
  }
}

// Function to send special commands
function sendSpecialCommand(commandType) {
  if (socket && socket.readyState === WebSocket.OPEN) {
    // Send a JSON object to differentiate special commands
    socket.send(JSON.stringify({
      type: 'special',
      command: commandType
    }));
  } else {
    window.appendToTerminal("Error: Not connected to server. Trying to reconnect...");
    connectWebSocket();
  }
}

// Update the regular sendCommand function to identify regular commands
function sendCommand(command) {
  if (socket && socket.readyState === WebSocket.OPEN) {
    // Send a JSON object for regular commands too
    socket.send(JSON.stringify({
      type: 'regular',
      command: command
    }));
    window.appendToTerminal(`> ${command}`);
  } else {
    window.appendToTerminal("Error: Not connected to server. Trying to reconnect...");
    connectWebSocket(() => {
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({
          type: 'regular',
          command: command
        }));
        window.appendToTerminal(`> ${command}`);
      }
    });
  }
}

// Connect to WebSocket server
function connectWebSocket(callback) {
  // Close existing socket if any
  if (socket) {
    socket.close();
  }

  socket = new WebSocket("ws://localhost:8080/ws");

  socket.onopen = function () {
    terminalConnected = true;
    window.appendToTerminal("Connected to server.");

    if (callback && typeof callback === "function") {
      callback();
    }
  };

  socket.onmessage = function (event) {
    window.appendToTerminal(event.data);
  };

  socket.onerror = function () {
    window.appendToTerminal("WebSocket error occurred.");
    terminalConnected = false;
  };

  socket.onclose = function () {
    window.appendToTerminal("Disconnected from server.");
    terminalConnected = false;
  };
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

async function loadSettings() {
    try {
        const response = await fetch('/settings');
        if (!response.ok) {
            throw new Error('Failed to load settings');
        }
        
        const settings = await response.json();
        currentSettings = {
            provider: settings.provider || 'anthropic',
            model: settings.model || MODEL_OPTIONS.anthropic[0].id,
            apiKey: settings.apiKey || ''
        };
        
        document.getElementById('provider').value = currentSettings.provider;
        document.getElementById('apiKey').value = currentSettings.apiKey;
        updateModelOptions();
        document.getElementById('model').value = currentSettings.model;
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
    }
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
let isWaitingForResponse = false;

document.addEventListener('DOMContentLoaded', function() {
    // Chat panel toggle
    const openChatBtn = document.getElementById('openChatBtn');
    const closeChatBtn = document.getElementById('closeChatBtn');
    const chatPanel = document.getElementById('chatPanel');
    const chatInput = document.getElementById('chatInput');
    const sendChatBtn = document.getElementById('sendChatBtn');
    const chatContent = document.getElementById('chatContent');
    const chatModelInfo = document.getElementById('chatModelInfo');

    // Initialize chat panel
    function initChatPanel() {
        // Add welcome message
        addMessageToChat('assistant', 'Hello! I\'m your AI assistant. How can I help you with your code today?');
        
        // Update model info
        updateChatModelInfo();
    }

    // Open chat panel
    openChatBtn.addEventListener('click', function() {
        chatPanel.classList.add('open');
        chatInput.focus();
    });

    // Close chat panel
    closeChatBtn.addEventListener('click', function() {
        chatPanel.classList.remove('open');
    });

    // Send message when button is clicked
    sendChatBtn.addEventListener('click', sendChatMessage);

    // Send message when Enter is pressed (but allow Shift+Enter for new lines)
    chatInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendChatMessage();
        }
    });

    // Auto-resize textarea as user types
    chatInput.addEventListener('input', function() {
        this.style.height = 'auto';
        this.style.height = (this.scrollHeight) + 'px';
    });

    // Function to send chat message
    function sendChatMessage() {
        if (isWaitingForResponse) return;
        
        const message = chatInput.value.trim();
        if (!message) return;
        
        // Add user message to chat
        addMessageToChat('user', message);
        
        // Clear input
        chatInput.value = '';
        chatInput.style.height = 'auto';
        
        // Show loading indicator
        showLoadingIndicator();
        
        // Send to backend
        sendToLLM(message);
    }

    // Function to add message to chat
    function addMessageToChat(role, content) {
        const messageDiv = document.createElement('div');
        messageDiv.classList.add('chat-message');
        messageDiv.classList.add(role === 'user' ? 'user-message' : 'assistant-message');
        
        // Store in history
        chatHistory.push({ role, content });
        
        // For assistant messages, we'll need to render markdown
        if (role === 'assistant') {
            // Simple markdown rendering for code blocks
            // A more complete solution would use a library like marked.js
            const formattedContent = content.replace(/```(\w*)([\s\S]*?)```/g, 
                '<pre><code class="language-$1">$2</code></pre>');
            messageDiv.innerHTML = formattedContent;
        } else {
            messageDiv.textContent = content;
        }
        
        chatContent.appendChild(messageDiv);
        chatContent.scrollTop = chatContent.scrollHeight;
    }

    // Function to show loading indicator
    function showLoadingIndicator() {
        isWaitingForResponse = true;
        sendChatBtn.disabled = true;
        
        const loadingDiv = document.createElement('div');
        loadingDiv.classList.add('chat-message', 'assistant-message', 'loading-indicator');
        loadingDiv.id = 'loadingIndicator';
        loadingDiv.innerHTML = 'Thinking <span></span><span></span><span></span>';
        
        chatContent.appendChild(loadingDiv);
        chatContent.scrollTop = chatContent.scrollHeight;
    }

    // Function to hide loading indicator
    function hideLoadingIndicator() {
        isWaitingForResponse = false;
        sendChatBtn.disabled = false;
        
        const loadingIndicator = document.getElementById('loadingIndicator');
        if (loadingIndicator) {
            loadingIndicator.remove();
        }
    }

    // Function to send message to LLM
    function sendToLLM(message) {
        // Prepare the request data
        const requestData = {
            message: message,
            history: chatHistory.slice(0, -1) // Exclude the latest user message as it's sent separately
        };

        // Send to backend
        fetch('/api/chat', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            return response.json();
        })
        .then(data => {
            hideLoadingIndicator();
            addMessageToChat('assistant', data.response);
        })
        .catch(error => {
            console.error('Error:', error);
            hideLoadingIndicator();
            addMessageToChat('assistant', 'Sorry, I encountered an error. Please try again.');
        });
    }

    // Function to update model info in the chat footer
    function updateChatModelInfo() {
        fetch('/api/settings')
        .then(response => response.json())
        .then(data => {
            chatModelInfo.textContent = data.model;
        })
        .catch(error => {
            console.error('Error fetching settings:', error);
        });
    }

    // Initialize chat when page loads
    initChatPanel();
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

// Call this when the page loads
document.addEventListener('DOMContentLoaded', function() {
    setupDragAndDrop();
    setupEventListeners();
});
