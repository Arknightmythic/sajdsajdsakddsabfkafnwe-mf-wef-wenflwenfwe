package helpdesk

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type HelpdeskRepository struct {
	db *sqlx.DB
}

func NewHelpdeskRepository(db *sqlx.DB) *HelpdeskRepository {
	return &HelpdeskRepository{db: db}
}

func (r *HelpdeskRepository) GetSwitchStatus() (*SwitchHelpdesk, error) {
	var sh SwitchHelpdesk

	query := `SELECT id, status FROM switch_helpdesk LIMIT 1`
	err := r.db.Get(&sh, query)

	if err != nil {
		if err == sql.ErrNoRows {
			insertQuery := `INSERT INTO switch_helpdesk (status) VALUES (false) RETURNING id, status`
			err = r.db.QueryRowx(insertQuery).StructScan(&sh)
			if err != nil {
				return nil, fmt.Errorf("failed to insert default switch status: %w", err)
			}
			return &sh, nil
		}
		return nil, err
	}

	return &sh, nil
}

func (r *HelpdeskRepository) UpdateSwitchStatus(status bool) (*SwitchHelpdesk, error) {
	_, err := r.GetSwitchStatus()
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE switch_helpdesk 
		SET status = $1 
		WHERE id = (SELECT id FROM switch_helpdesk LIMIT 1)
		RETURNING id, status
	`

	var sh SwitchHelpdesk
	err = r.db.QueryRowx(query, status).StructScan(&sh)
	if err != nil {
		return nil, err
	}

	return &sh, nil
}

func (r *HelpdeskRepository) Create(helpdesk *Helpdesk) error {
	query := `INSERT INTO helpdesk (session_id, platform, platform_unique_id, status, user_id) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return r.db.QueryRow(query, helpdesk.SessionID, helpdesk.Platform, helpdesk.PlatformUniqueID, helpdesk.Status, helpdesk.UserID).
		Scan(&helpdesk.ID, &helpdesk.CreatedAt)
}

func (r *HelpdeskRepository) GetAll(limit, offset int, search string, status string) ([]Helpdesk, int, error) {
	helpdesks := []Helpdesk{}
	var conditions []string
	var args []interface{}
	argIdx := 1

	query := `SELECT id, session_id, platform, platform_unique_id, status, user_id, created_at 
			  FROM helpdesk`

	if search != "" {
		placeholder := "$" + fmt.Sprint(argIdx)
		conditions = append(conditions, "(platform ILIKE "+placeholder+" OR platform_unique_id ILIKE "+placeholder+" OR session_id::text ILIKE "+placeholder+")")
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if status != "" {
		conditions = append(conditions, "status ILIKE $"+fmt.Sprint(argIdx))
		args = append(args, status)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM helpdesk" + where
	var total int
	if err := r.db.Get(&total, countQuery, args...); err != nil {
		return []Helpdesk{}, 0, err
	}

	if total == 0 {
		return []Helpdesk{}, 0, nil
	}

	limitPlaceholder := "$" + fmt.Sprint(argIdx)
	offsetPlaceholder := "$" + fmt.Sprint(argIdx+1)

	fullQuery := query + where + " ORDER BY created_at DESC LIMIT " + limitPlaceholder + " OFFSET " + offsetPlaceholder
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

func (r *HelpdeskRepository) UpdateStatus(id int, status string, userID any) error {
	if status == "in_progress" && userID != nil {
		query := `UPDATE helpdesk SET status = $1, user_id = $2 WHERE id = $3`
		_, err := r.db.Exec(query, status, userID, id)
		return err
	} else {
		query := `UPDATE helpdesk SET status = $1 WHERE id = $2`
		_, err := r.db.Exec(query, status, id)
		return err
	}
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


func (r *HelpdeskRepository) GetSummary() (*HelpdeskSummary, error) {
	var summary HelpdeskSummary
		
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN status ILIKE 'queue' OR status ILIKE 'open' THEN 1 ELSE 0 END), 0) as queue,
			COALESCE(SUM(CASE WHEN status ILIKE 'in_progress' THEN 1 ELSE 0 END), 0) as active,
			COALESCE(SUM(CASE WHEN status ILIKE 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN status ILIKE 'resolved' OR status ILIKE 'closed' THEN 1 ELSE 0 END), 0) as resolved
		FROM helpdesk
	`
	
	err := r.db.Get(&summary, query)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}