package user

import (
	"dokuprime-be/util"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	isInvalidInput     = "Invalid input"
	isInvalidUser      = "Invalid user ID"
	isNotAuthenticated = "User not authenticated"
)

type UserHandler struct {
	Service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{
		Service: service,
	}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidInput)
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
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidUser)
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
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidUser)
		return
	}

	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidInput)
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
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidUser)
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
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidInput)
		return
	}

	response, err := h.Service.Login(req.Email, req.Password)
	if err != nil {
		util.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	util.SuccessResponse(c, "Login successful", response)
}

func (h *UserHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, isNotAuthenticated)
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		var req struct {
			SessionID string `json:"session_id"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			sessionID = req.SessionID
		}
	}

	if sessionID == "" {
		util.ErrorResponse(c, http.StatusBadRequest, "Session ID not found")
		return
	}

	err := h.Service.Logout(userID.(int64), sessionID)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	util.SuccessResponse(c, "Logout successful", nil)
}

func (h *UserHandler) LogoutAllSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, isNotAuthenticated)
		return
	}

	err := h.Service.LogoutAllSessions(userID.(int64))
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout all sessions")
		return
	}

	util.SuccessResponse(c, "All sessions logged out successfully", nil)
}

func (h *UserHandler) GetActiveSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, isNotAuthenticated)
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

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Refresh token not found in request body")
		return
	}

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		refreshToken = c.GetHeader("X-Refresh-Token")
	}

	if refreshToken == "" {
		util.ErrorResponse(c, http.StatusBadRequest, "Refresh token not found")
		return
	}

	accessToken, err := h.Service.RefreshAccessToken(refreshToken)
	if err != nil {
		util.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	util.SuccessResponse(c, "Token refreshed successfully", gin.H{
		"access_token": accessToken,
	})
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, isNotAuthenticated)
		return
	}

	user, err := h.Service.GetUserByID(int(userID.(int64)))
	if err != nil {
		util.ErrorResponse(c, http.StatusNotFound, "User not found")
		return
	}

	util.SuccessResponse(c, "Current user fetched successfully", user)
}
