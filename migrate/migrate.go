package migrate

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func RunMigrations(db *sqlx.DB) {
	log.Println("Starting migrations...")

	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL,
		account_type VARCHAR(50),
		phone VARCHAR(50),
		role_id INT
	);

	CREATE TABLE IF NOT EXISTS permissions (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS teams (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		pages TEXT[]
	);

	CREATE TABLE IF NOT EXISTS roles (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		permissions TEXT[],
		team_id INT REFERENCES teams(id) ON DELETE SET NULL
	);
	`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully.")
}
