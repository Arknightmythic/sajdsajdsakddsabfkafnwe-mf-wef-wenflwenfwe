package chat

import (
	"math"
	"time"

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

func (s *ChatService) GetAllChatHistory(filter ChatHistoryFilter) (*ChatHistoryWithPagination, error) {
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	histories, total, err := s.repo.GetAllChatHistory(filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	page := 1
	if filter.Limit > 0 {
		page = (filter.Offset / filter.Limit) + 1
	}

	return &ChatHistoryWithPagination{
		Data:       histories,
		Total:      total,
		Page:       page,
		PageSize:   filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *ChatService) GetChatHistoryByID(id int) (*ChatHistory, error) {
	return s.repo.GetChatHistoryByID(id)
}

func (s *ChatService) GetChatHistoryBySessionID(sessionID uuid.UUID, filter ChatHistoryFilter) (*ChatHistoryWithPagination, error) {
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	histories, total, err := s.repo.GetChatHistoryBySessionID(sessionID, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	page := 1
	if filter.Limit > 0 {
		page = (filter.Offset / filter.Limit) + 1
	}

	return &ChatHistoryWithPagination{
		Data:       histories,
		Total:      total,
		Page:       page,
		PageSize:   filter.Limit,
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

func (s *ChatService) GetAllConversations(filter ConversationFilter) (*ConversationWithPagination, error) {
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	conversations, total, err := s.repo.GetAllConversations(filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	page := 1
	if filter.Limit > 0 {
		page = (filter.Offset / filter.Limit) + 1
	}

	return &ConversationWithPagination{
		Data:       conversations,
		Total:      total,
		Page:       page,
		PageSize:   filter.Limit,
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

func (s *ChatService) GetChatPairsBySessionID(sessionID *uuid.UUID, filter ChatHistoryFilter) (*ChatPairsWithPagination, error) {
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	pairs, total, err := s.repo.GetChatPairsBySessionID(sessionID, filter)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))

	page := 1
	if filter.Limit > 0 {
		page = (filter.Offset / filter.Limit) + 1
	}

	return &ChatPairsWithPagination{
		Data:       pairs,
		Total:      total,
		Page:       page,
		PageSize:   filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (s *ChatService) UpdateIsAnsweredStatus(questionID, answerID int, revision string, isValidated bool, userID any) error {
	return s.repo.UpdateIsAnsweredStatus(questionID, answerID, revision, isValidated, userID)
}

func (s *ChatService) Feedback(answerID int, sessionID uuid.UUID, feedback bool) error {
	if answerID != 0 {
		err := s.repo.UpdateFeedback(answerID, feedback)
		if err != nil {
			return err
		}
	} else {
		err := s.repo.UpdateChatFeedback(sessionID, feedback)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *ChatService) GetChatHistoriesForDownload(startDate, endDate *time.Time, typeFilter string) ([]ChatHistory, error) {
	return s.repo.GetChatHistoriesForDownload(startDate, endDate, typeFilter)
}
