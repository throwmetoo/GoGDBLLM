# GoGDBLLM Design Document

## Overview

GoGDBLLM is an interactive debugging tool that combines the power of GDB (GNU Debugger) with LLM (Large Language Model) assistance. The application provides a web-based interface for uploading, executing, and debugging programs while leveraging AI to help users understand code behavior, debug issues, and learn programming concepts.

## Architecture

The application follows a client-server architecture:

1. **Backend**: Go server that handles file uploads, GDB execution, and LLM API communication
2. **Frontend**: HTML/CSS/JavaScript web interface that provides terminal emulation and chat capabilities

## Core Components

### 1. File Upload System

- Allows users to upload executable files for debugging
- Stores files temporarily on the server
- Provides feedback on upload status

### 2. Terminal Emulation

- Interactive terminal interface for GDB commands
- Real-time output display using WebSockets
- Support for special key combinations (CTRL+C, arrow keys, etc.)
- Command history navigation

### 3. Settings Management

- Provider selection (Anthropic, OpenAI, OpenRouter)
- Model selection based on provider
- API key storage and management
- Connection testing functionality

### 4. LLM Chat Integration

The chat panel provides AI assistance during debugging sessions:

#### Phase 1: Basic Chat Functionality (Current)
- Slide-in panel with conversation UI
- Connection to LLM API using configured settings
- Simple Q&A capabilities
- Support for markdown rendering in responses

#### Phase 2: Context-Aware Responses (Planned)
- Include terminal context in LLM prompts
- Improved markdown rendering for code blocks
- Streaming responses for better user experience
- Ability to reference specific parts of terminal output

#### Phase 3: Advanced Features (Planned)
- Specialized buttons for common debugging tasks
- Code/command suggestions
- Conversation management features
- Session history and export options

## User Interface

### Main Sections

1. **Upload Section**
   - File upload interface with drag-and-drop support
   - Upload button and status indicators

2. **Terminal Section**
   - GDB terminal emulation
   - Command input area
   - Execute button for running programs

3. **Settings Section**
   - Provider and model selection dropdowns
   - API key input field
   - Test connection button

4. **Chat Panel**
   - Floating chat button in bottom-right corner
   - Slide-in panel with conversation history
   - Message input area
   - Send button
   - Model information display

## Data Flow

1. User uploads an executable file
2. Server stores the file and makes it available for GDB
3. User interacts with GDB through the terminal interface
4. WebSocket connection streams GDB output to the frontend
5. User can open the chat panel to ask questions about the code or debugging process
6. LLM requests are sent to the configured provider's API
7. Responses are displayed in the chat panel with proper formatting

## Technical Implementation

### Backend (Go)

- HTTP server for static files and API endpoints
- WebSocket handling for real-time terminal communication
- Process management for GDB execution
- Settings persistence using JSON files
- LLM API integration with multiple providers

### Frontend (HTML/CSS/JavaScript)

- Responsive design with mobile support
- Terminal emulation using plain JavaScript
- WebSocket client for real-time updates
- Chat interface with markdown rendering
- Settings UI with dynamic model options

## Security Considerations

- API keys are stored locally and never exposed to other users
- Temporary file storage with proper permissions
- Input validation for all API endpoints
- Process isolation for executed programs

## Future Enhancements

1. **Enhanced Context Awareness**
   - Automatically include relevant code snippets in LLM prompts
   - Parse GDB output to provide more targeted assistance

2. **Debugging Assistance**
   - Suggest breakpoints and watch variables
   - Explain complex data structures
   - Identify common bugs and suggest fixes

3. **Learning Features**
   - Explain code line-by-line
   - Provide conceptual explanations of programming constructs
   - Suggest improvements and best practices

4. **Collaboration Tools**
   - Share debugging sessions
   - Export conversations and debugging steps
   - Annotated debugging playback

## Implementation Roadmap

1. **Phase 1 (In Progress)**
   - Basic file upload and execution
   - GDB integration with terminal emulation
   - Settings management
   - Simple chat interface

2. **Phase 2 (Planned)**
   - Improved chat UI with better formatting
   - Context-aware LLM prompts
   - Enhanced error handling

3. **Phase 3 (Planned)**
   - Advanced debugging assistance
   - Code explanation features
   - Performance optimizations

4. **Phase 4 (Future)**
   - Collaboration tools
   - Session management
   - Extended provider support
