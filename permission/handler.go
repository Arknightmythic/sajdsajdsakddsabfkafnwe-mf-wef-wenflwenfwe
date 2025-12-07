package permission

import (
	"dokuprime-be/util"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const isInvalidPermissionID = "Invalid permission ID"

type PermissionHandler struct {
	service *PermissionService
}

func NewPermissionHandler(service *PermissionService) *PermissionHandler {
	return &PermissionHandler{service: service}
}

func (h *PermissionHandler) CreatePermission(ctx *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	permission := &Permission{
		Name: req.Name,
	}

	if err := h.service.Create(permission); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(ctx, "Permission created successfully", permission)
}

func (h *PermissionHandler) GetPermissions(ctx *gin.Context) {
	permissions, err := h.service.GetAll()
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Permissions retrieved successfully", permissions)
}

func (h *PermissionHandler) GetPermissionByID(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, isInvalidPermissionID)
		return
	}

	permission, err := h.service.GetByID(id)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusNotFound, "Permission not found")
		return
	}

	util.SuccessResponse(ctx, "Permission retrieved successfully", permission)
}

func (h *PermissionHandler) UpdatePermission(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, isInvalidPermissionID)
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	permission := &Permission{
		ID:   id,
		Name: req.Name,
	}

	if err := h.service.Update(permission); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Permission updated successfully", permission)
}

func (h *PermissionHandler) DeletePermission(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, isInvalidPermissionID)
		return
	}

	if err := h.service.Delete(id); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Permission deleted successfully", nil)
}
