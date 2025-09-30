# WhatsApp Go Server

A Golang server that provides WhatsApp functionality using the `go.mau.fi/whatsmeow` library with direct HTTP API endpoints.

## Features

- **Contact Management**: Search for contacts by name or phone number
- **Message Handling**: Send and receive WhatsApp messages with media support
- **Chat Management**: List and manage WhatsApp chats and conversations
- **Media Support**: Send images, videos, audio files, and documents
- **Voice Messages**: Send audio files as WhatsApp voice messages
- **🎤 Voice Note Processing**: Automatic voice note transcription and AI-powered responses
- **Message History**: Store and retrieve message history with SQLite
- **Direct HTTP API**: RESTful API endpoints for all WhatsApp operations
- **QR Code Authentication**: Terminal-based QR code scanning for WhatsApp login

## Architecture

The project is organized into several packages for better maintainability:

- `main.go` - Main server entry point with HTTP API
- `models/` - Database models and SQLite operations
- `whatsapp/` - WhatsApp client wrapper using whatsmeow
- `handlers/` - HTTP request handlers for API endpoints
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
go build -o whatsapp-server
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
./whatsapp-server
```

2. The server will start and display a QR code in the terminal for WhatsApp authentication.

3. Scan the QR code with your WhatsApp mobile app to authenticate.

4. Once authenticated, the server will be ready to handle HTTP API requests.

## OpenAPI Documentation

The server includes automatic OpenAPI documentation generation:

- **OpenAPI UI**: Visit `http://localhost:8080/openapi` to view the interactive API documentation
- **OpenAPI JSON**: Access the raw OpenAPI specification at `http://localhost:8080/openapi.json`
- **OpenAPI YAML**: Access the raw OpenAPI specification at `http://localhost:8080/openapi.yaml`

The documentation is automatically generated from code annotations and includes:
- API endpoint descriptions
- Request/response schemas
- Example values
- Interactive testing interface

## API Endpoints

All endpoints run on the same port (default: 8080):

### Health Check
- `GET /health` - Server health status

### WhatsApp API
- `POST /api/list-messages` - List messages from a chat
- `POST /api/search-contacts` - Search for contacts
- `POST /api/send-message` - Send a WhatsApp message
- `POST /api/send-voice-note` - Send a voice note (multipart/form-data)
- `POST /send` - Send voice message (Python-style API with media_path)

### Documentation
- `GET /openapi` - OpenAPI documentation UI
- `GET /openapi.json` - OpenAPI specification in JSON format
- `GET /openapi.yaml` - OpenAPI specification in YAML format

## API Methods

The server provides the following HTTP API methods:

### Contact Management
- `POST /api/search-contacts` - Search for contacts by name or phone number

### Message Management
- `POST /api/list-messages` - Retrieve messages with optional filters and context
- `POST /api/send-message` - Send a WhatsApp message to a specified recipient
- `POST /api/send-voice-note` - Send an audio file as a WhatsApp voice message
- `POST /send` - Send voice message (Python-style API with media_path)

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
```bash
curl -X POST http://localhost:8080/api/search-contacts \
  -H "Content-Type: application/json" \
  -d '{"query": "John"}'
```

### Send Message
```bash
curl -X POST http://localhost:8080/api/send-message \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "1234567890@s.whatsapp.net",
    "message": "Hello from WhatsApp API!"
  }'
```

### Send Voice Note
```bash
curl -X POST http://localhost:8080/api/send-voice-note \
  -F "recipient=1234567890@s.whatsapp.net" \
  -F "file=@/path/to/audio.ogg"
```

### Send Voice Message (Python-style)
```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "1234567890@s.whatsapp.net",
    "media_path": "/path/to/audio.ogg"
  }'
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

## API Documentation

The server provides OpenAPI 3.0 documentation with the following endpoints:

- **Interactive UI**: `GET /openapi` - Modern Swagger UI for exploring the API
- **JSON Specification**: `GET /openapi.json` - OpenAPI 3.0 specification in JSON format
- **YAML Specification**: `GET /openapi.yaml` - OpenAPI 3.0 specification in YAML format

The documentation is automatically generated from Go code annotations using the `swag-openapi3` tool.

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
- `github.com/mdp/qrterminal/v3` v3.2.0 - QR code terminal display
- `github.com/mattn/go-sqlite3` v1.14.32 - SQLite database driver
- `github.com/gorilla/mux` v1.8.1 - HTTP router and URL matcher
- `github.com/llamastack/llama-stack-client-go` v0.1.0-alpha.1 - AI client for voice processing

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
