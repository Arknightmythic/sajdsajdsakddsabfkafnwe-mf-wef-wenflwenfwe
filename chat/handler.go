package chat

import (
	"database/sql"
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/helpdesk"
	"dokuprime-be/messaging"
	"dokuprime-be/util"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	invalidRequestBody    = "Invalid request body"
	invalidDateFormat     = "Invalid date format: %v"
	invalidChatHistoryID  = "Invalid chat history ID"
	invalidSessionID      = "Invalid session ID"
	invalidConversationID = "Invalid conversation ID"
	isNotAuthenticated    = "User not authenticated"
)

type ChatHandler struct {
	service         *ChatService
	externalClient  *external.Client
	wsClient        *config.WebSocketClient
	helpdeskService helpdesk.HelpdeskService
	messageService  messaging.MessageService
}

func NewChatHandler(service *ChatService, externalClient *external.Client, wsURL, wsToken string, helpdeskService helpdesk.HelpdeskService, messageService messaging.MessageService) *ChatHandler {
	handler := &ChatHandler{
		service:         service,
		externalClient:  externalClient,
		wsClient:        config.NewWebSocketClient(wsURL, wsToken),
		helpdeskService: helpdeskService,
		messageService:  messageService,
	}

	if err := handler.wsClient.Connect(); err != nil {
		log.Printf("Failed to connect to WebSocket: %v", err)
	}

	return handler
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
		IsAnswered          *bool                  `json:"is_answered"`
		Revision            *string                `json:"revision"`
		IsValidated         *bool                  `json:"is_validated"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidRequestBody)
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
		IsAnswered:          req.IsAnswered,
		Revision:            req.Revision,
		IsValidated:         req.IsValidated,
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

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	startDatePtr, endDatePtr, err := parseDateRange(ctx.Query("start_date"), ctx.Query("end_date"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, fmt.Sprintf(invalidDateFormat, err))
		return
	}

	filter := ChatHistoryFilter{
		SortBy:        ctx.Query("sort_by"),
		SortDirection: ctx.Query("sort_direction"),
		StartDate:     startDatePtr,
		EndDate:       endDatePtr,
		Limit:         pageSize,
		Offset:        (page - 1) * pageSize,
	}

	result, err := h.service.GetAllChatHistory(filter)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat histories retrieved successfully", result)
}

func (h *ChatHandler) GetChatHistoryByID(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidChatHistoryID)
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
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidSessionID)
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	startDatePtr, endDatePtr, err := parseDateRange(ctx.Query("start_date"), ctx.Query("end_date"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, fmt.Sprintf(invalidDateFormat, err))
		return
	}

	filter := ChatHistoryFilter{
		SortBy:        ctx.Query("sort_by"),
		SortDirection: ctx.Query("sort_direction"),
		StartDate:     startDatePtr,
		EndDate:       endDatePtr,
		Limit:         pageSize,
		Offset:        (page - 1) * pageSize,
	}

	result, err := h.service.GetChatHistoryBySessionID(sessionID, filter)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat history retrieved successfully", result)
}

func (h *ChatHandler) UpdateChatHistory(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidChatHistoryID)
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
		IsAnswered          *bool                  `json:"is_answered"`
		Revision            *string                `json:"revision"`
		IsValidated         *bool                  `json:"is_validated"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidRequestBody)
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
		IsAnswered:          req.IsAnswered,
		Revision:            req.Revision,
		IsValidated:         req.IsValidated,
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
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidChatHistoryID)
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
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidRequestBody)
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

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	startDatePtr, endDatePtr, err := parseDateRange(ctx.Query("start_date"), ctx.Query("end_date"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, fmt.Sprintf(invalidDateFormat, err))
		return
	}

	var platformUniqueIDPtr *string
	if val := ctx.Query("platform_unique_id"); val != "" {
		platformUniqueIDPtr = &val
	}

	filter := ConversationFilter{
		SortBy:           ctx.Query("sort_by"),
		SortDirection:    ctx.Query("sort_direction"),
		StartDate:        startDatePtr,
		EndDate:          endDatePtr,
		Limit:            pageSize,
		Offset:           (page - 1) * pageSize,
		PlatformUniqueID: platformUniqueIDPtr,
	}

	result, err := h.service.GetAllConversations(filter)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Conversations retrieved successfully", result)
}

func (h *ChatHandler) GetConversationByID(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidConversationID)
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
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidConversationID)
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
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidRequestBody)
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
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidConversationID)
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
		StartTimestamp   string `json:"start_timestamp,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Println("Line 293", err)
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidRequestBody)
		return
	}

	h.ensureWebSocketConnection()

	conversation, err := h.resolveAskConversation(ctx, req.ConversationID)
	if err != nil {
		return
	}

	if conversation != nil && conversation.IsHelpdesk {
		if handled := h.handleExistingHelpdesk(ctx, conversation, req.Query, req.StartTimestamp); handled {
			return
		}
	}

	chatReq := external.ChatRequest{
		PlatformUniqueID: req.PlatformUniqueID,
		Query:            req.Query,
		ConversationID:   req.ConversationID,
		Platform:         req.Platform,
		StartTimestamp:   req.StartTimestamp,
	}

	resp, err := h.externalClient.SendChatMessage(chatReq)
	if err != nil {
		log.Println("Line 307", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	finalConversation, err := h.ensureConversationFromResponse(req.Platform, req.PlatformUniqueID, resp)
	if err != nil {
		log.Println("Line 331", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Error creating conversation")
		return
	}

	responseAsk := h.processAskResponseData(finalConversation, resp)
	util.SuccessResponse(ctx, "Message sent successfully", responseAsk)
	h.broadcastAskResponse(ctx, finalConversation, responseAsk)
}

func (h *ChatHandler) ensureWebSocketConnection() {
	if !h.wsClient.IsConnected() {
		log.Println("WebSocket not connected, attempting to reconnect...")
		if err := h.wsClient.Connect(); err != nil {
			log.Printf("Failed to connect to WebSocket: %v", err)
		}
	}
}

func (h *ChatHandler) resolveAskConversation(ctx *gin.Context, conversationID string) (*Conversation, error) {
	if conversationID == "" {
		return nil, nil
	}

	parsedUUID, err := uuid.Parse(conversationID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid uuid format")
		return nil, err
	}

	conversation, err := h.service.GetConversationByID(parsedUUID)
	if err != nil && err != sql.ErrNoRows {
		log.Println(err)
		util.ErrorResponse(ctx, http.StatusBadRequest, "Error fetching conversation")
		return nil, err
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return conversation, nil
}

func (h *ChatHandler) handleExistingHelpdesk(ctx *gin.Context, conversation *Conversation, query, startTimestamp string) bool {
	err := h.messageService.HandleHelpdeskMessage(
		conversation.ID,
		query,
		"user",
		conversation.Platform,
		&conversation.PlatformUniqueID,
		startTimestamp,
	)

	if err != nil {
		log.Println("Error handling helpdesk message:", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Error sending message")
		return true
	}

	existingHelpdesk, err := h.helpdeskService.GetBySessionID(conversation.ID.String())
	if err != nil && err != sql.ErrNoRows {
		log.Println("Error checking helpdesk:", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Error get existing helpdesk")
		return true
	}

	if existingHelpdesk == nil {
		err = h.helpdeskService.Create(&helpdesk.Helpdesk{
			SessionID:        conversation.ID.String(),
			Platform:         conversation.Platform,
			PlatformUniqueID: &conversation.PlatformUniqueID,
			Status:           "queue",
		})

		if err != nil {
			log.Println("Error creating helpdesk:", err)
			util.ErrorResponse(ctx, http.StatusInternalServerError, "Error creating helpdesk")
			return true
		}
	}

	responseAsk := ResponseAsk{
		User:             conversation.PlatformUniqueID,
		ConversationID:   conversation.ID.String(),
		Query:            query,
		Answer:           "",
		IsHelpdesk:       true,
		Platform:         conversation.Platform,
		PlatformUniqueID: conversation.PlatformUniqueID,
	}

	util.SuccessResponse(ctx, "Message sent to agent queue", responseAsk)
	return true
}

func (h *ChatHandler) ensureConversationFromResponse(reqPlatform, reqUniqueID string, resp *external.ChatResponse) (*Conversation, error) {
	conversationID, err := uuid.Parse(resp.ConversationID)
	if err != nil {
		log.Println("Line 314", err)
		return nil, fmt.Errorf("invalid conversation ID from external API")
	}

	conversation, err := h.service.GetConversationByID(conversationID)
	if err != nil {

		conversation = &Conversation{
			ID:               conversationID,
			StartTimestamp:   time.Now(),
			Platform:         reqPlatform,
			PlatformUniqueID: reqUniqueID,
			IsHelpdesk:       resp.IsHelpdesk,
			Context:          nil,
		}
		if err := h.service.CreateConversation(conversation); err != nil {
			return nil, err
		}
	}
	return conversation, nil
}

func (h *ChatHandler) processAskResponseData(conversation *Conversation, resp *external.ChatResponse) ResponseAsk {
	var responseAnswer string
	var responseCitations external.FlexibleCitationArray
	var responseQuestionCategory []string

	if resp.IsHelpdesk {
		responseAnswer = "Pesan Anda telah dikirim ke agen. Mohon tunggu balasan."
		responseCitations = external.FlexibleCitationArray{}
		responseQuestionCategory = []string{}

		existingHelpdesk, err := h.helpdeskService.GetBySessionID(resp.ConversationID)
		if err != nil || existingHelpdesk == nil {
			err = h.helpdeskService.Create(&helpdesk.Helpdesk{
				SessionID:        resp.ConversationID,
				Platform:         conversation.Platform,
				PlatformUniqueID: &conversation.PlatformUniqueID,
				Status:           "Queue",
			})
			if err != nil {
				log.Printf("Error creating helpdesk: %v", err)
			}
		}
		if !conversation.IsHelpdesk {
			conversation.IsHelpdesk = true
			if err := h.service.UpdateConversation(conversation); err != nil {
				log.Printf("Failed to update conversation is_helpdesk status: %v", err)
			}
		}
	} else {
		responseAnswer = resp.Answer
		responseCitations = resp.Citations
		responseQuestionCategory = resp.QuestionCategory
	}

	return ResponseAsk{
		User:             resp.User,
		ConversationID:   resp.ConversationID,
		Query:            resp.Query,
		RewrittenQuery:   resp.RewrittenQuery,
		Category:         resp.Category,
		QuestionCategory: responseQuestionCategory,
		Answer:           responseAnswer,
		Citations:        responseCitations,
		IsHelpdesk:       resp.IsHelpdesk,
		IsAnswered:       resp.IsAnswered,
		Platform:         conversation.Platform,
		PlatformUniqueID: conversation.PlatformUniqueID,
		QuestionID:       resp.QuestionID,
		AnswerID:         resp.AnswerID,
	}
}

func (h *ChatHandler) broadcastAskResponse(ctx *gin.Context, conversation *Conversation, responseAsk ResponseAsk) {
	if conversation.Platform == "web" {
		if h.wsClient.IsConnected() {
			channelName := conversation.ID.String()
			publishData := map[string]interface{}{
				"user":               responseAsk.User,
				"conversation_id":    responseAsk.ConversationID,
				"query":              responseAsk.Query,
				"rewritten_query":    responseAsk.RewrittenQuery,
				"category":           responseAsk.Category,
				"question_category":  responseAsk.QuestionCategory,
				"answer":             responseAsk.Answer,
				"citations":          responseAsk.Citations,
				"is_helpdesk":        responseAsk.IsHelpdesk,
				"is_answered":        responseAsk.IsAnswered,
				"platform":           conversation.Platform,
				"platform_unique_id": conversation.PlatformUniqueID,
				"timestamp":          time.Now().Unix(),
				"question_id":        responseAsk.QuestionID,
				"answer_id":          responseAsk.AnswerID,
			}

			if err := h.wsClient.Publish(channelName, publishData); err != nil {
				log.Printf("Failed to publish to channel %s: %v", channelName, err)
			} else {
				log.Printf("✅ Published message to channel: %s", channelName)
			}
		}
	} else {
		err := h.externalClient.SendMessageToAPI(responseAsk)
		if err != nil {
			log.Printf("Error sending to Multi Channel API: %v", err)

		}
	}
}

func (h *ChatHandler) GetChatPairsBySessionID(ctx *gin.Context) {
	sessionIDParam := ctx.Param("session_id")

	var sessionID *uuid.UUID
	if sessionIDParam != "" && sessionIDParam != "all" {
		parsed, err := uuid.Parse(sessionIDParam)
		if err != nil {
			util.ErrorResponse(ctx, http.StatusBadRequest, invalidSessionID)
			return
		}
		sessionID = &parsed
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	startDatePtr, endDatePtr, err := parseDateRange(ctx.Query("start_date"), ctx.Query("end_date"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, fmt.Sprintf(invalidDateFormat, err))
		return
	}

	var isValidatedFilter *string
	if val := ctx.Query("is_validated"); val != "" {
		if val == "null" || val == "0" || val == "1" {
			isValidatedFilter = &val
		}
	}

	var isAnsweredFilter *bool
	if val := ctx.Query("is_answered"); val != "" {
		boolVal := val == "true" || val == "1"
		isAnsweredFilter = &boolVal
	} else {
		defaultFalse := false
		isAnsweredFilter = &defaultFalse
	}

	filter := ChatHistoryFilter{
		SortBy:        ctx.Query("sort_by"),
		SortDirection: ctx.Query("sort_direction"),
		StartDate:     startDatePtr,
		EndDate:       endDatePtr,
		Limit:         pageSize,
		Offset:        (page - 1) * pageSize,
		IsValidated:   isValidatedFilter,
		IsAnswered:    isAnsweredFilter,
		Search:        ctx.Query("search"),
	}

	result, err := h.service.GetChatPairsBySessionID(sessionID, filter)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Chat pairs retrieved successfully", result)
}

func (h *ChatHandler) DebugChatHistory(ctx *gin.Context) {
	sessionID, err := uuid.Parse(ctx.Param("session_id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidSessionID)
		return
	}

	var histories []ChatHistory
	query := `
		SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
			   category, feedback, question_category, question_sub_category, is_answered, revision
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

func (h *ChatHandler) ValidateAnswer(ctx *gin.Context) {
	var req struct {
		QuestionID int    `json:"question_id" binding:"required"`
		Question   string `json:"question" binding:"required"`
		AnswerID   int    `json:"answer_id" binding:"required"`
		Answer     string `json:"answer" binding:"required"`
		Revision   string `json:"revision"`
		Validate   bool   `json:"validate"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidRequestBody)
		return
	}

	if req.Revision == "" {
		req.Revision = req.Answer
	}

	userID, exists := ctx.Get("user_id")
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, isNotAuthenticated)
		return
	}

	if err := h.service.UpdateIsAnsweredStatus(req.QuestionID, req.AnswerID, req.Revision, req.Validate, userID); err != nil {
		log.Println("Error updating is_answered status:", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to update validation status")
		return
	}

	if req.Validate {
		tempFileName := fmt.Sprintf("qa_%d_%d.txt", req.QuestionID, req.AnswerID)
		tempFilePath := filepath.Join(os.TempDir(), tempFileName)

		content := fmt.Sprintf("Q:%s\nA:%s", req.Question, "")
		if err := os.WriteFile(tempFilePath, []byte(content), 0644); err != nil {
			log.Println("Error creating temp file:", err)
			util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to create temporary file")
			return
		}

		extractReq := external.ExtractRequest{
			ID:       "faq-" + strconv.Itoa(req.AnswerID),
			Category: "qna",
			Filename: tempFileName,
			FilePath: tempFilePath,
		}

		if err := h.externalClient.ExtractDocument(extractReq); err != nil {
			log.Println("Error extracting document:", err)
			os.Remove(tempFilePath)
			util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to upload document to external API")
			return
		}

		if err := os.Remove(tempFilePath); err != nil {
			log.Println("Warning: Failed to delete temp file:", err)
		}
	}

	util.SuccessResponse(ctx, "Answer validation updated successfully", gin.H{
		"question_id": req.QuestionID,
		"answer_id":   req.AnswerID,
		"validate":    req.Validate,
	})
}

func parseDate(s string) (time.Time, error) {
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}

func parseDateRange(startDateStr, endDateStr string) (*time.Time, *time.Time, error) {
	var startDatePtr, endDatePtr *time.Time

	if startDateStr != "" {
		t, err := parseDate(startDateStr)
		if err != nil {
			return nil, nil, err
		}
		startDatePtr = &t
	}

	if endDateStr != "" {
		t, err := parseDate(endDateStr)
		if err != nil {
			return nil, nil, err
		}

		endDatePtr = &t
	}

	return startDatePtr, endDatePtr, nil
}

func (h *ChatHandler) Feedback(ctx *gin.Context) {
	var req struct {
		AnswerID  int       `json:"answer_id,omitempty"`
		SessionID uuid.UUID `json:"session_id,omitempty"`
		Feedback  bool      `json:"feedback,omitempty"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, invalidRequestBody)
		return
	}

	err := h.service.Feedback(req.AnswerID, req.SessionID, req.Feedback)
	if err != nil {
		log.Println("Error updating feedback status:", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to update feedback status")
		return
	}

	util.SuccessResponse(ctx, "Feedback updated successfully", nil)
}

func (h *ChatHandler) Close() error {
	return h.wsClient.Close()
}

func (h *ChatHandler) DownloadChatHistory(ctx *gin.Context) {
	// Parse query parameters
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")
	typeFilter := ctx.DefaultQuery("type", "all") // all, human, ai

	// Validate type parameter
	if typeFilter != "all" && typeFilter != "human" && typeFilter != "ai" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid type parameter. Must be 'all', 'human', or 'ai'")
		return
	}

	// Parse dates
	startDatePtr, endDatePtr, err := parseDateRange(startDateStr, endDateStr)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, fmt.Sprintf(invalidDateFormat, err))
		return
	}

	// Get chat histories from service
	histories, err := h.service.GetChatHistoriesForDownload(startDatePtr, endDatePtr, typeFilter)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate CSV
	csvData, err := generateChatHistoryCSV(histories)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to generate CSV: "+err.Error())
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("chat_history_%s_%s_%s.csv",
		typeFilter,
		time.Now().Format("20060102"),
		time.Now().Format("150405"))

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", "attachment; filename="+filename)
	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Transfer-Encoding", "binary")

	// Add BOM for Excel UTF-8 compatibility
	ctx.Data(http.StatusOK, "text/csv; charset=utf-8", append([]byte{0xEF, 0xBB, 0xBF}, csvData...))
}

func generateChatHistoryCSV(histories []ChatHistory) ([]byte, error) {
	var buf strings.Builder

	// Write CSV header
	headers := []string{
		"ID",
		"Session ID",
		"Type",
		"Content",
		"Created At",
		"User ID",
		"Is Cannot Answer",
		"Category",
		"Feedback",
		"Question Category",
		"Question Sub Category",
		"Is Answered",
		"Revision",
		"Is Validated",
		"Start Timestamp",
	}
	buf.WriteString(strings.Join(headers, ",") + "\n")

	// Write data rows
	for _, history := range histories {
		// Extract message type and content from JSON
		messageType := ""
		content := ""

		if dataMap, ok := history.Message["data"].(map[string]interface{}); ok {
			if t, ok := dataMap["type"].(string); ok {
				messageType = t
			}
			if c, ok := dataMap["content"].(string); ok {
				content = escapeCSV(c)
			}
		}

		row := []string{
			fmt.Sprintf("%d", history.ID),
			history.SessionID.String(),
			messageType,
			content,
			history.CreatedAt.Format("2006-01-02 15:04:05"),
			formatNullableInt64(history.UserID),
			formatNullableBool(history.IsCannotAnswer),
			escapeCSV(formatNullableString(history.Category)),
			formatNullableBool(history.Feedback),
			escapeCSV(formatNullableString(history.QuestionCategory)),
			escapeCSV(formatNullableString(history.QuestionSubCategory)),
			formatNullableBool(history.IsAnswered),
			escapeCSV(formatNullableString(history.Revision)),
			formatNullableBool(history.IsValidated),
			formatTimestamp(history.StartTimestamp),
		}

		buf.WriteString(strings.Join(row, ",") + "\n")
	}

	return []byte(buf.String()), nil
}

func escapeCSV(field string) string {
	// If field contains comma, newline, or quotes, wrap it in quotes
	if strings.Contains(field, ",") || strings.Contains(field, "\n") || strings.Contains(field, "\"") {
		// Escape existing quotes by doubling them
		field = strings.ReplaceAll(field, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", field)
	}
	return field
}

func formatNullableInt64(val *int64) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%d", *val)
}

func formatNullableBool(val *bool) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%t", *val)
}

func formatNullableString(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}

func formatTimestamp(val string) string {
	if val == "" {
		return ""
	}
	return val
}
