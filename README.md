# GoGDBLLM

GoGDBLLM is an interactive debugging tool that combines GDB (GNU Debugger) with LLM (Large Language Model) assistance. It provides a web-based interface for uploading, executing, and debugging programs while leveraging AI to help users understand code behavior, debug issues, and learn programming concepts.

## Features

- **Interactive GDB Terminal**: Access GDB through a web-based terminal with command history and special key combinations support
- **File Upload System**: Upload executables for debugging with drag-and-drop support
- **LLM-Assisted Debugging**: Get AI assistance for understanding code behavior, debugging issues, and learning GDB commands
- **Multi-Provider Support**: Choose from different LLM providers (Anthropic, OpenAI, OpenRouter) for AI assistance
- **Clean UI**: Modern, responsive interface with light/dark mode support

## Project Structure

The project follows a clean architecture pattern with clear separation of concerns:

```
.
├── cmd/
│   └── gogdbllm/        # Application entry point
├── internal/
│   ├── api/             # API interfaces for LLM integration
│   ├── gdb/             # GDB process management
│   ├── handlers/        # HTTP request handlers
│   ├── settings/        # Application settings management
│   └── websocket/       # WebSocket communication
├── uploads/             # Directory for uploaded executables
└── web/
    ├── static/          # Static assets (JS, CSS)
    │   ├── css/
    │   └── js/
    └── templates/       # HTML templates
```

## Installation

1. Make sure you have Go 1.22+ installed
2. Clone the repository
3. Install dependencies:

```bash
go mod download
go mod tidy
```

## Running the Application

1. Using the provided script:

```bash
./start.sh
```

2. Or manually:

```bash
go build -o gogdbllm ./cmd/gogdbllm
./gogdbllm
```

3. Using Docker:

```bash
docker-compose up -d
```

The application will be available at http://localhost:8080

## Usage

1. **Upload an Executable**: Drag and drop an executable file or use the file browser
2. **Debug Your Program**: Use standard GDB commands in the terminal
3. **Get AI Assistance**: Click the chat button to ask questions about your debugging session

## API Integration

The application supports multiple LLM providers:

- **Anthropic**: Supports Claude models
- **OpenAI**: Supports GPT models
- **OpenRouter**: Provides access to multiple models from different providers

To use the AI features, you need to configure your API key in the settings.

## Development

### Prerequisites

- Go 1.22+
- GDB
- Web browser

### Building from Source

```bash
git clone https://github.com/yourusername/gogdbllm.git
cd gogdbllm
go build -o gogdbllm ./cmd/gogdbllm
```

## Design Document

For information about the design principles and architecture decisions, see the [Design Document](DesignDocument.md).

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [Gorilla WebSocket](https://github.com/gorilla/websocket) for WebSocket implementation
- [Gorilla Mux](https://github.com/gorilla/mux) for HTTP routing 