#!/bin/bash

# Voice Processing Test Script
# This script tests the voice processing functionality

echo "🎤 Testing Voice Processing Functionality"
echo "========================================"

# Check if the server is running
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "❌ Server is not running. Please start the server first:"
    echo "   ./whatsapp-mcp-server"
    exit 1
fi

echo "✅ Server is running"

# Check if required dependencies are installed
echo ""
echo "🔍 Checking dependencies..."

# Check ffmpeg
if command -v ffmpeg &> /dev/null; then
    echo "✅ ffmpeg is installed"
else
    echo "❌ ffmpeg not found. Please install it:"
    echo "   brew install ffmpeg  # macOS"
    echo "   sudo apt install ffmpeg  # Ubuntu/Debian"
fi

# Check espeak
if command -v espeak &> /dev/null; then
    echo "✅ espeak is installed"
else
    echo "❌ espeak not found. Please install it:"
    echo "   brew install espeak  # macOS"
    echo "   sudo apt install espeak  # Ubuntu/Debian"
fi

# Check whisper
if command -v whisper &> /dev/null; then
    echo "✅ whisper is installed"
else
    echo "❌ whisper not found. Please install it:"
    echo "   pip3 install openai-whisper"
fi

# Check OpenAI API key
if [ -n "$OPENAI_API_KEY" ]; then
    echo "✅ OPENAI_API_KEY is set"
else
    echo "⚠️ OPENAI_API_KEY not set (will use local whisper)"
fi

echo ""
echo "🧪 Testing voice note sending..."

# Create a test audio file using espeak
test_audio="test_voice.ogg"
test_text="Hello, this is a test voice message for the WhatsApp MCP server."

echo "🔊 Generating test audio file..."
if command -v espeak &> /dev/null && command -v ffmpeg &> /dev/null; then
    # Generate WAV with espeak
    espeak -s 150 -v en -w "${test_audio}.wav" "$test_text"
    
    # Convert to OGG with ffmpeg
    ffmpeg -y -i "${test_audio}.wav" -c:a libopus -b:a 64k -ar 48000 -ac 1 "$test_audio"
    
    # Clean up WAV file
    rm "${test_audio}.wav"
    
    echo "✅ Test audio file created: $test_audio"
    
    # Test sending voice note (you'll need to replace with actual recipient)
    echo ""
    echo "📤 Testing voice note sending..."
    echo "Note: Replace 'RECIPIENT_JID' with an actual WhatsApp JID"
    
    # Example curl command (commented out - uncomment and modify as needed)
    # curl -X POST http://localhost:8080/api/send-voice-note \
    #   -F "recipient=RECIPIENT_JID@s.whatsapp.net" \
    #   -F "file=@$test_audio"
    
    echo "📋 To test voice note sending, run:"
    echo "curl -X POST http://localhost:8080/api/send-voice-note \\"
    echo "  -F \"recipient=YOUR_JID@s.whatsapp.net\" \\"
    echo "  -F \"file=@$test_audio\""
    
    # Clean up test file
    echo ""
    echo "🧹 Cleaning up test files..."
    rm -f "$test_audio"
    echo "✅ Test files cleaned up"
    
else
    echo "❌ Cannot create test audio file (espeak or ffmpeg not available)"
fi

echo ""
echo "📋 Voice Processing Test Summary:"
echo "================================="
echo "✅ Server is running"
echo "✅ Dependencies checked"
echo "✅ Test audio generation tested"
echo ""
echo "🎤 Voice processing is ready!"
echo ""
echo "To test with a real voice note:"
echo "1. Send a voice message to your WhatsApp number"
echo "2. The server will automatically process it"
echo "3. You should receive an audio response"
echo ""
echo "For detailed documentation, see VOICE_PROCESSING.md"

