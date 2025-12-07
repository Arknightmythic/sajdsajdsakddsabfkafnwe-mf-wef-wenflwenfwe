package document

import (
	"context"
	"dokuprime-be/config"
	"dokuprime-be/external"
	"dokuprime-be/util"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const isBatchUpload = "batch_upload:"

type DocumentService struct {
	repo           *DocumentRepository
	redis          *redis.Client
	asyncProcessor *AsyncProcessor
	externalClient *external.Client
}

type FileData struct {
	Filename string
	Size     int64
	Content  []byte
}

func NewDocumentService(repo *DocumentRepository, redisClient *redis.Client, asyncProcessor *AsyncProcessor, externalClient *external.Client) *DocumentService {
	return &DocumentService{
		repo:           repo,
		redis:          redisClient,
		asyncProcessor: asyncProcessor,
		externalClient: externalClient,
	}
}

func (s *DocumentService) GenerateViewToken(filename string) (string, error) {
	token := util.RandString(32)
	key := "view_token:" + token

	ctx := context.Background()
	err := s.redis.Set(ctx, key, filename, 5*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store view token: %w", err)
	}

	return token, nil
}


func (s *DocumentService) GenerateViewTokenByID(id int) (string, error) {
	detail, err := s.repo.GetDocumentDetailByID(id)
	if err != nil {
		return "", fmt.Errorf("document detail not found: %w", err)
	}
	return s.GenerateViewToken(detail.Filename)
}

func (s *DocumentService) CreateDocument(document *Document, detail *DocumentDetail) error {
	if err := s.repo.CreateDocument(document); err != nil {
		return err
	}

	detail.DocumentID = document.ID
	return s.repo.CreateDocumentDetail(detail)
}

func (s *DocumentService) GetAllDocuments(filter DocumentFilter) ([]DocumentWithDetail, int, error) {
	documents, err := s.repo.GetAllDocuments(filter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.GetTotalDocuments(filter)
	if err != nil {
		return nil, 0, err
	}

	return documents, total, nil
}

func (s *DocumentService) GetDocumentDetailsByDocumentID(documentID int) ([]DocumentDetail, error) {
	return s.repo.GetDocumentDetailsByDocumentID(documentID)
}

func (s *DocumentService) UpdateDocument(documentID int, detail *DocumentDetail) error {
	_, err := s.repo.GetDocumentByID(documentID)
	if err != nil {
		return err
	}

	falseValue := false
	detail.IsLatest = &falseValue

	detail.DocumentID = documentID
	return s.repo.CreateDocumentDetail(detail)
}

func (s *DocumentService) ApproveDocument(detailID int) error {
	detail, err := s.repo.GetDocumentDetailByID(detailID)
	if err != nil {
		return fmt.Errorf("failed to get document detail: %w", err)
	}

	if detail.Status != nil && *detail.Status == "Approved" {
		return fmt.Errorf("document is already approved")
	}

	document, err := s.repo.GetDocumentByID(detail.DocumentID)
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	filePath := config.GetDocumentPath(detail.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("document file not found: %s", detail.Filename)
	}

	deleteReq := external.DeleteRequest{
		ID:       detail.DocumentID,
		Category: document.Category,
	}

	if err := s.externalClient.DeleteDocument(deleteReq); err != nil {
		log.Printf("Warning: Failed to delete document from external API (ID: %d): %v", detail.DocumentID, err)

	} else {
		log.Printf("Successfully deleted document from external API (ID: %d)", detail.DocumentID)
	}

	if err := s.repo.UpdateAllDocumentDetailsApprove(detail.DocumentID, false); err != nil {
		return fmt.Errorf("failed to update is_approve for other documents: %w", err)
	}

	if err := s.repo.UpdateDocumentDetailApprove(detailID, true); err != nil {
		return fmt.Errorf("failed to set is_approve: %w", err)
	}

	if err := s.repo.UpdateDocumentDetailStatus(detailID, "Approved"); err != nil {
		return fmt.Errorf("failed to set status to Approved: %w", err)
	}

	if err := s.repo.UpdateDocumentDetailLatest(detail.DocumentID); err != nil {
		return fmt.Errorf("failed to update is_latest for other documents: %w", err)
	}

	if err := s.repo.UpdateDocumentDetailLatestByID(detailID, true); err != nil {
		return fmt.Errorf("failed to set is_latest for approved document: %w", err)
	}

	extractReq := external.ExtractRequest{
		ID:       strconv.Itoa(detail.DocumentID),
		Category: document.Category,
		Filename: detail.DocumentName,
		FilePath: filePath,
	}

	job := ExtractionJob{
		DetailID: detailID,
		Request:  extractReq,
	}

	if err := s.asyncProcessor.SubmitJob(job); err != nil {
		log.Printf("Warning: Failed to submit extraction job for detail ID %d: %v", detailID, err)
	}

	return nil
}

func (s *DocumentService) RejectDocument(detailID int) error {
	if err := s.repo.UpdateDocumentDetailApprove(detailID, false); err != nil {
		return fmt.Errorf("failed to set is_approve to false: %w", err)
	}

	if err := s.repo.UpdateDocumentDetailStatus(detailID, "Rejected"); err != nil {
		return fmt.Errorf("failed to set status to Rejected: %w", err)
	}

	if err := s.repo.UpdateDocumentDetailIngestStatus(detailID, "unprocessed"); err != nil {
		return fmt.Errorf("failed to set ingest_status to unprocessed: %w", err)
	}

	return nil
}

func (s *DocumentService) DeleteDocument(documentID int) error {

	document, err := s.repo.GetDocumentByID(documentID)
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	deleteReq := external.DeleteRequest{
		ID:       documentID,
		Category: document.Category,
	}

	if err := s.externalClient.DeleteDocument(deleteReq); err != nil {
		log.Printf("Warning: Failed to delete document from external API (ID: %d): %v", documentID, err)

	} else {
		log.Printf("Successfully deleted document from external API (ID: %d)", documentID)
	}

	details, err := s.repo.GetDocumentDetailsByDocumentID(documentID)
	if err != nil {
		return fmt.Errorf("failed to get document details: %w", err)
	}

	for _, detail := range details {
		filePath := config.GetDocumentPath(detail.Filename)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: Failed to delete file %s: %v\n", filePath, err)
		}
	}

	if err := s.repo.DeleteDocumentDetails(documentID); err != nil {
		return fmt.Errorf("failed to delete document details: %w", err)
	}

	if err := s.repo.DeleteDocument(documentID); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

func GenerateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	timestamp := time.Now().Unix()
	uniqueID := uuid.New().String()
	return fmt.Sprintf("%d_%s%s", timestamp, uniqueID, ext)
}

func (s *DocumentService) GetAllDocumentDetails(filter DocumentDetailFilter) ([]DocumentDetail, int, error) {
	details, err := s.repo.GetAllDocumentDetails(filter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.GetTotalDocumentDetails(filter)
	if err != nil {
		return nil, 0, err
	}

	return details, total, nil
}

func (s *DocumentService) GetExtractionQueueSize() int {
	return s.asyncProcessor.GetQueueSize()
}

func (s *DocumentService) StartBatchUpload(files []*multipart.FileHeader, category, email, accountType string, autoApprove bool) (string, error) {
	batchID := util.RandString(16)

	fileDataList := make([]FileData, 0, len(files))
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("Failed to open file %s during preparation: %v", fileHeader.Filename, err)
			continue
		}

		content, err := io.ReadAll(file)
		file.Close()

		if err != nil {
			log.Printf("Failed to read file %s during preparation: %v", fileHeader.Filename, err)
			continue
		}

		fileDataList = append(fileDataList, FileData{
			Filename: fileHeader.Filename,
			Size:     fileHeader.Size,
			Content:  content,
		})
	}

	if len(fileDataList) == 0 {
		return "", fmt.Errorf("no valid files to process")
	}

	if err := s.setBatchStatus(batchID, map[string]interface{}{
		"total":        len(fileDataList),
		"processed":    0,
		"successful":   0,
		"failed":       0,
		"extracted":    0,
		"status":       "processing",
		"auto_approve": autoApprove,
		"started_at":   time.Now().Format(time.RFC3339),
	}); err != nil {
		return "", fmt.Errorf("failed to set batch status: %w", err)
	}

	go s.processBatchUpload(batchID, fileDataList, category, email, accountType, autoApprove)

	return batchID, nil
}

type batchStats struct {
	processed   int
	successful  int
	failed      int
	extracted   int
	total       int
	batchID     string
	autoApprove bool
	mu          sync.Mutex
}

func (s *DocumentService) processBatchUpload(batchID string, files []FileData, category, email, accountType string, autoApprove bool) {
	// 1. Prepare Environment
	uploadDir, maxFileSize, validTypes, err := s.prepareBatchEnv(batchID)
	if err != nil {
		return
	}

	// 2. Initialize Stats
	stats := &batchStats{
		total:       len(files),
		batchID:     batchID,
		autoApprove: autoApprove,
	}

	workerCount := 10
	jobs := make(chan FileData, len(files))
	var wg sync.WaitGroup

	// 3. Start Workers
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go s.runBatchWorker(w, jobs, &wg, stats, category, email, accountType, uploadDir, validTypes, maxFileSize)
	}

	// 4. Distribute Jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// 5. Wait and Finalize
	wg.Wait()
	s.finalizeBatch(stats)
}

// ==================================================================================
// HELPER FUNCTIONS (Tambahkan di bawah processBatchUpload)
// ==================================================================================

func (s *DocumentService) prepareBatchEnv(batchID string) (string, int, map[string]bool, error) {
	uploadDir := config.GetUploadPath()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Batch %s: Failed to create upload directory: %v", batchID, err)
		return "", 0, nil, err
	}

	validTypes := map[string]bool{"pdf": true, "docx": true, "txt": true, "doc": true}

	maxFileSizeFromEnv, err := strconv.Atoi(os.Getenv("MAX_FILE_SIZE_ALLOWED"))
	if err != nil {
		maxFileSizeFromEnv = 70
	}
	maxFileSize := maxFileSizeFromEnv * 1024 * 1024

	return uploadDir, maxFileSize, validTypes, nil
}

func (s *DocumentService) runBatchWorker(workerID int, jobs <-chan FileData, wg *sync.WaitGroup, stats *batchStats, category, email, accountType, uploadDir string, validTypes map[string]bool, maxFileSize int) {
	defer wg.Done()

	for file := range jobs {
		documentID, detailID, success := s.processFileDataWithExtraction(
			file, category, email, accountType, uploadDir,
			validTypes, maxFileSize, stats.batchID, workerID, stats.autoApprove,
		)

		s.updateBatchStats(stats, success, documentID, detailID)
	}
}

func (s *DocumentService) updateBatchStats(stats *batchStats, success bool, documentID, detailID int) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.processed++
	if success {
		stats.successful++
		if stats.autoApprove && documentID > 0 && detailID > 0 {
			stats.extracted++
		}
	} else {
		stats.failed++
	}

	if stats.processed%10 == 0 || stats.processed == stats.total {
		s.setBatchStatus(stats.batchID, map[string]interface{}{
			"total":        stats.total,
			"processed":    stats.processed,
			"successful":   stats.successful,
			"failed":       stats.failed,
			"extracted":    stats.extracted,
			"status":       "processing",
			"auto_approve": stats.autoApprove,
			"started_at":   s.getBatchStartTime(stats.batchID),
		})
	}
}

func (s *DocumentService) finalizeBatch(stats *batchStats) {
	s.setBatchStatus(stats.batchID, map[string]interface{}{
		"total":        stats.total,
		"processed":    stats.processed,
		"successful":   stats.successful,
		"failed":       stats.failed,
		"extracted":    stats.extracted,
		"status":       "completed",
		"auto_approve": stats.autoApprove,
		"started_at":   s.getBatchStartTime(stats.batchID),
		"completed_at": time.Now().Format(time.RFC3339),
	})

	log.Printf("Batch %s completed: %d/%d successful, %d failed, %d extracted",
		stats.batchID, stats.successful, stats.total, stats.failed, stats.extracted)
}

func (s *DocumentService) processFileDataWithExtraction(
	fileData FileData, category, email, accountType, uploadDir string,
	validTypes map[string]bool, maxFileSize int, batchID string,
	workerID int, autoApprove bool,
) (int, int, bool) {
	originalFilename := fileData.Filename

	if fileData.Size > int64(maxFileSize) {
		log.Printf("Batch %s Worker %d: File %s exceeds size limit", batchID, workerID, originalFilename)
		return 0, 0, false
	}

	ext := strings.ToLower(filepath.Ext(originalFilename))
	dataType := strings.TrimPrefix(ext, ".")
	if !validTypes[dataType] {
		log.Printf("Batch %s Worker %d: File %s has invalid type", batchID, workerID, originalFilename)
		return 0, 0, false
	}

	uniqueFilename := GenerateUniqueFilename(originalFilename)
	filePath := filepath.Join(uploadDir, uniqueFilename)

	if err := os.WriteFile(filePath, fileData.Content, 0644); err != nil {
		log.Printf("Batch %s Worker %d: Failed to write file %s: %v", batchID, workerID, originalFilename, err)
		return 0, 0, false
	}

	document := &Document{
		Category: category,
	}

	isLatest := true
	var status string
	var isApprove *bool

	if autoApprove {
		status = "Approved"
		approveTrue := true
		isApprove = &approveTrue
	} else {
		status = "Pending"
		isApprove = nil
	}

	detail := &DocumentDetail{
		DocumentName: originalFilename,
		Filename:     uniqueFilename,
		DataType:     dataType,
		Staff:        email,
		Team:         accountType,
		Status:       &status,
		IsLatest:     &isLatest,
		IsApprove:    isApprove,
	}

	if err := s.CreateDocument(document, detail); err != nil {
		log.Printf("Batch %s Worker %d: Database error for file %s: %v", batchID, workerID, originalFilename, err)
		os.Remove(filePath)
		return 0, 0, false
	}

	if autoApprove {
		extractReq := external.ExtractRequest{
			ID:       strconv.Itoa(document.ID),
			Category: category,
			Filename: originalFilename,
			FilePath: filePath,
		}

		if err := s.externalClient.ExtractDocument(extractReq); err != nil {
			log.Printf("Batch %s Worker %d: Failed to extract file %s (ID: %d) to external API: %v",
				batchID, workerID, originalFilename, document.ID, err)

			return document.ID, detail.ID, true
		}

		log.Printf("Batch %s Worker %d: Successfully extracted file %s (ID: %d) to external API",
			batchID, workerID, originalFilename, document.ID)
	}

	return document.ID, detail.ID, true
}

func (s *DocumentService) setBatchStatus(batchID string, status map[string]interface{}) error {
	key := isBatchUpload + batchID
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	ctx := context.Background()
	return s.redis.Set(ctx, key, data, 24*time.Hour).Err()
}

func (s *DocumentService) getBatchStartTime(batchID string) string {
	key := isBatchUpload + batchID
	ctx := context.Background()

	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return time.Now().Format(time.RFC3339)
	}

	var status map[string]interface{}
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return time.Now().Format(time.RFC3339)
	}

	if startTime, ok := status["started_at"].(string); ok {
		return startTime
	}
	return time.Now().Format(time.RFC3339)
}

func (s *DocumentService) GetBatchStatus(batchID string) (map[string]interface{}, error) {
	key := isBatchUpload + batchID
	ctx := context.Background()

	data, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("batch not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get batch status: %w", err)
	}

	var status map[string]interface{}
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("failed to parse batch status: %w", err)
	}

	return status, nil
}

func (s *DocumentService) GetTeamNameByUserID(userID int64) (string, error) {
	return s.repo.GetTeamNameByUserID(userID)
}

func (s *DocumentService) BatchDeleteDocuments(ids []int) (int, []string) {
	successCount := 0
	var errorMessages []string

	for _, id := range ids {
		err := s.DeleteDocument(id)
		if err != nil {
			log.Printf("Batch Delete: Failed to delete document ID %d: %v", id, err)
			errorMessages = append(errorMessages, fmt.Sprintf("ID %d: %v", id, err))
		} else {
			successCount++
		}
	}

	return successCount, errorMessages
}


func (s *DocumentService) GenerateViewTokenByDocumentID(documentID int) (string, error) {
	// Cari detail dokumen yang latest & approved berdasarkan document_id
	detail, err := s.repo.GetApprovedLatestDocumentDetailByDocumentID(documentID)
	if err != nil {
		return "", fmt.Errorf("approved and latest document detail not found for document_id %d: %w", documentID, err)
	}

	// Generate token menggunakan filename yang ditemukan (reuse fungsi yang ada)
	return s.GenerateViewToken(detail.Filename)
}