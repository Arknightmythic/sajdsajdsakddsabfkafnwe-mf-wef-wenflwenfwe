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

	CREATE TABLE IF NOT EXISTS conversations (
		id UUID PRIMARY KEY,
		start_timestamp TIMESTAMP NOT NULL,
		end_timestamp TIMESTAMP,
		platform TEXT NOT NULL,
		platform_unique_id TEXT NOT NULL,
		is_helpdesk BOOLEAN DEFAULT false NOT NULL,
		context TEXT NULL
	);

	-- Updated chat_history using schema bkpm
	CREATE SCHEMA IF NOT EXISTS bkpm;

	CREATE TABLE IF NOT EXISTS chat_history (
		id SERIAL PRIMARY KEY,
		session_id UUID NOT NULL,
		message JSONB NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		user_id BIGINT,
		is_cannot_answer BOOLEAN,
		category TEXT,
		feedback BOOLEAN,
		question_category TEXT,
		question_sub_category TEXT,
		is_answered BOOLEAN,
		revision TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_chat_history_session_id
		ON bkpm.chat_history(session_id);

	CREATE INDEX IF NOT EXISTS idx_chat_history_user_id
		ON bkpm.chat_history(user_id);


	CREATE INDEX IF NOT EXISTS idx_document_details_document_id ON document_details(document_id);
	CREATE INDEX IF NOT EXISTS idx_document_details_is_latest ON document_details(is_latest);
	CREATE INDEX IF NOT EXISTS idx_document_details_data_type ON document_details(data_type);
	CREATE INDEX IF NOT EXISTS idx_document_details_status ON document_details(status);
	CREATE INDEX IF NOT EXISTS idx_conversations_platform_unique_id ON conversations(platform_unique_id);

	-- Add name column if missing
	DO $$ 
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
					   WHERE table_name='users' AND column_name='name') THEN
			ALTER TABLE users ADD COLUMN name VARCHAR(255);
			UPDATE users SET name = 'User' WHERE name IS NULL;
			ALTER TABLE users ALTER COLUMN name SET NOT NULL;
		END IF;
	END $$;

	-- Add data_type column if missing
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
