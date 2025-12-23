package azure

import (
	"dokuprime-be/util"
	"fmt"
	"net/http"
	"net/url"

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

	fmt.Printf("=== Login Successful ===\n")
	fmt.Printf("Access Token: %s...\n", loginResp.AccessToken[:20])
	fmt.Printf("Refresh Token: %s...\n", loginResp.RefreshToken[:20])

	// Redirect to frontend with tokens as query parameters
	frontendURL := fmt.Sprintf("%s?status=login-success&access_token=%s&refresh_token=%s&session_id=%s",
		h.Service.Config.FrontendCallbackURL,
		url.QueryEscape(loginResp.AccessToken),
		url.QueryEscape(loginResp.RefreshToken),
		url.QueryEscape(loginResp.SessionID))

	fmt.Printf("Redirecting to: %s\n", frontendURL)
	c.Redirect(http.StatusFound, frontendURL)
}

func (h *AzureHandler) Logout(c *gin.Context) {
	logoutURL := h.Service.GetLogoutURL()

	util.SuccessResponse(c, "Azure AD logout URL generated", AuthURLResponse{
		AuthURL: logoutURL,
	})
}
