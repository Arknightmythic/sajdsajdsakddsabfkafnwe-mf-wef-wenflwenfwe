package seeder

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func RunSeeder(db *sqlx.DB) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM permissions")
	if err != nil {
		log.Fatalf("Failed to check permissions table: %v", err)
	}

	if count > 0 {
		log.Println("Permissions already seeded.")
		return
	}

	permissions := []string{
		"agent-dashboard:access",
		"dashboard:create",
		"dashboard:read",
		"dashboard:update",
		"dashboard:delete",
		"dashboard:manager",
		"dashboard:master",
		"knowledge-base:create",
		"knowledge-base:read",
		"knowledge-base:update",
		"knowledge-base:delete",
		"knowledge-base:manager",
		"knowledge-base:master",
		"market-competitor-insight:create",
		"market-competitor-insight:read",
		"market-competitor-insight:update",
		"market-competitor-insight:delete",
		"market-competitor-insight:manager",
		"market-competitor-insight:master",
		"prompt-management:create",
		"prompt-management:read",
		"prompt-management:update",
		"prompt-management:delete",
		"prompt-management:manager",
		"prompt-management:master",
		"upload-document:create",
		"upload-document:read",
		"upload-document:update",
		"upload-document:delete",
		"upload-document:manager",
		"upload-document:master",
		"sipp-case-details:create",
		"sipp-case-details:read",
		"sipp-case-details:update",
		"sipp-case-details:delete",
		"sipp-case-details:manager",
		"sipp-case-details:master",
		"user-management:create",
		"user-management:read",
		"user-management:update",
		"user-management:delete",
		"user-management:manager",
		"user-management:master",
		"team-management:create",
		"team-management:read",
		"team-management:update",
		"team-management:delete",
		"team-management:manager",
		"team-management:master",
		"role-management:create",
		"role-management:read",
		"role-management:update",
		"role-management:delete",
		"role-management:manager",
		"role-management:master",
		"document-management:master",
		"document-management:manager",
		"document-management:create",
		"document-management:read",
		"document-management:update",
		"document-management:delete",
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}

	stmt, err := tx.Prepare("INSERT INTO permissions (name) VALUES ($1) ON CONFLICT (name) DO NOTHING")
	if err != nil {
		log.Fatalf("Failed to prepare statement: %v", err)
	}

	for _, p := range permissions {
		_, err = stmt.Exec(p)
		if err != nil {
			log.Fatalf("Failed to insert permission '%s': %v", p, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Printf("Seeded %d permissions successfully.", len(permissions))
}
