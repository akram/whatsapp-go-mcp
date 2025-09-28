package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	llamastack "github.com/llamastack/llama-stack-client-go"
	"github.com/llamastack/llama-stack-client-go/option"
	"github.com/llamastack/llama-stack-client-go/packages/param"
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
	client         *whatsmeow.Client
	db             *models.Database
	deviceStore    *store.Device
	eventHandlerID uint32
	mediaDir       string
	ttsUrl         string
}

// NewClient creates a new WhatsApp client
func NewClient(dbPath, mediaDir, ttsUrl string) (*Client, error) {
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
		ttsUrl:      ttsUrl,
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
		Content:   fmt.Sprintf("[%s Message]", strings.ToUpper(messageType[:1])+messageType[1:]),
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
	c.updateChatInfo(info.Chat, fmt.Sprintf("[%s Message]", strings.ToUpper(messageType[:1])+messageType[1:]), info.Timestamp)

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

	resp, err := c.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Store the sent message in the database
	sentMessage := &models.Message{
		Time:      time.Now(),
		Sender:    c.client.Store.ID.String(), // Our own JID
		Content:   message,
		IsFromMe:  true,
		MediaType: "text",
		Filename:  "",
		ChatJID:   recipientJID.String(),
		MessageID: resp.ID, // Use the actual message ID from WhatsApp response
	}

	if err := c.db.StoreMessage(sentMessage); err != nil {
		log.Printf("⚠️ Failed to store sent message in database: %v", err)
	} else {
		log.Printf("✅ Sent message stored in database")
	}

	// Update chat info
	c.updateChatInfo(recipientJID, message, time.Now())

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
		// Estimate duration (rough estimate: assume 1 second per 16KB for opus)
		estimatedDuration := float64(fileInfo.Size()) / 16000.0
		if estimatedDuration < 1 {
			estimatedDuration = 1
		}
		duration = estimatedDuration
		log.Printf("⏱️ Using estimated duration: %.2f seconds", duration)
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
			URL:               &uploaded.URL,
			Mimetype:          stringPtr("audio/ogg; codecs=opus"), // Use proper MIME type for voice messages
			FileLength:        &fileSizePtr,
			Seconds:           uint32Ptr(uint32(duration)), // Use actual duration
			PTT:               boolPtr(true),               // Mark as voice message
			FileSHA256:        uploaded.FileSHA256,
			FileEncSHA256:     uploaded.FileEncSHA256,
			MediaKey:          uploaded.MediaKey,
			DirectPath:        &uploaded.DirectPath,        // Add missing DirectPath
			MediaKeyTimestamp: int64Ptr(time.Now().Unix()), // Add missing MediaKeyTimestamp
		},
	}

	resp, err := c.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		log.Printf("❌ Failed to send audio message: %v", err)
		return fmt.Errorf("failed to send audio message: %w", err)
	}

	// Store the sent audio message in the database
	audioMessage := &models.Message{
		Time:      time.Now(),
		Sender:    c.client.Store.ID.String(), // Our own JID
		Content:   "[Voice Message]",          // Placeholder content for audio messages
		IsFromMe:  true,
		MediaType: "voice",
		Filename:  filepath.Base(filePath),
		ChatJID:   recipientJID.String(),
		MessageID: resp.ID, // Use the actual message ID from WhatsApp response
	}

	if err := c.db.StoreMessage(audioMessage); err != nil {
		log.Printf("⚠️ Failed to store sent audio message in database: %v", err)
	} else {
		log.Printf("✅ Sent audio message stored in database")
	}

	// Update chat info
	c.updateChatInfo(recipientJID, "[Voice Message]", time.Now())

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

func int64Ptr(i int64) *int64 {
	return &i
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

	// Different handling for voice vs regular audio
	if messageType == "voice" {
		log.Printf("🎤 Voice message received - processing with AI agent")
		c.processVoiceMessage(evt, audioMsg)
	} else {
		log.Printf("🎵 Regular audio message received - could trigger audio analysis")
		// TODO: Add audio analysis logic here
	}
}

// processVoiceMessage handles the complete voice message processing pipeline
func (c *Client) processVoiceMessage(evt *events.Message, audioMsg *waE2E.AudioMessage) {
	info := evt.Info

	log.Printf("🎤 Starting voice message processing pipeline")

	// Step 0: Set voice recording presence to indicate we're processing
	if err := c.setVoiceRecordingPresence(info.Chat.String()); err != nil {
		log.Printf("⚠️ Failed to set voice recording presence: %v", err)
	}

	// Step 1: Download the voice message
	audioFilePath, err := c.downloadVoiceMessage(evt, audioMsg)
	if err != nil {
		log.Printf("❌ Failed to download voice message: %v", err)
		c.clearChatPresence(info.Chat.String()) // Clear presence on error
		c.sendAutoReply(info.Chat.String(), "Sorry, I couldn't download your voice message. Please try again.")
		return
	}
	defer os.Remove(audioFilePath) // Clean up downloaded file

	log.Printf("✅ Voice message downloaded to: %s", audioFilePath)

	// Step 2: Convert speech to text
	transcribedText, err := c.speechToText(audioFilePath)
	if err != nil {
		log.Printf("❌ Failed to transcribe voice message: %v", err)
		c.clearChatPresence(info.Chat.String()) // Clear presence on error
		c.sendAutoReply(info.Chat.String(), "Sorry, I couldn't understand your voice message. Please try speaking more clearly.")
		return
	}

	log.Printf("✅ Voice transcribed: %s", transcribedText)

	// Step 3: Process with AI agent
	responseText, err := c.processWithLlamaStackAgent(transcribedText)
	if err != nil {
		log.Printf("❌ Failed to process with AI agent: %v", err)
		c.clearChatPresence(info.Chat.String()) // Clear presence on error
		c.sendAutoReply(info.Chat.String(), "Sorry, I'm having trouble processing your request right now. Please try again later.")
		return
	}

	log.Printf("✅ AI agent response: %s", responseText)

	// Step 4: Convert response to speech
	responseAudioPath, err := c.textToSpeech(responseText)
	if err != nil {
		log.Printf("❌ Failed to convert response to speech: %v", err)
		// Fallback to text response
		c.clearChatPresence(info.Chat.String()) // Clear presence on error
		c.sendAutoReply(info.Chat.String(), responseText)
		return
	}
	defer os.Remove(responseAudioPath) // Clean up generated audio file

	log.Printf("✅ Response converted to speech: %s", responseAudioPath)
	log.Printf("🔍 DEBUG: Generated audio file exists: %v", fileExists(responseAudioPath))
	if fileExists(responseAudioPath) {
		if stat, err := os.Stat(responseAudioPath); err == nil {
			log.Printf("🔍 DEBUG: Audio file size: %d bytes", stat.Size())
		}
	}

	// Step 5: Send audio response
	err = c.SendAudioMessage(info.Chat.String(), responseAudioPath)
	if err != nil {
		log.Printf("❌ Failed to send audio response: %v", err)
		// Fallback to text response
		c.clearChatPresence(info.Chat.String()) // Clear presence on error
		c.sendAutoReply(info.Chat.String(), responseText)
		return
	}

	// Step 6: Also send text response for debugging purposes
	log.Printf("🔍 DEBUG: Sending text response for debugging")
	debugText := fmt.Sprintf("🔍 DEBUG - Transcribed: \"%s\"\n\n🤖 AI Response: \"%s\"", transcribedText, responseText)
	c.sendAutoReply(info.Chat.String(), debugText)

	// Step 7: Clear voice recording presence
	if err := c.clearChatPresence(info.Chat.String()); err != nil {
		log.Printf("⚠️ Failed to clear chat presence: %v", err)
	}

	log.Printf("✅ Voice response and debug text sent successfully")
}

// downloadVoiceMessage downloads a voice message from WhatsApp
func (c *Client) downloadVoiceMessage(evt *events.Message, audioMsg *waE2E.AudioMessage) (string, error) {
	info := evt.Info

	log.Printf("📥 Downloading voice message from %s", info.Sender.String())

	// Create media directory if it doesn't exist
	if err := os.MkdirAll(c.mediaDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	// Generate filename for the downloaded audio
	filename := fmt.Sprintf("voice_%s_%s.ogg", info.ID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(c.mediaDir, filename)

	// Download the media using WhatsApp client
	ctx := context.Background()
	data, err := c.client.Download(ctx, audioMsg)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	// Write the downloaded data to file
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("✅ Voice message downloaded successfully: %s", filePath)
	return filePath, nil
}

// speechToText converts audio file to text using speech recognition
func (c *Client) speechToText(audioFilePath string) (string, error) {
	log.Printf("🎙️ Converting speech to text: %s", audioFilePath)

	// Use OpenAI Whisper API for speech-to-text conversion
	// You can also use local solutions like whisper.cpp or other STT services
	transcribedText, err := c.transcribeWithWhisper(audioFilePath)
	if err != nil {
		return "", fmt.Errorf("speech-to-text conversion failed: %w", err)
	}

	log.Printf("✅ Speech transcribed: %s", transcribedText)
	return transcribedText, nil
}

// transcribeWithWhisper uses OpenAI Whisper API for transcription
func (c *Client) transcribeWithWhisper(audioFilePath string) (string, error) {
	// For now, we'll use a simple implementation
	// In production, you would integrate with OpenAI Whisper API or local whisper

	// Check if we have OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Fallback to local whisper if available
		return c.transcribeWithLocalWhisper(audioFilePath)
	}

	// TODO: Implement OpenAI Whisper API integration
	// For now, return a placeholder
	return "Voice message transcribed (placeholder)", nil
}

// transcribeWithLocalWhisper uses local whisper installation for transcription
func (c *Client) transcribeWithLocalWhisper(audioFilePath string) (string, error) {
	log.Printf("🎙️ Using local whisper for transcription")

	// Check if whisper is available
	cmd := exec.Command("whisper", "--help")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("whisper not found, please install whisper: %w", err)
	}

	// Create output directory for transcription
	outputDir := filepath.Join(c.mediaDir, "transcriptions")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create transcription directory: %w", err)
	}

	// Run whisper transcription with smaller, faster model
	// Set a timeout for whisper transcription (5 minutes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd = exec.CommandContext(ctx, "whisper", audioFilePath, "--model", "base", "--output_dir", outputDir, "--output_format", "txt")

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("whisper transcription timed out after 5 minutes")
		}
		return "", fmt.Errorf("whisper transcription failed: %w", err)
	}

	// Whisper creates output file with same name as input (without extension) + .txt
	inputBaseName := strings.TrimSuffix(filepath.Base(audioFilePath), filepath.Ext(audioFilePath))
	outputFile := filepath.Join(outputDir, inputBaseName+".txt")

	// Read the transcription result
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription file: %w", err)
	}

	// Clean up transcription file
	os.Remove(outputFile)

	return strings.TrimSpace(string(content)), nil
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// textToSpeech converts text to speech audio file
func (c *Client) textToSpeech(text string) (string, error) {
	log.Printf("🔊 Converting text to speech: %s", text)

	// Create output directory for TTS
	outputDir := filepath.Join(c.mediaDir, "tts")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create TTS directory: %w", err)
	}

	// Generate filename for TTS output
	filename := fmt.Sprintf("tts_%d.ogg", time.Now().Unix())
	outputPath := filepath.Join(outputDir, filename)

	// Use local TTS service
	err := c.generateSpeechWithLocalService(text, outputPath)
	if err != nil {
		return "", fmt.Errorf("text-to-speech conversion failed: %w", err)
	}

	log.Printf("✅ Text converted to speech: %s", outputPath)
	return outputPath, nil
}

// generateSpeechWithLocalService uses local TTS service for text-to-speech conversion
func (c *Client) generateSpeechWithLocalService(text, outputPath string) error {
	log.Printf("🔊 Using local TTS service for generation")

	// Create a temporary WAV file first
	tempWavPath := outputPath + ".wav"
	defer os.Remove(tempWavPath) // Clean up temporary WAV file

	// Use curl to call the TTS service
	cmd := exec.Command("curl", "-X", "POST", "-F", fmt.Sprintf("text=%s", text), c.ttsUrl, "--output", tempWavPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("local TTS service call failed: %w", err)
	}

	// Check if the WAV file was created and has content
	if stat, err := os.Stat(tempWavPath); err != nil || stat.Size() == 0 {
		return fmt.Errorf("TTS service did not generate valid audio file")
	}

	// Convert WAV to OGG using ffmpeg
	cmd = exec.Command("ffmpeg", "-y", "-i", tempWavPath, "-c:a", "libopus", "-b:a", "64k", "-ar", "48000", "-ac", "1", outputPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	return nil
}

// processWithLlamaStackAgent processes transcribed text with LlamaStack agent
func (c *Client) processWithLlamaStackAgent(transcribedText string) (string, error) {
	log.Printf("🤖 Processing transcribed text with LlamaStack agent: %s", transcribedText)

	// Create LlamaStack client
	client, modelID, err := c.createLlamaStackClient()
	if err != nil {
		return "", fmt.Errorf("failed to create LlamaStack client: %w", err)
	}

	// Create agent with tools and instructions
	agent, err := c.createLlamaStackAgent(client, modelID)
	if err != nil {
		return "", fmt.Errorf("failed to create LlamaStack agent: %w", err)
	}

	// Generate response using the agent
	response, err := c.generateAgentResponse(client, agent.AgentID, transcribedText)
	if err != nil {
		return "", fmt.Errorf("failed to generate agent response: %w", err)
	}

	return response, nil
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

	resp, err := c.client.SendMessage(ctx, recipientJID, msg)
	if err != nil {
		log.Printf("❌ Failed to send auto-reply: %v", err)
		return
	}

	// Store the auto-reply message in the database
	autoReplyMessage := &models.Message{
		Time:      time.Now(),
		Sender:    c.client.Store.ID.String(), // Our own JID
		Content:   message,
		IsFromMe:  true,
		MediaType: "text",
		Filename:  "",
		ChatJID:   chatJID,
		MessageID: resp.ID, // Use the actual message ID from WhatsApp response
	}

	if err := c.db.StoreMessage(autoReplyMessage); err != nil {
		log.Printf("⚠️ Failed to store auto-reply message in database: %v", err)
	} else {
		log.Printf("✅ Auto-reply message stored in database")
	}

	// Update chat info
	c.updateChatInfo(recipientJID, message, time.Now())

	log.Printf("✅ Auto-reply sent: %s", message)
}

// createLlamaStackClient creates and configures a LlamaStack client
func (c *Client) createLlamaStackClient() (llamastack.Client, string, error) {
	log.Printf("🔗 Creating LlamaStack client")

	// Get LlamaStack configuration from environment variables
	baseURL := os.Getenv("LLAMASTACK_BASE_URL")
	if baseURL == "" {
		baseURL = "http://ragathon-team-1-ragathon-team-1.apps.llama-rag-pool-b84hp.aws.rh-ods.com"
	}

	apiKey := os.Getenv("LLAMASTACK_API_KEY")
	modelID := os.Getenv("LLAMASTACK_MODEL")
	if modelID == "" {
		modelID = "vllm-inference/llama-4-scout-17b-16e-w4a16"
	}

	// Ensure model ID has provider prefix if not already present
	if !strings.Contains(modelID, "/") {
		modelID = "vllm-inference/" + modelID
	}

	log.Printf("🔗 LlamaStack Base URL: %s", baseURL)
	log.Printf("🤖 LlamaStack Model: %s", modelID)
	if apiKey != "" {
		log.Printf("🔑 Using API Key: %s", apiKey[:min(len(apiKey), 8)]+"...")
	} else {
		log.Printf("⚠️ No API Key provided")
	}

	// Create the official LlamaStack client with configuration
	client := llamastack.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)

	log.Printf("✅ LlamaStack client created successfully")
	return client, modelID, nil
}

// listAvailableToolGroups lists all available tool groups on the LlamaStack server
func (c *Client) listAvailableToolGroups(client llamastack.Client) error {
	log.Printf("🔍 Listing available tool groups...")

	toolgroups, err := client.Toolgroups.List(context.TODO())
	if err != nil {
		log.Printf("❌ Failed to list tool groups: %v", err)
		return err
	}

	log.Printf("📋 Available tool groups: %+v", toolgroups)

	return nil
}

// createLlamaStackAgent creates an agent with tools and instructions
func (c *Client) createLlamaStackAgent(client llamastack.Client, modelID string) (*llamastack.AgentNewResponse, error) {
	log.Printf("🤖 Creating LlamaStack agent with model: %s", modelID)

	// List available tool groups for debugging
	c.listAvailableToolGroups(client)

	// Banking assistant instructions with available tools
	instructions := `You are a helpful banking assistant. You MUST use the available tools to retrieve information immediately when needed. that includes using knowledge_search and mcp tools.
For write operation as updates do only knowledge_search , do not try to update the data.
IMPORTANT: When a user asks for their personal information (like address, account details, statements, etc.), you MUST call the appropriate mcp tool with phone number "+353 85 148 0072" to retrieve their data. Do not just describe what you would do - actually execute the tool call.

For banking knowledge questions, use the knowledge_search tool to find relevant information from the banking knowledge base.

Always make the necessary tool calls first, then provide the user with the actual retrieved information. Never show tool calls as text - execute them and use the results to answer the user's question.

If you need to search for current banking information, use the knowledge search tool. If you need user-specific data, use the MCP tools with the phone number +353 85 148 0072.`

	// Create agent configuration with available tools
	agentConfig := llamastack.AgentConfigParam{
		Instructions: instructions,
		Model:        modelID, // Use the model from environment (vllm-inference/llama-3-2-3b-instruct)
		Name:         llamastack.String("WhatsApp Banking Assistant"),
		Toolgroups: []llamastack.AgentConfigToolgroupUnionParam{
			// Knowledge search tool with vector database
			{
				OfAgentToolGroupWithArgs: &llamastack.AgentConfigToolgroupAgentToolGroupWithArgsParam{
					Name: "builtin::rag/knowledge_search",
					Args: map[string]llamastack.AgentConfigToolgroupAgentToolGroupWithArgsArgUnionParam{
						"vector_db_ids": {
							OfAnyArray: []any{"vs_1f1dd1b7-49ad-4ceb-8e8d-f0bf9afe2179"},
						},
					},
				},
			},
			// WhatsApp MCP tools for user information (using only one to avoid conflicts)
			{
				OfString: llamastack.String("mcp::redbank-financials"),
			},
		},
		ToolConfig: llamastack.AgentConfigToolConfigParam{
			ToolChoice: "auto", // Use "auto" to let the agent decide when to use tools
		},
	}

	// Create the agent
	agent, err := client.Agents.New(context.TODO(), llamastack.AgentNewParams{
		AgentConfig: agentConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	log.Printf("✅ Agent created successfully with ID: %s", agent.AgentID)
	return agent, nil
}

// processWithLlamaStack processes a text message using LlamaStack agent
func (c *Client) processWithLlamaStack(evt *events.Message, content string) {
	info := evt.Info

	log.Printf("🤖 Processing message with LlamaStack agent: %s", content)

	// Create LlamaStack client
	client, modelID, err := c.createLlamaStackClient()
	if err != nil {
		log.Printf("❌ Failed to create LlamaStack client: %v", err)
		c.sendAutoReply(info.Chat.String(), "Sorry, I'm having trouble connecting to my AI assistant right now. Please try again later.")
		return
	}

	// Create agent with tools and instructions
	agent, err := c.createLlamaStackAgent(client, modelID)
	if err != nil {
		log.Printf("❌ Failed to create LlamaStack agent: %v", err)
		c.sendAutoReply(info.Chat.String(), "Sorry, I'm having trouble setting up my AI assistant right now. Please try again later.")
		return
	}

	log.Printf("✅ Agent created: %s", agent.AgentID)

	// Generate response using the agent
	response, err := c.generateAgentResponse(client, agent.AgentID, content)
	if err != nil {
		log.Printf("❌ Failed to generate agent response: %v", err)
		// Fall back to simple response
		response = c.generateFallbackResponse(content)
		log.Printf("🔄 Using fallback response: %s", response)
	} else {
		log.Printf("🤖 LlamaStack agent response: %s", response)
	}

	// Send the generated response
	c.sendAutoReply(info.Chat.String(), response)
}

// generateAgentResponse generates a response using the LlamaStack agent
func (c *Client) generateAgentResponse(client llamastack.Client, agentID, userMessage string) (string, error) {
	log.Printf("🤖 Generating agent response using agent: %s", agentID)
	log.Printf("💬 User message: %s", userMessage)

	// Create a new session for the agent
	session, err := client.Agents.Session.New(context.TODO(), agentID, llamastack.AgentSessionNewParams{
		SessionName: "WhatsApp Banking Session",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create agent session: %w", err)
	}

	log.Printf("✅ Agent session created: %s", session.SessionID)

	// Create a streaming turn with the user message
	stream := client.Agents.Turn.NewStreaming(context.TODO(), session.SessionID, llamastack.AgentTurnNewParams{
		AgentID: agentID,
		Messages: []llamastack.AgentTurnNewParamsMessageUnion{
			{
				OfUserMessage: &llamastack.UserMessageParam{
					Content: llamastack.InterleavedContentUnionParam{
						OfString: param.Opt[string]{Value: userMessage},
					},
				},
			},
		},
	})

	log.Printf("✅ Agent streaming turn created")

	// Process the streaming response
	var finalResponse string
	var turnID string
	var hasError bool
	var errorMessage string

	for stream.Next() {
		chunk := stream.Current()

		// Log the chunk type for debugging
		log.Printf("📦 Received chunk: %+v", chunk)

		// Check for errors in the chunk
		if errorField, exists := chunk.JSON.ExtraFields["error"]; exists && errorField.Valid() {
			hasError = true
			errorMessage = fmt.Sprintf("Agent error: %v", errorField)
			log.Printf("❌ %s", errorMessage)
			break
		}

		// Handle different types of streaming events
		event := chunk.Event
		switch event.Payload.EventType {
		case "turn_start":
			if event.Payload.TurnID != "" {
				turnID = event.Payload.TurnID
				log.Printf("✅ Turn started: %s", turnID)
			}
		case "step_complete":
			step := event.Payload.StepDetails
			log.Printf("🔧 Step completed - Type: %s, StepID: %s", step.StepType, step.StepID)

			if step.StepType == "inference" && step.ModelResponse.Role == "assistant" {
				// Extract the response content
				if step.ModelResponse.Content.OfString != "" {
					finalResponse = step.ModelResponse.Content.OfString
					log.Printf("🤖 Received assistant response: %s", finalResponse)
				} else if len(step.ModelResponse.Content.OfInterleavedContentItemArray) > 0 {
					for _, contentItem := range step.ModelResponse.Content.OfInterleavedContentItemArray {
						if contentItem.Text != "" {
							finalResponse = contentItem.Text
							log.Printf("🤖 Received assistant response: %s", finalResponse)
							break
						}
					}
				}
			} else if step.StepType == "tool_execution" {
				log.Printf("🔧 Tool execution completed - StepID: %s", step.StepID)
				// Log tool responses for debugging
				if len(step.ToolResponses) > 0 {
					for i, toolResp := range step.ToolResponses {
						log.Printf("🔧 Tool response %d: %+v", i, toolResp)
					}
				}
			}
		case "step_progress":
			delta := event.Payload.Delta
			log.Printf("🔄 Step progress - Type: %s", delta.Type)

			if delta.Type == "tool_call" && delta.ToolCall.ToolName != "" {
				log.Printf("🔧 Tool call in progress: %s with args: %+v", delta.ToolCall.ToolName, delta.ToolCall.Arguments)
			} else if delta.Type == "text" && delta.Text != "" {
				log.Printf("📝 Text progress: %s", delta.Text)
			}
		case "turn_complete":
			log.Printf("✅ Turn completed")
			goto streamComplete
		}
	}

streamComplete:

	if err := stream.Err(); err != nil {
		return "", fmt.Errorf("streaming error: %w", err)
	}

	if hasError {
		return "", fmt.Errorf("%s", errorMessage)
	}

	if finalResponse == "" {
		return "", fmt.Errorf("no response received from agent")
	}

	log.Printf("✅ Agent response generated successfully")
	return finalResponse, nil
}

// generateFallbackResponse generates a simple fallback response when LlamaStack is unavailable
func (c *Client) generateFallbackResponse(content string) string {
	lowerContent := strings.ToLower(strings.TrimSpace(content))

	// Simple keyword-based responses
	switch {
	case strings.Contains(lowerContent, "hello") || strings.Contains(lowerContent, "hi"):
		return "Hello! 👋 I'm here to help you with WhatsApp. How can I assist you today?"
	case strings.Contains(lowerContent, "help"):
		return "I can help you with WhatsApp operations like:\n• Searching contacts\n• Managing messages\n• Sending files\n• Getting chat information\n\nWhat would you like to do?"
	case strings.Contains(lowerContent, "thank"):
		return "You're welcome! 😊 Is there anything else I can help you with?"
	case strings.Contains(lowerContent, "bye") || strings.Contains(lowerContent, "goodbye"):
		return "Goodbye! 👋 Feel free to reach out anytime you need help with WhatsApp."
	case strings.Contains(lowerContent, "time"):
		return fmt.Sprintf("The current time is: %s", time.Now().Format("2006-01-02 15:04:05"))
	case strings.Contains(lowerContent, "weather"):
		return "I don't have access to weather information right now, but I can help you with WhatsApp-related tasks!"
	case strings.Contains(lowerContent, "how are you"):
		return "I'm doing well, thank you for asking! 😊 I'm here and ready to help you with WhatsApp operations."
	default:
		return "I received your message! While my AI assistant is temporarily unavailable, I'm still here to help you with WhatsApp operations. You can ask me about contacts, messages, or other WhatsApp features."
	}
}

// setVoiceRecordingPresence sets the chat presence to indicate voice recording
func (c *Client) setVoiceRecordingPresence(chatJID string) error {
	ctx := context.Background()
	if err := c.EnsureConnected(ctx); err != nil {
		return fmt.Errorf("failed to ensure connection for presence: %w", err)
	}

	recipientJID, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID for presence: %w", err)
	}

	log.Printf("🎤 Setting voice recording presence for %s", chatJID)
	err = c.client.SendChatPresence(recipientJID, types.ChatPresenceComposing, types.ChatPresenceMediaAudio)
	if err != nil {
		log.Printf("❌ Failed to set voice recording presence: %v", err)
		return fmt.Errorf("failed to set voice recording presence: %w", err)
	}

	log.Printf("✅ Voice recording presence set successfully")
	return nil
}

// clearChatPresence clears the chat presence indicator
func (c *Client) clearChatPresence(chatJID string) error {
	ctx := context.Background()
	if err := c.EnsureConnected(ctx); err != nil {
		return fmt.Errorf("failed to ensure connection for presence: %w", err)
	}

	recipientJID, err := types.ParseJID(chatJID)
	if err != nil {
		return fmt.Errorf("invalid chat JID for presence: %w", err)
	}

	log.Printf("🔄 Clearing chat presence for %s", chatJID)
	err = c.client.SendChatPresence(recipientJID, types.ChatPresencePaused, "")
	if err != nil {
		log.Printf("❌ Failed to clear chat presence: %v", err)
		return fmt.Errorf("failed to clear chat presence: %w", err)
	}

	log.Printf("✅ Chat presence cleared successfully")
	return nil
}
