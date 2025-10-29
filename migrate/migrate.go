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
		name VARCHAR(255) NOT NULL,
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

	CREATE TABLE IF NOT EXISTS documents (
		id SERIAL PRIMARY KEY,
		category VARCHAR(100) NOT NULL
	);

	CREATE TABLE IF NOT EXISTS document_details (
		id SERIAL PRIMARY KEY,
		document_id INT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
		document_name VARCHAR(255) NOT NULL,
		filename VARCHAR(255) NOT NULL,
		data_type VARCHAR(10) NOT NULL,
		staff VARCHAR(255) NOT NULL,
		team VARCHAR(100) NOT NULL,
		status VARCHAR(50),
		is_latest BOOLEAN,
		is_approve BOOLEAN,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_document_details_document_id ON document_details(document_id);
	CREATE INDEX IF NOT EXISTS idx_document_details_is_latest ON document_details(is_latest);
	CREATE INDEX IF NOT EXISTS idx_document_details_data_type ON document_details(data_type);
	CREATE INDEX IF NOT EXISTS idx_document_details_status ON document_details(status);

	-- Add name column if it doesn't exist (for existing databases)
	DO $$ 
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='users' AND column_name='name') THEN
			ALTER TABLE users ADD COLUMN name VARCHAR(255);
			UPDATE users SET name = 'User' WHERE name IS NULL;
			ALTER TABLE users ALTER COLUMN name SET NOT NULL;
		END IF;
	END $$;

	-- Add data_type column if it doesn't exist (for existing databases)
	DO $$ 
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='document_details' AND column_name='data_type') THEN
			ALTER TABLE document_details ADD COLUMN data_type VARCHAR(10);
			UPDATE document_details SET data_type = 'pdf' WHERE data_type IS NULL;
			ALTER TABLE document_details ALTER COLUMN data_type SET NOT NULL;
		END IF;
	END $$;
	`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully.")
}
