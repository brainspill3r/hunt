package Utils

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

// LoadEnv loads the environment variables from the given .env file
func LoadEnv(filePath string) {
    err := godotenv.Load(filePath)
    if err != nil {
        log.Fatalf("Error loading .env file: %v", err)
    }
}

// GetPortScanningWebhook retrieves the webhook URL for the port-scanning service from the environment variables
func GetPortScanningWebhook() string {
    webhook := os.Getenv("PORT_SCANNING_SERVICE_WEBHOOK")
    if webhook == "" {
        log.Fatalf("Error: PORT_SCANNING_SERVICE_WEBHOOK is not set in the environment variables")
    }
    return webhook
}
