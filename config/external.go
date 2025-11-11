package config

import (
	"os"
)

type ExternalAPIConfig struct {
	BaseURL string
	APIKey  string
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

	return &ExternalAPIConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}