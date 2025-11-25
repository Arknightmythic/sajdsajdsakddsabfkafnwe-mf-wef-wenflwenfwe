package seeder

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func teamSeeder(db *sqlx.DB) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM teams WHERE name = 'default'")
	if err != nil {
		log.Fatalf("Failed to check teams table: %v", err)
	}

	if count > 0 {
		log.Println("Default team already exists.")
		return
	}

	pages := pq.Array([]string{"dashboard"})

	_, err = db.Exec(
		"INSERT INTO teams (name, pages) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		"default",
		pages,
	)
	if err != nil {
		log.Fatalf("Failed to insert default team: %v", err)
	}

	log.Println("Seeded default team successfully.")
}

func roleSeeder(db *sqlx.DB) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM roles WHERE name = 'default'")
	if err != nil {
		log.Fatalf("Failed to check roles table: %v", err)
	}

	if count > 0 {
		log.Println("Default role already exists.")
		return
	}

	var permissionID int
	err = db.Get(&permissionID, "SELECT id FROM permissions WHERE name = 'dashboard:read'")
	if err != nil {
		log.Fatalf("Failed to get dashboard:read permission ID: %v", err)
	}

	var teamID int
	err = db.Get(&teamID, "SELECT id FROM teams WHERE name = 'default'")
	if err != nil {
		log.Fatalf("Failed to get default team ID: %v", err)
	}

	permissions := pq.Array([]string{fmt.Sprintf("%d", permissionID)})

	_, err = db.Exec(
		"INSERT INTO roles (name, permissions, team_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
		"default",
		permissions,
		teamID,
	)
	if err != nil {
		log.Fatalf("Failed to insert default role: %v", err)
	}

	log.Printf("Seeded default role successfully with permission ID: %d", permissionID)
}
