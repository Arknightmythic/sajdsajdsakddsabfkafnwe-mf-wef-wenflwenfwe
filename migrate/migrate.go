package migrate

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func RunMigrations(db *sqlx.DB) {
	log.Println("Starting migrations...")

	query := `
    -- ============================================================
    -- 1. INDEPENDENT TABLES (No Foreign Keys)
    -- ============================================================

    CREATE TABLE IF NOT EXISTS permissions (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100) NOT NULL UNIQUE
    );

    CREATE TABLE IF NOT EXISTS teams (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        pages TEXT[]
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

    CREATE TABLE IF NOT EXISTS conversations (
        id UUID PRIMARY KEY,
        start_timestamp TIMESTAMP NOT NULL,
        end_timestamp TIMESTAMP,
        platform TEXT NOT NULL,
        platform_unique_id TEXT NOT NULL,
        is_helpdesk BOOLEAN DEFAULT false NOT NULL,
        context TEXT,
        is_positive_feedback BOOLEAN,
        is_ask_helpdesk BOOLEAN
    );

    CREATE TABLE IF NOT EXISTS switch_helpdesk (
        id SERIAL PRIMARY KEY,
        status BOOLEAN
    );

    CREATE TABLE IF NOT EXISTS url_format (
        id SERIAL PRIMARY KEY,
        kode VARCHAR(50) NOT NULL UNIQUE,
        url TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS processed_messages (
        message_id TEXT NOT NULL,
        platform VARCHAR(50) NOT NULL,
        processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        CONSTRAINT processed_messages_pk PRIMARY KEY (message_id, platform)
    );

    CREATE TABLE IF NOT EXISTS run_times (
        id SERIAL PRIMARY KEY,
        dttm TIMESTAMPTZ DEFAULT NOW(),
        question_id SERIAL NOT NULL,
        answer_id SERIAL NOT NULL,
        qdrant_faq_time FLOAT8,
        qdrant_main_time FLOAT8,
        rerank_time FLOAT8,
        llm_time FLOAT8,
        duration_rewriter FLOAT8,
        duration_classify_collection FLOAT8,
        duration_question_classifier FLOAT8,
        duration_classify_kbli FLOAT8,
        duration_classify_specific FLOAT8
    );

    -- Reporting Tables
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

    -- ============================================================
    -- 2. TABLES WITH DEPENDENCIES (Foreign Keys)
    -- ============================================================

    -- Roles table depends on teams
    CREATE TABLE IF NOT EXISTS roles (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        permissions TEXT[],
        team_id INT
    );

    -- Users table depends on roles
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) NOT NULL UNIQUE,
        password VARCHAR(255) NOT NULL,
        account_type VARCHAR(50),
        phone VARCHAR(50),
        role_id INT
    );

    -- Document details depends on documents
    CREATE TABLE IF NOT EXISTS document_details (
        id SERIAL PRIMARY KEY,
        document_id INT NOT NULL,
        document_name VARCHAR(255) NOT NULL,
        filename VARCHAR(255) NOT NULL,
        data_type VARCHAR(10) NOT NULL,
        staff VARCHAR(255) NOT NULL,
        team VARCHAR(100) NOT NULL,
        status VARCHAR(50),
        is_latest BOOLEAN,
        is_approve BOOLEAN,
        created_at TIMESTAMP DEFAULT NOW(),
        ingest_status TEXT,
        request_type VARCHAR(20) DEFAULT 'NEW',
        requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        CONSTRAINT check_request_type CHECK (request_type IN ('NEW', 'UPDATE', 'DELETE'))
    );

    -- Chat history
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
        is_answered BOOLEAN DEFAULT false NOT NULL,
        revision TEXT,
        is_validated BOOLEAN,
        start_timestamp TIMESTAMP,
        citation JSONB,
        validator INT
    );

    CREATE TABLE IF NOT EXISTS chat_history_outside_oss (
        id BIGSERIAL PRIMARY KEY,
        message TEXT NOT NULL,
        question_category VARCHAR(100),
        question_sub_category VARCHAR(100),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        session_id UUID
    );

    -- Helpdesk depends on conversations
    CREATE TABLE IF NOT EXISTS helpdesk (
        id SERIAL PRIMARY KEY,
        session_id UUID NOT NULL,
        platform VARCHAR(50) NOT NULL,
        platform_unique_id VARCHAR(100),
        user_id INT,
        created_at TIMESTAMP DEFAULT NOW() NOT NULL,
        status VARCHAR(50)
    );

    -- Email metadata
    CREATE TABLE IF NOT EXISTS email_metadata (
        conversation_id UUID NOT NULL,
        subject TEXT,
        in_reply_to TEXT,
        "references" TEXT,
        thread_key TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        CONSTRAINT email_metadata_pk PRIMARY KEY (conversation_id)
    );

    -- ============================================================
    -- 3. ADD/UPDATE FOREIGN KEY CONSTRAINTS
    -- ============================================================
    
    -- Roles -> Teams
    DO $$ 
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name='roles_team_id_fkey' AND table_name='roles'
        ) THEN
            ALTER TABLE roles ADD CONSTRAINT roles_team_id_fkey 
                FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE;
        END IF;
    END $$;

    -- Users -> Roles (allow NULL if role is deleted)
    DO $$ 
    BEGIN
        IF EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name='users_role_id_fkey' AND table_name='users'
        ) THEN
            ALTER TABLE users DROP CONSTRAINT users_role_id_fkey;
        END IF;
        
        ALTER TABLE users ADD CONSTRAINT users_role_id_fkey 
            FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL;
    END $$;

    -- Document details -> Documents
    DO $$ 
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name='document_details_document_id_fkey' AND table_name='document_details'
        ) THEN
            ALTER TABLE document_details ADD CONSTRAINT document_details_document_id_fkey 
                FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE;
        END IF;
    END $$;

    -- Helpdesk -> Conversations
    DO $$ 
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name='helpdesk_session_id_fkey' AND table_name='helpdesk'
        ) THEN
            ALTER TABLE helpdesk ADD CONSTRAINT helpdesk_session_id_fkey 
                FOREIGN KEY (session_id) REFERENCES conversations(id);
        END IF;
    END $$;

    -- ============================================================
    -- 4. CREATE INDICES
    -- ============================================================
    
    CREATE INDEX IF NOT EXISTS idx_chat_history_session_id 
        ON chat_history USING btree (session_id);
    
    CREATE INDEX IF NOT EXISTS idx_chat_history_user_id 
        ON chat_history USING btree (user_id);
    
    CREATE INDEX IF NOT EXISTS idx_conversations_platform_unique_id 
        ON conversations USING btree (platform_unique_id);
    
    CREATE INDEX IF NOT EXISTS idx_document_details_data_type 
        ON document_details USING btree (data_type);
    
    CREATE INDEX IF NOT EXISTS idx_document_details_document_id 
        ON document_details USING btree (document_id);
    
    CREATE INDEX IF NOT EXISTS idx_document_details_is_latest 
        ON document_details USING btree (is_latest);
    
    CREATE INDEX IF NOT EXISTS idx_document_details_status 
        ON document_details USING btree (status);
    
    CREATE UNIQUE INDEX IF NOT EXISTS idx_email_thread_key 
        ON email_metadata USING btree (thread_key);

    -- ============================================================
    -- 5. COLUMN ALTERATIONS (Ensure all columns exist)
    -- ============================================================
    
    DO $$ 
    BEGIN
        -- Ensure data_type exists in document_details
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='document_details' AND column_name='data_type'
        ) THEN
            ALTER TABLE document_details ADD COLUMN data_type VARCHAR(10);
            UPDATE document_details SET data_type = 'pdf' WHERE data_type IS NULL;
            ALTER TABLE document_details ALTER COLUMN data_type SET NOT NULL;
        END IF;

        -- Ensure request_type exists in document_details
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='document_details' AND column_name='request_type'
        ) THEN
            ALTER TABLE document_details ADD COLUMN request_type VARCHAR(20) DEFAULT 'NEW';
        END IF;

        -- Ensure requested_at exists in document_details
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='document_details' AND column_name='requested_at'
        ) THEN
            ALTER TABLE document_details ADD COLUMN requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
        END IF;

        -- Ensure ingest_status exists in document_details
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='document_details' AND column_name='ingest_status'
        ) THEN
            ALTER TABLE document_details ADD COLUMN ingest_status TEXT;
        END IF;

        -- Ensure name column exists in users
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='users' AND column_name='name'
        ) THEN
            ALTER TABLE users ADD COLUMN name VARCHAR(255);
            UPDATE users SET name = 'User' WHERE name IS NULL;
            ALTER TABLE users ALTER COLUMN name SET NOT NULL;
        END IF;

        -- Ensure is_positive_feedback exists in conversations
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='conversations' AND column_name='is_positive_feedback'
        ) THEN
            ALTER TABLE conversations ADD COLUMN is_positive_feedback BOOLEAN;
        END IF;

        -- Ensure is_ask_helpdesk exists in conversations
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='conversations' AND column_name='is_ask_helpdesk'
        ) THEN
            ALTER TABLE conversations ADD COLUMN is_ask_helpdesk BOOLEAN;
        END IF;

        -- Ensure is_answered exists in chat_history
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='chat_history' AND column_name='is_answered'
        ) THEN
            ALTER TABLE chat_history ADD COLUMN is_answered BOOLEAN DEFAULT false NOT NULL;
        END IF;

        -- Ensure revision exists in chat_history
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='chat_history' AND column_name='revision'
        ) THEN
            ALTER TABLE chat_history ADD COLUMN revision TEXT;
        END IF;

        -- Ensure is_validated exists in chat_history
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='chat_history' AND column_name='is_validated'
        ) THEN
            ALTER TABLE chat_history ADD COLUMN is_validated BOOLEAN;
        END IF;

        -- Ensure start_timestamp exists in chat_history
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='chat_history' AND column_name='start_timestamp'
        ) THEN
            ALTER TABLE chat_history ADD COLUMN start_timestamp TIMESTAMP;
        END IF;

        -- Ensure citation exists in chat_history
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='chat_history' AND column_name='citation'
        ) THEN
            ALTER TABLE chat_history ADD COLUMN citation JSONB;
        END IF;

        -- Ensure validator exists in chat_history
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='chat_history' AND column_name='validator'
        ) THEN
            ALTER TABLE chat_history ADD COLUMN validator INT;
        END IF;

        -- Ensure all run_times columns exist
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='run_times' AND column_name='dttm'
        ) THEN
            ALTER TABLE run_times ADD COLUMN dttm TIMESTAMPTZ DEFAULT NOW();
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='run_times' AND column_name='duration_rewriter'
        ) THEN
            ALTER TABLE run_times ADD COLUMN duration_rewriter FLOAT8;
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='run_times' AND column_name='duration_classify_collection'
        ) THEN
            ALTER TABLE run_times ADD COLUMN duration_classify_collection FLOAT8;
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='run_times' AND column_name='duration_question_classifier'
        ) THEN
            ALTER TABLE run_times ADD COLUMN duration_question_classifier FLOAT8;
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='run_times' AND column_name='duration_classify_kbli'
        ) THEN
            ALTER TABLE run_times ADD COLUMN duration_classify_kbli FLOAT8;
        END IF;

        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='run_times' AND column_name='duration_classify_specific'
        ) THEN
            ALTER TABLE run_times ADD COLUMN duration_classify_specific FLOAT8;
        END IF;

        -- Ensure processed_at in processed_messages (renamed from created_at)
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='processed_messages' AND column_name='processed_at'
        ) AND EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='processed_messages' AND column_name='created_at'
        ) THEN
            ALTER TABLE processed_messages RENAME COLUMN created_at TO processed_at;
        END IF;
    END $$;

    `

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully.")
}