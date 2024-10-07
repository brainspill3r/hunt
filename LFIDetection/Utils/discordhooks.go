package Utils

import (
    "fmt"
    "os"
    "github.com/joho/godotenv"
)

// LoadEnv loads environment variables from a .env file
func LoadEnv(filePath string) {
    err := godotenv.Load(filePath)
    if err != nil {
        fmt.Println("Error loading .env file:", err)
    }
}

// GetLFIDetectionWebhook retrieves the webhook URL for LFI detection from the environment
func GetLFIDetectionWebhook() string {
    return os.Getenv("LFI_DETECTION_WEBHOOK")
}
