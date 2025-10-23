package role

import (
	"dokuprime-be/util"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service *RoleService
}

func NewRoleHandler(service *RoleService) *RoleHandler {
	return &RoleHandler{service: service}
}

func (h *RoleHandler) Create(c *gin.Context) {
	var input Role
	if err := c.ShouldBindJSON(&input); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid input")
		return
	}

	if err := h.service.Create(input); err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(c, "Role created successfully", input)
}

func (h *RoleHandler) GetAll(c *gin.Context) {
	roles, err := h.service.GetAll()
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "Roles fetched successfully", roles)
}

func (h *RoleHandler) GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	role, err := h.service.GetByID(id)
	if err != nil {
		util.ErrorResponse(c, http.StatusNotFound, "Role not found")
		return
	}

	util.SuccessResponse(c, "Role fetched successfully", role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var input Role
	if err := c.ShouldBindJSON(&input); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid input")
		return
	}

	if err := h.service.Update(id, input); err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "Role updated successfully", input)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.service.Delete(id); err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "Role deleted successfully", nil)
}
