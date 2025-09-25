// Package main implements a WhatsApp MCP server with OpenAPI documentation
// @title WhatsApp MCP Server API
// @version 1.0
// @description A WhatsApp MCP (Model Context Protocol) server providing WhatsApp functionality
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "whatsapp-go-mcp/docs" // Import generated docs
	"whatsapp-go-mcp/handlers"
	whatsappmcp "whatsapp-go-mcp/mcp"
	"whatsapp-go-mcp/tools"
	"whatsapp-go-mcp/whatsapp"
)

// handleSSERequest handles SSE requests for MCP communication
// @Summary Server-Sent Events endpoint for MCP communication
// @Description Establishes a Server-Sent Events connection for real-time MCP communication
// @Tags MCP
// @Accept text/event-stream
// @Produce text/event-stream
// @Success 200 {string} string "SSE connection established"
// @Router /events [get]
// @Router /sse [get]
func handleSSERequest(w http.ResponseWriter, r *http.Request, mcpServer *whatsappmcp.WhatsAppMCPServer) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: {\"status\":\"connected\"}\n\n")

	// Flush the response
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Keep connection alive with heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// Send heartbeat
			fmt.Fprintf(w, "event: heartbeat\n")
			fmt.Fprintf(w, "data: {\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// ListMessagesRequest represents a request to list messages
type ListMessagesRequest struct {
	ChatJID string `json:"chat_jid" example:"1234567890@s.whatsapp.net"`
	Limit   int    `json:"limit" example:"50"`
	Offset  int    `json:"offset" example:"0"`
}

// SearchContactsRequest represents a request to search contacts
type SearchContactsRequest struct {
	Query string `json:"query" example:"John"`
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Recipient string `json:"recipient" example:"1234567890@s.whatsapp.net"`
	Message   string `json:"message" example:"Hello World"`
}

// handleTools handles tools discovery requests
// @Summary List available MCP tools
// @Description Returns a list of all available MCP tools with their schemas
// @Tags MCP
// @Accept json
// @Produce json
// @Success 200 {object} tools.ToolsResponse "List of available tools"
// @Router /tools [get]
func handleTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	toolsList := tools.GetTools()
	response := tools.ToolsResponse{Tools: toolsList}
	json.NewEncoder(w).Encode(response)
}

// handleListMessages handles direct HTTP requests to list messages
// @Summary List messages from a chat
// @Description Retrieve messages from a specific WhatsApp chat
// @Tags API
// @Accept json
// @Produce json
// @Param request body ListMessagesRequest true "List messages request"
// @Success 200 {object} map[string]interface{} "List of messages"
// @Router /api/list-messages [post]
func handleListMessages(w http.ResponseWriter, r *http.Request) {
	var req ListMessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 50
	}

	// For now, return a mock response since we need the MCP server instance
	// In a real implementation, you'd call the MCP server's list_messages tool
	response := map[string]interface{}{
		"chat_jid": req.ChatJID,
		"limit":    req.Limit,
		"offset":   req.Offset,
		"messages": []map[string]interface{}{
			{
				"id":      "msg_1",
				"content": "Sample message 1",
				"sender":  "1234567890@s.whatsapp.net",
				"time":    "2025-09-25T00:00:00Z",
			},
			{
				"id":      "msg_2",
				"content": "Sample message 2",
				"sender":  "0987654321@s.whatsapp.net",
				"time":    "2025-09-25T00:01:00Z",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSearchContacts handles direct HTTP requests to search contacts
// @Summary Search for contacts
// @Description Search for WhatsApp contacts by name or phone number
// @Tags API
// @Accept json
// @Produce json
// @Param request body SearchContactsRequest true "Search contacts request"
// @Success 200 {object} map[string]interface{} "List of contacts"
// @Router /api/search-contacts [post]
func handleSearchContacts(w http.ResponseWriter, r *http.Request) {
	var req SearchContactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Mock response - in real implementation, call MCP server
	response := map[string]interface{}{
		"query": req.Query,
		"contacts": []map[string]interface{}{
			{
				"jid":  "1234567890@s.whatsapp.net",
				"name": "John Doe",
			},
			{
				"jid":  "0987654321@s.whatsapp.net",
				"name": "Jane Smith",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSendMessage handles direct HTTP requests to send messages
// @Summary Send a WhatsApp message
// @Description Send a message to a WhatsApp contact or group
// @Tags API
// @Accept json
// @Produce json
// @Param request body SendMessageRequest true "Send message request"
// @Success 200 {object} map[string]interface{} "Message sent confirmation"
// @Router /api/send-message [post]
func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Mock response - in real implementation, call MCP server
	response := map[string]interface{}{
		"status":    "sent",
		"recipient": req.Recipient,
		"message":   req.Message,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Get configuration from environment variables
	dbPath := os.Getenv("WHATSAPP_DB_PATH")
	if dbPath == "" {
		dbPath = "./whatsapp.db"
	}

	mediaDir := os.Getenv("WHATSAPP_MEDIA_DIR")
	if mediaDir == "" {
		mediaDir = "./media"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create WhatsApp client
	client, err := whatsapp.NewClient(dbPath, mediaDir)
	if err != nil {
		log.Fatalf("Failed to create WhatsApp client: %v", err)
	}
	defer client.Close()

	// Connect to WhatsApp
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to WhatsApp: %v", err)
	}

	// Create MCP server
	mcpServer := whatsappmcp.NewWhatsAppMCPServer(client)

	// Create router with gorilla/mux
	router := mux.NewRouter()

	// Add routes
	router.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		handleSSERequest(w, r, mcpServer)
	}).Methods("GET")

	router.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		handleSSERequest(w, r, mcpServer)
	}).Methods("GET")

	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET")

	router.HandleFunc("/tools", handleTools).Methods("GET")

	// Dynamic tool execution endpoint
	router.HandleFunc("/tools/{tool_name}/execute", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleExecuteTool(w, r, mcpServer)
	}).Methods("POST")

	// API endpoints for direct HTTP access to MCP tools
	router.HandleFunc("/api/list-messages", handleListMessages).Methods("POST")
	router.HandleFunc("/api/search-contacts", handleSearchContacts).Methods("POST")
	router.HandleFunc("/api/send-message", handleSendMessage).Methods("POST")
	router.HandleFunc("/api/send-voice-note", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleSendVoiceNote(w, r, mcpServer)
	}).Methods("POST")

	// Python-style API endpoint
	router.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleSend(w, r, mcpServer)
	}).Methods("POST")

	// Swagger documentation
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start HTTP server with all routes
	go func() {
		log.Printf("Starting MCP server with SSE transport on port %s", port)
		log.Printf("Available endpoints:")
		log.Printf("🔌 - GET /events - SSE endpoint for MCP communication")
		log.Printf("🔌 - GET /sse - Alias for /events")
		log.Printf("🔌 - GET /health - Health check")
		log.Printf("🔌 - GET /tools - List available MCP tools")
		log.Printf("🔌 - POST /tools/{tool_name}/execute - Execute any MCP tool dynamically")
		log.Printf("🔌 - POST /api/list-messages - List messages from a chat")
		log.Printf("🔌 - POST /api/search-contacts - Search for contacts")
		log.Printf("🔌 - POST /api/send-message - Send a WhatsApp message")
		log.Printf("🔌 - POST /api/send-voice-note - Send a voice note (multipart/form-data)")
		log.Printf("🔌 - POST /send - Send voice message (Python-style API with media_path)")
		log.Printf("🔌 - GET /swagger/ - OpenAPI documentation")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop MCP server
	if err := mcpServer.Stop(); err != nil {
		log.Printf("Error stopping MCP server: %v", err)
	}

	// Stop HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
