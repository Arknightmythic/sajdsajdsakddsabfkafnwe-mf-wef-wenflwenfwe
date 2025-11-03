package document

import (
	"context"
	"dokuprime-be/util"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type DocumentService struct {
	repo  *DocumentRepository
	redis *redis.Client
}

func NewDocumentService(repo *DocumentRepository, redisClient *redis.Client) *DocumentService {
	return &DocumentService{
		repo:  repo,
		redis: redisClient,
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
