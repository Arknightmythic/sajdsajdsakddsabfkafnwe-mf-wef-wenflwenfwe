package config

import (
	"os"
)

type ExternalAPIConfig struct {
	BaseURL string
}

func LoadExternalAPIConfig() *ExternalAPIConfig {
	baseURL := os.Getenv("EXTERNAL_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://172.16.12.98:9534"
	}

	return &ExternalAPIConfig{
		BaseURL: baseURL,
	}
}
