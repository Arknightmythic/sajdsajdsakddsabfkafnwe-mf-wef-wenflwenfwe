package user

import (
	"dokuprime-be/util"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	Service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{
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

func (h *UserHandler) CreateUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid input")
		return
	}

	createdUser, err := h.Service.CreateUser(&user)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(c, "User created successfully", createdUser)
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	var query GetUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.Service.GetUsers(&query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) GetUserByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.Service.GetUserByID(id)
	if err != nil {
		util.ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	util.SuccessResponse(c, "User fetched successfully", user)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid input")
		return
	}

	updatedUser, err := h.Service.UpdateUser(id, &user)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "User updated successfully", updatedUser)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = h.Service.DeleteUser(id)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "User deleted successfully", nil)
}

func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid input")
		return
	}

	response, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		util.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	domain, path, secure, httpOnly, sameSite, accessMaxAge, refreshMaxAge := getCookieSettings()

	c.SetSameSite(sameSite)
	c.SetCookie(
		"access_token",
		response.AccessToken,
		accessMaxAge,
		path,
		domain,
		secure,
		httpOnly,
	)

	c.SetSameSite(sameSite)
	c.SetCookie(
		"refresh_token",
		response.RefreshToken,
		refreshMaxAge,
		path,
		domain,
		secure,
		httpOnly,
	)

	c.SetSameSite(sameSite)
	c.SetCookie(
		"session_id",
		response.SessionID,
		refreshMaxAge,
		path,
		domain,
		secure,
		httpOnly,
	)

	util.SuccessResponse(c, "Login successful", response)
}

func (h *UserHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	sessionID, err := c.Cookie("session_id")
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Session ID not found")
		return
	}

	err = h.Service.Logout(userID.(int64), sessionID)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	domain, path, secure, httpOnly, sameSite, _, _ := getCookieSettings()

	c.SetSameSite(sameSite)
	c.SetCookie("access_token", "", -1, path, domain, secure, httpOnly)

	c.SetSameSite(sameSite)
	c.SetCookie("refresh_token", "", -1, path, domain, secure, httpOnly)

	c.SetSameSite(sameSite)
	c.SetCookie("session_id", "", -1, path, domain, secure, httpOnly)

	util.SuccessResponse(c, "Logout successful", nil)
}

func (h *UserHandler) LogoutAllSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err := h.Service.LogoutAllSessions(userID.(int64))
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout all sessions")
		return
	}

	domain, path, secure, httpOnly, sameSite, _, _ := getCookieSettings()

	c.SetSameSite(sameSite)
	c.SetCookie("access_token", "", -1, path, domain, secure, httpOnly)

	c.SetSameSite(sameSite)
	c.SetCookie("refresh_token", "", -1, path, domain, secure, httpOnly)

	c.SetSameSite(sameSite)
	c.SetCookie("session_id", "", -1, path, domain, secure, httpOnly)

	util.SuccessResponse(c, "All sessions logged out successfully", nil)
}

func (h *UserHandler) GetActiveSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	sessions, err := h.Service.GetActiveSessions(userID.(int64))
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve sessions")
		return
	}

	util.SuccessResponse(c, "Active sessions retrieved successfully", gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

func (h *UserHandler) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Refresh token not found")
		return
	}

	accessToken, err := h.Service.RefreshAccessToken(refreshToken)
	if err != nil {
		util.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	domain, path, secure, httpOnly, sameSite, accessMaxAge, _ := getCookieSettings()

	c.SetSameSite(sameSite)
	c.SetCookie(
		"access_token",
		accessToken,
		accessMaxAge,
		path,
		domain,
		secure,
		httpOnly,
	)

	util.SuccessResponse(c, "Token refreshed successfully", gin.H{
		"access_token": accessToken,
	})
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	user, err := h.Service.GetUserByID(int(userID.(int64)))
	if err != nil {
		util.ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	util.SuccessResponse(c, "Current user fetched successfully", user)
}
