package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// GenerateID generates a random ID of the specified length
func GenerateID(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ValidateJID validates a WhatsApp JID format
func ValidateJID(jid string) error {
	// Basic JID validation
	if jid == "" {
		return fmt.Errorf("JID cannot be empty")
	}

	// Check if it's a phone number format
	phoneRegex := regexp.MustCompile(`^\d+@s\.whatsapp\.net$`)
	if phoneRegex.MatchString(jid) {
		return nil
	}

	// Check if it's a group JID format
	groupRegex := regexp.MustCompile(`^\d+@g\.us$`)
	if groupRegex.MatchString(jid) {
		return nil
	}

	// Check if it's a broadcast JID format
	broadcastRegex := regexp.MustCompile(`^\d+@broadcast$`)
	if broadcastRegex.MatchString(jid) {
		return nil
	}

	return fmt.Errorf("invalid JID format: %s", jid)
}

// NormalizePhoneNumber normalizes a phone number to WhatsApp format
func NormalizePhoneNumber(phone string) string {
	// Remove all non-digit characters
	phone = regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	// Remove leading zeros
	phone = strings.TrimLeft(phone, "0")

	// Add country code if not present (assuming +1 for US)
	if len(phone) == 10 {
		phone = "1" + phone
	}

	return phone
}

// FormatPhoneNumber formats a phone number for display
func FormatPhoneNumber(phone string) string {
	phone = NormalizePhoneNumber(phone)

	if len(phone) == 11 && phone[0] == '1' {
		// US format: +1 (XXX) XXX-XXXX
		return fmt.Sprintf("+1 (%s) %s-%s", phone[1:4], phone[4:7], phone[7:])
	}

	// Default format: +XXXXXXXXXX
	return "+" + phone
}

// ExtractPhoneFromJID extracts the phone number from a JID
func ExtractPhoneFromJID(jid string) string {
	parts := strings.Split(jid, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// IsGroupJID checks if a JID represents a group
func IsGroupJID(jid string) bool {
	return strings.HasSuffix(jid, "@g.us")
}

// IsBroadcastJID checks if a JID represents a broadcast
func IsBroadcastJID(jid string) bool {
	return strings.HasSuffix(jid, "@broadcast")
}

// IsIndividualJID checks if a JID represents an individual contact
func IsIndividualJID(jid string) bool {
	return strings.HasSuffix(jid, "@s.whatsapp.net")
}

// SanitizeMessageContent sanitizes message content for storage
func SanitizeMessageContent(content string) string {
	// Remove null bytes and control characters
	content = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`).ReplaceAllString(content, "")

	// Trim whitespace
	content = strings.TrimSpace(content)

	return content
}

// ValidateMessageContent validates message content
func ValidateMessageContent(content string) error {
	if len(content) == 0 {
		return fmt.Errorf("message content cannot be empty")
	}

	if len(content) > 4096 {
		return fmt.Errorf("message content too long (max 4096 characters)")
	}

	return nil
}

// ValidateFilePath validates a file path
func ValidateFilePath(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("invalid file path: path traversal not allowed")
	}

	return nil
}

// GetFileExtension returns the file extension (without the dot)
func GetFileExtension(filename string) string {
	lastDot := strings.LastIndex(filename, ".")
	if lastDot == -1 {
		return ""
	}
	return strings.ToLower(filename[lastDot+1:])
}

// IsValidImageExtension checks if a file extension is a valid image format
func IsValidImageExtension(ext string) bool {
	validExts := []string{"jpg", "jpeg", "png", "gif", "webp", "bmp", "tiff"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// IsValidVideoExtension checks if a file extension is a valid video format
func IsValidVideoExtension(ext string) bool {
	validExts := []string{"mp4", "avi", "mov", "mkv", "wmv", "flv", "webm", "m4v"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// IsValidAudioExtension checks if a file extension is a valid audio format
func IsValidAudioExtension(ext string) bool {
	validExts := []string{"mp3", "wav", "ogg", "opus", "aac", "flac", "m4a", "wma"}
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// ChunkSlice splits a slice into chunks of the specified size
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// Contains checks if a slice contains a specific element
func Contains[T comparable](slice []T, element T) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate elements from a slice
func RemoveDuplicates[T comparable](slice []T) []T {
	keys := make(map[T]bool)
	var result []T

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
