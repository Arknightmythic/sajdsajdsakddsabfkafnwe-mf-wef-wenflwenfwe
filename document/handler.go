package document

import (
	"context"
	"dokuprime-be/config"
	"dokuprime-be/util"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	urlViewFile             = "%s/api/documents/view-file?token=%s"
	successViewResponse     = "View URL generated successfully"
	emailNotFoundResponse   = "User email not found"
	accountNotFoundResponse = "Account type not found"
	failedParseFormResponse = "Failed to parse multipart form"
)

type FileUploadConfig struct {
	MaxFileSize int
	ValidTypes  map[string]bool
}

type UploadContext struct {
	Category  string
	Email     string
	TeamName  string
	UploadDir string
}

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
	viewURL := fmt.Sprintf(urlViewFile, baseURL, token)

	util.SuccessResponse(ctx, successViewResponse, gin.H{
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

	viewURL := fmt.Sprintf(urlViewFile, baseURL, token)

	util.SuccessResponse(ctx, successViewResponse, gin.H{
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
		util.ErrorResponse(ctx, http.StatusBadRequest, failedParseFormResponse)
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
		util.ErrorResponse(ctx, http.StatusUnauthorized, emailNotFoundResponse)
		return
	}
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, accountNotFoundResponse)
		return
	}
	teamName := h.getTeamNameForUser(ctx)
	uploadDir := config.GetUploadPath()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	maxFileSize, validTypes := h.getUploadConfig()

	// Create configuration structs
	uploadConfig := FileUploadConfig{
		MaxFileSize: maxFileSize,
		ValidTypes:  validTypes,
	}

	uploadCtx := UploadContext{
		Category:  category,
		Email:     email.(string),
		TeamName:  teamName,
		UploadDir: uploadDir,
	}

	var uploadedDocuments []map[string]interface{}
	var failedUploads []map[string]string

	for _, file := range files {
		success, failure := h.processSingleFile(ctx, file, uploadCtx, uploadConfig)
		if failure != nil {
			failedUploads = append(failedUploads, failure)
		} else {
			uploadedDocuments = append(uploadedDocuments, success)
		}
	}
	h.sendUploadResponse(ctx, uploadedDocuments, failedUploads)
}

func (h *DocumentHandler) getUploadConfig() (int, map[string]bool) {
	validTypes := map[string]bool{"pdf": true, "txt": true}

	maxFileSizeFromEnv, err := strconv.Atoi(os.Getenv("MAX_FILE_SIZE_ALLOWED"))
	if err != nil {
		maxFileSizeFromEnv = 70
	}
	maxFileSize := maxFileSizeFromEnv * 1024 * 1024
	return maxFileSize, validTypes
}

func (h *DocumentHandler) processSingleFile(
	ctx *gin.Context,
	file *multipart.FileHeader,
	uploadCtx UploadContext,
	config FileUploadConfig,
) (map[string]interface{}, map[string]string) {
	originalFilename := file.Filename

	if file.Size > int64(config.MaxFileSize) {
		return nil, map[string]string{
			"filename": originalFilename,
			"reason":   fmt.Sprintf("File size exceeds maximum limit of %d MB", config.MaxFileSize/(1024*1024)),
		}
	}

	ext := strings.ToLower(filepath.Ext(originalFilename))
	dataType := strings.TrimPrefix(ext, ".")
	if !config.ValidTypes[dataType] {
		return nil, map[string]string{
			"filename": originalFilename,
			"reason":   "Invalid file type. Only PDF and TXT are allowed",
		}
	}

	uniqueFilename := GenerateUniqueFilename(originalFilename)
	filePath := filepath.Join(uploadCtx.UploadDir, uniqueFilename)
	if err := ctx.SaveUploadedFile(file, filePath); err != nil {
		return nil, map[string]string{
			"filename": originalFilename,
			"reason":   fmt.Sprintf("Failed to save file: %v", err),
		}
	}

	document := &Document{Category: uploadCtx.Category}
	isLatest := true
	pendingStatus := "Pending"
	detail := &DocumentDetail{
		DocumentName: originalFilename,
		Filename:     uniqueFilename,
		DataType:     dataType,
		Staff:        uploadCtx.Email,
		Team:         uploadCtx.TeamName,
		Status:       &pendingStatus,
		IsLatest:     &isLatest,
		IsApprove:    nil,
	}

	if err := h.service.CreateDocument(document, detail); err != nil {
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Printf("Warning: Failed to remove file %s after DB error: %v", filePath, removeErr)
		}
		return nil, map[string]string{
			"filename": originalFilename,
			"reason":   err.Error(),
		}
	}

	return map[string]interface{}{
		"document":        document,
		"document_detail": detail,
	}, nil
}

func (h *DocumentHandler) sendUploadResponse(ctx *gin.Context, uploadedDocuments []map[string]interface{}, failedUploads []map[string]string) {
	response := gin.H{
		"uploaded_count": len(uploadedDocuments),
		"failed_count":   len(failedUploads),
		"uploaded":       uploadedDocuments,
	}

	if len(failedUploads) > 0 {
		response["failed"] = failedUploads
	}

	if len(uploadedDocuments) == 0 {
		statusCode := http.StatusBadRequest
		message := "No files were uploaded successfully"

		if len(failedUploads) > 0 {
			message = failedUploads[0]["reason"]
		}

		util.ErrorResponse(ctx, statusCode, message)
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
		IngestStatus:  ctx.Query("ingest_status"),
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
		util.ErrorResponse(ctx, http.StatusUnauthorized, emailNotFoundResponse)
		return
	}

	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, accountNotFoundResponse)
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

	util.SuccessResponse(ctx, "Request hapus berhasil dikirim. Menunggu persetujuan Admin.", nil)
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
		RequestType:   ctx.Query("request_type"),
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
		util.ErrorResponse(ctx, http.StatusBadRequest, failedParseFormResponse)
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
		util.ErrorResponse(ctx, http.StatusUnauthorized, emailNotFoundResponse)
		return
	}

	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, accountNotFoundResponse)
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
		util.ErrorResponse(c, http.StatusInternalServerError, "Failed to request delete for all selected documents")
		return
	}

	responseMsg := fmt.Sprintf("Successfully requested delete for %d documents", successCount)
	if len(errors) > 0 {
		responseMsg = fmt.Sprintf("Requested %d documents with %d failures", successCount, len(errors))
	}

	util.SuccessResponse(c, responseMsg, gin.H{
		"success_count": successCount,
		"failed_count":  len(errors),
		"errors":        errors,
	})
}

func (h *DocumentHandler) GenerateViewURLByDocumentID(ctx *gin.Context) {
	var req struct {
		DocumentID int `json:"document_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "document_id is required")
		return
	}

	token, err := h.service.GenerateViewTokenByDocumentID(req.DocumentID)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusNotFound, err.Error())
		return
	}

	scheme := "https"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, ctx.Request.Host)
	viewURL := fmt.Sprintf(urlViewFile, baseURL, token)

	util.SuccessResponse(ctx, successViewResponse, gin.H{
		"url": viewURL,
	})
}

func (h *DocumentHandler) CrawlerBatchUpload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		util.ErrorResponse(c, http.StatusBadRequest, failedParseFormResponse)
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		util.ErrorResponse(c, http.StatusBadRequest, "No files uploaded")
		return
	}

	category := c.DefaultPostForm("category", "crawling-data")

	results, err := h.service.ProcessCrawlerBatch(files, category)
	if err != nil {
		util.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	successCount := 0
	replacedCount := 0
	skippedCount := 0
	errorCount := 0

	for _, r := range results {
		switch r.Status {
		case "Uploaded":
			successCount++
		case "Replaced":
			replacedCount++
		case "Skipped":
			skippedCount++
		case "Error":
			errorCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Processed %d files", len(files)),
		"summary": gin.H{
			"uploaded": successCount,
			"replaced": replacedCount,
			"skipped":  skippedCount,
			"errors":   errorCount,
		},
		"details": results,
	})
}

func (h *DocumentHandler) CheckDuplicates(ctx *gin.Context) {
	var req struct {
		Filenames []string `json:"filenames" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid request body")
		return
	}

	duplicates, err := h.service.CheckDuplicates(req.Filenames)
	if err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.SuccessResponse(ctx, "Scan completed", gin.H{
		"duplicates": duplicates,
	})
}
