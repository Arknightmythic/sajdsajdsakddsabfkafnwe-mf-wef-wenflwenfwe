package helpdesk

import (
	"dokuprime-be/messaging"
	"dokuprime-be/util"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type HelpdeskHandler struct {
	service        *HelpdeskService
	messageService *messaging.MessageService
}

func NewHelpdeskHandler(service *HelpdeskService, messageService *messaging.MessageService) *HelpdeskHandler {
	return &HelpdeskHandler{
		service:        service,
		messageService: messageService,
	}
}

func (h *HelpdeskHandler) CreateHelpdesk(ctx *gin.Context) {
	var req struct {
		SessionID        string  `json:"session_id"`
		Platform         string  `json:"platform"`
		PlatformUniqueID *string `json:"platform_unique_id"`
		Status           string  `json:"status"`
		UserID           int     `json:"user_id"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	helpdesk := &Helpdesk{
		SessionID:        req.SessionID,
		Platform:         req.Platform,
		PlatformUniqueID: req.PlatformUniqueID,
		Status:           req.Status,
		UserID:           req.UserID,
	}

	if err := h.service.Create(helpdesk); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(ctx, "Helpdesk created successfully", helpdesk)
}

func (h *HelpdeskHandler) GetAll(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	search := ctx.DefaultQuery("search", "")

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	helpdesks, total, err := h.service.GetAll(limit, offset, search)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"helpdesks": helpdesks,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"search":    search,
	}

	util.SuccessResponse(ctx, "Helpdesks retrieved successfully", response)
}

func (h *HelpdeskHandler) GetHelpdeskByID(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid helpdesk ID")
		return
	}

	helpdesk, err := h.service.GetByID(id)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusNotFound, "Helpdesk not found")
		return
	}

	util.SuccessResponse(ctx, "Helpdesk retrieved successfully", helpdesk)
}

func (h *HelpdeskHandler) UpdateHelpdesk(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid helpdesk ID")
		return
	}

	var req map[string]interface{}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req) == 1 && req["status"] != nil {
		status, ok := req["status"].(string)
		if !ok {
			util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid status value")
			return
		}

		if err := h.service.UpdateStatus(id, status); err != nil {
			util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
			return
		}

		helpdesk, err := h.service.GetByID(id)
		if err != nil {
			util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
			return
		}

		util.SuccessResponse(ctx, "Helpdesk status updated successfully", helpdesk)
		return
	}

	var fullReq struct {
		SessionID        string  `json:"session_id"`
		Platform         string  `json:"platform"`
		PlatformUniqueID *string `json:"platform_unique_id"`
		Status           string  `json:"status"`
		UserID           int     `json:"user_id"`
	}

	if err := ctx.ShouldBindJSON(&fullReq); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	helpdesk := &Helpdesk{
		ID:               id,
		SessionID:        fullReq.SessionID,
		Platform:         fullReq.Platform,
		PlatformUniqueID: fullReq.PlatformUniqueID,
		Status:           fullReq.Status,
		UserID:           fullReq.UserID,
	}

	if err := h.service.Update(helpdesk); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	updatedHelpdesk, err := h.service.GetByID(id)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Helpdesk updated successfully", updatedHelpdesk)
}

func (h *HelpdeskHandler) DeleteHelpdesk(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid helpdesk ID")
		return
	}

	if err := h.service.Delete(id); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Helpdesk deleted successfully", nil)
}

func (h *HelpdeskHandler) AskHelpdesk(ctx *gin.Context) {
	var req struct {
		SessionID      string `json:"session_id" binding:"required"`
		Message        string `json:"message" binding:"required"`
		UserType       string `json:"user_type" binding:"required"`
		StartTimestamp string `json:"start_timestamp"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	sessionID, err := uuid.Parse(req.SessionID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid session ID format")
		return
	}

	helpdesk, err := h.service.GetBySessionID(req.SessionID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusNotFound, "Helpdesk session not found")
		return
	}

	err = h.messageService.HandleHelpdeskMessage(
		sessionID,
		req.Message,
		req.UserType,
		helpdesk.Platform,
		helpdesk.PlatformUniqueID,
		req.StartTimestamp,
	)

	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Message sent successfully", gin.H{
		"session_id": req.SessionID,
		"message":    req.Message,
		"user_type":  req.UserType,
	})
}
