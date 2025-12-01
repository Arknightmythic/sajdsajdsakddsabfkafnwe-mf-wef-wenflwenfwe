package helpdesk

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"fmt"
	"strings"
)

type HelpdeskRepository struct {
	db *sqlx.DB
}

func NewHelpdeskRepository(db *sqlx.DB) *HelpdeskRepository {
	return &HelpdeskRepository{db: db}
}

func (r *HelpdeskRepository) Create(helpdesk *Helpdesk) error {
	query := `INSERT INTO helpdesk (session_id, platform, platform_unique_id, status, user_id) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return r.db.QueryRow(query, helpdesk.SessionID, helpdesk.Platform, helpdesk.PlatformUniqueID, helpdesk.Status, helpdesk.UserID).
		Scan(&helpdesk.ID, &helpdesk.CreatedAt)
}

func (r *HelpdeskRepository) GetAll(limit, offset int, search string, status string) ([]Helpdesk, int, error) {
	helpdesks := []Helpdesk{} // Inisialisasi slice kosong
	var conditions []string
	var args []interface{}
	argIdx := 1

	query := `SELECT id, session_id, platform, platform_unique_id, status, user_id, created_at 
			  FROM helpdesk`

	// Filter Search (General)
	if search != "" {
		// PERBAIKAN DI SINI: Tambahkan "OR session_id::text ILIKE ..."
		// Kita perlu cast UUID ke text agar bisa di-ILIKE
		conditions = append(conditions, fmt.Sprintf("(platform ILIKE $%d OR platform_unique_id ILIKE $%d OR session_id::text ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	// Filter Status (Specific)
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status ILIKE $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM helpdesk %s", where)
	var total int
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return []Helpdesk{}, 0, err
	}

	if total == 0 {
		return []Helpdesk{}, 0, nil
	}

	// Main query
	fullQuery := fmt.Sprintf("%s %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", query, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	if err := r.db.Select(&helpdesks, fullQuery, args...); err != nil {
		return []Helpdesk{}, 0, err
	}

	return helpdesks, total, nil
}

func (r *HelpdeskRepository) GetTotal(search string) (int, error) {
	var total int
	query := `SELECT COUNT(*) FROM helpdesk`

	if search != "" {
		query += ` WHERE platform ILIKE $1 OR status ILIKE $1`
		err := r.db.Get(&total, query, "%"+search+"%")
		return total, err
	}

	err := r.db.Get(&total, query)
	return total, err
}

func (r *HelpdeskRepository) GetByID(id int) (*Helpdesk, error) {
	var helpdesk Helpdesk
	query := `SELECT id, session_id, platform, platform_unique_id, status, user_id, created_at 
			  FROM helpdesk WHERE id = $1`
	err := r.db.Get(&helpdesk, query, id)
	if err != nil {
		return nil, err
	}
	return &helpdesk, nil
}

func (r *HelpdeskRepository) Update(helpdesk *Helpdesk) error {
	query := `UPDATE helpdesk 
			  SET session_id = $1, platform = $2, platform_unique_id = $3, status = $4, user_id = $5 
			  WHERE id = $6`
	_, err := r.db.Exec(query, helpdesk.SessionID, helpdesk.Platform, helpdesk.PlatformUniqueID, helpdesk.Status, helpdesk.UserID, helpdesk.ID)
	return err
}

func (r *HelpdeskRepository) UpdateStatus(id int, status string) error {
	query := `UPDATE helpdesk SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(query, status, id)
	return err
}

func (r *HelpdeskRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM helpdesk WHERE id = $1`, id)
	return err
}

func (r *HelpdeskRepository) GetBySessionID(sessionID string) (*Helpdesk, error) {
	var helpdesk Helpdesk
	query := `SELECT id, session_id, platform, platform_unique_id, status, user_id, created_at 
			  FROM helpdesk WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.Get(&helpdesk, query, sessionID)
	if err != nil {
		return nil, err
	}
	return &helpdesk, nil
}

func (r *HelpdeskRepository) SolvedConversation(id uuid.UUID) error {
	_, err := r.db.Exec(`UPDATE helpdesk SET status = $1 WHERE session_id = $2`, "resolved", id)
	return err
}

func (r *HelpdeskRepository) EndTimestampConversation(id uuid.UUID, endTimestamp string) error {
	_, err := r.db.Exec(`UPDATE conversations SET end_timestamp = $1 WHERE id = $2`, endTimestamp, id)
	return err
}
