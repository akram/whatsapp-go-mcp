// Package main implements a WhatsApp server with OpenAPI documentation
// @title WhatsApp Server API
// @version 1.0
// @description A WhatsApp server providing WhatsApp functionality
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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"whatsapp-go-mcp/handlers"
	"whatsapp-go-mcp/whatsapp"

	"github.com/gorilla/mux"
)

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

// handleListMessages handles direct HTTP requests to list messages
// @Summary List messages from a chat
// @Description Retrieve messages from a specific WhatsApp chat
// @Tags API
// @Accept json
// @Produce json
// @Param request body ListMessagesRequest true "List messages request"
// @Success 200 {object} map[string]interface{} "List of messages"
// @Router /api/list-messages [post]
func handleListMessages(w http.ResponseWriter, r *http.Request, client *whatsapp.Client) {
	var req ListMessagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 50
	}

	// Call the WhatsApp client directly
	result, err := client.ListMessages(req.ChatJID, req.Limit, req.Offset)
	if err != nil {
		log.Printf("❌ Failed to list messages: %v", err)
		http.Error(w, "Failed to list messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
func handleSearchContacts(w http.ResponseWriter, r *http.Request, client *whatsapp.Client) {
	var req SearchContactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Call the WhatsApp client directly
	result, err := client.SearchContacts(req.Query)
	if err != nil {
		log.Printf("❌ Failed to search contacts: %v", err)
		http.Error(w, "Failed to search contacts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
func handleSendMessage(w http.ResponseWriter, r *http.Request, client *whatsapp.Client) {
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Call the WhatsApp client directly
	err := client.SendMessage(req.Recipient, req.Message)
	if err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		http.Error(w, "Failed to send message", http.StatusInternalServerError)
		return
	}

	// Return success response
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
	ttsUrl := os.Getenv("TTS_URL")
	if ttsUrl == "" {
		ttsUrl = "http://localhost:8001/text-to-speech"
	}
	sttUrl := os.Getenv("STT_URL")
	client, err := whatsapp.NewClient(dbPath, mediaDir, ttsUrl, sttUrl)
	if err != nil {
		log.Fatalf("Failed to create WhatsApp client: %v", err)
	}
	defer client.Close()

	// Connect to WhatsApp
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to WhatsApp: %v", err)
	}

	// Create router with gorilla/mux
	router := mux.NewRouter()

	// Add routes
	router.HandleFunc("/health", handlers.HandleHealth).Methods("GET")

	// API endpoints for direct HTTP access to WhatsApp functionality
	router.HandleFunc("/api/list-messages", func(w http.ResponseWriter, r *http.Request) {
		handleListMessages(w, r, client)
	}).Methods("POST")
	router.HandleFunc("/api/search-contacts", func(w http.ResponseWriter, r *http.Request) {
		handleSearchContacts(w, r, client)
	}).Methods("POST")
	router.HandleFunc("/api/send-message", func(w http.ResponseWriter, r *http.Request) {
		handleSendMessage(w, r, client)
	}).Methods("POST")
	router.HandleFunc("/api/send-voice-note", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleSendVoiceNote(w, r, client)
	}).Methods("POST")

	// Python-style API endpoint
	router.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleSend(w, r, client)
	}).Methods("POST")

	// OpenAPI 3.0 documentation
	router.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/openapi.json")
	}).Methods("GET")
	router.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		http.ServeFile(w, r, "./docs/openapi.yaml")
	}).Methods("GET")
	router.HandleFunc("/openapi", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/openapi-ui.html")
	}).Methods("GET")

	// Redirect swagger-ui to openapi for compatibility
	router.HandleFunc("/swagger-ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/openapi", http.StatusMovedPermanently)
	}).Methods("GET")
	router.HandleFunc("/swagger-ui/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/openapi", http.StatusMovedPermanently)
	}).Methods("GET")

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start HTTP server with all routes
	go func() {
		log.Printf("Starting WhatsApp server on port %s", port)
		log.Printf("Available endpoints:")
		log.Printf("🔌 - GET /health - Health check")
		log.Printf("🔌 - POST /api/list-messages - List messages from a chat")
		log.Printf("🔌 - POST /api/search-contacts - Search for contacts")
		log.Printf("🔌 - POST /api/send-message - Send a WhatsApp message")
		log.Printf("🔌 - POST /api/send-voice-note - Send a voice note (multipart/form-data)")
		log.Printf("🔌 - POST /send - Send voice message (Python-style API with media_path)")
		log.Printf("🔌 - GET /openapi - OpenAPI 3.0 documentation (Interactive UI)")
		log.Printf("🔌 - GET /openapi.json - OpenAPI 3.0 specification (JSON)")
		log.Printf("🔌 - GET /openapi.yaml - OpenAPI 3.0 specification (YAML)")
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

	// Stop HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
