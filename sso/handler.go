package sso

import (
	"dokuprime-be/util"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SSOHandler struct {
	service *SSOService
}

func NewSSOHandler(service *SSOService) *SSOHandler {
	return &SSOHandler{service: service}
}

// Login: Redirect user to Microsoft
func (h *SSOHandler) Login(c *gin.Context) {
	// State harusnya acak untuk mencegah CSRF
	state := util.RandString(16)
	// Simpan state di cookie sementara untuk validasi di callback (opsional tapi disarankan)
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	
	url := h.service.GetLoginURL(state)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// Callback: Handle return from Microsoft
func (h *SSOHandler) Callback(c *gin.Context) {
	// Validasi State (Opsional)
	state := c.Query("state")
	cookieState, err := c.Cookie("oauth_state")
	if err != nil || state != cookieState {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid oauth state")
		return
	}
	// Hapus cookie state
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	code := c.Query("code")
	if code == "" {
		util.ErrorResponse(c, http.StatusBadRequest, "Code not found")
		return
	}

	response, err := h.service.HandleCallback(c.Request.Context(), code)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Set Cookies (Copy logic from user/handler.go)
	domain := os.Getenv("COOKIE_DOMAIN")
	path := os.Getenv("COOKIE_PATH")
	if path == "" { path = "/" }
	
	secure, _ := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))
	httpOnly := true // Selalu true untuk keamanan
	
	sameSiteStr := os.Getenv("COOKIE_SAME_SITE")
	var sameSite http.SameSite
	switch sameSiteStr {
	case "Strict": sameSite = http.SameSiteStrictMode
	case "None": sameSite = http.SameSiteNoneMode
	default: sameSite = http.SameSiteLaxMode
	}

	accessMaxAge := 3600
	refreshMaxAge := 604800

	c.SetSameSite(sameSite)
	c.SetCookie("access_token", response.AccessToken, accessMaxAge, path, domain, secure, httpOnly)
	c.SetCookie("refresh_token", response.RefreshToken, refreshMaxAge, path, domain, secure, httpOnly)
	c.SetCookie("session_id", response.SessionID, refreshMaxAge, path, domain, secure, httpOnly)

	// Redirect ke Frontend Dashboard setelah login sukses
	frontendURL := os.Getenv("ALLOWED_ORIGINS") // Atau variabel env khusus FRONTEND_URL
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // Default
	}
	c.Redirect(http.StatusFound, frontendURL+"/")
}