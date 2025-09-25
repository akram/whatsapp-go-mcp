package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"whatsapp-go-mcp/models"

	_ "github.com/mattn/go-sqlite3"
)

// Client wraps the WhatsApp client with additional functionality
type Client struct {
	client           *whatsmeow.Client
	db               *models.Database
	deviceStore      *store.Device
	eventHandlerID   uint32
	mediaDir         string
	llamastackClient *LlamaStackClient
}

// LlamaStackClient represents a client for LlamaStack API
type LlamaStackClient struct {
	BaseURL     string
	HTTPClient  *http.Client
	Model       string
	Temperature float64
	MaxTokens   int
}

// LlamaStackToolgroup represents a toolgroup in LlamaStack
type LlamaStackToolgroup struct {
	Identifier  string            `json:"identifier"`
	ProviderID  string            `json:"provider_id"`
	MCPEndpoint map[string]string `json:"mcp_endpoint"`
}

// LlamaStackAgent represents an agent in LlamaStack
type LlamaStackAgent struct {
	Client    *LlamaStackClient
	Model     string
	SessionID string
	Tools     []string
}

// LlamaStackRequest represents a request to LlamaStack
type LlamaStackRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Tools       []string  `json:"tools,omitempty"`
}

// Message represents a message in LlamaStack format
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LlamaStackResponse represents a response from LlamaStack
type LlamaStackResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice represents a choice in LlamaStack response
type Choice struct {
	Message Message `json:"message"`
}

// NewClient creates a new WhatsApp client
func NewClient(dbPath, mediaDir string) (*Client, error) {
	// Create device store
	ctx := context.Background()
	logger := waLog.Noop
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+dbPath+"?_foreign_keys=on", logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create device store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Create WhatsApp client
	client := whatsmeow.NewClient(deviceStore, nil)

	// Create database
	database, err := models.NewDatabase(dbPath + "_messages.db")
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Create media directory
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create media directory: %w", err)
	}

	c := &Client{
		client:      client,
		db:          database,
		deviceStore: deviceStore,
		mediaDir:    mediaDir,
	}

	// Add event handler
	c.eventHandlerID = client.AddEventHandler(c.eventHandler)

	return c, nil
}

// Connect connects to WhatsApp
func (c *Client) Connect(ctx context.Context) error {
	log.Printf("🔌 Attempting to connect to WhatsApp...")

	if c.client.Store.ID == nil {
		// No ID stored, new login
		log.Printf("📱 No stored session found, initiating new login...")
		qrChan, _ := c.client.GetQRChannel(ctx)
		err := c.client.Connect()
		if err != nil {
			log.Printf("❌ Failed to connect: %v", err)
			return fmt.Errorf("failed to connect: %w", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				// Print QR code to terminal using qrterminal library
				fmt.Println("Scan the QR code below with WhatsApp:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else if evt.Event == "success" {
				fmt.Println("Successfully logged in!")
				log.Printf("✅ WhatsApp login successful")
				break
			}
		}
	} else {
		// Already logged in, just connect
		log.Printf("🔄 Using stored session, connecting...")
		err := c.client.Connect()
		if err != nil {
			log.Printf("❌ Failed to connect: %v", err)
			return fmt.Errorf("failed to connect: %w", err)
		}
		log.Printf("✅ WhatsApp connected successfully")
	}

	return nil
}

// Disconnect disconnects from WhatsApp
func (c *Client) Disconnect() {
	c.client.Disconnect()
}

// IsConnected checks if the WhatsApp client is connected
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// EnsureConnected ensures the client is connected, reconnecting if necessary
func (c *Client) EnsureConnected(ctx context.Context) error {
	if !c.IsConnected() {
		log.Printf("⚠️ WhatsApp client not connected, attempting to reconnect...")
		return c.Connect(ctx)
	}
	return nil
}

// Close closes the client and database
func (c *Client) Close() error {
	c.client.RemoveEventHandler(c.eventHandlerID)
	return c.db.Close()
}

// eventHandler handles WhatsApp events
func (c *Client) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		log.Printf("🔔 Processing message event")
		c.handleMessage(v)
	case *events.Receipt:
		log.Printf("🔔 Processing receipt event")
		c.handleReceipt(v)
	case *events.Presence:
		log.Printf("🔔 Processing presence event")
		c.handlePresence(v)
	default:
		log.Printf("🔔 Processing unknown event type: %T", evt)
	}
}

// handleMessage processes incoming messages and routes them to appropriate handlers
func (c *Client) handleMessage(evt *events.Message) {
	msg := evt.Message
	info := evt.Info

	// Log message received
	log.Printf("📨 Message received from %s in chat %s (ID: %s)",
		info.Sender.String(),
		info.Chat.String(),
		info.ID)

	// Route message to appropriate handler based on type
	if msg.GetConversation() != "" {
		c.handleTextMessage(evt, msg.GetConversation())
	} else if msg.GetExtendedTextMessage() != nil {
		c.handleTextMessage(evt, msg.GetExtendedTextMessage().GetText())
	} else if msg.GetImageMessage() != nil {
		c.handleImageMessage(evt, msg.GetImageMessage())
	} else if msg.GetVideoMessage() != nil {
		c.handleVideoMessage(evt, msg.GetVideoMessage())
	} else if msg.GetAudioMessage() != nil {
		c.handleAudioMessage(evt, msg.GetAudioMessage())
	} else if msg.GetDocumentMessage() != nil {
		c.handleDocumentMessage(evt, msg.GetDocumentMessage())
	} else {
		log.Printf("❓ Unknown message type")
		c.handleUnknownMessage(evt)
	}
}

// handleTextMessage processes text messages
func (c *Client) handleTextMessage(evt *events.Message, content string) {
	info := evt.Info

	log.Printf("💬 Text message: %s", content)

	// Store message in database
	message := &models.Message{
		Time:      info.Timestamp,
		Sender:    info.Sender.String(),
		Content:   content,
		IsFromMe:  info.IsFromMe,
		MediaType: "text",
		Filename:  "",
		ChatJID:   info.Chat.String(),
		MessageID: info.ID,
	}

	if err := c.db.StoreMessage(message); err != nil {
		log.Printf("❌ Failed to store text message: %v", err)
	} else {
		log.Printf("✅ Text message stored successfully")
	}

	// Update chat info
	c.updateChatInfo(info.Chat, content, info.Timestamp)

	// Process text message for commands or auto-replies
	c.processTextMessage(evt, content)
}

// handleAudioMessage processes audio/voice messages
func (c *Client) handleAudioMessage(evt *events.Message, audioMsg *waE2E.AudioMessage) {
	info := evt.Info

	log.Printf("🎵 Audio message received")
	log.Printf("📊 Audio details - Duration: %d seconds, PTT: %v, MIME: %s",
		audioMsg.GetSeconds(), audioMsg.GetPTT(), audioMsg.GetMimetype())

	// Determine if it's a voice message (PTT) or regular audio
	messageType := "audio"
	if audioMsg.GetPTT() {
		messageType = "voice"
		log.Printf("🎤 Voice message (PTT)")
	} else {
		log.Printf("🎵 Regular audio message")
	}

	// Store message in database
	message := &models.Message{
		Time:      info.Timestamp,
		Sender:    info.Sender.String(),
		Content:   fmt.Sprintf("[%s Message]", strings.Title(messageType)),
		IsFromMe:  info.IsFromMe,
		MediaType: messageType,
		Filename:  "",
		ChatJID:   info.Chat.String(),
		MessageID: info.ID,
	}

	if err := c.db.StoreMessage(message); err != nil {
		log.Printf("❌ Failed to store audio message: %v", err)
	} else {
		log.Printf("✅ Audio message stored successfully")
	}

	// Update chat info
	c.updateChatInfo(info.Chat, fmt.Sprintf("[%s Message]", strings.Title(messageType)), info.Timestamp)

	// Process audio/voice message
	c.processAudioMessage(evt, audioMsg, messageType)
}

// handleImageMessage processes image messages
func (c *Client) handleImageMessage(evt *events.Message, imageMsg *waE2E.ImageMessage) {
	info := evt.Info
	caption := imageMsg.GetCaption()

	log.Printf("🖼️ Image message (caption: %s)", caption)

	// Store message in database
	message := &models.Message{
		Time:      info.Timestamp,
		Sender:    info.Sender.String(),
		Content:   caption,
		IsFromMe:  info.IsFromMe,
		MediaType: "image",
		Filename:  "",
		ChatJID:   info.Chat.String(),
		MessageID: info.ID,
	}

	if err := c.db.StoreMessage(message); err != nil {
		log.Printf("❌ Failed to store image message: %v", err)
	} else {
		log.Printf("✅ Image message stored successfully")
	}

	// Update chat info
	c.updateChatInfo(info.Chat, caption, info.Timestamp)

	// TODO: Add custom image processing logic here
	// e.g., OCR, image analysis, etc.
}

// handleVideoMessage processes video messages
func (c *Client) handleVideoMessage(evt *events.Message, videoMsg *waE2E.VideoMessage) {
	info := evt.Info
	caption := videoMsg.GetCaption()

	log.Printf("🎥 Video message (caption: %s)", caption)

	// Store message in database
	message := &models.Message{
		Time:      info.Timestamp,
		Sender:    info.Sender.String(),
		Content:   caption,
		IsFromMe:  info.IsFromMe,
		MediaType: "video",
		Filename:  "",
		ChatJID:   info.Chat.String(),
		MessageID: info.ID,
	}

	if err := c.db.StoreMessage(message); err != nil {
		log.Printf("❌ Failed to store video message: %v", err)
	} else {
		log.Printf("✅ Video message stored successfully")
	}

	// Update chat info
	c.updateChatInfo(info.Chat, caption, info.Timestamp)

	// TODO: Add custom video processing logic here
	// e.g., video analysis, thumbnail generation, etc.
}

// handleDocumentMessage processes document messages
func (c *Client) handleDocumentMessage(evt *events.Message, docMsg *waE2E.DocumentMessage) {
	info := evt.Info
	filename := docMsg.GetFileName()
	caption := docMsg.GetCaption()

	log.Printf("📄 Document message (filename: %s, caption: %s)", filename, caption)

	// Store message in database
	message := &models.Message{
		Time:      info.Timestamp,
		Sender:    info.Sender.String(),
		Content:   caption,
		IsFromMe:  info.IsFromMe,
		MediaType: "document",
		Filename:  filename,
		ChatJID:   info.Chat.String(),
		MessageID: info.ID,
	}

	if err := c.db.StoreMessage(message); err != nil {
		log.Printf("❌ Failed to store document message: %v", err)
	} else {
		log.Printf("✅ Document message stored successfully")
	}

	// Update chat info
	c.updateChatInfo(info.Chat, caption, info.Timestamp)

	// TODO: Add custom document processing logic here
	// e.g., file type detection, content extraction, etc.
}

// handleUnknownMessage processes unknown message types
func (c *Client) handleUnknownMessage(evt *events.Message) {
	info := evt.Info

	log.Printf("❓ Unknown message type from %s", info.Sender.String())

	// Store as unknown message type
	message := &models.Message{
		Time:      info.Timestamp,
		Sender:    info.Sender.String(),
		Content:   "[Unknown Message Type]",
		IsFromMe:  info.IsFromMe,
		MediaType: "unknown",
		Filename:  "",
		ChatJID:   info.Chat.String(),
		MessageID: info.ID,
	}

	if err := c.db.StoreMessage(message); err != nil {
		log.Printf("❌ Failed to store unknown message: %v", err)
	} else {
		log.Printf("✅ Unknown message stored successfully")
	}

	// Update chat info
	c.updateChatInfo(info.Chat, "[Unknown Message Type]", info.Timestamp)
}

// handleReceipt processes message receipts
func (c *Client) handleReceipt(evt *events.Receipt) {
	log.Printf("📋 Receipt received - Type: %s, MessageIDs: %v",
		evt.Type, evt.MessageIDs)
	// Handle read receipts, delivery receipts, etc.
}

// handlePresence processes presence updates
func (c *Client) handlePresence(evt *events.Presence) {
	log.Printf("👤 Presence update - From: %s, LastSeen: %s",
		evt.From.String(), evt.LastSeen.String())
	// Handle online/offline status updates
}

// updateChatInfo updates chat information in the database
func (c *Client) updateChatInfo(chatJID types.JID, lastMessage string, timestamp time.Time) {
	chat := &models.Chat{
		JID:             chatJID.String(),
		LastMessage:     lastMessage,
		LastMessageTime: timestamp,
		IsGroup:         chatJID.Server == types.GroupServer,
	}

	// Try to get chat name
	if chatJID.Server == types.GroupServer {
		// For groups, we might need to get the group info
		// For now, we'll use the JID as the name
		chat.Name = chatJID.String()
	} else {
		// For individual chats, try to get contact name
		ctx := context.Background()
		contact, err := c.client.Store.Contacts.GetContact(ctx, chatJID)
		if err == nil && contact.FullName != "" {
			chat.Name = contact.FullName
		} else {
			chat.Name = chatJID.String()
		}
	}

	c.db.StoreChat(chat)
}

// SearchContacts searches for contacts by name or phone number
func (c *Client) SearchContacts(query string) ([]*models.Contact, error) {
	// First search in our database
	dbContacts, err := c.db.SearchContacts(query)
	if err != nil {
		return nil, err
	}

	// Also search in WhatsApp client's contact list
	ctx := context.Background()
	allContacts, err := c.client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return dbContacts, nil // Return database results if client search fails
	}

	var clientContacts []*models.Contact
	for jid, contact := range allContacts {
		if strings.Contains(strings.ToLower(contact.FullName), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(contact.PushName), strings.ToLower(query)) ||
			strings.Contains(jid.String(), query) {

			clientContact := &models.Contact{
				JID:       jid.String(),
				Name:      contact.FullName,
				PushName:  contact.PushName,
				IsGroup:   false,
				IsBlocked: contact.BusinessName != "",
			}
			clientContacts = append(clientContacts, clientContact)
		}
	}

	// Merge and deduplicate results
	contactMap := make(map[string]*models.Contact)
	for _, contact := range dbContacts {
		contactMap[contact.JID] = contact
	}
	for _, contact := range clientContacts {
		if _, exists := contactMap[contact.JID]; !exists {
			contactMap[contact.JID] = contact
		}
	}

	var result []*models.Contact
	for _, contact := range contactMap {
		result = append(result, contact)
	}

	return result, nil
}

// ListMessages retrieves messages with optional filters
func (c *Client) ListMessages(chatJID string, limit, offset int) ([]*models.Message, error) {
	return c.db.GetMessages(chatJID, limit, offset)
}

// ListChats lists available chats with metadata
func (c *Client) ListChats() ([]*models.Chat, error) {
	return c.db.GetChats()
}

// GetChat gets information about a specific chat
func (c *Client) GetChat(chatJID string) (*models.Chat, error) {
	return c.db.GetChatByJID(chatJID)
}

// GetDirectChatByContact finds a direct chat with a specific contact
func (c *Client) GetDirectChatByContact(contactJID string) (*models.Chat, error) {
	// For direct chats, the chat JID is the same as the contact JID
	return c.db.GetChatByJID(contactJID)
}

// GetContactChats lists all chats involving a specific contact
func (c *Client) GetContactChats(contactJID string) ([]*models.Chat, error) {
	return c.db.GetChatsByContact(contactJID)
}

// GetLastInteraction gets the most recent message with a contact
func (c *Client) GetLastInteraction(contactJID string) (*models.Message, error) {
	return c.db.GetLastMessageWithContact(contactJID)
}

// GetMessageContext retrieves context around a specific message
func (c *Client) GetMessageContext(messageID string, contextSize int) ([]*models.Message, error) {
	// Get the target message
	targetMsg, err := c.db.GetMessageByID(messageID)
	if err != nil {
		return nil, err
	}

	// Get messages before and after
	beforeMsgs, err := c.db.GetMessages(targetMsg.ChatJID, contextSize, 0)
	if err != nil {
		return nil, err
	}

	// Filter to get context around the target message
	var context []*models.Message
	for _, msg := range beforeMsgs {
		if msg.MessageID == messageID {
			// Found the target message, add surrounding context
			startIdx := max(0, len(beforeMsgs)-contextSize)
			endIdx := min(len(beforeMsgs), len(beforeMsgs)+contextSize)
			context = beforeMsgs[startIdx:endIdx]
			break
		}
	}

	return context, nil
}

// SendMessage sends a WhatsApp message to a specified phone number or group JID
func (c *Client) SendMessage(recipient string, message string) error {
	// Ensure client is connected before sending
	ctx := context.Background()
	if err := c.EnsureConnected(ctx); err != nil {
		return fmt.Errorf("failed to ensure connection: %w", err)
	}

	log.Printf("📤 Sending message to %s: %s", recipient, message)

	recipientJID, err := types.ParseJID(recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient JID: %w", err)
	}

	msg := &waE2E.Message{
		Conversation: &message,
	}

	_, err = c.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("✅ Message sent successfully to %s", recipient)
	return nil
}

// SendFile sends a file to a specified recipient
func (c *Client) SendFile(recipient string, filePath string, caption string) error {
	// Ensure client is connected before sending
	ctx := context.Background()
	if err := c.EnsureConnected(ctx); err != nil {
		return fmt.Errorf("failed to ensure connection: %w", err)
	}

	log.Printf("📤 Sending file to %s: %s (caption: %s)", recipient, filePath, caption)

	recipientJID, err := types.ParseJID(recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient JID: %w", err)
	}

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Read file content - for now we'll skip the actual file upload
	// In a real implementation, you would upload the file data
	_, err = io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine media type based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	var mediaType string
	var msg *waE2E.Message

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		mediaType = "image"
		fileSizePtr := uint64(fileInfo.Size())
		msg = &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption:    &caption,
				Mimetype:   &mediaType,
				FileLength: &fileSizePtr,
			},
		}
	case ".mp4", ".avi", ".mov", ".mkv":
		mediaType = "video"
		fileSizePtr := uint64(fileInfo.Size())
		msg = &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption:    &caption,
				Mimetype:   &mediaType,
				FileLength: &fileSizePtr,
			},
		}
	case ".ogg", ".opus":
		mediaType = "audio"
		fileSizePtr := uint64(fileInfo.Size())
		msg = &waE2E.Message{
			AudioMessage: &waE2E.AudioMessage{
				Mimetype:   &mediaType,
				FileLength: &fileSizePtr,
			},
		}
	default:
		// Default to document
		mediaType = "application/octet-stream"
		fileName := fileInfo.Name()
		fileSizePtr := uint64(fileInfo.Size())
		msg = &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Caption:    &caption,
				Mimetype:   &mediaType,
				FileName:   &fileName,
				FileLength: &fileSizePtr,
			},
		}
	}

	_, err = c.client.SendMessage(context.Background(), recipientJID, msg)
	return err
}

// SendAudioMessage sends an audio file as a WhatsApp voice message
func (c *Client) SendAudioMessage(recipient string, filePath string) error {
	// Ensure client is connected before sending
	ctx := context.Background()
	if err := c.EnsureConnected(ctx); err != nil {
		return fmt.Errorf("failed to ensure connection: %w", err)
	}

	log.Printf("📤 Sending audio message to %s: %s", recipient, filePath)

	recipientJID, err := types.ParseJID(recipient)
	if err != nil {
		return fmt.Errorf("invalid recipient JID: %w", err)
	}

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Read file content
	fileData, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	log.Printf("📊 Audio file details - Size: %d bytes, Name: %s", fileInfo.Size(), fileInfo.Name())

	// Determine MIME type based on file extension
	mimeType := getAudioMimeType(filePath)
	log.Printf("🎵 Detected MIME type: %s", mimeType)

	// Get audio duration using ffprobe
	duration, err := getAudioDuration(filePath)
	if err != nil {
		log.Printf("⚠️ Could not determine audio duration: %v", err)
		duration = 0 // Fallback to 0 if we can't determine duration
	} else {
		log.Printf("⏱️ Audio duration: %.2f seconds", duration)
	}

	// Upload media to WhatsApp servers with retry logic
	var uploaded whatsmeow.UploadResponse
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("🔄 Upload attempt %d/%d", attempt, maxRetries)

		uploaded, err = c.client.Upload(ctx, fileData, whatsmeow.MediaAudio)
		if err == nil {
			log.Printf("✅ Audio file uploaded successfully, URL: %s", uploaded.URL)
			break
		}

		log.Printf("❌ Upload attempt %d failed: %v", attempt, err)
		if attempt < maxRetries {
			log.Printf("⏳ Retrying in 2 seconds...")
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		log.Printf("❌ Failed to upload audio file after %d attempts: %v", maxRetries, err)
		return fmt.Errorf("failed to upload audio file after %d attempts: %w", maxRetries, err)
	}

	// Create audio message
	fileSizePtr := uint64(fileInfo.Size())
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           &uploaded.URL,
			Mimetype:      stringPtr(mimeType),
			FileLength:    &fileSizePtr,
			Seconds:       uint32Ptr(uint32(duration)), // Use actual duration
			PTT:           boolPtr(true),               // Mark as voice message
			FileSHA256:    uploaded.FileSHA256,
			FileEncSHA256: uploaded.FileEncSHA256,
			MediaKey:      uploaded.MediaKey,
		},
	}

	_, err = c.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		log.Printf("❌ Failed to send audio message: %v", err)
		return fmt.Errorf("failed to send audio message: %w", err)
	}

	log.Printf("✅ Audio message sent successfully to %s", recipient)
	return nil
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func uint32Ptr(u uint32) *uint32 {
	return &u
}

func boolPtr(b bool) *bool {
	return &b
}

// getAudioMimeType determines the MIME type based on file extension
func getAudioMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".ogg":
		return "audio/ogg" // WhatsApp prefers simple MIME type for voice messages
	case ".opus":
		return "audio/ogg" // Treat opus as ogg for WhatsApp compatibility
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".m4a":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".flac":
		return "audio/flac"
	case ".wma":
		return "audio/x-ms-wma"
	case ".mp4":
		return "audio/mp4"
	case ".3gp":
		return "audio/3gpp"
	case ".amr":
		return "audio/amr"
	default:
		return "audio/ogg" // Default fallback for voice messages
	}
}

// getAudioDuration gets the duration of an audio file using ffprobe
func getAudioDuration(filePath string) (float64, error) {
	// Use ffprobe to get audio duration
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	// Parse JSON output
	var probeResult struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probeResult); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	// Convert duration string to float64
	duration, err := strconv.ParseFloat(probeResult.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

// DownloadMedia downloads media from a WhatsApp message
func (c *Client) DownloadMedia(messageID string) (string, error) {
	// Get message from database
	msg, err := c.db.GetMessageByID(messageID)
	if err != nil {
		return "", fmt.Errorf("message not found: %w", err)
	}

	if msg.MediaType == "" {
		return "", fmt.Errorf("message has no media")
	}

	// For now, return a placeholder path
	// In a real implementation, you would need to store the actual media data
	// and provide a way to retrieve it
	filename := fmt.Sprintf("%s_%s", messageID, msg.Filename)
	filePath := filepath.Join(c.mediaDir, filename)

	return filePath, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// processTextMessage handles text message processing (commands, auto-replies, etc.)
func (c *Client) processTextMessage(evt *events.Message, content string) {
	info := evt.Info
	// Skip processing messages from ourselves
	if info.IsFromMe {
		return
	}

	// Convert to lowercase for command matching
	lowerContent := strings.ToLower(strings.TrimSpace(content))

	// Example command handling
	switch {
	case strings.HasPrefix(lowerContent, "/help"):
		c.sendAutoReply(info.Chat.String(), "Available commands:\n/help - Show this help\n/ping - Test connection\n/time - Get current time")
	case strings.HasPrefix(lowerContent, "/ping"):
		c.sendAutoReply(info.Chat.String(), "Pong! 🏓")
	case strings.HasPrefix(lowerContent, "/time"):
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		c.sendAutoReply(info.Chat.String(), fmt.Sprintf("Current time: %s", currentTime))
	case strings.Contains(lowerContent, "hello") || strings.Contains(lowerContent, "hi"):
		c.sendAutoReply(info.Chat.String(), "Hello! 👋 How can I help you?")
	default:
		// No specific command matched, use LlamaStack to generate response
		log.Printf("💬 Text message processed: %s", content)
		c.processWithLlamaStack(evt, content)
	}
}

// processAudioMessage handles audio/voice message processing
func (c *Client) processAudioMessage(evt *events.Message, audioMsg *waE2E.AudioMessage, messageType string) {
	info := evt.Info
	// Skip processing messages from ourselves
	if info.IsFromMe {
		return
	}

	log.Printf("🎵 Processing %s message from %s", messageType, info.Sender.String())

	// Example: Different handling for voice vs regular audio
	if messageType == "voice" {
		log.Printf("🎤 Voice message received - could trigger transcription or voice commands")
		// TODO: Add voice transcription logic here
		// TODO: Add voice command processing here
	} else {
		log.Printf("🎵 Regular audio message received - could trigger audio analysis")
		// TODO: Add audio analysis logic here
	}

	// Example: Auto-reply for voice messages
	if messageType == "voice" {
		c.sendAutoReply(info.Chat.String(), "🎤 Voice message received! I heard you loud and clear.")
	}
}

// sendAutoReply sends an automatic reply to a chat
func (c *Client) sendAutoReply(chatJID string, message string) {
	ctx := context.Background()
	if err := c.EnsureConnected(ctx); err != nil {
		log.Printf("❌ Failed to ensure connection for auto-reply: %v", err)
		return
	}

	recipientJID, err := types.ParseJID(chatJID)
	if err != nil {
		log.Printf("❌ Invalid chat JID for auto-reply: %v", err)
		return
	}

	msg := &waE2E.Message{
		Conversation: &message,
	}

	_, err = c.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		log.Printf("❌ Failed to send auto-reply: %v", err)
	} else {
		log.Printf("✅ Auto-reply sent: %s", message)
	}
}

// createLlamaStackClient creates and configures a LlamaStack client
func (c *Client) createLlamaStackClient() (*LlamaStackAgent, error) {
	// Get LlamaStack configuration from environment variables
	llamastackBaseURL := os.Getenv("LLAMASTACK_BASE_URL")
	if llamastackBaseURL == "" {
		llamastackBaseURL = "http://ragathon-team-3-ragathon-team-3.apps.llama-rag-pool-b84hp.aws.rh-ods.com/"
	}

	whatsappMCPSSEURL := os.Getenv("WHATSAPP_MCP_SSE_URL")
	if whatsappMCPSSEURL == "" {
		whatsappMCPSSEURL = "http://localhost:8080/sse"
	}

	llamastackModel := os.Getenv("LLAMASTACK_MODEL")
	if llamastackModel == "" {
		llamastackModel = "vllm-inference/llama-3-2-3b-instruct"
	}

	llamastackTemperature := 0.7
	if tempStr := os.Getenv("LLAMASTACK_TEMPERATURE"); tempStr != "" {
		if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
			llamastackTemperature = temp
		}
	}

	llamastackMaxTokens := 200
	if tokensStr := os.Getenv("LLAMASTACK_MAX_TOKENS"); tokensStr != "" {
		if tokens, err := strconv.Atoi(tokensStr); err == nil {
			llamastackMaxTokens = tokens
		}
	}

	log.Printf("🔗 LlamaStack Base URL: %s", llamastackBaseURL)
	log.Printf("🔗 WhatsApp MCP SSE URL: %s", whatsappMCPSSEURL)
	log.Printf("🤖 Using model: %s", llamastackModel)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 120 * time.Second,
	}

	// Create LlamaStack client
	client := &LlamaStackClient{
		BaseURL:     llamastackBaseURL,
		HTTPClient:  httpClient,
		Model:       llamastackModel,
		Temperature: llamastackTemperature,
		MaxTokens:   llamastackMaxTokens,
	}

	log.Printf("✅ LlamaStack client created successfully")
	log.Printf("🔗 Connected to LlamaStack service at: %s", llamastackBaseURL)

	// Test if LlamaStack service is accessible
	if err := c.testLlamaStackConnection(client); err != nil {
		log.Printf("⚠️ LlamaStack service test failed: %v", err)
		log.Printf("⚠️ Continuing anyway - toolgroup registration might still work")
	}

	// Try to register the WhatsApp MCP toolgroup (optional - continue even if it fails)
	toolgroupID := "mcp::whatsapp-mcp-auto-reply"
	err := c.registerToolgroup(client, toolgroupID, whatsappMCPSSEURL)
	if err != nil {
		log.Printf("⚠️ Failed to register toolgroup (continuing without MCP tools): %v", err)
		log.Printf("ℹ️ LlamaStack will work in basic mode without WhatsApp MCP tools")
		// Continue without MCP tools - LlamaStack can still work without them
		toolgroupID = ""
	} else {
		log.Printf("✅ WhatsApp MCP toolgroup registered: %s", toolgroupID)
	}

	// Create agent
	var tools []string
	if toolgroupID != "" {
		tools = []string{toolgroupID}
	}

	agent := &LlamaStackAgent{
		Client:    client,
		Model:     llamastackModel,
		SessionID: fmt.Sprintf("whatsapp_auto_reply_session_%d", time.Now().Unix()),
		Tools:     tools,
	}

	log.Printf("✅ Agent created successfully")
	log.Printf("🤖 Using AI model: %s", llamastackModel)
	log.Printf("📱 Created session: %s", agent.SessionID)

	return agent, nil
}

// registerToolgroup registers a toolgroup with LlamaStack
func (c *Client) registerToolgroup(client *LlamaStackClient, toolgroupID, mcpEndpoint string) error {
	// First, try to unregister any existing toolgroup
	c.unregisterToolgroup(client, toolgroupID)

	// Register the new toolgroup
	toolgroup := LlamaStackToolgroup{
		Identifier: toolgroupID,
		ProviderID: "model-context-protocol",
		MCPEndpoint: map[string]string{
			"uri": mcpEndpoint,
		},
	}

	jsonData, err := json.Marshal(toolgroup)
	if err != nil {
		return fmt.Errorf("failed to marshal toolgroup: %w", err)
	}

	// Try different possible endpoints for toolgroup registration
	possibleEndpoints := []string{
		fmt.Sprintf("%s/toolgroups", client.BaseURL),
		fmt.Sprintf("%s/api/toolgroups", client.BaseURL),
		fmt.Sprintf("%s/v1/toolgroups", client.BaseURL),
		fmt.Sprintf("%s/tools", client.BaseURL),
		fmt.Sprintf("%s/api/tools", client.BaseURL),
	}

	var lastErr error
	for _, url := range possibleEndpoints {
		log.Printf("🔗 Trying toolgroup registration at URL: %s", url)
		log.Printf("📤 Toolgroup data: %s", string(jsonData))

		req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := client.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to register toolgroup: %w", err)
			continue
		}

		log.Printf("📊 Toolgroup registration response status: %d", resp.StatusCode)

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			resp.Body.Close()
			log.Printf("✅ Toolgroup registered successfully at: %s", url)
			return nil
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("❌ Toolgroup registration failed at %s with status %d: %s", url, resp.StatusCode, string(body))
		lastErr = fmt.Errorf("toolgroup registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	return lastErr
}

// testLlamaStackConnection tests if the LlamaStack service is accessible
func (c *Client) testLlamaStackConnection(client *LlamaStackClient) error {
	// Try to access the root endpoint or health endpoint
	testURLs := []string{
		fmt.Sprintf("%s/", client.BaseURL),
		fmt.Sprintf("%s/health", client.BaseURL),
		fmt.Sprintf("%s/api/health", client.BaseURL),
		fmt.Sprintf("%s/v1/health", client.BaseURL),
	}

	for _, url := range testURLs {
		resp, err := client.HTTPClient.Get(url)
		if err != nil {
			log.Printf("⚠️ Failed to test %s: %v", url, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
			log.Printf("✅ LlamaStack service accessible at: %s (status: %d)", url, resp.StatusCode)
			return nil
		}

		log.Printf("⚠️ LlamaStack service returned status %d at: %s", resp.StatusCode, url)
	}

	return fmt.Errorf("could not establish connection to LlamaStack service")
}

// unregisterToolgroup unregisters a toolgroup from LlamaStack
func (c *Client) unregisterToolgroup(client *LlamaStackClient, toolgroupID string) error {
	// List existing toolgroups
	url := fmt.Sprintf("%s/toolgroups", client.BaseURL)
	resp, err := client.HTTPClient.Get(url)
	if err != nil {
		log.Printf("⚠️ Failed to list toolgroups: %v", err)
		return nil // Non-critical error
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil // Non-critical error
	}

	var toolgroups []LlamaStackToolgroup
	if err := json.NewDecoder(resp.Body).Decode(&toolgroups); err != nil {
		log.Printf("⚠️ Failed to decode toolgroups: %v", err)
		return nil // Non-critical error
	}

	// Find and unregister matching toolgroups
	for _, tg := range toolgroups {
		if strings.Contains(tg.Identifier, toolgroupID) {
			log.Printf("🗑️ Unregistering existing toolgroup: %s", tg.Identifier)

			deleteURL := fmt.Sprintf("%s/toolgroups/%s", client.BaseURL, tg.Identifier)
			req, err := http.NewRequest("DELETE", deleteURL, nil)
			if err != nil {
				log.Printf("⚠️ Failed to create delete request: %v", err)
				continue
			}

			deleteResp, err := client.HTTPClient.Do(req)
			if err != nil {
				log.Printf("⚠️ Failed to delete toolgroup: %v", err)
				continue
			}
			deleteResp.Body.Close()
		}
	}

	return nil
}

// generateResponse generates a response using LlamaStack
func (c *Client) generateResponse(agent *LlamaStackAgent, userMessage string) (string, error) {
	// Create system message based on available tools
	systemMessage := "You are a helpful WhatsApp assistant. "
	if len(agent.Tools) > 0 {
		systemMessage += "You can use the WhatsApp MCP tools to:\n- Search and manage WhatsApp contacts\n- List and read WhatsApp messages\n- Manage WhatsApp chats\n- Send messages and files\n- Get message context and interactions\n\nAlways be helpful and provide clear information about WhatsApp operations."
	} else {
		systemMessage += "You are here to help users with general questions and provide helpful responses. Keep your answers concise and friendly."
	}

	// Create messages for the conversation
	messages := []Message{
		{
			Role:    "system",
			Content: systemMessage,
		},
		{
			Role:    "user",
			Content: userMessage,
		},
	}

	// Create request
	request := LlamaStackRequest{
		Model:       agent.Model,
		Messages:    messages,
		Temperature: agent.Client.Temperature,
		MaxTokens:   agent.Client.MaxTokens,
		Tools:       agent.Tools,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request to LlamaStack
	url := fmt.Sprintf("%s/chat/completions", agent.Client.BaseURL)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := agent.Client.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response LlamaStackResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices received")
	}

	return response.Choices[0].Message.Content, nil
}

// processWithLlamaStack processes a text message using LlamaStack
func (c *Client) processWithLlamaStack(evt *events.Message, content string) {
	info := evt.Info

	log.Printf("🤖 Processing message with LlamaStack: %s", content)

	// Create or get LlamaStack client
	if c.llamastackClient == nil {
		agent, err := c.createLlamaStackClient()
		if err != nil {
			log.Printf("❌ Failed to create LlamaStack client: %v", err)
			c.sendAutoReply(info.Chat.String(), "Sorry, I'm having trouble connecting to my AI assistant right now. Please try again later.")
			return
		}
		c.llamastackClient = agent.Client
	}

	// Create agent for this request
	agent := &LlamaStackAgent{
		Client:    c.llamastackClient,
		Model:     c.llamastackClient.Model,
		SessionID: fmt.Sprintf("whatsapp_session_%d", time.Now().Unix()),
		Tools:     []string{"mcp::whatsapp-mcp-auto-reply"},
	}

	// Generate response using LlamaStack
	response, err := c.generateResponse(agent, content)
	if err != nil {
		log.Printf("❌ Failed to generate LlamaStack response: %v", err)
		c.sendAutoReply(info.Chat.String(), "Sorry, I'm having trouble generating a response right now. Please try again later.")
		return
	}

	log.Printf("🤖 LlamaStack response: %s", response)

	// Send the generated response
	c.sendAutoReply(info.Chat.String(), response)
}
