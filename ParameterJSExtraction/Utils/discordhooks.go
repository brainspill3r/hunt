package Utils

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

// GetParamJSExtractionWebhook retrieves the webhook URL for ParameterJSExtraction from the environment variables
func GetParamJSExtractionWebhook() string {
    webhook := os.Getenv("PARAM_JS_EXTRACTION_WEBHOOK")
    if webhook == "" {
        log.Fatalf("Error: PARAM_JS_EXTRACTION_WEBHOOK is not set in the environment variables")
    }
    return webhook
}
