package chat

import (
	"math"

	"github.com/google/uuid"
)

type ChatService struct {
	repo *ChatRepository
}

func NewChatService(repo *ChatRepository) *ChatService {
	return &ChatService{repo: repo}
}

func (s *ChatService) CreateChatHistory(history *ChatHistory) error {
	return s.repo.CreateChatHistory(history)
}

func (s *ChatService) GetAllChatHistory(page, pageSize int) (*ChatHistoryWithPagination, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	histories, total, err := s.repo.GetAllChatHistory(page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &ChatHistoryWithPagination{
		Data:       histories,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ChatService) GetChatHistoryByID(id int) (*ChatHistory, error) {
	return s.repo.GetChatHistoryByID(id)
}

func (s *ChatService) GetChatHistoryBySessionID(sessionID uuid.UUID, page, pageSize int) (*ChatHistoryWithPagination, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	histories, total, err := s.repo.GetChatHistoryBySessionID(sessionID, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &ChatHistoryWithPagination{
		Data:       histories,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ChatService) UpdateChatHistory(history *ChatHistory) error {
	return s.repo.UpdateChatHistory(history)
}

func (s *ChatService) DeleteChatHistory(id int) error {
	return s.repo.DeleteChatHistory(id)
}

func (s *ChatService) CreateConversation(conv *Conversation) error {
	return s.repo.CreateConversation(conv)
}

func (s *ChatService) GetAllConversations(page, pageSize int) (*ConversationWithPagination, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	conversations, total, err := s.repo.GetAllConversations(page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &ConversationWithPagination{
		Data:       conversations,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ChatService) GetConversationByID(id uuid.UUID) (*Conversation, error) {
	return s.repo.GetConversationByID(id)
}

func (s *ChatService) UpdateConversation(conv *Conversation) error {
	return s.repo.UpdateConversation(conv)
}

func (s *ChatService) DeleteConversation(id uuid.UUID) error {
	return s.repo.DeleteConversation(id)
}

func (s *ChatService) GetOrCreateConversation(platform, platformUniqueID string) (*Conversation, error) {
	conv, err := s.repo.GetConversationByPlatformAndUser(platform, platformUniqueID)
	if err != nil {

		return conv, err
	}
	return conv, nil
}

func (s *ChatService) GetChatPairsBySessionID(sessionID *uuid.UUID, page, pageSize int) (*ChatPairsWithPagination, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	pairs, total, err := s.repo.GetChatPairsBySessionID(sessionID, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &ChatPairsWithPagination{
		Data:       pairs,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *ChatService) UpdateIsAnsweredStatus(questionID, answerID int, isAnswered bool) error {
	return s.repo.UpdateIsAnsweredStatus(questionID, answerID, isAnswered)
}
