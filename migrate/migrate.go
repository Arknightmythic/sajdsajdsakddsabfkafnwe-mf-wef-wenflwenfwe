package migrate

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func RunMigrations(db *sqlx.DB) {
	log.Println("Starting migrations...")

	var currentDB string
	err := db.Get(&currentDB, "SELECT current_database()")
	if err != nil {
		log.Fatalf("Failed to get current database: %v", err)
	}
	log.Printf("Current database: %s", currentDB)

	query := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        email VARCHAR(255) NOT NULL UNIQUE,
        password VARCHAR(255) NOT NULL,
        account_type VARCHAR(50),
        phone VARCHAR(50)
    );
    `

	result, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	var tableExists bool
	err = db.Get(&tableExists, `
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name = 'users'
        )
    `)
	if err != nil {
		log.Printf("Warning: Could not verify table creation: %v", err)
	} else {
		log.Printf("Table 'users' exists: %v", tableExists)
	}

	log.Printf("Migration completed successfully. Result: %v", result)
}
