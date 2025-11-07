package chat

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Message map[string]interface{}

func (m Message) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *Message) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, m)
}

type ChatHistory struct {
	ID                  int       `db:"id" json:"id"`
	SessionID           uuid.UUID `db:"session_id" json:"session_id"`
	Message             Message   `db:"message" json:"message"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UserID              *int64    `db:"user_id" json:"user_id,omitempty"`
	IsCannotAnswer      *bool     `db:"is_cannot_answer" json:"is_cannot_answer,omitempty"`
	Category            *string   `db:"category" json:"category,omitempty"`
	Feedback            *bool     `db:"feedback" json:"feedback,omitempty"`
	QuestionCategory    *string   `db:"question_category" json:"question_category,omitempty"`
	QuestionSubCategory *string   `db:"question_sub_category" json:"question_sub_category,omitempty"`
}

type Conversation struct {
	ID               uuid.UUID     `db:"id" json:"id"`
	StartTimestamp   time.Time     `db:"start_timestamp" json:"start_timestamp"`
	EndTimestamp     *time.Time    `db:"end_timestamp" json:"end_timestamp,omitempty"`
	Platform         string        `db:"platform" json:"platform"`
	PlatformUniqueID string        `db:"platform_unique_id" json:"platform_unique_id"`
	IsHelpdesk       bool          `db:"is_helpdesk" json:"is_helpdesk"`
	ChatHistory      []ChatHistory `json:"chat_history,omitempty"`
}

type ConversationWithPagination struct {
	Data       []Conversation `json:"data"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

type ChatHistoryWithPagination struct {
	Data       []ChatHistory `json:"data"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}
