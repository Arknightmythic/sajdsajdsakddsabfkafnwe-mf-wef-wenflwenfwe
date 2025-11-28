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
	IsAnswered          *bool     `db:"is_answered" json:"is_answered,omitempty"`
	Revision            *string   `db:"revision" json:"revision,omitempty"`
	IsValidated         *bool     `db:"is_validated" json:"is_validated"`
}

type Metadata struct {
	Subject    string `json:"subject"`
	InReplyTo  string `json:"in_reply_to"`
	References string `json:"references"`
	ThreadKey  string `json:"thread_key"`
}

type ResponseAsk struct {
	User             string   `json:"user"`
	ConversationID   string   `json:"conversation_id"`
	Query            string   `json:"query"`
	RewrittenQuery   string   `json:"rewritten_query"`
	Category         string   `json:"category"`
	QuestionCategory []string `json:"question_category"`
	Answer           string   `json:"answer"`
	Citations        []string `json:"citations"`
	IsHelpdesk       bool     `json:"is_helpdesk"`
	IsAnswered       *bool    `json:"is_answered"`
	Platform         string   `json:"platform"`
	PlatformUniqueID string   `json:"platform_unique_id"`
	Metadata         Metadata `json:"metadata,omitempty"`
}

type ChatPair struct {
	QuestionID       int       `json:"question_id"`
	QuestionContent  string    `json:"question_content"`
	QuestionTime     time.Time `json:"question_time"`
	AnswerID         int       `json:"answer_id"`
	AnswerContent    string    `json:"answer_content"`
	AnswerTime       time.Time `json:"answer_time"`
	Category         *string   `json:"category,omitempty"`
	QuestionCategory *string   `json:"question_category,omitempty"`
	Feedback         *bool     `json:"feedback,omitempty"`
	IsCannotAnswer   *bool     `json:"is_cannot_answer,omitempty"`
	Revision         *string   `json:"revision,omitempty"`
	SessionID        uuid.UUID `json:"session_id"`
	IsValidated      *bool     `json:"is_validated"`
	IsAnswered       *bool     `json:"is_answered"`
	CreatedAt        time.Time `json:"created_at"`
}

type ChatPairsWithPagination struct {
	Data       []ChatPair `json:"data"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

type Conversation struct {
	ID               uuid.UUID     `db:"id" json:"id"`
	StartTimestamp   time.Time     `db:"start_timestamp" json:"start_timestamp"`
	EndTimestamp     *time.Time    `db:"end_timestamp" json:"end_timestamp,omitempty"`
	Platform         string        `db:"platform" json:"platform"`
	PlatformUniqueID string        `db:"platform_unique_id" json:"platform_unique_id"`
	IsHelpdesk       bool          `db:"is_helpdesk" json:"is_helpdesk"`
	Context          *string       `db:"context" json:"context"`
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

type ChatHistoryFilter struct {
	SortBy        string
	SortDirection string
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int
	Offset        int
	IsValidated   *string
	IsAnswered    *bool
}

type ConversationFilter struct {
	SortBy           string
	SortDirection    string
	StartDate        *time.Time
	EndDate          *time.Time
	Limit            int
	Offset           int
	PlatformUniqueID *string
}
