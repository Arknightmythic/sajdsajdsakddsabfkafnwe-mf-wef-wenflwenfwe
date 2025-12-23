package user

import (
	"fmt"

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
		baseQuery += ` AND account_type = $` + fmt.Sprint(argIndex)
		args = append(args, *query.AccountType)
		argIndex++
	}

	if query.RoleID != nil {
		baseQuery += ` AND role_id = $` + fmt.Sprint(argIndex)
		args = append(args, *query.RoleID)
		argIndex++
	}

	if query.TeamID != nil {
		baseQuery += ` AND role_id IN (SELECT id FROM roles WHERE team_id = $` + fmt.Sprint(argIndex) + `)`
		args = append(args, *query.TeamID)
		argIndex++
	}

	if query.Search != nil && *query.Search != "" {
		searchPattern := "%" + *query.Search + "%"
		placeholder := "$" + fmt.Sprint(argIndex)
		baseQuery += ` AND (name ILIKE ` + placeholder + ` OR email ILIKE ` + placeholder + `)`
		args = append(args, searchPattern)
		argIndex++
	}

	limitPlaceholder := "$" + fmt.Sprint(argIndex)
	offsetPlaceholder := "$" + fmt.Sprint(argIndex+1)
	baseQuery += ` ORDER BY id ASC LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
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
		baseQuery += ` AND account_type = $` + fmt.Sprint(argIndex)
		args = append(args, *query.AccountType)
		argIndex++
	}

	if query.RoleID != nil {
		baseQuery += ` AND role_id = $` + fmt.Sprint(argIndex)
		args = append(args, *query.RoleID)
		argIndex++
	}

	if query.TeamID != nil {
		baseQuery += ` AND role_id IN (SELECT id FROM roles WHERE team_id = $` + fmt.Sprint(argIndex) + `)`
		args = append(args, *query.TeamID)
		argIndex++
	}

	if query.Search != nil && *query.Search != "" {
		searchPattern := "%" + *query.Search + "%"
		placeholder := "$" + fmt.Sprint(argIndex)
		baseQuery += ` AND (name ILIKE ` + placeholder + ` OR email ILIKE ` + placeholder + `)`
		args = append(args, searchPattern)
		argIndex++
	}

	err := r.DB.Get(&count, baseQuery, args...)
	return count, err
}

func (r *UserRepository) GetUserByID(id int) (*User, error) {
	var user User
	query := `SELECT id, name, email, account_type, phone, role_id FROM users WHERE id = $1;`
	err := r.DB.Get(&user, query, id)
	return &user, err
}

func (r *UserRepository) UpdateUser(id int, user *User) (*User, error) {
	var query string
	var args []interface{}
	if user.Password == "" {
		query = `
			UPDATE users SET name = $1, account_type = $2, phone = $3, role_id = $4
			WHERE id = $5
			RETURNING id, name, email, account_type, phone, role_id;
		`
		args = []interface{}{user.Name, user.AccountType, user.Phone, user.RoleID, id}
	} else {
		query = `
			UPDATE users SET name = $1, password = $2, account_type = $3, phone = $4, role_id = $5
			WHERE id = $6
			RETURNING id, name, email, account_type, phone, role_id;
		`
		args = []interface{}{user.Name, user.Password, user.AccountType, user.Phone, user.RoleID, id}
	}

	var updatedUser User
	err := r.DB.Get(&updatedUser, query, args...)
	return &updatedUser, err
}

func (r *UserRepository) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id = $1;`
	_, err := r.DB.Exec(query, id)
	return err
}

func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	var user User
	query := `SELECT id, name, email, password, account_type, phone, role_id FROM users WHERE email = $1;`
	err := r.DB.Get(&user, query, email)
	return &user, err
}
