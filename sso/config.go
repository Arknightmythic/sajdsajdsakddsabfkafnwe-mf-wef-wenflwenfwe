package sso

import (
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

func GetMicrosoftOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
		ClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("MICROSOFT_REDIRECT_URI"),
		Endpoint:     microsoft.AzureADEndpoint(os.Getenv("MICROSOFT_TENANT_ID")),
		Scopes:       []string{"User.Read", "email", "profile", "openid"},
	}
}