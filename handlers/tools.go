package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	whatsappmcp "whatsapp-go-mcp/mcp"
)

// ExecuteToolRequest represents a request to execute a specific MCP tool
type ExecuteToolRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
}

// ExecuteToolResponse represents the response from tool execution
type ExecuteToolResponse struct {
	Success  bool        `json:"success" example:"true"`
	ToolName string      `json:"tool_name" example:"search_contacts"`
	Result   interface{} `json:"result"`
	Error    string      `json:"error,omitempty" example:""`
}

// HandleExecuteTool handles dynamic tool execution requests
// @Summary Execute a specific MCP tool
// @Description Execute any available MCP tool with the provided parameters
// @Tags API
// @Accept json
// @Produce json
// @Param tool_name path string true "Name of the tool to execute"
// @Param request body ExecuteToolRequest true "Tool execution request"
// @Success 200 {object} ExecuteToolResponse "Tool execution result"
// @Failure 404 {object} map[string]string "Tool not found"
// @Failure 500 {object} map[string]string "Tool execution error"
// @Router /tools/{tool_name}/execute [post]
func HandleExecuteTool(w http.ResponseWriter, r *http.Request, mcpServer *whatsappmcp.WhatsAppMCPServer) {
	// Extract tool name from URL path
	vars := mux.Vars(r)
	toolName := vars["tool_name"]

	var req ExecuteToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Map tool names to their corresponding functions
	toolFunctions := map[string]func(map[string]interface{}) (interface{}, error){
		"search_contacts": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.SearchContacts(context.Background(), params)
		},
		"list_messages": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.ListMessages(context.Background(), params)
		},
		"list_chats": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.ListChats(context.Background(), params)
		},
		"get_chat": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.GetChat(context.Background(), params)
		},
		"get_direct_chat_by_contact": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.GetDirectChatByContact(context.Background(), params)
		},
		"get_contact_chats": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.GetContactChats(context.Background(), params)
		},
		"get_last_interaction": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.GetLastInteraction(context.Background(), params)
		},
		"get_message_context": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.GetMessageContext(context.Background(), params)
		},
		"send_message": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.SendMessage(context.Background(), params)
		},
		"send_file": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.SendFile(context.Background(), params)
		},
		"send_audio_message": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.SendAudioMessage(context.Background(), params)
		},
		"download_media": func(params map[string]interface{}) (interface{}, error) {
			return mcpServer.DownloadMedia(context.Background(), params)
		},
	}

	// Check if tool exists
	toolFunc, exists := toolFunctions[toolName]
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Tool '%s' not found", toolName),
		})
		return
	}

	// Execute the tool
	result, err := toolFunc(req.Parameters)
	if err != nil {
		log.Printf("Error executing tool %s: %v", toolName, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := ExecuteToolResponse{
			Success:  false,
			ToolName: toolName,
			Error:    err.Error(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return successful result
	w.Header().Set("Content-Type", "application/json")
	response := ExecuteToolResponse{
		Success:  true,
		ToolName: toolName,
		Result:   result,
	}
	json.NewEncoder(w).Encode(response)
}
