package azure

import (
	"dokuprime-be/config"
	"fmt"
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
	tenantID := config.AppConfig.AzureADTenantID
	backendURI := config.AppConfig.BackendURI
	if backendURI == "" {
		backendURI = "http://localhost:8000"
	}

	redirectPath := config.AppConfig.AzureADRedirectURI
	if redirectPath == "" {
		redirectPath = "/api/authazure/callback"
	}
	frontendCallbackURL := config.AppConfig.FrontendAzureAuthCallbackURI
	if frontendCallbackURL == "" {
		frontendCallbackURL = "http://localhost:5173/auth-microsoft/callback"
	}

	return &AzureConfig{
		ClientID:            config.AppConfig.AzureADClientID,
		ClientSecret:        config.AppConfig.AzureADClientSecret,
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
	var team = config.AppConfig.DefaultTeam
	if team == "" {
		return "finance"
	}
	return team
}
