package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads the .discordhooks.env file from a specified path
func LoadEnv(filePath string) {
	err := godotenv.Load(filePath)
	if err != nil {
		log.Fatalf("Error loading .discordhooks.env file: %v", err)
	}
}

// GetScanCompletionWebhook retrieves the scan completion webhook from environment variables
func GetScanCompletionWebhook() string {
	webhook := os.Getenv("DISCORD_SCAN_COMPLETION_WEBHOOK")
	if webhook == "" {
		log.Fatalf("Error: DISCORD_SCAN_COMPLETION_WEBHOOK is not set in the environment variables")
	}
	return webhook
}
