package user

import (
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
		INSERT INTO users (email, password, account_type, phone, role_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, account_type, phone, role_id;
	`
	var createdUser User
	err := r.DB.Get(&createdUser, query, user.Email, user.Password, user.AccountType, user.Phone, user.RoleID)
	return &createdUser, err
}

func (r *UserRepository) GetUsers() ([]User, error) {
	var users []User
	query := `SELECT id, email, account_type, phone, role_id FROM users ORDER BY id ASC;`
	err := r.DB.Select(&users, query)
	return users, err
}

func (r *UserRepository) GetUserByID(id int) (*User, error) {
	var user User
	query := `SELECT id, email, account_type, phone, role_id FROM users WHERE id=$1;`
	err := r.DB.Get(&user, query, id)
	return &user, err
}

func (r *UserRepository) UpdateUser(id int, user *User) (*User, error) {
	query := `
		UPDATE users SET email=$1, password=$2, account_type=$3, phone=$4, role_id=$5
		WHERE id=$6
		RETURNING id, email, account_type, phone, role_id;
	`
	var updatedUser User
	err := r.DB.Get(&updatedUser, query, user.Email, user.Password, user.AccountType, user.Phone, user.RoleID, id)
	return &updatedUser, err
}

func (r *UserRepository) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id=$1;`
	_, err := r.DB.Exec(query, id)
	return err
}

func (r *UserRepository) GetUserByEmail(email string) (*User, error) {
	var user User
	query := `SELECT id, email, password, account_type, phone FROM users WHERE email=$1;`
	err := r.DB.Get(&user, query, email)
	return &user, err
}
