package chat

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ChatRepository struct {
	db *sqlx.DB
}

func NewChatRepository(db *sqlx.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateChatHistory(history *ChatHistory) error {
	query := `
		INSERT INTO chat_history
		(session_id, message, user_id, is_cannot_answer, category, feedback, question_category, question_sub_category, is_answered, revision)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	return r.db.QueryRow(
		query,
		history.SessionID,
		history.Message,
		history.UserID,
		history.IsCannotAnswer,
		history.Category,
		history.Feedback,
		history.QuestionCategory,
		history.QuestionSubCategory,
		history.IsAnswered,
		history.Revision,
	).Scan(&history.ID, &history.CreatedAt)
}

func (r *ChatRepository) GetAllChatHistory(filter ChatHistoryFilter) ([]ChatHistory, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM chat_history %s", where)
	var total int
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	allowedSort := map[string]bool{"created_at": true, "user_id": true, "id": true, "session_id": true}
	sortBy := "created_at"
	if filter.SortBy != "" && allowedSort[filter.SortBy] {
		sortBy = filter.SortBy
	}

	sortDirection := "DESC"
	if strings.ToUpper(filter.SortDirection) == "ASC" {
		sortDirection = "ASC"
	}

	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
			   category, feedback, question_category, question_sub_category, is_answered, revision
		FROM chat_history %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortBy, sortDirection, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	var histories []ChatHistory
	if err := r.db.Select(&histories, query, args...); err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

func (r *ChatRepository) GetChatHistoryByID(id int) (*ChatHistory, error) {
	var history ChatHistory
	query := `
		SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
			   category, feedback, question_category, question_sub_category, is_answered, revision
		FROM chat_history
		WHERE id = $1
	`
	err := r.db.Get(&history, query, id)
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func (r *ChatRepository) GetChatHistoryBySessionID(sessionID uuid.UUID, filter ChatHistoryFilter) ([]ChatHistory, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIdx))
	args = append(args, sessionID)
	argIdx++

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM chat_history %s", where)
	var total int
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	allowedSort := map[string]bool{"created_at": true, "user_id": true, "id": true}
	sortBy := "created_at"
	if filter.SortBy != "" && allowedSort[filter.SortBy] {
		sortBy = filter.SortBy
	}

	sortDirection := "ASC"
	if strings.ToUpper(filter.SortDirection) == "DESC" {
		sortDirection = "DESC"
	}

	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
			   category, feedback, question_category, question_sub_category, is_answered, revision
		FROM chat_history
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortBy, sortDirection, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	var histories []ChatHistory
	if err := r.db.Select(&histories, query, args...); err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

func (r *ChatRepository) UpdateChatHistory(history *ChatHistory) error {
	query := `
		UPDATE chat_history
		SET message = $1, user_id = $2, is_cannot_answer = $3, category = $4,
			feedback = $5, question_category = $6, question_sub_category = $7, is_answered = $8, revision = $9
		WHERE id = $10
	`
	_, err := r.db.Exec(
		query,
		history.Message,
		history.UserID,
		history.IsCannotAnswer,
		history.Category,
		history.Feedback,
		history.QuestionCategory,
		history.QuestionSubCategory,
		history.IsAnswered,
		history.Revision,
		history.ID,
	)
	return err
}

func (r *ChatRepository) DeleteChatHistory(id int) error {
	_, err := r.db.Exec(`DELETE FROM chat_history WHERE id = $1`, id)
	return err
}

func (r *ChatRepository) CreateConversation(conv *Conversation) error {
	if conv.ID == uuid.Nil {
		conv.ID = uuid.New()
	}

	query := `
		INSERT INTO conversations (id, start_timestamp, end_timestamp, platform, platform_unique_id, is_helpdesk, context)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	return r.db.QueryRow(
		query,
		conv.ID,
		conv.StartTimestamp,
		conv.EndTimestamp,
		conv.Platform,
		conv.PlatformUniqueID,
		conv.IsHelpdesk,
		conv.Context,
	).Scan(&conv.ID)
}

func (r *ChatRepository) GetAllConversations(filter ConversationFilter) ([]Conversation, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "platform = 'web'")

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("start_timestamp >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("start_timestamp <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM conversations %s", where)
	var total int
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	allowedSort := map[string]bool{"start_timestamp": true, "end_timestamp": true, "id": true}
	sortBy := "start_timestamp"
	if filter.SortBy != "" && allowedSort[filter.SortBy] {
		sortBy = filter.SortBy
	}

	sortDirection := "DESC"
	if strings.ToUpper(filter.SortDirection) == "ASC" {
		sortDirection = "ASC"
	}

	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, start_timestamp, end_timestamp, platform, platform_unique_id, is_helpdesk, 
			   COALESCE(context, '') as context
		FROM conversations
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, sortBy, sortDirection, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	var conversations []Conversation
	if err := r.db.Select(&conversations, query, args...); err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

func (r *ChatRepository) GetConversationByID(id uuid.UUID) (*Conversation, error) {
	var conv Conversation
	query := `
		SELECT id, start_timestamp, end_timestamp, platform, platform_unique_id, is_helpdesk, context
		FROM conversations
		WHERE id = $1
	`
	err := r.db.Get(&conv, query, id)
	if err != nil {
		return nil, err
	}

	var histories []ChatHistory
	historyQuery := `
		SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
			   category, feedback, question_category, question_sub_category, is_answered, revision
		FROM chat_history
		WHERE session_id = $1
		ORDER BY created_at ASC
	`
	err = r.db.Select(&histories, historyQuery, conv.ID)
	if err == nil {
		conv.ChatHistory = histories
	}

	return &conv, nil
}

func (r *ChatRepository) UpdateConversation(conv *Conversation) error {
	query := `
		UPDATE conversations
		SET end_timestamp = $1, platform = $2, platform_unique_id = $3, is_helpdesk = $4, context = $5
		WHERE id = $6
	`
	_, err := r.db.Exec(
		query,
		conv.EndTimestamp,
		conv.Platform,
		conv.PlatformUniqueID,
		conv.IsHelpdesk,
		conv.Context,
		conv.ID,
	)
	return err
}

func (r *ChatRepository) DeleteConversation(id uuid.UUID) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM chat_history WHERE session_id = $1`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM conversations WHERE id = $1`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *ChatRepository) GetConversationByPlatformAndUser(platform, platformUniqueID string) (*Conversation, error) {
	var conv Conversation
	query := `
		SELECT id, start_timestamp, end_timestamp, platform, platform_unique_id, is_helpdesk, context
		FROM conversations
		WHERE platform = $1 AND platform_unique_id = $2 AND end_timestamp IS NULL
		ORDER BY start_timestamp DESC
		LIMIT 1
	`
	err := r.db.Get(&conv, query, platform, platformUniqueID)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func (r *ChatRepository) GetChatPairsBySessionID(sessionID *uuid.UUID, filter ChatHistoryFilter) ([]ChatPair, int, error) {
	var histories []ChatHistory
	var conditions []string
	var args []interface{}
	argIdx := 1

	if sessionID != nil {
		conditions = append(conditions, fmt.Sprintf("session_id = $%d", argIdx))
		args = append(args, *sessionID)
		argIdx++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	order := "ORDER BY session_id, created_at ASC"

	query := fmt.Sprintf(`
			SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
				   category, feedback, question_category, question_sub_category, is_answered, revision
			FROM chat_history
			%s
			%s
		`, where, order)

	if err := r.db.Select(&histories, query, args...); err != nil {
		return nil, 0, err
	}

	var pairs []ChatPair
	for i := 0; i < len(histories); i += 2 {
		if i+1 >= len(histories) {
			break
		}

		userRole := getMessageRole(histories[i].Message)
		assistantRole := getMessageRole(histories[i+1].Message)

		if userRole == "user" && assistantRole == "assistant" {
			questionContent := extractContent(histories[i].Message)
			answerContent := extractContent(histories[i+1].Message)

			pairs = append(pairs, ChatPair{
				QuestionID:       histories[i].ID,
				QuestionContent:  questionContent,
				QuestionTime:     histories[i].CreatedAt,
				AnswerID:         histories[i+1].ID,
				AnswerContent:    answerContent,
				AnswerTime:       histories[i+1].CreatedAt,
				Category:         histories[i].Category,
				QuestionCategory: histories[i].QuestionCategory,
				Feedback:         histories[i+1].Feedback,
				IsCannotAnswer:   histories[i+1].IsCannotAnswer,
				SessionID:        histories[i].SessionID,
			})
		}
	}

	total := len(pairs)

	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	offset := filter.Offset
	end := offset + filter.Limit

	if offset >= total {
		return []ChatPair{}, total, nil
	}

	if end > total {
		end = total
	}

	return pairs[offset:end], total, nil
}

func extractContent(msg Message) string {

	if data, ok := msg["data"].(map[string]interface{}); ok {
		if content, ok := data["content"].(string); ok {
			return content
		}
	}

	if content, ok := msg["content"].(string); ok {
		return content
	}

	return ""
}

func getMessageRole(msg Message) string {
	if msgType, ok := msg["type"].(string); ok {
		if msgType == "human" {
			return "user"
		}
		if msgType == "ai" {
			return "assistant"
		}
	}

	if data, ok := msg["data"].(map[string]interface{}); ok {
		if msgType, ok := data["type"].(string); ok {
			if msgType == "human" {
				return "user"
			}
			if msgType == "ai" {
				return "assistant"
			}
		}
	}

	if role, ok := msg["role"].(string); ok {
		return role
	}

	return ""
}

func (r *ChatRepository) UpdateIsAnsweredStatus(questionID, answerID int, revision string, isAnswered bool) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryQuestion := `UPDATE chat_history SET is_answered = $1, is_cannot_answer = $2 WHERE id = $3`
	_, err = tx.Exec(queryQuestion, isAnswered, !isAnswered, questionID)
	if err != nil {
		return err
	}

	queryAnswer := `UPDATE chat_history SET is_answered = $1, is_cannot_answer = $2, revision = $3 WHERE id = $4`
	_, err = tx.Exec(queryAnswer, isAnswered, !isAnswered, revision, answerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
