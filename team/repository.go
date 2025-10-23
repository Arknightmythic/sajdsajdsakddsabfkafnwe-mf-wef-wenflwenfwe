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

func (r *TeamRepository) GetAll() ([]Team, error) {
	var teams []Team
	err := r.db.Select(&teams, `SELECT id, name, pages FROM teams`)
	if err != nil {
		return nil, err
	}
	return teams, nil
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
