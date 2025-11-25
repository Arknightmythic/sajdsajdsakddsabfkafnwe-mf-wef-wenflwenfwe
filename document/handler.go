package document

import (
	"context"
	"dokuprime-be/config"
	"dokuprime-be/util"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type DocumentHandler struct {
	service *DocumentService
	redis   *redis.Client
}

func NewDocumentHandler(service *DocumentService, redisClient *redis.Client) *DocumentHandler {
	return &DocumentHandler{
		service: service,
		redis:   redisClient,
	}
}

func (h *DocumentHandler) GenerateViewURL(ctx *gin.Context) {
	var req struct {
		Filename string `json:"filename" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Filename is required")
		return
	}

	token, err := h.service.GenerateViewToken(req.Filename)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	baseURL := "https://" + ctx.Request.Host
	viewURL := fmt.Sprintf("%s/api/documents/view-file?token=%s", baseURL, token)

	util.SuccessResponse(ctx, "View URL generated successfully", gin.H{
		"url": viewURL,
	})
}

func (h *DocumentHandler) GenerateViewURLByID(ctx *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Document ID is required")
		return
	}

	
	token, err := h.service.GenerateViewTokenByID(req.ID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	
	scheme := "https"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, ctx.Request.Host)
	
	
	viewURL := fmt.Sprintf("%s/api/documents/view-file?token=%s", baseURL, token)

	util.SuccessResponse(ctx, "View URL generated successfully", gin.H{
		"url": viewURL,
	})
}

func (h *DocumentHandler) ViewDocument(ctx *gin.Context) {
	token := ctx.Query("token")
	if token == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Token is required")
		return
	}

	key := "view_token:" + token
	ctxRedis := context.Background()

	filename, err := h.redis.Get(ctxRedis, key).Result()
	if err == redis.Nil {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to validate token with GET: %v", err)
		util.ErrorResponse(ctx, http.StatusInternalServerError, errorMsg)
		return
	}

	h.redis.Del(ctxRedis, key)

	filePath := config.GetDocumentPath(filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		util.ErrorResponse(ctx, http.StatusNotFound, "File not found")
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to open file")
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to get file info")
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))
	contentType := "application/octet-stream"
	if ext == ".pdf" {
		contentType = "application/pdf"
	} else if ext == ".txt" {
		contentType = "text/plain"
	}

	ctx.Header("Content-Description", "File View")
	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	ctx.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))

	io.Copy(ctx.Writer, file)
}

func (h *DocumentHandler) getTeamNameForUser(ctx *gin.Context) string {
	userID, exists := ctx.Get("user_id")
	if !exists {
		return ""
	}

	
	teamName, err := h.service.GetTeamNameByUserID(userID.(int64))
	if err == nil && teamName != "" {
		return teamName
	}

	
	accountType, exists := ctx.Get("account_type")
	if exists {
		return accountType.(string)
	}
	
	return "Unknown"
}

func (h *DocumentHandler) UploadDocument(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		util.ErrorResponse(ctx, http.StatusBadRequest, "At least one file is required")
		return
	}

	category := ctx.PostForm("category")
	if category == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Category is required")
		return
	}

	email, exists := ctx.Get("email")
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "User email not found")
		return
	}

	
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "Account type not found")
		return
	}
	teamName := h.getTeamNameForUser(ctx)

	uploadDir := config.GetUploadPath()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	validTypes := map[string]bool{"pdf": true, "txt": true}

	maxFileSizeFromEnv, err := strconv.Atoi(os.Getenv("MAX_FILE_SIZE_ALLOWED"))
	if err != nil {
		maxFileSizeFromEnv = 70
	}
	maxFileSize := maxFileSizeFromEnv * 1024 * 1024

	var uploadedDocuments []map[string]interface{}
	var failedUploads []map[string]string

	for _, file := range files {
		originalFilename := file.Filename

		if file.Size > int64(maxFileSize) {
			failedUploads = append(failedUploads, map[string]string{
				"filename": originalFilename,
				"reason":   fmt.Sprintf("File size exceeds maximum limit of %d MB", maxFileSize/(1024*1024)),
			})
			continue
		}

		ext := strings.ToLower(filepath.Ext(originalFilename))
		dataType := strings.TrimPrefix(ext, ".")

		if !validTypes[dataType] {
			failedUploads = append(failedUploads, map[string]string{
				"filename": originalFilename,
				"reason":   "Invalid file type. Only PDF and TXT are allowed",
			})
			continue
		}

		uniqueFilename := GenerateUniqueFilename(originalFilename)
		filePath := filepath.Join(uploadDir, uniqueFilename)

		if err := ctx.SaveUploadedFile(file, filePath); err != nil {
			failedUploads = append(failedUploads, map[string]string{
				"filename": originalFilename,
				"reason":   fmt.Sprintf("Failed to save file: %v", err),
			})
			continue
		}

		document := &Document{
			Category: category,
		}

		isLatest := true
		pendingStatus := "Pending"
		detail := &DocumentDetail{
			DocumentName: originalFilename,
			Filename:     uniqueFilename,
			DataType:     dataType,
			Staff:        email.(string),
			Team:         teamName,
			Status:       &pendingStatus,
			IsLatest:     &isLatest,
			IsApprove:    nil,
		}

		if err := h.service.CreateDocument(document, detail); err != nil {
			if removeErr := os.Remove(filePath); removeErr != nil {
				log.Printf("Warning: Failed to remove file %s after DB error: %v", filePath, removeErr)
			}
			failedUploads = append(failedUploads, map[string]string{
				"filename": originalFilename,
				"reason":   fmt.Sprintf("Database error: %v", err),
			})
			continue
		}

		uploadedDocuments = append(uploadedDocuments, map[string]interface{}{
			"document":        document,
			"document_detail": detail,
		})
	}

	response := gin.H{
		"uploaded_count": len(uploadedDocuments),
		"failed_count":   len(failedUploads),
		"uploaded":       uploadedDocuments,
	}

	if len(failedUploads) > 0 {
		response["failed"] = failedUploads
	}

	if len(uploadedDocuments) == 0 {
		util.ErrorResponse(ctx, http.StatusBadRequest, "No files were uploaded successfully")
		return
	}

	statusCode := http.StatusCreated
	message := "Documents uploaded successfully"

	if len(failedUploads) > 0 {
		statusCode = http.StatusMultiStatus
		message = "Some documents uploaded successfully, some failed"
	}

	ctx.JSON(statusCode, gin.H{
		"success": true,
		"message": message,
		"data":    response,
	})
}

func (h *DocumentHandler) GetDocuments(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var startDatePtr, endDatePtr *time.Time
	if sd := ctx.Query("start_date"); sd != "" {
		if t, err := parseDate(sd); err == nil {
			startDatePtr = &t
		}
	}
	if ed := ctx.Query("end_date"); ed != "" {
		if t, err := parseDate(ed); err == nil {
			endDatePtr = &t
		}
	}

	filter := DocumentFilter{
		Search:        ctx.Query("search"),
		DataType:      ctx.Query("data_type"),
		Category:      ctx.Query("category"),
		Status:        ctx.Query("status"),
		Limit:         limit,
		Offset:        offset,
		SortBy:        ctx.Query("sort_by"),
		SortDirection: ctx.Query("sort_direction"),
		StartDate:     startDatePtr,
		EndDate:       endDatePtr,
	}

	documents, total, err := h.service.GetAllDocuments(filter)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"documents": documents,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"filters": map[string]interface{}{
			"search":    filter.Search,
			"data_type": filter.DataType,
			"category":  filter.Category,
			"status":    filter.Status,
		},
	}

	util.SuccessResponse(ctx, "Documents retrieved successfully", response)
}

func (h *DocumentHandler) GetDocumentDetails(ctx *gin.Context) {
	documentID, err := strconv.Atoi(ctx.Query("document_id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid document_id")
		return
	}

	details, err := h.service.GetDocumentDetailsByDocumentID(documentID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Document details retrieved successfully", details)
}

func (h *DocumentHandler) UpdateDocument(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "File is required")
		return
	}

	idStr := ctx.PostForm("id")
	if idStr == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Document ID is required")
		return
	}

	documentID, err := strconv.Atoi(idStr)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid document ID")
		return
	}

	email, exists := ctx.Get("email")
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "User email not found")
		return
	}

	
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "Account type not found")
		return
	}

	teamName := h.getTeamNameForUser(ctx)

	originalFilename := file.Filename
	ext := strings.ToLower(filepath.Ext(originalFilename))
	dataType := strings.TrimPrefix(ext, ".")

	validTypes := map[string]bool{"pdf": true, "txt": true}
	if !validTypes[dataType] {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid file type. Only PDF and TXT are allowed")
		return
	}

	uniqueFilename := GenerateUniqueFilename(originalFilename)

	uploadDir := config.GetUploadPath()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	filePath := filepath.Join(uploadDir, uniqueFilename)
	if err := ctx.SaveUploadedFile(file, filePath); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to save file")
		return
	}

	pendingStatus := "Pending"
	detail := &DocumentDetail{
		DocumentName: originalFilename,
		Filename:     uniqueFilename,
		DataType:     dataType,
		Staff:        email.(string),
		Team:         teamName,
		Status:       &pendingStatus,
		IsApprove:    nil,
	}

	if err := h.service.UpdateDocument(documentID, detail); err != nil {
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Printf("Warning: Failed to remove file %s after DB error: %v", filePath, removeErr)
		}
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Document updated successfully", detail)
}

func (h *DocumentHandler) ApproveDocument(ctx *gin.Context) {
	detailID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid document detail ID")
		return
	}

	if err := h.service.ApproveDocument(detailID); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Document approved successfully", nil)
}

func (h *DocumentHandler) RejectDocument(ctx *gin.Context) {
	detailID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid document detail ID")
		return
	}

	if err := h.service.RejectDocument(detailID); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Document rejected successfully", nil)
}

func (h *DocumentHandler) DeleteDocument(ctx *gin.Context) {
	documentID, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid document ID")
		return
	}

	if err := h.service.DeleteDocument(documentID); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Document and all related details deleted successfully", nil)
}

func (h *DocumentHandler) DownloadDocument(ctx *gin.Context) {
	filename := ctx.Param("filename")
	if filename == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Filename is required")
		return
	}

	filePath := config.GetDocumentPath(filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		util.ErrorResponse(ctx, http.StatusNotFound, "File not found")
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to open file")
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to get file info")
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Transfer-Encoding", "binary")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	io.Copy(ctx.Writer, file)
}

func (h *DocumentHandler) GetAllDocumentDetails(ctx *gin.Context) {
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var startDatePtr, endDatePtr *time.Time
	if sd := ctx.Query("start_date"); sd != "" {
		if t, err := parseDate(sd); err == nil {
			startDatePtr = &t
		}
	}
	if ed := ctx.Query("end_date"); ed != "" {
		if t, err := parseDate(ed); err == nil {
			endDatePtr = &t
		}
	}

	filter := DocumentDetailFilter{
		Search:        ctx.Query("search"),
		DataType:      ctx.Query("data_type"),
		Category:      ctx.Query("category"),
		Status:        ctx.Query("status"),
		DocumentName:  ctx.Query("document_name"),
		Limit:         limit,
		Offset:        offset,
		SortBy:        ctx.Query("sort_by"),
		SortDirection: ctx.Query("sort_direction"),
		StartDate:     startDatePtr,
		EndDate:       endDatePtr,
	}

	details, total, err := h.service.GetAllDocumentDetails(filter)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]interface{}{
		"document_details": details,
		"total":            total,
		"limit":            limit,
		"offset":           offset,
		"filters": map[string]interface{}{
			"search":        filter.Search,
			"data_type":     filter.DataType,
			"category":      filter.Category,
			"status":        filter.Status,
			"document_name": filter.DocumentName,
		},
	}

	util.SuccessResponse(ctx, "Document details retrieved successfully", response)
}

func (h *DocumentHandler) GetQueueStatus(ctx *gin.Context) {
	queueSize := h.service.GetExtractionQueueSize()

	response := gin.H{
		"pending_jobs": queueSize,
		"message":      fmt.Sprintf("There are %d extraction jobs in the queue", queueSize),
	}

	util.SuccessResponse(ctx, "Queue status retrieved successfully", response)
}

func (h *DocumentHandler) BatchUploadDocument(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		util.ErrorResponse(ctx, http.StatusBadRequest, "At least one file is required")
		return
	}

	category := ctx.PostForm("category")
	if category == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Category is required")
		return
	}

	email, exists := ctx.Get("email")
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "User email not found")
		return
	}

	
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "Account type not found")
		return
	}
	teamName := h.getTeamNameForUser(ctx)

	autoApproveStr := ctx.DefaultPostForm("auto_approve", "false")
	autoApprove := autoApproveStr == "true"

	batchID, err := h.service.StartBatchUpload(files, category, email.(string), teamName, autoApprove)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Batch upload started", gin.H{
		"batch_id":     batchID,
		"total_files":  len(files),
		"auto_approve": autoApprove,
		"message":      "Files are being processed. Use the batch_id to check status",
	})
}

func (h *DocumentHandler) GetBatchUploadStatus(ctx *gin.Context) {
	batchID := ctx.Query("batch_id")
	if batchID == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "batch_id is required")
		return
	}

	status, err := h.service.GetBatchStatus(batchID)
	if err != nil {
		if err.Error() == "batch not found" {
			util.ErrorResponse(ctx, http.StatusNotFound, "Batch ID not found or expired")
			return
		}
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Batch status retrieved successfully", status)
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


type BatchDeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

func (h *DocumentHandler) BatchDeleteDocument(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, "Invalid request body. 'ids' array is required.")
		return
	}

	successCount, errors := h.service.BatchDeleteDocuments(req.IDs)

	if len(errors) > 0 && successCount == 0 {
		// Jika gagal semua
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete all selected documents")
		return
	}

	responseMsg := fmt.Sprintf("Successfully deleted %d documents", successCount)
	if len(errors) > 0 {
		responseMsg = fmt.Sprintf("Deleted %d documents with %d failures", successCount, len(errors))
	}

	// Menggunakan StatusMultiStatus (207) atau OK (200) tergantung preferensi, di sini pakai 200 agar umum
	util.SuccessResponse(c, responseMsg, gin.H{
		"success_count": successCount,
		"failed_count":  len(errors),
		"errors":        errors, // Opsional: kirim detail error jika perlu
	})
}