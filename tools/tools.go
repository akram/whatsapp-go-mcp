package tools

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name" example:"search_contacts"`
	Description string                 `json:"description" example:"Search for contacts by name or phone number"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolsResponse represents the tools list response
type ToolsResponse struct {
	Tools []Tool `json:"tools"`
}

// GetTools returns the list of available MCP tools
func GetTools() []Tool {
	return []Tool{
		{
			Name:        "search_contacts",
			Description: "Search for contacts by name or phone number",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query for contacts",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "list_messages",
			Description: "Retrieve messages with optional filters and context",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_jid": map[string]interface{}{
						"type":        "string",
						"description": "Chat JID to retrieve messages from",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of messages to retrieve",
						"default":     50,
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Number of messages to skip",
						"default":     0,
					},
				},
				"required": []string{"chat_jid"},
			},
		},
		{
			Name:        "list_chats",
			Description: "List available chats with metadata",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_chat",
			Description: "Get information about a specific chat",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_jid": map[string]interface{}{
						"type":        "string",
						"description": "Chat JID to get information for",
					},
				},
				"required": []string{"chat_jid"},
			},
		},
		{
			Name:        "get_direct_chat_by_contact",
			Description: "Find a direct chat with a specific contact",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"contact_jid": map[string]interface{}{
						"type":        "string",
						"description": "Contact JID to find direct chat for",
					},
				},
				"required": []string{"contact_jid"},
			},
		},
		{
			Name:        "get_contact_chats",
			Description: "List all chats involving a specific contact",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"contact_jid": map[string]interface{}{
						"type":        "string",
						"description": "Contact JID to find chats for",
					},
				},
				"required": []string{"contact_jid"},
			},
		},
		{
			Name:        "get_last_interaction",
			Description: "Get the most recent message with a contact",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"contact_jid": map[string]interface{}{
						"type":        "string",
						"description": "Contact JID to get last interaction for",
					},
				},
				"required": []string{"contact_jid"},
			},
		},
		{
			Name:        "get_message_context",
			Description: "Retrieve context around a specific message",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message_id": map[string]interface{}{
						"type":        "string",
						"description": "Message ID to get context for",
					},
					"context_size": map[string]interface{}{
						"type":        "integer",
						"description": "Number of messages before and after to include",
						"default":     10,
					},
				},
				"required": []string{"message_id"},
			},
		},
		{
			Name:        "send_message",
			Description: "Send a WhatsApp message to a specified recipient",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recipient": map[string]interface{}{
						"type":        "string",
						"description": "Recipient JID (phone number or group JID)",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message content to send",
					},
				},
				"required": []string{"recipient", "message"},
			},
		},
		{
			Name:        "send_file",
			Description: "Send a file to a specified recipient",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recipient": map[string]interface{}{
						"type":        "string",
						"description": "Recipient JID (phone number or group JID)",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to send",
					},
					"caption": map[string]interface{}{
						"type":        "string",
						"description": "Optional caption for the file",
					},
				},
				"required": []string{"recipient", "file_path"},
			},
		},
		{
			Name:        "send_audio_message",
			Description: "Send an audio file as a WhatsApp voice message",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recipient": map[string]interface{}{
						"type":        "string",
						"description": "Recipient JID (phone number or group JID)",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the audio file (.ogg opus format recommended)",
					},
				},
				"required": []string{"recipient", "file_path"},
			},
		},
		{
			Name:        "download_media",
			Description: "Download media from a WhatsApp message",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message_id": map[string]interface{}{
						"type":        "string",
						"description": "Message ID containing the media",
					},
				},
				"required": []string{"message_id"},
			},
		},
	}
}
