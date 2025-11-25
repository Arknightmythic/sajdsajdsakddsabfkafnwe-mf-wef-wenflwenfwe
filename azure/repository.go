package azure

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type AzureRepository struct {
	DB *sqlx.DB
}

func NewAzureRepository(db *sqlx.DB) *AzureRepository {
	return &AzureRepository{DB: db}
}

type UserData struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func (r *AzureRepository) FindOrCreateUser(displayName, email string) (*UserData, error) {
	var user UserData

	query := `SELECT id, name, email FROM users WHERE email = $1`
	err := r.DB.Get(&user, query, email)

	if err == nil {
		return &user, nil
	}

	roleId, err := r.findDefaultRole()
	if err != nil {
		fmt.Printf("Warning: failed to create user_management entry: %v\n", err)
	}

	unusablePassword := generateSecurePassword()
	accountType := "microsoft"

	insertQuery := `
		INSERT INTO users (name, email, password, account_type, role_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, email
	`

	err = r.DB.Get(&user, insertQuery, displayName, email, unusablePassword, accountType, roleId)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (r *AzureRepository) findDefaultRole() (int, error) {
	var rolesId int
	teamQuery := `SELECT id FROM roles WHERE name = 'default' LIMIT 1`
	err := r.DB.Get(&rolesId, teamQuery)
	if err != nil {
		return 0, fmt.Errorf("default team 'default' not found: %w", err)
	}

	return rolesId, nil
}

func (r *AzureRepository) GetUserByID(id int64) (*UserData, error) {
	var user UserData
	query := `SELECT id, username, email FROM users WHERE id = $1`
	err := r.DB.Get(&user, query, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AzureRepository) GetUserByEmail(email string) (*UserData, error) {
	var user UserData
	query := `SELECT id, username, email FROM users WHERE email = $1`
	err := r.DB.Get(&user, query, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func generateSecurePassword() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func (r *AzureRepository) GetRoleIDByUserID(userID int64) (*int, error) {
	var roleID *int
	query := `SELECT role_id FROM users WHERE id = $1`
	err := r.DB.Get(&roleID, query, userID)
	if err != nil {
		return nil, err
	}
	return roleID, nil
}

func (r *AzureRepository) GetAccountTypeByUserID(userID int64) (*string, error) {
	var accountType *string
	query := `SELECT account_type FROM users WHERE id = $1`
	err := r.DB.Get(&accountType, query, userID)
	if err != nil {
		return nil, err
	}
	return accountType, nil
}
