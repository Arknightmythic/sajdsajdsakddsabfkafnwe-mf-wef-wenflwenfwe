package migrate

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func RunMigrations(db *sqlx.DB) {
	log.Println("Starting migrations...")

	// Catatan: Urutan pembuatan tabel dipertahankan agar tidak error saat fresh install
	// Users dibuat lebih dulu, kemudian Roles. Oleh karena itu Constraint Users -> Roles 
	// sebaiknya ditangani via ALTER TABLE di bawah agar aman dari urutan pembuatan.

	query := `
    -- 1. Independent Tables
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

    -- Updated: Menambahkan ON DELETE CASCADE pada definisi awal untuk fresh install
    CREATE TABLE IF NOT EXISTS roles (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        permissions TEXT[],
        team_id INT REFERENCES teams(id) ON DELETE CASCADE
    );
    
    CREATE TABLE IF NOT EXISTS documents (
        id SERIAL PRIMARY KEY,
        category VARCHAR(100) NOT NULL
    );

    CREATE TABLE IF NOT EXISTS greetings (
        id SERIAL PRIMARY KEY,
        greetings_text TEXT
    );

    CREATE TABLE IF NOT EXISTS guides (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        filename VARCHAR(255) NOT NULL,
        original_filename VARCHAR(255),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS user_query_classifications (
        id SERIAL PRIMARY KEY,
        category TEXT NOT NULL,
        sub_category TEXT NOT NULL,
        detail TEXT
    );

    -- 2. Tables with Foreign Keys or Dependencies
    -- Note: Table 'roles' sudah didefinisikan di atas, duplikasi IF NOT EXISTS aman tapi redundant.

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
        created_at TIMESTAMP DEFAULT NOW(),
        ingest_status TEXT
    );

    CREATE TABLE IF NOT EXISTS conversations (
        id UUID PRIMARY KEY,
        start_timestamp TIMESTAMP NOT NULL,
        end_timestamp TIMESTAMP,
        platform TEXT NOT NULL,
        platform_unique_id TEXT NOT NULL,
        is_helpdesk BOOLEAN DEFAULT false NOT NULL,
        context TEXT NULL,
        is_positive_feedback BOOLEAN,
        helpdesk_count INT
    );

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
        revision TEXT,
        is_validated BOOLEAN,
        start_timestamp TIMESTAMP,
        citation JSONB
    );

    CREATE TABLE IF NOT EXISTS chat_history_outside_oss (
        id BIGSERIAL PRIMARY KEY,
        message TEXT NOT NULL,
        question_category VARCHAR(100),
        question_sub_category VARCHAR(100),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        session_id UUID
    );

    CREATE TABLE IF NOT EXISTS helpdesk (
        id SERIAL PRIMARY KEY,
        session_id UUID NOT NULL REFERENCES conversations(id),
        platform VARCHAR(50) NOT NULL,
        platform_unique_id VARCHAR(100),
        status VARCHAR(50) NOT NULL,
        user_id INT,
        created_at TIMESTAMP DEFAULT NOW() NOT NULL
    );

    -- 3. Reporting / TBL Tables (No PKs defined in DDL, but creating as requested)
    CREATE TABLE IF NOT EXISTS tbl_agent_conv (
        user_id VARCHAR(50),
        conversation_id VARCHAR(50),
        start_date VARCHAR(50),
        end_date VARCHAR(50),
        is_positive_feedback INT
    );

    CREATE TABLE IF NOT EXISTS tbl_agent_conv_detail (
        conversation_id VARCHAR(50),
        message_id VARCHAR(50),
        start_date VARCHAR(50),
        end_date VARCHAR(50),
        question VARCHAR(50),
        answer VARCHAR(50)
    );

    CREATE TABLE IF NOT EXISTS tbl_user_conv (
        user_id VARCHAR(50),
        conversation_id VARCHAR(50),
        created_at VARCHAR(50),
        is_helpdesk INT,
        channel VARCHAR(50)
    );

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

    -- ============================================================
    -- UPDATE FOREIGN KEY CONSTRAINTS (CASCADE & SET NULL)
    -- ============================================================
    
    -- 1. Roles: Hapus FK lama jika ada, buat baru dengan ON DELETE CASCADE
    ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_team_id_fkey;
    ALTER TABLE roles ADD CONSTRAINT roles_team_id_fkey 
        FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;

    -- 2. Users: Hapus FK lama jika ada, buat baru dengan ON DELETE SET NULL
    ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_id_fkey;
    ALTER TABLE users ADD CONSTRAINT users_role_id_fkey 
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL;
    
    -- Masukkan tabel ini ke dalam string query
    CREATE TABLE IF NOT EXISTS tbl_user_conv_detail (
        conversation_id VARCHAR(50),
        message_id VARCHAR(50),
        start_date VARCHAR(50),
        end_date VARCHAR(50),
        question VARCHAR(128),
        answer VARCHAR(128),
        is_cannot_answer INT,
        is_positive_feedback INT,
        is_validated INT,
        category VARCHAR(50),
        sub_category VARCHAR(50)
    );

    -- 4. Indices
    CREATE INDEX IF NOT EXISTS idx_chat_history_session_id ON chat_history(session_id);
    CREATE INDEX IF NOT EXISTS idx_chat_history_user_id ON chat_history(user_id);
    CREATE INDEX IF NOT EXISTS idx_conversations_platform_unique_id ON conversations(platform_unique_id);
    CREATE INDEX IF NOT EXISTS idx_document_details_data_type ON document_details(data_type);
    CREATE INDEX IF NOT EXISTS idx_document_details_document_id ON document_details(document_id);
    CREATE INDEX IF NOT EXISTS idx_document_details_is_latest ON document_details(is_latest);
    CREATE INDEX IF NOT EXISTS idx_document_details_status ON document_details(status);

    -- 5. Alterations (Idempotency Checks)
    -- If tables exist from previous runs, ensure new columns are added.

    -- Updates for 'conversations'
    DO $$ 
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='conversations' AND column_name='is_positive_feedback') THEN
            ALTER TABLE conversations ADD COLUMN is_positive_feedback BOOLEAN;
        END IF;
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='conversations' AND column_name='helpdesk_count') THEN
            ALTER TABLE conversations ADD COLUMN helpdesk_count INT;
        END IF;
    END $$;

    -- Updates for 'chat_history'
    DO $$ 
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='chat_history' AND column_name='is_validated') THEN
            ALTER TABLE chat_history ADD COLUMN is_validated BOOLEAN;
        END IF;
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='chat_history' AND column_name='start_timestamp') THEN
            ALTER TABLE chat_history ADD COLUMN start_timestamp TIMESTAMP;
        END IF;
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='chat_history' AND column_name='citation') THEN
            ALTER TABLE chat_history ADD COLUMN citation JSONB;
        END IF;
    END $$;

    -- Updates for 'document_details'
    DO $$ 
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='document_details' AND column_name='ingest_status') THEN
            ALTER TABLE document_details ADD COLUMN ingest_status TEXT;
        END IF;
    END $$;

    -- Updates for 'users' (Legacy check from old migrate)
    DO $$ 
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='name') THEN
            ALTER TABLE users ADD COLUMN name VARCHAR(255);
            UPDATE users SET name = 'User' WHERE name IS NULL;
            ALTER TABLE users ALTER COLUMN name SET NOT NULL;
        END IF;
    END $$;
    `

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully.")
}