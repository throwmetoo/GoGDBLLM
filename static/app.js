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
  uploadBtn.addEventListener("click", handleFileUpload);

  // Execute button
  executeBtn.addEventListener("click", executeUploadedFile);

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

// Update the handleFileUpload function to validate settings first
async function handleFileUpload() {
  const file = fileInput.files[0];
  if (!file) {
    alert("Please select a file first");
    return;
  }

  // Validate settings before proceeding
  const settingsValid = await validateSettings();
  if (!settingsValid) {
    return;
  }

  // Switch to terminal view
  showSection("terminalSection");

  // Clear terminal and show upload message
  terminalDiv.textContent = "";
  window.appendToTerminal("Uploading file...");

  // Upload the file
  const success = await uploadFile(file);

  // Execute the file immediately after upload
  if (success) {
    executeUploadedFile();
  }
}

// Upload file to server
async function uploadFile(file) {
  const formData = new FormData();
  formData.append("file", file);

  try {
    const response = await fetch("/upload", {
      method: "POST",
      body: formData,
    });

    if (!response.ok) {
      window.appendToTerminal(`Server error: ${response.statusText}`);
      return false;
    }

    uploadedFileName = file.name;
    window.appendToTerminal("File uploaded successfully.");
    return true;
  } catch (error) {
    window.appendToTerminal(`Upload failed: ${error.message}`);
    return false;
  }
}

// Execute the uploaded file
function executeUploadedFile() {
  if (!uploadedFileName) {
    window.appendToTerminal("No file has been uploaded yet.");
    return;
  }

  window.appendToTerminal(`Executing ${uploadedFileName}...`);

  if (!terminalConnected) {
    connectWebSocket(() => {
      sendCommand(`/tmp/${uploadedFileName}`);
    });
  } else {
    sendCommand(`/tmp/${uploadedFileName}`);
  }
}

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

async function saveSettings() {
    const provider = document.getElementById('provider').value;
    const model = document.getElementById('model').value;
    const apiKey = document.getElementById('apiKey').value;
    
    currentSettings = { provider, model, apiKey };
    
    try {
        const response = await fetch('/settings', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(currentSettings)
        });
        
        if (!response.ok) {
            throw new Error('Failed to save settings');
        }
        
        showNotification('Settings saved successfully', 'success');
    } catch (error) {
        console.error('Error saving settings:', error);
        showNotification('Failed to save settings: ' + error.message, 'error');
    }
}

// Add new function to test the connection
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
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText);
        }
        
        const result = await response.json();
        window.appendToTerminal('Connection successful! Model is available.');
        showNotification('Connection test successful! The API key and model are valid.', 'success');
        
    } catch (error) {
        window.appendToTerminal(`Connection failed: ${error.message}`);
        showNotification(`Connection test failed: ${error.message}`, 'error');
    } finally {
        testBtn.disabled = false;
        saveBtn.disabled = false;
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
