package role

import (
	"github.com/jmoiron/sqlx"
)

type RoleRepository struct {
	db *sqlx.DB
}

func NewRoleRepository(db *sqlx.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(role Role) error {
	_, err := r.db.Exec(`
		INSERT INTO roles (name, permissions, team_id)
		VALUES ($1, $2, $3)
	`, role.Name, role.Permissions, role.TeamID)
	return err
}

func (r *RoleRepository) GetAll() ([]Role, error) {
	var roles []Role
	err := r.db.Select(&roles, "SELECT * FROM roles")
	return roles, err
}

func (r *RoleRepository) GetByID(id int) (*Role, error) {
	var role Role
	err := r.db.Get(&role, "SELECT * FROM roles WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) Update(id int, role Role) error {
	_, err := r.db.Exec(`
		UPDATE roles 
		SET name=$1, permissions=$2, team_id=$3
		WHERE id=$4
	`, role.Name, role.Permissions, role.TeamID, id)
	return err
}

func (r *RoleRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM roles WHERE id=$1", id)
	return err
}