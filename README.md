# WhatsApp Go MCP Server

A Golang MCP (Model Context Protocol) server that provides WhatsApp functionality using the `go.mau.fi/whatsmeow` library and the `gomcp-sdk` for proper MCP implementation.

## Features

- **Contact Management**: Search for contacts by name or phone number
- **Message Handling**: Send and receive WhatsApp messages with media support
- **Chat Management**: List and manage WhatsApp chats and conversations
- **Media Support**: Send images, videos, audio files, and documents
- **Voice Messages**: Send audio files as WhatsApp voice messages
- **🎤 Voice Note Processing**: Automatic voice note transcription and AI-powered responses
- **Message History**: Store and retrieve message history with SQLite
- **MCP Compliance**: Full MCP 2024-11-05 specification support
- **QR Code Authentication**: Terminal-based QR code scanning for WhatsApp login

## Architecture

The project is organized into several packages for better maintainability:

- `main.go` - Main server entry point with SSE transport
- `models/` - Database models and SQLite operations
- `whatsapp/` - WhatsApp client wrapper using whatsmeow
- `mcp/` - MCP server implementation using gomcp-sdk
- `config/` - Configuration management
- `utils/` - Utility functions for file handling, logging, and validation

## Installation

1. Clone the repository:
```bash
git clone https://github.com/mcp/whatsapp-go-mcp.git
cd whatsapp-go-mcp
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o whatsapp-mcp-server
```

## Configuration

The server can be configured using environment variables:

- `PORT` - Server port (default: 8080)
- `WHATSAPP_DB_PATH` - Path to WhatsApp database (default: ./whatsapp.db)
- `WHATSAPP_MEDIA_DIR` - Directory for media files (default: ./media)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `QR_CODE_DIR` - Directory for QR codes (default: ./qr_codes)

## Usage

1. Start the server:
```bash
./whatsapp-mcp-server
```

2. The server will start and display a QR code in the terminal for WhatsApp authentication.

3. Scan the QR code with your WhatsApp mobile app to authenticate.

4. Once authenticated, the server will be ready to handle MCP requests.

## OpenAPI Documentation

The server includes automatic OpenAPI/Swagger documentation generation:

- **Swagger UI**: Visit `http://localhost:8080/swagger/` to view the interactive API documentation
- **OpenAPI JSON**: Access the raw OpenAPI specification at `http://localhost:8080/swagger/doc.json`
- **OpenAPI YAML**: Access the raw OpenAPI specification at `http://localhost:8080/swagger/doc.yaml`

The documentation is automatically generated from code annotations and includes:
- API endpoint descriptions
- Request/response schemas
- Example values
- Interactive testing interface

## Tools Discovery

The server provides a tools discovery endpoint that returns all available MCP tools:

- **Tools List**: Visit `http://localhost:8080/tools` to get a JSON list of all available tools
- **Tool Schemas**: Each tool includes its input schema with parameter descriptions and requirements
- **Dynamic Discovery**: Clients can programmatically discover available tools and their capabilities

Example tools response:
```json
{
  "tools": [
    {
      "name": "search_contacts",
      "description": "Search for contacts by name or phone number",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": {
            "type": "string",
            "description": "Search query for contacts"
          }
        },
        "required": ["query"]
      }
    }
  ]
}
```

## API Endpoints

All endpoints run on the same port (default: 8080):

### MCP Server (SSE Transport)
- `GET /events` - Server-Sent Events endpoint for MCP communication
- `GET /sse` - Alias for /events endpoint

### Health Check
- `GET /health` - Server health status

### Tools Discovery
- `GET /tools` - List all available MCP tools with their schemas

### Documentation
- `GET /swagger/` - OpenAPI/Swagger documentation UI
- `GET /swagger/doc.json` - OpenAPI specification in JSON format
- `GET /swagger/doc.yaml` - OpenAPI specification in YAML format

## MCP Methods

The server implements the following MCP methods:

### Contact Management
- `search_contacts` - Search for contacts by name or phone number
- `get_contact_chats` - List all chats involving a specific contact

### Message Management
- `list_messages` - Retrieve messages with optional filters and context
- `get_message_context` - Retrieve context around a specific message
- `get_last_interaction` - Get the most recent message with a contact
- `send_message` - Send a WhatsApp message to a specified recipient
- `send_file` - Send a file (image, video, document) to a recipient
- `send_audio_message` - Send an audio file as a WhatsApp voice message
- `download_media` - Download media from a WhatsApp message

### Chat Management
- `list_chats` - List available chats with metadata
- `get_chat` - Get information about a specific chat
- `get_direct_chat_by_contact` - Find a direct chat with a specific contact

## Database Schema

The server uses SQLite to store messages and metadata:

### Messages Table
```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    time DATETIME NOT NULL,
    sender TEXT NOT NULL,
    content TEXT,
    is_from_me BOOLEAN NOT NULL,
    media_type TEXT,
    filename TEXT,
    chat_jid TEXT NOT NULL,
    message_id TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Contacts Table
```sql
CREATE TABLE contacts (
    jid TEXT PRIMARY KEY,
    name TEXT,
    push_name TEXT,
    is_group BOOLEAN NOT NULL DEFAULT FALSE,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Chats Table
```sql
CREATE TABLE chats (
    jid TEXT PRIMARY KEY,
    name TEXT,
    last_message TEXT,
    last_message_time DATETIME,
    unread_count INTEGER DEFAULT 0,
    is_group BOOLEAN NOT NULL DEFAULT FALSE,
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Example Usage

### Search Contacts
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "search_contacts",
  "params": {
    "query": "John"
  }
}
```

### Send Message
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "send_message",
  "params": {
    "recipient": "1234567890@s.whatsapp.net",
    "message": "Hello from MCP!"
  }
}
```

### Send File
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "send_file",
  "params": {
    "recipient": "1234567890@s.whatsapp.net",
    "file_path": "/path/to/image.jpg",
    "caption": "Check out this image!"
  }
}
```

## Voice Note Processing

The server now includes automatic voice note processing! When a voice note (PTT message) is received, the system will:

1. **Download** the voice message
2. **Transcribe** it to text using Whisper (speech-to-text)
3. **Process** the text with an AI agent (LlamaStack)
4. **Convert** the response back to speech using espeak
5. **Send** the audio response back as a voice message

### Prerequisites

Install the required dependencies:

```bash
# Run the installation script
./install_voice_deps.sh
```

Or install manually:
- **ffmpeg**: Audio processing
- **espeak**: Text-to-speech synthesis  
- **whisper**: Speech-to-text transcription

### Configuration

Set optional environment variables:

```bash
# Use OpenAI Whisper API instead of local whisper
export OPENAI_API_KEY="your-openai-api-key"
```

For detailed voice processing documentation, see [VOICE_PROCESSING.md](VOICE_PROCESSING.md).

## Media Support

The server supports various media types:

### Images
- JPEG, PNG, GIF, WebP, BMP, TIFF

### Videos
- MP4, AVI, MOV, MKV, WMV, FLV, WebM, M4V

### Audio
- MP3, WAV, OGG, Opus, AAC, FLAC, M4A, WMA

### Documents
- PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, ZIP, RAR, 7Z

## Security Considerations

- The server stores WhatsApp session data locally
- Media files are stored in the specified media directory
- JID validation is performed for all contact operations
- File path validation prevents directory traversal attacks

## Dependencies

- `go.mau.fi/whatsmeow` v0.0.0-20250922112717-258fd9454b95 - WhatsApp client library
- `github.com/fredcamaral/gomcp-sdk` v1.2.0 - MCP server implementation
- `github.com/mdp/qrterminal/v3` v3.2.0 - QR code terminal display
- `github.com/mattn/go-sqlite3` v1.14.32 - SQLite database driver
- `github.com/swaggo/http-swagger` v1.3.4 - Swagger UI for HTTP servers
- `github.com/swaggo/swag` v1.16.6 - Swagger documentation generator
- `github.com/gorilla/mux` v1.8.1 - HTTP router and URL matcher

## License

This project is licensed under the MIT License.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Troubleshooting

### Common Issues

1. **QR Code not displaying**: Ensure your terminal supports UTF-8 characters
2. **Authentication fails**: Make sure your phone has internet connection and WhatsApp is updated
3. **Database errors**: Check file permissions for the database directory
4. **Media upload fails**: Verify file permissions and available disk space

### Logs

The server provides structured JSON logging. Set the `LOG_LEVEL` environment variable to control verbosity:

- `debug` - All messages
- `info` - Informational messages and above
- `warn` - Warning messages and above
- `error` - Error messages only

## Support

For issues and questions, please open an issue on the GitHub repository.
