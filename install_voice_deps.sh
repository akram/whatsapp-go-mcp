#!/bin/bash

# Voice Processing Dependencies Installation Script
# This script installs the required dependencies for voice note processing

echo "🎤 Installing voice processing dependencies..."

# Check if running on macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "📱 Detected macOS"
    
    # Check if Homebrew is installed
    if ! command -v brew &> /dev/null; then
        echo "❌ Homebrew not found. Please install Homebrew first:"
        echo "   /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
        exit 1
    fi
    
    echo "🍺 Installing dependencies with Homebrew..."
    
    # Install ffmpeg for audio processing
    if ! command -v ffmpeg &> /dev/null; then
        echo "📹 Installing ffmpeg..."
        brew install ffmpeg
    else
        echo "✅ ffmpeg already installed"
    fi
    
    # Install espeak for text-to-speech
    if ! command -v espeak &> /dev/null; then
        echo "🔊 Installing espeak..."
        brew install espeak
    else
        echo "✅ espeak already installed"
    fi
    
    # Install pipx for better Python package management
    if ! command -v pipx &> /dev/null; then
        echo "📦 Installing pipx for Python package management..."
        brew install pipx
    else
        echo "✅ pipx already installed"
    fi
    
    # Install whisper for speech-to-text (optional, requires Python)
    if ! command -v whisper &> /dev/null; then
        echo "🎙️ Installing whisper (requires Python)..."
        if command -v pipx &> /dev/null; then
            echo "📦 Using pipx to install whisper..."
            pipx install openai-whisper
        elif command -v pip3 &> /dev/null; then
            echo "📦 Using pip3 to install whisper..."
            pip3 install --user openai-whisper || pip3 install --break-system-packages openai-whisper
        elif command -v pip &> /dev/null; then
            echo "📦 Using pip to install whisper..."
            pip install --user openai-whisper || pip install --break-system-packages openai-whisper
        else
            echo "⚠️ Python pip not found. Please install Python and pip first."
            echo "   You can install Python from https://python.org"
        fi
    else
        echo "✅ whisper already installed"
    fi

# Check if running on Ubuntu/Debian
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "🐧 Detected Linux"
    
    # Update package list
    sudo apt update
    
    echo "📦 Installing dependencies with apt..."
    
    # Install ffmpeg
    if ! command -v ffmpeg &> /dev/null; then
        echo "📹 Installing ffmpeg..."
        sudo apt install -y ffmpeg
    else
        echo "✅ ffmpeg already installed"
    fi
    
    # Install espeak
    if ! command -v espeak &> /dev/null; then
        echo "🔊 Installing espeak..."
        sudo apt install -y espeak
    else
        echo "✅ espeak already installed"
    fi
    
    # Install Python and pip for whisper
    if ! command -v python3 &> /dev/null; then
        echo "🐍 Installing Python3..."
        sudo apt install -y python3 python3-pip
    fi
    
    # Install whisper
    if ! command -v whisper &> /dev/null; then
        echo "🎙️ Installing whisper..."
        pip3 install openai-whisper
    else
        echo "✅ whisper already installed"
    fi

else
    echo "❌ Unsupported operating system: $OSTYPE"
    echo "Please install the following dependencies manually:"
    echo "  - ffmpeg (for audio processing)"
    echo "  - espeak (for text-to-speech)"
    echo "  - whisper (for speech-to-text, requires Python)"
    exit 1
fi

echo ""
echo "✅ Voice processing dependencies installation complete!"
echo ""
echo "📋 Installed tools:"
echo "  - ffmpeg: Audio/video processing"
echo "  - espeak: Text-to-speech conversion"
echo "  - whisper: Speech-to-text conversion"
echo ""
echo "🔧 Optional: Set OPENAI_API_KEY environment variable for cloud-based speech-to-text"
echo "   export OPENAI_API_KEY='your-api-key-here'"
echo ""
echo "🎤 Voice note processing is now ready!"
