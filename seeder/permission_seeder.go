package seeder

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func permissionSeeder(db *sqlx.DB) {
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
		"dashboard:access",
		"dashboard:manager",
		"dashboard:master",
		"knowledge-base:create",
		"knowledge-base:read",
		"knowledge-base:update",
		"knowledge-base:delete",
		"knowledge-base:manager",
		"knowledge-base:master",
		"document-management:create",
		"document-management:read",
		"document-management:update",
		"document-management:delete",
		"document-management:manager",
		"document-management:master",
		"public-service:access",
		"public-service:manager",
		"public-service:master",
		"validation-history:create",
		"validation-history:read",
		"validation-history:update",
		"validation-history:delete",
		"validation-history:manager",
		"validation-history:master",
		"guide:access",
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
		"helpdesk:access",
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
