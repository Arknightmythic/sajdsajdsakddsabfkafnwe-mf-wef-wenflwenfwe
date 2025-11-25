package azure

import (
	"fmt"
	"os"
)

type AzureConfig struct {
	ClientID            string
	ClientSecret        string
	TenantID            string
	Authority           string
	RedirectURI         string
	FrontendCallbackURL string
	Scope               []string
	TokenURL            string
	AuthorizationURL    string
	GraphAPIURL         string
	DefaultTeam         string
}

func NewAzureConfig() *AzureConfig {
	tenantID := os.Getenv("AZURE_AD_TENANT_ID")
	backendURI := os.Getenv("BACKEND_URI")
	if backendURI == "" {
		backendURI = "http://localhost:8000"
	}

	redirectPath := os.Getenv("AZURE_AD_REDIRECT_URI")
	if redirectPath == "" {
		redirectPath = "/api/authazure/callback"
	}

	// FIXED: Use the full callback URL including the path
	frontendCallbackURL := os.Getenv("FRONTEND_AZURE_AUTH_CALLBACK_URI")
	if frontendCallbackURL == "" {
		frontendCallbackURL = "http://localhost:5173/auth-microsoft/callback"
	}

	return &AzureConfig{
		ClientID:            os.Getenv("AZURE_AD_CLIENT_ID"),
		ClientSecret:        os.Getenv("AZURE_AD_CLIENT_SECRET"),
		TenantID:            tenantID,
		Authority:           fmt.Sprintf("https://login.microsoftonline.com/%s", tenantID),
		RedirectURI:         backendURI + redirectPath,
		FrontendCallbackURL: frontendCallbackURL,
		Scope:               []string{"openid", "profile", "email", "User.Read"},
		TokenURL:            fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID),
		AuthorizationURL:    fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID),
		GraphAPIURL:         "https://graph.microsoft.com/v1.0/me",
		DefaultTeam:         getDefaultTeam(),
	}
}

func getDefaultTeam() string {
	team := os.Getenv("DEFAULT_TEAM")
	if team == "" {
		return "finance"
	}
	return team
}
