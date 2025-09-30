package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"whatsapp-go-mcp/whatsapp"
)

// SendVoiceNoteRequest represents a request to send a voice note
type SendVoiceNoteRequest struct {
	Recipient string `json:"recipient" example:"1234567890@s.whatsapp.net"`
}

// SendVoiceNoteResponse represents the response from voice note sending
type SendVoiceNoteResponse struct {
	Success   bool   `json:"success" example:"true"`
	Recipient string `json:"recipient" example:"1234567890@s.whatsapp.net"`
	Filename  string `json:"filename" example:"voice_note.ogg"`
	Timestamp string `json:"timestamp" example:"2025-09-25T00:50:40+02:00"`
	Error     string `json:"error,omitempty" example:""`
}

// HandleSendVoiceNote handles voice note upload and sending
// @Summary Send a voice note via WhatsApp
// @Description Upload and send an audio file as a WhatsApp voice message
// @Tags API
// @Accept multipart/form-data
// @Produce json
// @Param recipient formData string true "Recipient JID"
// @Param file formData file true "Audio file (.ogg opus format)"
// @Success 200 {object} SendVoiceNoteResponse "Voice note sent successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/send-voice-note [post]
func HandleSendVoiceNote(w http.ResponseWriter, r *http.Request, client *whatsapp.Client) {
	// Parse multipart form (max 32MB)
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		log.Printf("❌ Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get recipient from form
	recipient := r.FormValue("recipient")
	if recipient == "" {
		log.Printf("❌ Missing recipient parameter")
		http.Error(w, "Missing recipient parameter", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("❌ Failed to get uploaded file: %v", err)
		http.Error(w, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	if !isValidAudioFile(header.Filename) {
		log.Printf("❌ Invalid audio file type: %s", header.Filename)
		http.Error(w, "Invalid audio file type. Supported formats: .ogg, .opus, .mp3, .wav, .m4a, .aac, .flac, .wma, .mp4, .3gp, .amr", http.StatusBadRequest)
		return
	}

	// Create temporary file
	tempDir := "./temp"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Printf("❌ Failed to create temp directory: %v", err)
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}

	tempFile, err := os.CreateTemp(tempDir, "voice_note_*.ogg")
	if err != nil {
		log.Printf("❌ Failed to create temp file: %v", err)
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file
	defer tempFile.Close()

	// Copy uploaded file to temp file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		log.Printf("❌ Failed to copy file: %v", err)
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
		return
	}

	// Send voice note
	log.Printf("📤 Sending voice note to %s: %s", recipient, header.Filename)
	err = client.SendAudioMessage(recipient, tempFile.Name())
	if err != nil {
		log.Printf("❌ Failed to send voice note: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := SendVoiceNoteResponse{
			Success:   false,
			Recipient: recipient,
			Filename:  header.Filename,
			Error:     err.Error(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("✅ Voice note sent successfully")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	response := SendVoiceNoteResponse{
		Success:   true,
		Recipient: recipient,
		Filename:  header.Filename,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	log.Printf("✅ Voice note sent successfully to %s", recipient)
	json.NewEncoder(w).Encode(response)
}

// SendRequest represents a request to send media (Python-style API)
type SendRequest struct {
	Recipient string `json:"recipient" example:"1234567890@s.whatsapp.net"`
	MediaPath string `json:"media_path" example:"/path/to/audio/file.ogg"`
}

// SendResponse represents the response from send operation
type SendResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Voice message sent successfully"`
}

// HandleSend handles the Python-style /send endpoint
// @Summary Send a voice message via WhatsApp (Python-style API)
// @Description Send an audio file as a WhatsApp voice message using media_path parameter
// @Tags API
// @Accept json
// @Produce json
// @Param request body SendRequest true "Send request with recipient and media_path"
// @Success 200 {object} SendResponse "Voice message sent successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /send [post]
func HandleSend(w http.ResponseWriter, r *http.Request, client *whatsapp.Client) {
	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to decode request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Recipient == "" {
		log.Printf("❌ Missing recipient parameter")
		http.Error(w, "Recipient must be provided", http.StatusBadRequest)
		return
	}

	if req.MediaPath == "" {
		log.Printf("❌ Missing media_path parameter")
		http.Error(w, "Media path must be provided", http.StatusBadRequest)
		return
	}

	// Check if file exists
	if _, err := os.Stat(req.MediaPath); os.IsNotExist(err) {
		log.Printf("❌ Media file not found: %s", req.MediaPath)
		http.Error(w, fmt.Sprintf("Media file not found: %s", req.MediaPath), http.StatusBadRequest)
		return
	}

	// Convert to Opus OGG if needed (matching Python implementation)
	mediaPath := req.MediaPath
	if !strings.HasSuffix(strings.ToLower(req.MediaPath), ".ogg") {
		log.Printf("🔄 Converting file to Opus OGG format: %s", req.MediaPath)

		convertedPath, err := convertToOpusOGG(req.MediaPath)
		if err != nil {
			log.Printf("❌ Error converting file to opus ogg: %v", err)
			http.Error(w, fmt.Sprintf("Error converting file to opus ogg. You likely need to install ffmpeg: %v", err), http.StatusInternalServerError)
			return
		}
		mediaPath = convertedPath

		// Clean up converted file after sending
		defer func() {
			if err := os.Remove(convertedPath); err != nil {
				log.Printf("⚠️ Failed to clean up converted file: %v", err)
			}
		}()
	}

	// Send voice message using WhatsApp client
	log.Printf("📤 Sending voice message to %s: %s", req.Recipient, mediaPath)
	err := client.SendAudioMessage(req.Recipient, mediaPath)
	if err != nil {
		log.Printf("❌ Failed to send voice message: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		response := SendResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to send voice message: %v", err),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("✅ Voice message sent successfully")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	response := SendResponse{
		Success: true,
		Message: "Voice message sent successfully",
	}
	json.NewEncoder(w).Encode(response)
}

// convertToOpusOGG converts any audio file to Opus OGG format
func convertToOpusOGG(inputPath string) (string, error) {
	// Create output path in the same directory as input
	fileDir := filepath.Dir(inputPath)
	fileName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	timestamp := time.Now().Unix()
	outputPath := filepath.Join(fileDir, fmt.Sprintf("%d_converted_%s.ogg", timestamp, fileName))

	// Use ffmpeg to convert to Opus OGG
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-c:a", "libopus", "-b:a", "64k", "-ar", "48000", "-ac", "1", outputPath)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	return outputPath, nil
}

// isValidAudioFile checks if the file is a valid audio file
func isValidAudioFile(filename string) bool {
	ext := filepath.Ext(filename)
	// Support common audio formats that WhatsApp can handle
	validExtensions := []string{
		".ogg", ".opus", ".mp3", ".wav", ".m4a", ".aac", ".flac", ".wma", ".mp4", ".3gp", ".amr",
	}

	for _, validExt := range validExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}
