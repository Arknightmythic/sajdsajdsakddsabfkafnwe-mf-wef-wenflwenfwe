package audit

import "time"

type AuditLog struct {
	ID           int       `db:"id" json:"id"`
	UserID       *string   `db:"user_id" json:"user_id,omitempty"`
	UserType     *string   `db:"user_type" json:"user_type,omitempty"`
	Action       string    `db:"action" json:"action"`
	Resource     string    `db:"resource" json:"resource"`
	Method       string    `db:"method" json:"method"`
	Path         string    `db:"path" json:"path"`
	StatusCode   *int      `db:"status_code" json:"status_code,omitempty"`
	RequestBody  *string   `db:"request_body" json:"request_body,omitempty"`
	ResponseBody *string   `db:"response_body" json:"response_body,omitempty"`
	IPAddress    string    `db:"ip_address" json:"ip_address"`
	UserAgent    string    `db:"user_agent" json:"user_agent"`
	ErrorMessage *string   `db:"error_message" json:"error_message,omitempty"`
	Duration     *int64    `db:"duration" json:"duration,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type CreateAuditLogRequest struct {
	UserID       *string
	UserType     *string
	Action       string
	Resource     string
	Method       string
	Path         string
	StatusCode   *int
	RequestBody  *string
	ResponseBody *string
	IPAddress    string
	UserAgent    string
	ErrorMessage *string
	Duration     *int64
}
