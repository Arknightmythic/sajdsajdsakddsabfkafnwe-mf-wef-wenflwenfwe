package seeder

import (
	"dokuprime-be/util"
	"log"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func superadminSeeder(db *sqlx.DB) {
	// Check if superadmin user already exists
	var userCount int
	err := db.Get(&userCount, "SELECT COUNT(*) FROM users WHERE email = 'superadmin@superadmin.com'")
	if err != nil {
		log.Fatalf("Failed to check superadmin user: %v", err)
	}

	if userCount > 0 {
		log.Println("Superadmin user already exists.")
		return
	}

	tx, err := db.Beginx()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// 1. Create Superadmin Team
	teamPages := pq.StringArray{
		"dashboard",
		"knowledge-base",
		"document-management",
		"chatbot",
		"validasi",
		"panduan",
		"user-management",
		"team-management",
		"role-management",
	}

	var teamID int
	err = tx.QueryRow(`
		INSERT INTO teams (name, pages) 
		VALUES ($1, $2) 
		RETURNING id
	`, "superadmin", teamPages).Scan(&teamID)
	if err != nil {
		log.Fatalf("Failed to create superadmin team: %v", err)
	}
	log.Printf("Created superadmin team with ID: %d", teamID)

	// 2. Get all permission IDs from the permissions table
	var permissionIDs []string
	rows, err := tx.Query("SELECT id FROM permissions ORDER BY id")
	if err != nil {
		log.Fatalf("Failed to fetch permissions: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var permID int
		if err := rows.Scan(&permID); err != nil {
			log.Fatalf("Failed to scan permission ID: %v", err)
		}
		permissionIDs = append(permissionIDs, strconv.Itoa(permID))
	}

	if len(permissionIDs) == 0 {
		log.Println("Warning: No permissions found. Make sure to run permission seeder first.")
	}

	// 3. Create Superadmin Role
	var roleID int
	err = tx.QueryRow(`
		INSERT INTO roles (name, permissions, team_id) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`, "superadmin", pq.StringArray(permissionIDs), teamID).Scan(&roleID)
	if err != nil {
		log.Fatalf("Failed to create superadmin role: %v", err)
	}
	log.Printf("Created superadmin role with ID: %d with %d permissions", roleID, len(permissionIDs))

	// 4. Create Superadmin User
	hashedPassword, err := util.GenerateDeterministicHash("superadmin")
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	var userID int
	err = tx.QueryRow(`
		INSERT INTO users (email, password, account_type, phone, role_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id
	`, "superadmin@superadmin.com", hashedPassword, "superadmin", "", roleID).Scan(&userID)
	if err != nil {
		log.Fatalf("Failed to create superadmin user: %v", err)
	}
	log.Printf("Created superadmin user with ID: %d", userID)

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Println("Superadmin seeder completed successfully.")
}
