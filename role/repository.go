package role

import (
	"fmt"

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

func (r *RoleRepository) GetRoleByTeamID(teamID int) ([]Role, error) {
	var roles []Role
	err := r.db.Select(&roles, "SELECT * FROM roles WHERE team_id=$1", teamID)
	return roles, err
}

func (r *RoleRepository) GetAll(limit, offset int, search string, teamID *int) ([]Role, error) {
	var roles []Role
	var args []interface{}
	argCount := 1

	query := `
		SELECT DISTINCT r.* 
		FROM roles r
		LEFT JOIN permissions p ON p.id::text = ANY(r.permissions)
		WHERE 1=1
	`

	if search != "" {
		query += fmt.Sprintf(` AND (
			r.name ILIKE $%d OR 
			p.name ILIKE $%d
		)`, argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	if teamID != nil {
		query += fmt.Sprintf(` AND r.team_id = $%d`, argCount)
		args = append(args, *teamID)
		argCount++
	}

	query += fmt.Sprintf(` ORDER BY r.id LIMIT $%d OFFSET $%d`, argCount, argCount+1)
	args = append(args, limit, offset)

	err := r.db.Select(&roles, query, args...)
	return roles, err
}

func (r *RoleRepository) GetTotal(search string, teamID *int) (int, error) {
	var total int
	var args []interface{}
	argCount := 1

	query := `
		SELECT COUNT(DISTINCT r.id) 
		FROM roles r
		LEFT JOIN permissions p ON p.id::text = ANY(r.permissions)
		WHERE 1=1
	`

	if search != "" {
		query += fmt.Sprintf(` AND (
			r.name ILIKE $%d OR 
			p.name ILIKE $%d
		)`, argCount, argCount)
		args = append(args, "%"+search+"%")
		argCount++
	}

	if teamID != nil {
		query += fmt.Sprintf(` AND r.team_id = $%d`, argCount)
		args = append(args, *teamID)
		argCount++
	}

	err := r.db.Get(&total, query, args...)
	return total, err
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
