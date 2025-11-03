package user

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	DB *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) CreateUser(user *User) (*User, error) {
	query := `
		INSERT INTO users (name, email, password, account_type, phone, role_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, email, account_type, phone, role_id;
	`
	var createdUser User
	err := r.DB.Get(&createdUser, query, user.Name, user.Email, user.Password, user.AccountType, user.Phone, user.RoleID)
	return &createdUser, err
}

func (r *UserRepository) GetUsers(query *GetUsersQuery) ([]User, error) {
	var users []User
	baseQuery := `SELECT id, name, email, account_type, phone, role_id FROM users WHERE 1=1`
	var args []interface{}
	argIndex := 1

	if query.AccountType != nil && *query.AccountType != "" {
		baseQuery += ` AND account_type = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, *query.AccountType)
		argIndex++
	}

	if query.RoleID != nil {
		baseQuery += ` AND role_id = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, *query.RoleID)
		argIndex++
	}

	if query.TeamID != nil {
		baseQuery += ` AND role_id IN (SELECT id FROM roles WHERE team_id = $` + fmt.Sprintf("%d", argIndex) + `)`
		args = append(args, *query.TeamID)
		argIndex++
	}

	if query.Search != nil && *query.Search != "" {
		searchPattern := "%" + *query.Search + "%"
		baseQuery += ` AND (name ILIKE $` + fmt.Sprintf("%d", argIndex) + ` OR email ILIKE $` + fmt.Sprintf("%d", argIndex) + `)`
		args = append(args, searchPattern)
		argIndex++
	}

	baseQuery += ` ORDER BY id ASC LIMIT $` + fmt.Sprintf("%d", argIndex) + ` OFFSET $` + fmt.Sprintf("%d", argIndex+1)
	args = append(args, query.Limit, query.Offset)

	err := r.DB.Select(&users, baseQuery, args...)
	return users, err
}

func (r *UserRepository) GetTotalUsers(query *GetUsersQuery) (int, error) {
	var count int
	baseQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	var args []interface{}
	argIndex := 1

	if query.AccountType != nil && *query.AccountType != "" {
		baseQuery += ` AND account_type = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, *query.AccountType)
		argIndex++
	}

	if query.RoleID != nil {
		baseQuery += ` AND role_id = $` + fmt.Sprintf("%d", argIndex)
		args = append(args, *query.RoleID)
		argIndex++
	}

	if query.TeamID != nil {
		baseQuery += ` AND role_id IN (SELECT id FROM roles WHERE team_id = $` + fmt.Sprintf("%d", argIndex) + `)`
		args = append(args, *query.TeamID)
		argIndex++
	}

	if query.Search != nil && *query.Search != "" {
		searchPattern := "%" + *query.Search + "%"
		baseQuery += ` AND (name ILIKE $` + fmt.Sprintf("%d", argIndex) + ` OR email ILIKE $` + fmt.Sprintf("%d", argIndex) + `)`
		args = append(args, searchPattern)
		argIndex++
	}

	err := r.DB.Get(&count, baseQuery, args...)
	return count, err
}

func (r *UserRepository) GetUserByID(id int) (*User, error) {
	var user User
	query := `SELECT id, name, email, account_type, phone, role_id FROM users WHERE id=$1;`
	err := r.DB.Get(&user, query, id)
	return &user, err
}

func (r *UserRepository) UpdateUser(id int, user *User) (*User, error) {
	query := `
		UPDATE users SET name=$1, password=$2, account_type=$3, phone=$4, role_id=$5
		WHERE id=$6
		RETURNING id, name, email, account_type, phone, role_id;
	`
	var updatedUser User
	err := r.DB.Get(&updatedUser, query, user.Name, user.Password, user.AccountType, user.Phone, user.RoleID, id)
	return &updatedUser, err
}

func (r *UserRepository) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id=$1;`
	_, err := r.DB.Exec(query, id)
	return err
}

func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	var user User
	query := `SELECT id, name, email, password, account_type, phone, role_id FROM users WHERE email=$1;`
	log.Println(query)
	err := r.DB.Get(&user, query, email)
	return &user, err
}
