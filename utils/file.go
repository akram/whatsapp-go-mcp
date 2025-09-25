package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// GetFileExtension returns the file extension (without the dot)
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext[1:])
}

// IsImageFile checks if a file is an image based on its extension
func IsImageFile(filename string) bool {
	ext := GetFileExtension(filename)
	imageExts := []string{"jpg", "jpeg", "png", "gif", "webp", "bmp", "tiff"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// IsVideoFile checks if a file is a video based on its extension
func IsVideoFile(filename string) bool {
	ext := GetFileExtension(filename)
	videoExts := []string{"mp4", "avi", "mov", "mkv", "wmv", "flv", "webm", "m4v"}
	for _, vidExt := range videoExts {
		if ext == vidExt {
			return true
		}
	}
	return false
}

// IsAudioFile checks if a file is an audio file based on its extension
func IsAudioFile(filename string) bool {
	ext := GetFileExtension(filename)
	audioExts := []string{"mp3", "wav", "ogg", "opus", "aac", "flac", "m4a", "wma"}
	for _, audExt := range audioExts {
		if ext == audExt {
			return true
		}
	}
	return false
}

// FormatFileSize formats a file size in bytes to a human-readable string
func FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats a duration to a human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}

// SanitizeFilename removes or replaces invalid characters in a filename
func SanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// GenerateUniqueFilename generates a unique filename by appending a number if the file exists
func GenerateUniqueFilename(basePath, filename string) string {
	fullPath := filepath.Join(basePath, filename)
	if !FileExists(fullPath) {
		return filename
	}

	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)

	for i := 1; ; i++ {
		newFilename := fmt.Sprintf("%s_%d%s", name, i, ext)
		newPath := filepath.Join(basePath, newFilename)
		if !FileExists(newPath) {
			return newFilename
		}
	}
}

// CleanupOldFiles removes files older than the specified duration from a directory
func CleanupOldFiles(dir string, maxAge time.Duration) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && time.Since(info.ModTime()) > maxAge {
			return os.Remove(path)
		}

		return nil
	})
}

// GetMediaTypeFromExtension returns the MIME type based on file extension
func GetMediaTypeFromExtension(filename string) string {
	ext := GetFileExtension(filename)

	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "mp4":
		return "video/mp4"
	case "avi":
		return "video/x-msvideo"
	case "mov":
		return "video/quicktime"
	case "mkv":
		return "video/x-matroska"
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "ogg":
		return "audio/ogg"
	case "opus":
		return "audio/opus"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "pdf":
		return "application/pdf"
	case "doc":
		return "application/msword"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "xls":
		return "application/vnd.ms-excel"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "ppt":
		return "application/vnd.ms-powerpoint"
	case "pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case "zip":
		return "application/zip"
	case "rar":
		return "application/x-rar-compressed"
	case "7z":
		return "application/x-7z-compressed"
	default:
		return "application/octet-stream"
	}
}
