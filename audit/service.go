package audit

import (
	"log"
)

type AuditService struct {
	repo *AuditRepository
}

func NewAuditService(repo *AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Log(req CreateAuditLogRequest) {

	reqCopy := CreateAuditLogRequest{
		UserID:       copyStringPtr(req.UserID),
		UserType:     copyStringPtr(req.UserType),
		Action:       req.Action,
		Resource:     req.Resource,
		Method:       req.Method,
		Path:         req.Path,
		StatusCode:   copyIntPtr(req.StatusCode),
		RequestBody:  copyStringPtr(req.RequestBody),
		ResponseBody: copyStringPtr(req.ResponseBody),
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		ErrorMessage: copyStringPtr(req.ErrorMessage),
		Duration:     copyInt64Ptr(req.Duration),
	}

	go func() {
		if err := s.repo.Create(reqCopy); err != nil {
			log.Printf("Failed to create audit log: %v", err)
		}
	}()
}

func (s *AuditService) GetAll(limit, offset int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetAll(limit, offset)
}

func (s *AuditService) GetByUserID(userID string, limit, offset int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetByUserID(userID, limit, offset)
}

func copyStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	copy := *s
	return &copy
}

func copyIntPtr(i *int) *int {
	if i == nil {
		return nil
	}
	copy := *i
	return &copy
}

func copyInt64Ptr(i *int64) *int64 {
	if i == nil {
		return nil
	}
	copy := *i
	return &copy
}
