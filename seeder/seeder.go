package seeder

import (
	"github.com/jmoiron/sqlx"
)

func RunSeeder(db *sqlx.DB) {
	permissionSeeder(db)
	superadminSeeder(db)
}
