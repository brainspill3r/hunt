package Utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadEnv loads the environment variables from the .env file
func LoadEnv(filePath string) {
	err := godotenv.Load(filePath)
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}

// GetURLCollectionWebhook fetches the Discord webhook for URL Collection
func GetURLCollectionWebhook() string {
	return os.Getenv("DISCORD_URL_COLLECTION_WEBHOOK")
}
