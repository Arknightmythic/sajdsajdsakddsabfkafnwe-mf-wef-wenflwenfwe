package config

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDB() *sqlx.DB {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbDriver := os.Getenv("DB_DRIVER")
	dbSchema := os.Getenv("DB_SCHEMA")

	defaultDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort)

	defaultDB, err := sqlx.Connect(dbDriver, defaultDSN)
	if err != nil {
		log.Fatalf("Failed to connect to default database: %v", err)
	}

	defaultDB.Close()

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSchema)

	db, err := sqlx.Connect(dbDriver, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	var searchPath string
	err = db.Get(&searchPath, "SHOW search_path")
	if err != nil {
		log.Fatalf("Failed to get search_path: %v", err)
	}
	log.Printf("Connected to database: %s with search_path: %s", dbName, searchPath)

	return db
}
