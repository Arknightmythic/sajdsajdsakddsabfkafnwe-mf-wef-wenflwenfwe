package chat

import (
	"dokuprime-be/external"
	"dokuprime-be/util"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	service        *ChatService
	externalClient *external.Client
}

func NewChatHandler(service *ChatService, externalClient *external.Client) *ChatHandler {
	return &ChatHandler{
		service:        service,
		externalClient: externalClient,
	}
}

func (h *ChatHandler) CreateChatHistory(ctx *gin.Context) {
	var req struct {
		SessionID           string                 `json:"session_id" binding:"required"`
		Message             map[string]interface{} `json:"message" binding:"required"`
		UserID              *int64                 `json:"user_id"`
		IsCannotAnswer      *bool                  `json:"is_cannot_answer"`
		Category            *string                `json:"category"`
		Feedback            *bool                  `json:"feedback"`
		QuestionCategory    *string                `json:"question_category"`
		QuestionSubCategory *string                `json:"question_sub_category"`
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

	history := &ChatHistory{
		SessionID:           sessionID,
		Message:             req.Message,
		UserID:              req.UserID,
		IsCannotAnswer:      req.IsCannotAnswer,
		Category:            req.Category,
		Feedback:            req.Feedback,
		QuestionCategory:    req.QuestionCategory,
		QuestionSubCategory: req.QuestionSubCategory,
	}

	if err := h.service.CreateChatHistory(history); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(ctx, "Chat history created successfully", history)
}

func (h *ChatHandler) GetChatHistories(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	result, err := h.service.GetAllChatHistory(page, pageSize)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat histories retrieved successfully", result)
}

func (h *ChatHandler) GetChatHistoryByID(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid chat history ID")
		return
	}

	history, err := h.service.GetChatHistoryByID(id)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusNotFound, "Chat history not found")
		return
	}

	util.SuccessResponse(ctx, "Chat history retrieved successfully", history)
}

func (h *ChatHandler) GetChatHistoryBySessionID(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("session_id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid session ID")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	result, err := h.service.GetChatHistoryBySessionID(sessionID, page, pageSize)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat history retrieved successfully", result)
}

func (h *ChatHandler) UpdateChatHistory(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid chat history ID")
		return
	}

	var req struct {
		Message             map[string]interface{} `json:"message"`
		UserID              *int64                 `json:"user_id"`
		IsCannotAnswer      *bool                  `json:"is_cannot_answer"`
		Category            *string                `json:"category"`
		Feedback            *bool                  `json:"feedback"`
		QuestionCategory    *string                `json:"question_category"`
		QuestionSubCategory *string                `json:"question_sub_category"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	history := &ChatHistory{
		ID:                  id,
		Message:             req.Message,
		UserID:              req.UserID,
		IsCannotAnswer:      req.IsCannotAnswer,
		Category:            req.Category,
		Feedback:            req.Feedback,
		QuestionCategory:    req.QuestionCategory,
		QuestionSubCategory: req.QuestionSubCategory,
	}

	if err := h.service.UpdateChatHistory(history); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat history updated successfully", history)
}

func (h *ChatHandler) DeleteChatHistory(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid chat history ID")
		return
	}

	if err := h.service.DeleteChatHistory(id); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat history deleted successfully", nil)
}

func (h *ChatHandler) CreateConversation(ctx *gin.Context) {
	var req struct {
		Platform         string  `json:"platform" binding:"required"`
		PlatformUniqueID string  `json:"platform_unique_id" binding:"required"`
		IsHelpdesk       bool    `json:"is_helpdesk"`
		Context          *string `json:"context"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	conv := &Conversation{
		ID:               uuid.New(),
		StartTimestamp:   time.Now(),
		Platform:         req.Platform,
		PlatformUniqueID: req.PlatformUniqueID,
		IsHelpdesk:       req.IsHelpdesk,
		Context:          req.Context,
	}

	if err := h.service.CreateConversation(conv); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(ctx, "Conversation created successfully", conv)
}

func (h *ChatHandler) GetConversations(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	result, err := h.service.GetAllConversations(page, pageSize)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Conversations retrieved successfully", result)
}

func (h *ChatHandler) GetConversationByID(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	conv, err := h.service.GetConversationByID(id)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusNotFound, "Conversation not found")
		return
	}

	util.SuccessResponse(ctx, "Conversation retrieved successfully", conv)
}

func (h *ChatHandler) UpdateConversation(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	var req struct {
		EndTimestamp     *time.Time `json:"end_timestamp"`
		Platform         string     `json:"platform"`
		PlatformUniqueID string     `json:"platform_unique_id"`
		IsHelpdesk       bool       `json:"is_helpdesk"`
		Context          *string    `json:"context"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	conv := &Conversation{
		ID:               id,
		EndTimestamp:     req.EndTimestamp,
		Platform:         req.Platform,
		PlatformUniqueID: req.PlatformUniqueID,
		IsHelpdesk:       req.IsHelpdesk,
		Context:          req.Context,
	}

	if err := h.service.UpdateConversation(conv); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Conversation updated successfully", conv)
}

func (h *ChatHandler) DeleteConversation(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid conversation ID")
		return
	}

	if err := h.service.DeleteConversation(id); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Conversation deleted successfully", nil)
}

func (h *ChatHandler) Ask(ctx *gin.Context) {
	var req struct {
		PlatformUniqueID string `json:"platform_unique_id" binding:"required"`
		Query            string `json:"query" binding:"required"`
		ConversationID   string `json:"conversation_id"`
		Platform         string `json:"platform" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	chatReq := external.ChatRequest{
		PlatformUniqueID: req.PlatformUniqueID,
		Query:            req.Query,
		ConversationID:   req.ConversationID,
		Platform:         req.Platform,
	}

	resp, err := h.externalClient.SendChatMessage(chatReq)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	conversationID, err := uuid.Parse(resp.ConversationID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Invalid conversation ID from external API")
		return
	}

	conv, err := h.service.GetConversationByID(conversationID)
	if err != nil {

		conv = &Conversation{
			ID:               conversationID,
			StartTimestamp:   time.Now(),
			Platform:         req.Platform,
			PlatformUniqueID: req.PlatformUniqueID,
			IsHelpdesk:       resp.IsHelpdesk,
			Context:          nil,
		}
		if err := h.service.CreateConversation(conv); err != nil {

		}
	}

	userMessage := Message{
		"role":    "user",
		"content": req.Query,
	}
	userHistory := &ChatHistory{
		SessionID:           conversationID,
		Message:             userMessage,
		Category:            &resp.Category,
		QuestionCategory:    stringSliceToString(resp.QuestionCategory),
		QuestionSubCategory: nil,
	}
	if err := h.service.CreateChatHistory(userHistory); err != nil {

	}

	assistantMessage := Message{
		"role":    "assistant",
		"content": resp.Answer,
	}
	isCannotAnswer := resp.IsAnswered != nil && !*resp.IsAnswered
	assistantHistory := &ChatHistory{
		SessionID:      conversationID,
		Message:        assistantMessage,
		IsCannotAnswer: &isCannotAnswer,
		Category:       &resp.Category,
	}
	if err := h.service.CreateChatHistory(assistantHistory); err != nil {

	}

	util.SuccessResponse(ctx, "Message sent successfully", resp)
}

func stringSliceToString(slice []string) *string {
	if len(slice) == 0 {
		return nil
	}
	result := strings.Join(slice, ", ")
	return &result
}

func (h *ChatHandler) GetChatPairsBySessionID(ctx *gin.Context) {
	sessionIDParam := ctx.Param("session_id")

	var sessionID *uuid.UUID
	if sessionIDParam != "" && sessionIDParam != "all" {
		parsed, err := uuid.Parse(sessionIDParam)
		if err != nil {
			util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid session ID")
			return
		}
		sessionID = &parsed
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	result, err := h.service.GetChatPairsBySessionID(sessionID, page, pageSize)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat pairs retrieved successfully", result)
}

func (h *ChatHandler) DebugChatHistory(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("session_id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid session ID")
		return
	}

	var histories []ChatHistory
	query := `
		SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
		       category, feedback, question_category, question_sub_category
		FROM chat_history
		WHERE session_id = $1
		ORDER BY created_at ASC
	`
	err = h.service.repo.db.Select(&histories, query, sessionID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	debug := make([]map[string]interface{}, 0)
	for _, h := range histories {
		info := map[string]interface{}{
			"id":      h.ID,
			"message": h.Message,
			"role":    getMessageRole(h.Message),
		}
		debug = append(debug, info)
	}

	util.SuccessResponse(ctx, "Debug info", debug)
}
