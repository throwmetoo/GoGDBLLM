/**
 * upload.js - Handles file upload functionality
 */

// Initialize upload section
function initUploadSection() {
    const dropZone = document.getElementById('dropZone');
    const fileInput = document.getElementById('fileInput');
    const uploadBtn = document.getElementById('uploadBtn');
    const uploadStatus = document.getElementById('uploadStatus');
    
    let selectedFile = null;
    
    // Handle drop zone click
    dropZone.addEventListener('click', () => {
        fileInput.click();
    });
    
    // Handle file selection
    fileInput.addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            selectedFile = e.target.files[0];
            dropZone.textContent = `Selected: ${selectedFile.name}`;
            uploadBtn.disabled = false;
        }
    });
    
    // Handle drag and drop events
    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('dragover');
    });
    
    dropZone.addEventListener('dragleave', () => {
        dropZone.classList.remove('dragover');
    });
    
    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('dragover');
        
        if (e.dataTransfer.files.length > 0) {
            selectedFile = e.dataTransfer.files[0];
            dropZone.textContent = `Selected: ${selectedFile.name}`;
            uploadBtn.disabled = false;
        }
    });
    
    // Handle upload button click
    uploadBtn.addEventListener('click', async () => {
        if (!selectedFile) {
            AppUtils.showNotification('Please select a file first', 'error');
            return;
        }
        
        // Create form data
        const formData = new FormData();
        formData.append('executable', selectedFile);
        
        // Update UI during upload
        uploadBtn.disabled = true;
        uploadStatus.textContent = 'Uploading...';
        uploadStatus.className = 'status-message';
        
        try {
            // Send upload request
            const response = await fetch('/upload', {
                method: 'POST',
                body: formData
            });
            
            const result = await response.json();
            
            if (result.success) {
                // Show success message
                uploadStatus.textContent = `Upload successful: ${selectedFile.name}`;
                uploadStatus.classList.add('success');
                
                // Start GDB with the uploaded file
                await startGDB(result.data.filename);
                
                // Switch to terminal tab
                document.getElementById('terminalTabBtn').click();
                
                // Show notification
                AppUtils.showNotification('File uploaded and debugger started', 'success');
            } else {
                // Show error message
                uploadStatus.textContent = `Upload failed: ${result.error}`;
                uploadStatus.classList.add('error');
                AppUtils.showNotification('Upload failed', 'error');
            }
        } catch (error) {
            console.error('Upload error:', error);
            uploadStatus.textContent = `Upload error: ${error.message}`;
            uploadStatus.classList.add('error');
            AppUtils.showNotification('Upload error', 'error');
        } finally {
            uploadBtn.disabled = false;
        }
    });
    
    console.log('Upload section initialized');
}

// Start GDB with the uploaded file
async function startGDB(filename) {
    try {
        const response = await fetch('/start-gdb', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ filename })
        });
        
        const result = await response.json();
        
        if (!result.success) {
            throw new Error(result.error || 'Failed to start GDB');
        }
        
        return true;
    } catch (error) {
        console.error('Error starting GDB:', error);
        AppUtils.showNotification('Failed to start debugger', 'error');
        return false;
    }
}

// Make available globally
window.AppUpload = {
    initUploadSection
}; 