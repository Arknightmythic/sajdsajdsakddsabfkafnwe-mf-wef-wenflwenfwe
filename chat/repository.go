package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const isWHERE = "WHERE "

type ChatRepository struct {
	db *sqlx.DB
}

type chatHistoryWithPlatform struct {
	ChatHistory
	PlatformUniqueID string `db:"platform_unique_id"`
}

func NewChatRepository(db *sqlx.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateChatHistory(history *ChatHistory) error {
	query := `
		       INSERT INTO chat_history
		       (session_id, message, user_id, is_cannot_answer, category, feedback, question_category, question_sub_category, is_answered, revision, is_validated)
		       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
		nil,
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
		where = isWHERE + strings.Join(conditions, " AND ")
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
			 category, feedback, question_category, question_sub_category, is_answered, revision, is_validated
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
			 category, feedback, question_category, question_sub_category, is_answered, revision, is_validated
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

	where := isWHERE + strings.Join(conditions, " AND ")

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
			 category, feedback, question_category, question_sub_category, is_answered, revision, is_validated
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

	if filter.PlatformUniqueID != nil {
		conditions = append(conditions, fmt.Sprintf("platform_unique_id = $%d", argIdx))
		args = append(args, *filter.PlatformUniqueID)
		argIdx++
	}

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

	where := isWHERE + strings.Join(conditions, " AND ")

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
			 category, feedback, question_category, question_sub_category, is_answered, revision, is_validated
		 FROM chat_history
		 WHERE session_id = $1
		 ORDER BY id ASC
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

	histories, err := r.fetchRawHistories(sessionID, filter)
	if err != nil {
		return nil, 0, err
	}

	pairs := r.buildChatPairs(histories, filter)

	return r.sortAndPaginatePairs(pairs, filter)
}

func (r *ChatRepository) fetchRawHistories(sessionID *uuid.UUID, filter ChatHistoryFilter) ([]chatHistoryWithPlatform, error) {
	var histories []chatHistoryWithPlatform
	var conditions []string
	var args []interface{}
	argIdx := 1

	if sessionID != nil {
		conditions = append(conditions, fmt.Sprintf("ch.session_id = $%d", argIdx))
		args = append(args, *sessionID)
		argIdx++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("ch.created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("ch.created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf(`(
            c.platform_unique_id ILIKE $%d OR 
            ch.session_id::text ILIKE $%d OR 
            ch.message ->> 'content' ILIKE $%d OR 
            ch.message -> 'data' ->> 'content' ILIKE $%d
        )`, argIdx, argIdx, argIdx, argIdx))
		args = append(args, searchPattern)
		argIdx++
	}

	conditions = append(conditions, "c.is_helpdesk = false")

	where := ""
	if len(conditions) > 0 {
		where = isWHERE + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT ch.id, ch.session_id, ch.message, ch.created_at, ch.user_id, ch.is_cannot_answer,
			ch.category, ch.feedback, ch.question_category, ch.question_sub_category, ch.is_answered, 
			ch.revision, ch.is_validated, 
			c.platform_unique_id
		FROM chat_history ch 
		JOIN conversations c ON ch.session_id = c.id
		%s
		ORDER BY ch.session_id ASC, ch.created_at ASC, ch.id ASC
	`, where)

	if err := r.db.Select(&histories, query, args...); err != nil {
		return nil, err
	}

	return histories, nil
}

func (r *ChatRepository) buildChatPairs(histories []chatHistoryWithPlatform, filter ChatHistoryFilter) []ChatPair {
	var pairs []ChatPair
	for i := 0; i < len(histories); i += 2 {
		if i+1 >= len(histories) {
			break
		}

		pair, isValidPair := r.createPairFromHistory(histories[i], histories[i+1])

		if isValidPair {
			if r.shouldIncludePair(pair, filter) {
				pairs = append(pairs, pair)
			}
		} else {

			i--
		}
	}
	return pairs
}

func (r *ChatRepository) createPairFromHistory(q, a chatHistoryWithPlatform) (ChatPair, bool) {
	userRole := getMessageRole(q.ChatHistory.Message)
	assistantRole := getMessageRole(a.ChatHistory.Message)

	if userRole == "user" && assistantRole == "assistant" && q.SessionID == a.SessionID {
		return ChatPair{
			QuestionID:       q.ID,
			QuestionContent:  extractContent(q.ChatHistory.Message),
			QuestionTime:     q.CreatedAt,
			AnswerID:         a.ID,
			AnswerContent:    extractContent(a.ChatHistory.Message),
			AnswerTime:       a.CreatedAt,
			Category:         q.Category,
			QuestionCategory: q.QuestionCategory,
			Feedback:         a.Feedback,
			IsCannotAnswer:   a.IsCannotAnswer,
			Revision:         a.Revision,
			SessionID:        q.SessionID,
			IsValidated:      a.IsValidated,
			IsAnswered:       q.IsAnswered,
			CreatedAt:        q.CreatedAt,
			PlatformUniqueID: q.PlatformUniqueID,
		}, true
	}
	return ChatPair{}, false
}

func (r *ChatRepository) shouldIncludePair(pair ChatPair, filter ChatHistoryFilter) bool {
	if !r.matchValidatedFilter(pair, filter.IsValidated) {
		return false
	}

	if !r.matchAnsweredFilter(pair, filter.IsAnswered) {
		return false
	}

	return true
}

func (r *ChatRepository) matchValidatedFilter(pair ChatPair, filterVal *string) bool {
	if filterVal == nil {
		return true
	}

	reqVal := *filterVal
	isValidated := pair.IsValidated

	if reqVal == "null" && isValidated != nil {
		return false
	}
	if reqVal == "1" && (isValidated == nil || !*isValidated) {
		return false
	}
	if reqVal == "0" && (isValidated == nil || *isValidated) {
		return false
	}
	return true
}

func (r *ChatRepository) matchAnsweredFilter(pair ChatPair, filterVal *bool) bool {
	if filterVal == nil {
		return true
	}

	reqVal := *filterVal
	currentVal := false
	if pair.IsAnswered != nil {
		currentVal = *pair.IsAnswered
	}

	return currentVal == reqVal
}

func (r *ChatRepository) sortAndPaginatePairs(pairs []ChatPair, filter ChatHistoryFilter) ([]ChatPair, int, error) {

	if strings.ToUpper(filter.SortDirection) == "DESC" || filter.SortDirection == "" {
		sort.SliceStable(pairs, func(i, j int) bool {
			return pairs[i].CreatedAt.After(pairs[j].CreatedAt)
		})
	} else {
		sort.SliceStable(pairs, func(i, j int) bool {
			return pairs[i].CreatedAt.Before(pairs[j].CreatedAt)
		})
	}

	total := len(pairs)
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	start := filter.Offset
	end := start + filter.Limit

	if start >= total {
		return []ChatPair{}, total, nil
	}
	if end > total {
		end = total
	}

	return pairs[start:end], total, nil
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

func (r *ChatRepository) UpdateIsAnsweredStatus(questionID, answerID int, revision string, isValidated bool, userID any) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queryQuestion := `UPDATE chat_history SET is_validated = $1, is_cannot_answer = $2, validator = $3 WHERE id = $4`
	_, err = tx.Exec(queryQuestion, isValidated, !isValidated, userID, questionID)
	if err != nil {
		return err
	}

	queryAnswer := `UPDATE chat_history SET is_validated = $1, is_cannot_answer = $2, revision = $3, validator = $4 WHERE id = $5`
	_, err = tx.Exec(queryAnswer, isValidated, !isValidated, revision, userID, answerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *ChatRepository) GetHelpdeskMessages(sessionID uuid.UUID, limit, offset int) ([]ChatHistory, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM chat_history WHERE session_id = $1`
	if err := r.db.Get(&total, countQuery, sessionID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, session_id, message, created_at, user_id, is_cannot_answer,
			category, feedback, question_category, question_sub_category, is_answered, revision, is_validated
		FROM chat_history
		WHERE session_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	var histories []ChatHistory
	if err := r.db.Select(&histories, query, sessionID, limit, offset); err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

func (r *ChatRepository) UpdateFeedback(answerID int, feedback bool) error {
	query := `
		UPDATE chat_history 
		SET feedback = CASE 
			WHEN feedback = $1 THEN NULL 
			ELSE $1 
		END 
		WHERE id = $2
	`
	_, err := r.db.Exec(query, feedback, answerID)
	return err
}

func (r *ChatRepository) UpdateChatFeedback(sessionID uuid.UUID, feedback bool) error {
	query := `
		WITH random_unfeedback_ai AS (
			SELECT id
			FROM chat_history
			WHERE session_id = $1
				AND feedback IS NULL
				AND message->>'type' = 'ai'
			ORDER BY RANDOM()
			LIMIT 1
		)
		UPDATE chat_history
		SET feedback = $2
		WHERE id IN (SELECT id FROM random_unfeedback_ai)
	`

	_, err := r.db.Exec(query, sessionID, feedback)
	return err
}
