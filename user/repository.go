package user

import (
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

func (r *UserRepository) GetUsers(limit, offset int) ([]User, error) {
	var users []User
	query := `SELECT id, name, email, account_type, phone, role_id FROM users ORDER BY id ASC LIMIT $1 OFFSET $2;`
	err := r.DB.Select(&users, query, limit, offset)
	return users, err
}

func (r *UserRepository) GetTotalUsers() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users;`
	err := r.DB.Get(&count, query)
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