package utils

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

// LoadEnv loads the environment variables from a given file
func LoadEnv(filePath string) {
    err := godotenv.Load(filePath)
    if err != nil {
        log.Fatalf("Error loading .env file: %v", err)
    }
}

// GetOpenRedirectWebhook retrieves the Open Redirect webhook URL from the environment variables
func GetOpenRedirectWebhook() string {
    webhook := os.Getenv("DISCORD_OPEN_REDIRECT_WEBHOOK")
    if webhook == "" {
        log.Fatalf("Error: DISCORD_OPEN_REDIRECT_WEBHOOK is not set in the environment variables")
    }
    return webhook
}
