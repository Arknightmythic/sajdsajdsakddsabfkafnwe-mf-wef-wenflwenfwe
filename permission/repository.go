package permission

import (
	"github.com/jmoiron/sqlx"
)

type PermissionRepository struct {
	db *sqlx.DB
}

func NewPermissionRepository(db *sqlx.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) Create(permission *Permission) error {
	query := `INSERT INTO permissions (name) VALUES ($1) RETURNING id`
	return r.db.QueryRow(query, permission.Name).Scan(&permission.ID)
}

func (r *PermissionRepository) GetAll() ([]Permission, error) {
	var permissions []Permission
	err := r.db.Select(&permissions, `SELECT id, name FROM permissions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *PermissionRepository) GetByID(id int) (*Permission, error) {
	var permission Permission
	err := r.db.Get(&permission, `SELECT id, name FROM permissions WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *PermissionRepository) Update(permission *Permission) error {
	_, err := r.db.Exec(`UPDATE permissions SET name = $1 WHERE id = $2`, permission.Name, permission.ID)
	return err
}

func (r *PermissionRepository) Delete(id int) error {
	_, err := r.db.Exec(`DELETE FROM permissions WHERE id = $1`, id)
	return err
}
