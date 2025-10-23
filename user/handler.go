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
	users, err := h.Service.GetUsers()
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "Users fetched successfully", users)
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

	util.SuccessResponse(c, "Logout successful", nil)
}

func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid input")
		return
	}

	accessToken, err := h.Service.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		util.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	util.SuccessResponse(c, "Token refreshed successfully", gin.H{
		"access_token": accessToken,
	})
}
