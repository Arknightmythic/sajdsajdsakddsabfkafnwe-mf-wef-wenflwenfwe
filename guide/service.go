package guide

import (
	"context"
	"dokuprime-be/config"
	"dokuprime-be/util"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type GuideService struct {
	repo  *GuideRepository
	redis *redis.Client
}

func NewGuideService(repo *GuideRepository, redisClient *redis.Client) *GuideService {
	return &GuideService{
		repo:  repo,
		redis: redisClient,
	}
}

func generateUniqueFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	timestamp := time.Now().Unix()
	uniqueID := uuid.New().String()
	return fmt.Sprintf("%d_%s%s", timestamp, uniqueID, ext)
}

func (s *GuideService) UploadGuide(title, description string, file *multipart.FileHeader) (*Guide, error) {

	uniqueFilename := generateUniqueFilename(file.Filename)
	uploadDir := config.GetUploadPath()
	filePath := filepath.Join(uploadDir, uniqueFilename)

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file on disk: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file content: %w", err)
	}

	guide := &Guide{
		Title:            title,
		Description:      description,
		Filename:         uniqueFilename,
		OriginalFilename: file.Filename,
	}

	if err := s.repo.Create(guide); err != nil {
		os.Remove(filePath)
		return nil, err
	}

	return guide, nil
}

func (s *GuideService) GetAll(filter GuideFilter) ([]Guide, int, error) {
	return s.repo.GetAll(filter)
}

func (s *GuideService) GetByID(id int) (*Guide, error) {
	return s.repo.GetByID(id)
}

func (s *GuideService) UpdateGuide(id int, title, description string, file *multipart.FileHeader) (*Guide, error) {

	existingGuide, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	existingGuide.Title = title
	existingGuide.Description = description

	if file != nil {
		uploadDir := config.GetUploadPath()

		oldFilePath := filepath.Join(uploadDir, existingGuide.Filename)
		os.Remove(oldFilePath)

		uniqueFilename := generateUniqueFilename(file.Filename)
		newFilePath := filepath.Join(uploadDir, uniqueFilename)

		src, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer src.Close()

		dst, err := os.Create(newFilePath)
		if err != nil {
			return nil, err
		}
		defer dst.Close()

		if _, err := dst.ReadFrom(src); err != nil {
			os.Remove(newFilePath)
			return nil, err
		}

		existingGuide.Filename = uniqueFilename
		existingGuide.OriginalFilename = file.Filename
	}

	if err := s.repo.Update(existingGuide); err != nil {
		return nil, err
	}

	return existingGuide, nil
}

func (s *GuideService) DeleteGuide(id int) error {

	guide, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	uploadDir := config.GetUploadPath()
	filePath := filepath.Join(uploadDir, guide.Filename)
	os.Remove(filePath)

	return s.repo.Delete(id)
}

func (s *GuideService) GenerateViewTokenByID(id int) (string, error) {
	guide, err := s.repo.GetByID(id)
	if err != nil {
		return "", fmt.Errorf("guide not found: %w", err)
	}

	token := util.RandString(32)
	key := "view_guide_token:" + token

	ctx := context.Background()

	err = s.redis.Set(ctx, key, guide.Filename, 5*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store view token: %w", err)
	}

	return token, nil
}
