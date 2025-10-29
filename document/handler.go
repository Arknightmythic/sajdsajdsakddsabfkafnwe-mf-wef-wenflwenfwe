package document

import (
	"dokuprime-be/util"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type DocumentHandler struct {
	service *DocumentService
}

func NewDocumentHandler(service *DocumentService) *DocumentHandler {
	return &DocumentHandler{service: service}
}

func (h *DocumentHandler) UploadDocument(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		util.ErrorResponse(ctx, http.StatusBadRequest, "File is required")
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

	accountType, exists := ctx.Get("account_type")
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "Account type not found")
		return
	}

	originalFilename := file.Filename
	ext := strings.ToLower(filepath.Ext(originalFilename))
	dataType := strings.TrimPrefix(ext, ".")

	validTypes := map[string]bool{"pdf": true, "docx": true, "txt": true, "doc": true}
	if !validTypes[dataType] {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid file type. Only PDF, DOCX, and TXT are allowed")
		return
	}

	uniqueFilename := GenerateUniqueFilename(originalFilename)

	uploadDir := "./uploads/documents"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	filePath := filepath.Join(uploadDir, uniqueFilename)
	if err := ctx.SaveUploadedFile(file, filePath); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to save file")
		return
	}

	document := &Document{
		Category: category,
	}

	isLatest := true
	detail := &DocumentDetail{
		DocumentName: originalFilename,
		Filename:     uniqueFilename,
		DataType:     dataType,
		Staff:        email.(string),
		Team:         accountType.(string),
		Status:       nil,
		IsLatest:     &isLatest,
		IsApprove:    nil,
	}

	if err := h.service.CreateDocument(document, detail); err != nil {
		os.Remove(filePath)
		util.ErrorResponse(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	util.CreatedResponse(ctx, "Document uploaded successfully", gin.H{
		"document":        document,
		"document_detail": detail,
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

	filter := DocumentFilter{
		Search:   ctx.Query("search"),
		DataType: ctx.Query("data_type"),
		Category: ctx.Query("category"),
		Status:   ctx.Query("status"),
		Limit:    limit,
		Offset:   offset,
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

	accountType, exists := ctx.Get("account_type")
	if !exists {
		util.ErrorResponse(ctx, http.StatusUnauthorized, "Account type not found")
		return
	}

	originalFilename := file.Filename
	ext := strings.ToLower(filepath.Ext(originalFilename))
	dataType := strings.TrimPrefix(ext, ".")

	validTypes := map[string]bool{"pdf": true, "docx": true, "txt": true, "doc": true}
	if !validTypes[dataType] {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Invalid file type. Only PDF, DOCX, and TXT are allowed")
		return
	}

	uniqueFilename := GenerateUniqueFilename(originalFilename)

	uploadDir := "./uploads/documents"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to create upload directory")
		return
	}

	filePath := filepath.Join(uploadDir, uniqueFilename)
	if err := ctx.SaveUploadedFile(file, filePath); err != nil {
		util.ErrorResponse(ctx, http.StatusInternalServerError, "Failed to save file")
		return
	}

	detail := &DocumentDetail{
		DocumentName: originalFilename,
		Filename:     uniqueFilename,
		DataType:     dataType,
		Staff:        email.(string),
		Team:         accountType.(string),
		Status:       nil,
		IsApprove:    nil,
	}

	if err := h.service.UpdateDocument(documentID, detail); err != nil {
		os.Remove(filePath)
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

func (h *DocumentHandler) DownloadDocument(ctx *gin.Context) {
	filename := ctx.Param("filename")
	if filename == "" {
		util.ErrorResponse(ctx, http.StatusBadRequest, "Filename is required")
		return
	}

	filePath := filepath.Join("./uploads/documents", filename)

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
