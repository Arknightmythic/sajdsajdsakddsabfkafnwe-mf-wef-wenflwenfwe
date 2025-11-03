package user

import (
	"net/http"
	"strconv"

	"dokuprime-be/util"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	Service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
	return &UserHandler{Service: service}
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

	c.SetCookie(
		"access_token",
		response.AccessToken,
		3600,
		"/",
		"",
		false,
		true,
	)

	c.SetCookie(
		"refresh_token",
		response.RefreshToken,
		604800,
		"/",
		"",
		false,
		true,
	)

	util.SuccessResponse(c, "Login successful", response)
}

func (h *UserHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		util.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err := h.Service.Logout(userID.(int64))
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	util.SuccessResponse(c, "Logout successful", nil)
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

	c.SetCookie(
		"access_token",
		accessToken,
		3600,
		"/",
		"",
		false,
		true,
	)

	util.SuccessResponse(c, "Token refreshed successfully", gin.H{
		"access_token": accessToken,
	})
}
