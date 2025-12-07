package azure

import (
	"dokuprime-be/util"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

const loginFailedURLTemplate = "%s?status=login-failed&error=%s"

type AzureHandler struct {
	Service *AzureService
}

func NewAzureHandler(service *AzureService) *AzureHandler {
	return &AzureHandler{
		Service: service,
	}
}

func getCookieSettings() (domain string, path string, secure bool, httpOnly bool, sameSite http.SameSite, accessMaxAge int, refreshMaxAge int) {
	domain = os.Getenv("COOKIE_DOMAIN")

	path = os.Getenv("COOKIE_PATH")
	if path == "" {
		path = "/"
	}

	secure, _ = strconv.ParseBool(os.Getenv("COOKIE_SECURE"))

	httpOnly = true
	if httpOnlyStr := os.Getenv("COOKIE_HTTP_ONLY"); httpOnlyStr != "" {
		httpOnly, _ = strconv.ParseBool(httpOnlyStr)
	}

	sameSiteStr := os.Getenv("COOKIE_SAME_SITE")
	switch sameSiteStr {
	case "Strict":
		sameSite = http.SameSiteStrictMode
	case "None":
		sameSite = http.SameSiteNoneMode
	default:
		sameSite = http.SameSiteLaxMode
	}

	accessMaxAge = 3600
	if maxAgeStr := os.Getenv("COOKIE_ACCESS_TOKEN_MAX_AGE"); maxAgeStr != "" {
		if parsed, err := strconv.Atoi(maxAgeStr); err == nil {
			accessMaxAge = parsed
		}
	}

	refreshMaxAge = 604800
	if maxAgeStr := os.Getenv("COOKIE_REFRESH_TOKEN_MAX_AGE"); maxAgeStr != "" {
		if parsed, err := strconv.Atoi(maxAgeStr); err == nil {
			refreshMaxAge = parsed
		}
	}

	return
}

func (h *AzureHandler) Login(c *gin.Context) {
	authURL, _, err := h.Service.GetAuthURL()
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate auth URL")
		return
	}

	util.SuccessResponse(c, "Azure AD login URL generated", AuthURLResponse{
		AuthURL: authURL,
	})
}

func (h *AzureHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")
	errorDesc := c.Query("error_description")

	fmt.Printf("=== Azure Callback Received ===\n")
	fmt.Printf("Code: %s\n", code)
	fmt.Printf("State: %s\n", state)
	fmt.Printf("Error: %s\n", errorParam)
	fmt.Printf("Error Description: %s\n", errorDesc)
	fmt.Printf("Headers: %+v\n", c.Request.Header)

	if errorParam != "" {
		fmt.Printf("Azure returned error: %s - %s\n", errorParam, errorDesc)
		
		frontendURL := fmt.Sprintf(loginFailedURLTemplate,
			h.Service.Config.FrontendCallbackURL,
			url.QueryEscape(errorDesc))
		c.Redirect(http.StatusFound, frontendURL)
		return
	}

	if code == "" {
		fmt.Println("No authorization code received")
		
		frontendURL := fmt.Sprintf(loginFailedURLTemplate,
			h.Service.Config.FrontendCallbackURL,
			url.QueryEscape("No authorization code received"))
		c.Redirect(http.StatusFound, frontendURL)
		return
	}

	loginResp, err := h.Service.ProcessAzureLogin(code)
	if err != nil {
		fmt.Printf("Azure login processing error: %v\n", err)
		
		frontendURL := fmt.Sprintf(loginFailedURLTemplate,
			h.Service.Config.FrontendCallbackURL,
			url.QueryEscape(err.Error()))
		c.Redirect(http.StatusFound, frontendURL)
		return
	}

	domain, path, secure, httpOnly, sameSite, accessMaxAge, refreshMaxAge := getCookieSettings()

	fmt.Printf("=== Setting Cookies ===\n")
	fmt.Printf("Domain: %s, Path: %s, Secure: %v, SameSite: %v\n", domain, path, secure, sameSite)

	c.SetSameSite(sameSite)
	c.SetCookie(
		"access_token",
		loginResp.AccessToken,
		accessMaxAge,
		path,
		domain,
		secure,
		httpOnly,
	)

	c.SetSameSite(sameSite)
	c.SetCookie(
		"refresh_token",
		loginResp.RefreshToken,
		refreshMaxAge,
		path,
		domain,
		secure,
		httpOnly,
	)

	c.SetSameSite(sameSite)
	c.SetCookie(
		"session_id",
		loginResp.SessionID,
		refreshMaxAge,
		path,
		domain,
		secure,
		httpOnly,
	)

	frontendURL := fmt.Sprintf("%s?status=login-success",
		h.Service.Config.FrontendCallbackURL)

	fmt.Printf("Redirecting to: %s\n", frontendURL)
	c.Redirect(http.StatusFound, frontendURL)
}

func (h *AzureHandler) Logout(c *gin.Context) {
	logoutURL := h.Service.GetLogoutURL()

	domain, path, secure, httpOnly, sameSite, _, _ := getCookieSettings()

	c.SetSameSite(sameSite)
	c.SetCookie("access_token", "", -1, path, domain, secure, httpOnly)

	c.SetSameSite(sameSite)
	c.SetCookie("refresh_token", "", -1, path, domain, secure, httpOnly)

	c.SetSameSite(sameSite)
	c.SetCookie("session_id", "", -1, path, domain, secure, httpOnly)

	util.SuccessResponse(c, "Azure AD logout URL generated", AuthURLResponse{
		AuthURL: logoutURL,
	})
}