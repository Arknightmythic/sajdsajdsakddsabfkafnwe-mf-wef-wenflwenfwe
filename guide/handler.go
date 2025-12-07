package guide

import (
	"context"
	"dokuprime-be/config"
	"dokuprime-be/util"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const isInvalidID = "Invalid ID"

type GuideHandler struct {
	service *GuideService
	redis   *redis.Client
}

func NewGuideHandler(service *GuideService, redisClient *redis.Client) *GuideHandler {
	return &GuideHandler{
		service: service,
		redis:   redisClient,
	}
}

func (h *GuideHandler) UploadGuide(c *gin.Context) {
	title := c.PostForm("title")
	description := c.PostForm("description")
	
	file, err := c.FormFile("file")
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "File is required")
		return
	}

	if title == "" {
		util.ErrorResponse(c, http.StatusBadRequest, "Title is required")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		util.ErrorResponse(c, http.StatusBadRequest, "Only PDF files are allowed")
		return
	}

	guide, err := h.service.UploadGuide(title, description, file)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(c, "Guide uploaded successfully", guide)
}

func (h *GuideHandler) GetAll(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := GuideFilter{
		Search:        c.Query("search"),
		SortBy:        c.Query("sort_by"),
		SortDirection: c.Query("sort_direction"),
		Limit:         limit,
		Offset:        offset,
	}

	guides, total, err := h.service.GetAll(filter)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "Guides retrieved successfully", gin.H{
		"data":   guides,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *GuideHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidID)
		return
	}

	guide, err := h.service.GetByID(id)
	if err != nil {
		util.ErrorResponse(c, http.StatusNotFound, "Guide not found")
		return
	}

	util.SuccessResponse(c, "Guide retrieved successfully", guide)
}

func (h *GuideHandler) UpdateGuide(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidID)
		return
	}

	title := c.PostForm("title")
	description := c.PostForm("description")
	file, _ := c.FormFile("file")

	if title == "" {
		util.ErrorResponse(c, http.StatusBadRequest, "Title is required")
		return
	}

	if file != nil {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".pdf" {
			util.ErrorResponse(c, http.StatusBadRequest, "Only PDF files are allowed")
			return
		}
	}

	guide, err := h.service.UpdateGuide(id, title, description, file)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "Guide updated successfully", guide)
}

func (h *GuideHandler) DeleteGuide(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, isInvalidID)
		return
	}

	if err := h.service.DeleteGuide(id); err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(c, "Guide deleted successfully", nil)
}

func (h *GuideHandler) GenerateViewURL(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	token, err := h.service.GenerateViewTokenByID(req.ID)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	scheme := "https"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	viewURL := fmt.Sprintf("%s/api/guides/view-file?token=%s", baseURL, token)

	util.SuccessResponse(c, "View URL generated successfully", gin.H{
		"url": viewURL,
	})
}

func (h *GuideHandler) ViewFile(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		util.ErrorResponse(c, http.StatusBadRequest, "Token is required")
		return
	}

	key := "view_guide_token:" + token
	ctxRedis := context.Background()

	filename, err := h.redis.Get(ctxRedis, key).Result()
	if err == redis.Nil {
		util.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to validate token")
		return
	}

	h.redis.Del(ctxRedis, key)

	filePath := config.GetDocumentPath(filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		util.ErrorResponse(c, http.StatusNotFound, "File not found on server")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))
	c.File(filePath)
}