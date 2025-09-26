# Voice Note Processing

This WhatsApp MCP server now supports automatic voice note processing! When a voice note (PTT - Push-to-Talk message) is received, the system will:

1. **Download** the voice message
2. **Transcribe** it to text using speech-to-text
3. **Process** the text with an AI agent
4. **Convert** the response back to speech
5. **Send** the audio response back to the user

## Features

- 🎤 **Automatic Voice Processing**: Responds to voice notes automatically
- 🎙️ **Speech-to-Text**: Converts voice messages to text using Whisper
- 🤖 **AI Agent Integration**: Processes transcribed text with LlamaStack agent
- 🔊 **Text-to-Speech**: Converts AI responses back to audio using espeak
- 📱 **WhatsApp Integration**: Sends audio responses as voice messages

## Installation

### Prerequisites

Before using voice processing, you need to install the required dependencies:

```bash
# Run the installation script
./install_voice_deps.sh
```

Or install manually:

#### macOS (with Homebrew)
```bash
brew install ffmpeg espeak
pip3 install openai-whisper
```

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install -y ffmpeg espeak python3 python3-pip
pip3 install openai-whisper
```

### Required Tools

- **ffmpeg**: Audio processing and format conversion
- **espeak**: Text-to-speech synthesis
- **whisper**: Speech-to-text transcription (OpenAI Whisper)

## Configuration

### Environment Variables

Set the following environment variables for optimal performance:

```bash
# Optional: Use OpenAI Whisper API instead of local whisper
export OPENAI_API_KEY="your-openai-api-key"

# LlamaStack configuration (already configured)
export LLAMASTACK_BASE_URL="your-llamastack-url"
export LLAMASTACK_API_KEY="your-llamastack-api-key"
export LLAMASTACK_MODEL="your-model-name"
```

### Media Directory

The system uses the `WHATSAPP_MEDIA_DIR` environment variable (defaults to `./media`) to store:
- Downloaded voice messages
- Generated TTS audio files
- Transcription files (temporary)

## How It Works

### Voice Processing Pipeline

1. **Voice Message Detection**: The system detects incoming PTT (Push-to-Talk) messages
2. **Download**: Downloads the voice message to local storage
3. **Transcription**: Converts audio to text using Whisper (local or OpenAI API)
4. **AI Processing**: Sends transcribed text to LlamaStack agent for processing
5. **TTS Generation**: Converts AI response to speech using espeak
6. **Audio Response**: Sends the generated audio back as a voice message

### Supported Audio Formats

- **Input**: OGG Opus (WhatsApp voice messages)
- **Output**: OGG Opus (WhatsApp-compatible voice messages)

### Fallback Behavior

If any step fails, the system gracefully falls back:
- If transcription fails → Sends text message asking user to try again
- If TTS fails → Sends text response instead of audio
- If AI processing fails → Sends fallback text message

## Usage

### Automatic Processing

Voice processing happens automatically when voice notes are received. No additional configuration needed!

### Manual Testing

You can test the voice processing by:

1. Sending a voice note to the WhatsApp number
2. The system will automatically process and respond with an audio message

### API Endpoints

The existing voice note sending endpoints still work:

```bash
# Send voice note via multipart form
curl -X POST http://localhost:8080/api/send-voice-note \
  -F "recipient=1234567890@s.whatsapp.net" \
  -F "file=@voice_note.ogg"

# Send voice note via JSON (Python-style)
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": "1234567890@s.whatsapp.net",
    "media_path": "/path/to/voice_note.ogg"
  }'
```

## Troubleshooting

### Common Issues

1. **"whisper not found"**
   ```bash
   pip3 install openai-whisper
   ```

2. **"espeak not found"**
   ```bash
   # macOS
   brew install espeak
   
   # Ubuntu/Debian
   sudo apt install espeak
   ```

3. **"ffmpeg not found"**
   ```bash
   # macOS
   brew install ffmpeg
   
   # Ubuntu/Debian
   sudo apt install ffmpeg
   ```

4. **Audio quality issues**
   - Check microphone quality when recording voice notes
   - Ensure clear speech without background noise
   - Try shorter voice messages for better transcription

### Logs

Check the application logs for detailed processing information:

```bash
# Look for voice processing logs
tail -f logs/whatsapp-mcp.log | grep -E "(🎤|🎙️|🔊|🤖)"
```

### Performance Tips

1. **Use OpenAI Whisper API** for better transcription quality:
   ```bash
   export OPENAI_API_KEY="your-api-key"
   ```

2. **Optimize audio settings**:
   - Use clear, slow speech
   - Minimize background noise
   - Keep voice messages under 30 seconds for best results

3. **Monitor resource usage**:
   - Local whisper can be CPU-intensive
   - TTS generation is relatively fast
   - Consider using cloud services for production

## Advanced Configuration

### Custom TTS Settings

Modify the `generateSpeechWithEspeak` function to adjust:
- Speech rate (`-s` parameter)
- Voice (`-v` parameter)
- Audio quality (ffmpeg parameters)

### Custom STT Settings

Modify the `transcribeWithLocalWhisper` function to adjust:
- Whisper model size
- Language settings
- Output format

### Error Handling

The system includes comprehensive error handling:
- Graceful fallbacks at each step
- Detailed logging for debugging
- User-friendly error messages

## Security Considerations

- Voice messages are temporarily stored locally and automatically cleaned up
- No voice data is permanently stored
- Transcription and TTS processing happens locally (unless using OpenAI API)
- All temporary files are deleted after processing

## Future Enhancements

Potential improvements for future versions:
- Support for multiple languages
- Voice cloning for personalized responses
- Real-time voice processing
- Integration with other STT/TTS services
- Voice command recognition
- Audio quality enhancement

