package mcp

import (
	"context"
	"fmt"

	"github.com/fredcamaral/gomcp-sdk"
	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/fredcamaral/gomcp-sdk/server"
	"github.com/fredcamaral/gomcp-sdk/transport"

	"whatsapp-go-mcp/whatsapp"
)

// WhatsAppMCPServer implements the MCP server for WhatsApp functionality
type WhatsAppMCPServer struct {
	client *whatsapp.Client
	server *server.Server
}

// NewWhatsAppMCPServer creates a new MCP server instance
func NewWhatsAppMCPServer(client *whatsapp.Client) *WhatsAppMCPServer {
	server := mcp.NewServer("whatsapp-mcp-server", "1.0.0")

	mcpServer := &WhatsAppMCPServer{
		client: client,
		server: server,
	}

	mcpServer.registerTools()
	mcpServer.registerResources()

	return mcpServer
}

// registerTools registers all MCP tools
func (s *WhatsAppMCPServer) registerTools() {
	// Search contacts tool
	searchContactsTool := mcp.NewTool(
		"search_contacts",
		"Search for contacts by name or phone number",
		mcp.ObjectSchema("Search parameters", map[string]interface{}{
			"query": mcp.StringParam("Search query for contacts", true),
		}, []string{"query"}),
	)
	s.server.AddTool(searchContactsTool, mcp.ToolHandlerFunc(s.searchContacts))

	// List messages tool
	listMessagesTool := mcp.NewTool(
		"list_messages",
		"Retrieve messages with optional filters and context",
		mcp.ObjectSchema("Message list parameters", map[string]interface{}{
			"chat_jid": mcp.StringParam("Chat JID to retrieve messages from", true),
			"limit":    mcp.NumberParam("Maximum number of messages to retrieve", false),
			"offset":   mcp.NumberParam("Number of messages to skip", false),
		}, []string{"chat_jid"}),
	)
	s.server.AddTool(listMessagesTool, mcp.ToolHandlerFunc(s.listMessages))

	// List chats tool
	listChatsTool := mcp.NewTool(
		"list_chats",
		"List available chats with metadata",
		mcp.ObjectSchema("Chat list parameters", map[string]interface{}{}, []string{}),
	)
	s.server.AddTool(listChatsTool, mcp.ToolHandlerFunc(s.listChats))

	// Get chat tool
	getChatTool := mcp.NewTool(
		"get_chat",
		"Get information about a specific chat",
		mcp.ObjectSchema("Chat parameters", map[string]interface{}{
			"chat_jid": mcp.StringParam("Chat JID to get information for", true),
		}, []string{"chat_jid"}),
	)
	s.server.AddTool(getChatTool, mcp.ToolHandlerFunc(s.getChat))

	// Get direct chat by contact tool
	getDirectChatTool := mcp.NewTool(
		"get_direct_chat_by_contact",
		"Find a direct chat with a specific contact",
		mcp.ObjectSchema("Direct chat parameters", map[string]interface{}{
			"contact_jid": mcp.StringParam("Contact JID to find direct chat for", true),
		}, []string{"contact_jid"}),
	)
	s.server.AddTool(getDirectChatTool, mcp.ToolHandlerFunc(s.getDirectChatByContact))

	// Get contact chats tool
	getContactChatsTool := mcp.NewTool(
		"get_contact_chats",
		"List all chats involving a specific contact",
		mcp.ObjectSchema("Contact chats parameters", map[string]interface{}{
			"contact_jid": mcp.StringParam("Contact JID to find chats for", true),
		}, []string{"contact_jid"}),
	)
	s.server.AddTool(getContactChatsTool, mcp.ToolHandlerFunc(s.getContactChats))

	// Get last interaction tool
	getLastInteractionTool := mcp.NewTool(
		"get_last_interaction",
		"Get the most recent message with a contact",
		mcp.ObjectSchema("Last interaction parameters", map[string]interface{}{
			"contact_jid": mcp.StringParam("Contact JID to get last interaction for", true),
		}, []string{"contact_jid"}),
	)
	s.server.AddTool(getLastInteractionTool, mcp.ToolHandlerFunc(s.getLastInteraction))

	// Get message context tool
	getMessageContextTool := mcp.NewTool(
		"get_message_context",
		"Retrieve context around a specific message",
		mcp.ObjectSchema("Message context parameters", map[string]interface{}{
			"message_id":   mcp.StringParam("Message ID to get context for", true),
			"context_size": mcp.NumberParam("Number of messages before and after to include", false),
		}, []string{"message_id"}),
	)
	s.server.AddTool(getMessageContextTool, mcp.ToolHandlerFunc(s.getMessageContext))

	// Send message tool
	sendMessageTool := mcp.NewTool(
		"send_message",
		"Send a WhatsApp message to a specified recipient",
		mcp.ObjectSchema("Send message parameters", map[string]interface{}{
			"recipient": mcp.StringParam("Recipient JID (phone number or group JID)", true),
			"message":   mcp.StringParam("Message content to send", true),
		}, []string{"recipient", "message"}),
	)
	s.server.AddTool(sendMessageTool, mcp.ToolHandlerFunc(s.sendMessage))

	// Send file tool
	sendFileTool := mcp.NewTool(
		"send_file",
		"Send a file to a specified recipient",
		mcp.ObjectSchema("Send file parameters", map[string]interface{}{
			"recipient": mcp.StringParam("Recipient JID (phone number or group JID)", true),
			"file_path": mcp.StringParam("Path to the file to send", true),
			"caption":   mcp.StringParam("Optional caption for the file", false),
		}, []string{"recipient", "file_path"}),
	)
	s.server.AddTool(sendFileTool, mcp.ToolHandlerFunc(s.sendFile))

	// Send audio message tool
	sendAudioMessageTool := mcp.NewTool(
		"send_audio_message",
		"Send an audio file as a WhatsApp voice message",
		mcp.ObjectSchema("Send audio message parameters", map[string]interface{}{
			"recipient": mcp.StringParam("Recipient JID (phone number or group JID)", true),
			"file_path": mcp.StringParam("Path to the audio file (.ogg opus format recommended)", true),
		}, []string{"recipient", "file_path"}),
	)
	s.server.AddTool(sendAudioMessageTool, mcp.ToolHandlerFunc(s.sendAudioMessage))

	// Download media tool
	downloadMediaTool := mcp.NewTool(
		"download_media",
		"Download media from a WhatsApp message",
		mcp.ObjectSchema("Download media parameters", map[string]interface{}{
			"message_id": mcp.StringParam("Message ID containing the media", true),
		}, []string{"message_id"}),
	)
	s.server.AddTool(downloadMediaTool, mcp.ToolHandlerFunc(s.downloadMedia))
}

// registerResources registers MCP resources
func (s *WhatsAppMCPServer) registerResources() {
	// Register contacts as a resource
	contactsResource := mcp.NewResource("whatsapp://contacts", "contacts", "WhatsApp contacts", "application/json")
	s.server.AddResource(contactsResource, mcp.ResourceHandlerFunc(s.getContactsResource))

	// Register chats as a resource
	chatsResource := mcp.NewResource("whatsapp://chats", "chats", "WhatsApp chats", "application/json")
	s.server.AddResource(chatsResource, mcp.ResourceHandlerFunc(s.getChatsResource))
}

// Tool handlers
func (s *WhatsAppMCPServer) searchContacts(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}

	contacts, err := s.client.SearchContacts(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}

	return contacts, nil
}

func (s *WhatsAppMCPServer) listMessages(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	chatJID, ok := args["chat_jid"].(string)
	if !ok {
		return nil, fmt.Errorf("chat_jid parameter is required")
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	messages, err := s.client.ListMessages(chatJID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	return messages, nil
}

func (s *WhatsAppMCPServer) listChats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	chats, err := s.client.ListChats()
	if err != nil {
		return nil, fmt.Errorf("failed to list chats: %w", err)
	}

	return chats, nil
}

func (s *WhatsAppMCPServer) getChat(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	chatJID, ok := args["chat_jid"].(string)
	if !ok {
		return nil, fmt.Errorf("chat_jid parameter is required")
	}

	chat, err := s.client.GetChat(chatJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	return chat, nil
}

func (s *WhatsAppMCPServer) getDirectChatByContact(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	contactJID, ok := args["contact_jid"].(string)
	if !ok {
		return nil, fmt.Errorf("contact_jid parameter is required")
	}

	chat, err := s.client.GetDirectChatByContact(contactJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get direct chat: %w", err)
	}

	return chat, nil
}

func (s *WhatsAppMCPServer) getContactChats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	contactJID, ok := args["contact_jid"].(string)
	if !ok {
		return nil, fmt.Errorf("contact_jid parameter is required")
	}

	chats, err := s.client.GetContactChats(contactJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact chats: %w", err)
	}

	return chats, nil
}

func (s *WhatsAppMCPServer) getLastInteraction(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	contactJID, ok := args["contact_jid"].(string)
	if !ok {
		return nil, fmt.Errorf("contact_jid parameter is required")
	}

	message, err := s.client.GetLastInteraction(contactJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last interaction: %w", err)
	}

	return message, nil
}

func (s *WhatsAppMCPServer) getMessageContext(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	messageID, ok := args["message_id"].(string)
	if !ok {
		return nil, fmt.Errorf("message_id parameter is required")
	}

	contextSize := 10
	if cs, ok := args["context_size"].(float64); ok {
		contextSize = int(cs)
	}

	messages, err := s.client.GetMessageContext(messageID, contextSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get message context: %w", err)
	}

	return messages, nil
}

func (s *WhatsAppMCPServer) sendMessage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	recipient, ok := args["recipient"].(string)
	if !ok {
		return nil, fmt.Errorf("recipient parameter is required")
	}

	message, ok := args["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message parameter is required")
	}

	err := s.client.SendMessage(recipient, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return map[string]string{"status": "sent"}, nil
}

func (s *WhatsAppMCPServer) sendFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	recipient, ok := args["recipient"].(string)
	if !ok {
		return nil, fmt.Errorf("recipient parameter is required")
	}

	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter is required")
	}

	caption := ""
	if c, ok := args["caption"].(string); ok {
		caption = c
	}

	err := s.client.SendFile(recipient, filePath, caption)
	if err != nil {
		return nil, fmt.Errorf("failed to send file: %w", err)
	}

	return map[string]string{"status": "sent"}, nil
}

func (s *WhatsAppMCPServer) sendAudioMessage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	recipient, ok := args["recipient"].(string)
	if !ok {
		return nil, fmt.Errorf("recipient parameter is required")
	}

	filePath, ok := args["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter is required")
	}

	err := s.client.SendAudioMessage(recipient, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to send audio message: %w", err)
	}

	return map[string]string{"status": "sent"}, nil
}

func (s *WhatsAppMCPServer) downloadMedia(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	messageID, ok := args["message_id"].(string)
	if !ok {
		return nil, fmt.Errorf("message_id parameter is required")
	}

	filePath, err := s.client.DownloadMedia(messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to download media: %w", err)
	}

	return map[string]string{"file_path": filePath}, nil
}

// Resource handlers
func (s *WhatsAppMCPServer) getContactsResource(ctx context.Context, uri string) ([]protocol.Content, error) {
	// For now, return all contacts. In a real implementation, you might want to filter based on URI
	contacts, err := s.client.SearchContacts("")
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	// Convert contacts to MCP content format
	content := make([]protocol.Content, len(contacts))
	for i, contact := range contacts {
		content[i] = protocol.Content{
			Type: "text",
			Text: fmt.Sprintf("Contact: %s (%s)", contact.Name, contact.JID),
		}
	}

	return content, nil
}

func (s *WhatsAppMCPServer) getChatsResource(ctx context.Context, uri string) ([]protocol.Content, error) {
	chats, err := s.client.ListChats()
	if err != nil {
		return nil, fmt.Errorf("failed to get chats: %w", err)
	}

	// Convert chats to MCP content format
	content := make([]protocol.Content, len(chats))
	for i, chat := range chats {
		content[i] = protocol.Content{
			Type: "text",
			Text: fmt.Sprintf("Chat: %s (%s) - Last message: %s", chat.Name, chat.JID, chat.LastMessage),
		}
	}

	return content, nil
}

func (s *WhatsAppMCPServer) SearchContacts(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.searchContacts(ctx, args)
}

func (s *WhatsAppMCPServer) ListMessages(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.listMessages(ctx, args)
}

func (s *WhatsAppMCPServer) ListChats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.listChats(ctx, args)
}

func (s *WhatsAppMCPServer) GetChat(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.getChat(ctx, args)
}

func (s *WhatsAppMCPServer) GetDirectChatByContact(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.getDirectChatByContact(ctx, args)
}

func (s *WhatsAppMCPServer) GetContactChats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.getContactChats(ctx, args)
}

func (s *WhatsAppMCPServer) GetLastInteraction(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.getLastInteraction(ctx, args)
}

func (s *WhatsAppMCPServer) GetMessageContext(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.getMessageContext(ctx, args)
}

func (s *WhatsAppMCPServer) SendMessage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.sendMessage(ctx, args)
}

func (s *WhatsAppMCPServer) SendFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.sendFile(ctx, args)
}

func (s *WhatsAppMCPServer) SendAudioMessage(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.sendAudioMessage(ctx, args)
}

func (s *WhatsAppMCPServer) DownloadMedia(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return s.downloadMedia(ctx, args)
}

// GetServer returns the underlying MCP server
func (s *WhatsAppMCPServer) GetServer() *server.Server {
	return s.server
}

// Start starts the MCP server with SSE transport
func (s *WhatsAppMCPServer) Start(ctx context.Context, sseTransport *transport.SSETransport) error {
	return sseTransport.Start(ctx, s.server)
}

// Stop stops the MCP server
func (s *WhatsAppMCPServer) Stop() error {
	// The server doesn't have a Stop method, so we just return nil
	// The transport will handle cleanup
	return nil
}
