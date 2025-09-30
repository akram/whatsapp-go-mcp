package main

import (
	"context"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

func TestAudioSend(t *testing.T) {
	// Configuration
	dbPath := "whatsapp.db"
	mediaDir := "./media"
	recipient := "21656067876@s.whatsapp.net" // Your phone number
	audioFile := "media/tts/tts_1759043486.ogg"

	log.Printf("🧪 Testing WhatsApp audio message sending")
	log.Printf("📁 Database: %s", dbPath)
	log.Printf("📁 Media dir: %s", mediaDir)
	log.Printf("📱 Recipient: %s", recipient)
	log.Printf("🎵 Audio file: %s", audioFile)

	// Check if audio file exists
	if _, err := os.Stat(audioFile); os.IsNotExist(err) {
		t.Fatalf("❌ Audio file not found: %s", audioFile)
	}

	// Create device store
	ctx := context.Background()
	logger := waLog.Noop
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+dbPath+"?_foreign_keys=on", logger)
	if err != nil {
		t.Fatalf("❌ Failed to create device store: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		t.Fatalf("❌ Failed to get device: %v", err)
	}

	// Create WhatsApp client
	client := whatsmeow.NewClient(deviceStore, nil)

	// Connect to WhatsApp
	log.Printf("🔌 Connecting to WhatsApp...")
	err = client.Connect()
	if err != nil {
		t.Fatalf("❌ Failed to connect: %v", err)
	}

	// Wait for connection
	time.Sleep(2 * time.Second)

	if !client.IsConnected() {
		t.Fatalf("❌ Not connected to WhatsApp")
	}

	log.Printf("✅ Connected to WhatsApp")

	// Parse recipient JID
	recipientJID, err := types.ParseJID(recipient)
	if err != nil {
		t.Fatalf("❌ Invalid recipient JID: %v", err)
	}

	log.Printf("✅ Recipient JID parsed: %s", recipientJID.String())

	// Read audio file
	file, err := os.Open(audioFile)
	if err != nil {
		t.Fatalf("❌ Failed to open audio file: %v", err)
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("❌ Failed to read audio file: %v", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("❌ Failed to get file info: %v", err)
	}

	log.Printf("📊 Audio file details:")
	log.Printf("   Size: %d bytes", fileInfo.Size())
	log.Printf("   Name: %s", fileInfo.Name())

	// Upload media to WhatsApp servers
	log.Printf("🔄 Uploading audio file...")
	uploaded, err := client.Upload(ctx, fileData, whatsmeow.MediaAudio)
	if err != nil {
		t.Fatalf("❌ Failed to upload audio file: %v", err)
	}

	log.Printf("✅ Audio file uploaded successfully")
	log.Printf("   URL: %s", uploaded.URL)
	log.Printf("   FileSHA256: %x", uploaded.FileSHA256)
	log.Printf("   FileEncSHA256: %x", uploaded.FileEncSHA256)
	log.Printf("   MediaKey: %x", uploaded.MediaKey)

	// Get audio duration using ffprobe (simplified)
	duration := uint32(5) // Default duration
	log.Printf("⏱️ Using duration: %d seconds", duration)

	// Create audio message
	fileSizePtr := uint64(fileInfo.Size())
	mimeType := "audio/ogg"
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           &uploaded.URL,
			Mimetype:      &mimeType,
			FileLength:    &fileSizePtr,
			Seconds:       &duration,
			PTT:           boolPtr(true), // Mark as voice message
			FileSHA256:    uploaded.FileSHA256,
			FileEncSHA256: uploaded.FileEncSHA256,
			MediaKey:      uploaded.MediaKey,
		},
	}

	log.Printf("📤 Sending audio message...")
	log.Printf("   Message details:")
	log.Printf("     URL: %s", *msg.AudioMessage.URL)
	log.Printf("     MIME: %s", *msg.AudioMessage.Mimetype)
	log.Printf("     Size: %d", *msg.AudioMessage.FileLength)
	log.Printf("     Duration: %d", *msg.AudioMessage.Seconds)
	log.Printf("     PTT: %v", *msg.AudioMessage.PTT)

	// Send message
	_, err = client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		t.Fatalf("❌ Failed to send audio message: %v", err)
	}

	log.Printf("✅ Audio message sent successfully to %s", recipient)

	// Wait a bit to see if there are any errors
	time.Sleep(5 * time.Second)

	// Try to read messages from the chat using our database
	log.Printf("📖 Reading messages from database...")

	// Import our database models
	// For now, let's just log that we sent the message
	log.Printf("✅ Message sent to %s", recipient)
	log.Printf("   Audio file: %s", audioFile)
	log.Printf("   File size: %d bytes", fileInfo.Size())
	log.Printf("   Upload URL: %s", uploaded.URL)

	// Note: To read messages, we would need to use the database
	// or implement message event handling
	log.Printf("📝 Note: Message reading requires database integration")

	// Disconnect
	client.Disconnect()
	log.Printf("🔌 Disconnected from WhatsApp")
}

// Helper function for creating pointers
func boolPtr(b bool) *bool {
	return &b
}
