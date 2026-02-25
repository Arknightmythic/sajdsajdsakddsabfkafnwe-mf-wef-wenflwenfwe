package audit

import (
	"github.com/jmoiron/sqlx"
)

type AuditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(req CreateAuditLogRequest) error {
	query := `
		INSERT INTO audit_logs (
			user_id, user_type, action, resource, method, path, status_code,
			request_body, response_body, ip_address, user_agent,
			error_message, duration, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW()
		)
	`
	_, err := r.db.Exec(
		query,
		req.UserID,
		req.UserType,
		req.Action,
		req.Resource,
		req.Method,
		req.Path,
		req.StatusCode,
		req.RequestBody,
		req.ResponseBody,
		req.IPAddress,
		req.UserAgent,
		req.ErrorMessage,
		req.Duration,
	)
	return err
}

func (r *AuditRepository) GetAll(limit, offset int) ([]AuditLog, error) {
	var logs []AuditLog
	query := `
		SELECT id, user_id, user_type, action, resource, method, path, status_code,
		       request_body, response_body, ip_address, user_agent,
		       error_message, duration, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	err := r.db.Select(&logs, query, limit, offset)
	return logs, err
}

func (r *AuditRepository) GetByUserID(userID string, limit, offset int) ([]AuditLog, error) {
	var logs []AuditLog
	query := `
		SELECT id, user_id, user_type, action, resource, method, path, status_code,
		       request_body, response_body, ip_address, user_agent,
		       error_message, duration, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.Select(&logs, query, userID, limit, offset)
	return logs, err
}

func (r *AuditRepository) ArchiveOldLogs() error {
	query := `
		WITH moved_rows AS (
			DELETE FROM bkpm.audit_logs
			WHERE created_at < date_trunc('month', current_date)
			RETURNING *
		)
		INSERT INTO bkpm_archive.audit_logs
		SELECT * FROM moved_rows;
	`
	_, err := r.db.Exec(query)
	return err
}