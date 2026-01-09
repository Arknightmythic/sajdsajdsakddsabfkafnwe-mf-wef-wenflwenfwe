package config

type ExternalAPIConfig struct {
	BaseURL        string
	APIKey         string
	MessagesAPIURL string
	MessagesAPIKey string
}

func LoadExternalAPIConfig() *ExternalAPIConfig {
	baseURL := AppConfig.ExternalAPIBaseURL
	if baseURL == "" {
		baseURL = "http://172.16.12.98:9534"
	}

	apiKey := AppConfig.XAPIKey
	if apiKey == "" {
		apiKey = "BangJumAwesome"
	}

	messagesAPIURL := AppConfig.MessagesAPIURL
	if messagesAPIURL == "" {
		messagesAPIURL = "http://localhost:9798"
	}

	messagesAPIKey := AppConfig.MessagesAPIKey
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
