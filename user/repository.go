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
		INSERT INTO users (email, password, account_type, phone)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, account_type, phone;
	`
	var createdUser User
	err := r.DB.Get(&createdUser, query, user.Email, user.Password, user.AccountType, user.Phone)
	return &createdUser, err
}

func (r *UserRepository) GetUsers() ([]User, error) {
	var users []User
	query := `SELECT id, email, account_type, phone FROM users ORDER BY id ASC;`
	err := r.DB.Select(&users, query)
	return users, err
}

func (r *UserRepository) GetUserByID(id int) (*User, error) {
	var user User
	query := `SELECT id, email, account_type, phone FROM users WHERE id=$1;`
	err := r.DB.Get(&user, query, id)
	return &user, err
}

func (r *UserRepository) UpdateUser(id int, user *User) (*User, error) {
	query := `
		UPDATE users SET email=$1, password=$2, account_type=$3, phone=$4
		WHERE id=$5
		RETURNING id, email, account_type, phone;
	`
	var updatedUser User
	err := r.DB.Get(&updatedUser, query, user.Email, user.Password, user.AccountType, user.Phone, id)
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
