package team

import (
	"github.com/jmoiron/sqlx"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(team *Team) error {
	_, err := r.db.Exec(`INSERT INTO teams (name, pages) VALUES ($1, $2)`, team.Name, team.Pages)
	return err
}

func (r *TeamRepository) GetAll(limit, offset int, search string) ([]Team, error) {
	var teams []Team
	query := `SELECT id, name, pages FROM teams`

	if search != "" {
		query += ` WHERE name ILIKE $3`
		err := r.db.Select(&teams, query+` ORDER BY id LIMIT $1 OFFSET $2`, limit, offset, "%"+search+"%")
		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.Select(&teams, query+` ORDER BY id LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, err
		}
	}

	return teams, nil
}

func (r *TeamRepository) GetTotal(search string) (int, error) {
	var total int
	query := `SELECT COUNT(*) FROM teams`

	if search != "" {
		query += ` WHERE name ILIKE $1`
		err := r.db.Get(&total, query, "%"+search+"%")
		return total, err
	}

	err := r.db.Get(&total, query)
	return total, err
}

func (r *TeamRepository) GetByID(id int) (*Team, error) {
	var team Team
	err := r.db.Get(&team, `SELECT id, name, pages FROM teams WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepository) Update(team *Team) error {
	_, err := r.db.Exec(`UPDATE teams SET name = $1, pages = $2 WHERE id = $3`, team.Name, team.Pages, team.ID)
	return err
}

func (r *TeamRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM teams WHERE id = $1`, id)
	return err
}
