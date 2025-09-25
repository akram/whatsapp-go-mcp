package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	Port      string
	DBPath    string
	MediaDir  string
	LogLevel  string
	QRCodeDir string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Port:      getEnv("PORT", "8080"),
		DBPath:    getEnv("WHATSAPP_DB_PATH", "./whatsapp.db"),
		MediaDir:  getEnv("WHATSAPP_MEDIA_DIR", "./media"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		QRCodeDir: getEnv("QR_CODE_DIR", "./qr_codes"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool gets an environment variable as boolean with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
