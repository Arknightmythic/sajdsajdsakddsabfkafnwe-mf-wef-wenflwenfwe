package helpdesk

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

func (r *HelpdeskRepository) GetAll(limit, offset int, search string) ([]Helpdesk, error) {
	var helpdesks []Helpdesk
	query := `SELECT id, session_id, platform, platform_unique_id, status, user_id, created_at 
			  FROM helpdesk`

	if search != "" {
		query += ` WHERE platform ILIKE $3 OR status ILIKE $3`
		err := r.db.Select(&helpdesks, query+` ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset, "%"+search+"%")
		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.Select(&helpdesks, query+` ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, err
		}
	}

	return helpdesks, nil
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
