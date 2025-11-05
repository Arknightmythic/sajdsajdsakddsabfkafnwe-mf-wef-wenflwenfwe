package document

import (
	"context"
	"dokuprime-be/external"
	"dokuprime-be/util"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DocumentService struct {
	repo           *DocumentRepository
	redis          *redis.Client
	asyncProcessor *AsyncProcessor
}

func NewDocumentService(repo *DocumentRepository, redisClient *redis.Client, asyncProcessor *AsyncProcessor) *DocumentService {
	return &DocumentService{
		repo:           repo,
		redis:          redisClient,
		asyncProcessor: asyncProcessor,
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

	filePath := filepath.Join("./uploads/documents", detail.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("document file not found: %s", detail.Filename)
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
		ID:       detail.DocumentID,
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

	return nil
}

func (s *DocumentService) DeleteDocument(documentID int) error {

	details, err := s.repo.GetDocumentDetailsByDocumentID(documentID)
	if err != nil {
		return fmt.Errorf("failed to get document details: %w", err)
	}

	for _, detail := range details {
		filePath := filepath.Join("./uploads/documents", detail.Filename)
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
