package Utils

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
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

// GetXSSServiceWebhook retrieves the webhook URL for XSS detection service from the environment
func GetXSSServiceWebhook() string {
    return os.Getenv("XSS_SERVICE_WEBHOOK")
}

// SendDiscordNotification sends a notification to the provided Discord webhook URL
func SendDiscordNotification(webhookURL, message string) error {
    payload := map[string]string{"content": message}
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %v", err)
    }

    resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("failed to send POST request: %v", err)
    }
    defer resp.Body.Close()

    // Read the response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("failed to read response body: %v", err)
    }

    if resp.StatusCode != http.StatusNoContent {
        return fmt.Errorf("unexpected response from Discord: %s", body)
    }

    return nil
}
