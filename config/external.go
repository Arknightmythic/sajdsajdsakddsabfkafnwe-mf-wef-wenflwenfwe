package config

import (
	"os"
)

type ExternalAPIConfig struct {
	BaseURL        string
	APIKey         string
	MessagesAPIURL string
	MessagesAPIKey string
}

func LoadExternalAPIConfig() *ExternalAPIConfig {
	baseURL := os.Getenv("EXTERNAL_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://172.16.12.98:9534"
	}

	apiKey := os.Getenv("X_API_KEY")
	if apiKey == "" {
		apiKey = "BangJumAwesome"
	}

	messagesAPIURL := os.Getenv("MESSAGES_API_URL")
	if messagesAPIURL == "" {
		messagesAPIURL = "http://localhost:9798"
	}

	messagesAPIKey := os.Getenv("MESSAGES_API_KEY")
	if messagesAPIKey == "" {
		messagesAPIKey = "BangJorAwesome"
	}

	return &ExternalAPIConfig{
		BaseURL:        baseURL,
		APIKey:         apiKey,
		MessagesAPIURL: messagesAPIURL,
		MessagesAPIKey: messagesAPIKey,
	}
}
