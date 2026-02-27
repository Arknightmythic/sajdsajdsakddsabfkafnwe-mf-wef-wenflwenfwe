package audit

import (
	"github.com/jmoiron/sqlx"
	"log"
	"time"
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
	batchSize := 5000
	totalMoved := 0

	for {
		query := `
			WITH rows_to_move AS (
				SELECT id FROM bkpm.audit_logs
				WHERE created_at < CURRENT_DATE
				ORDER BY created_at ASC
				LIMIT $1
			),
			moved_rows AS (
				DELETE FROM bkpm.audit_logs
				WHERE id IN (SELECT id FROM rows_to_move)
				RETURNING *
			)
			INSERT INTO bkpm_archive.audit_logs
			SELECT * FROM moved_rows;
		`
		
		result, err := r.db.Exec(query, batchSize)
		if err != nil {
			log.Printf("[AuditRepository] Error saat mengeksekusi batch: %v", err)
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		totalMoved += int(rowsAffected)

		if rowsAffected == 0 {
			break
		}
		
		time.Sleep(1 * time.Second)
	}

	log.Printf("[AuditRepository] Selesai memindahkan %d baris data ke arsip.", totalMoved)
	
	_, _ = r.db.Exec("VACUUM ANALYZE bkpm.audit_logs;")
	
	return nil
}

func (r *AuditRepository) CleanupArchiveLogs() error {
	batchSize := 5000
	totalDeleted := 0
	for {
		query := `
			DELETE FROM bkpm_archive.audit_logs
			WHERE id IN (
				SELECT id FROM bkpm_archive.audit_logs
				WHERE created_at < CURRENT_DATE - INTERVAL '30 days'
				LIMIT $1
			)
		`
		result, err := r.db.Exec(query, batchSize)
		if err != nil {
			log.Printf("[AuditRepository] Error saat mengeksekusi batch hapus arsip: %v", err)
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		totalDeleted += int(rowsAffected)
		
		if rowsAffected == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	log.Printf("[AuditRepository] Selesai menghapus %d baris data arsip yang lebih dari 30 hari.", totalDeleted)
	
	return nil
}