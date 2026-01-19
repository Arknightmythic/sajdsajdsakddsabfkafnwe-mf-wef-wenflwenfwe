package config

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDB() *sqlx.DB {
	dbHost := AppConfig.DBHost
	dbPort := AppConfig.DBPort
	dbUser := AppConfig.DBUser
	dbPassword := AppConfig.DBPassword
	dbName := AppConfig.DBName
	dbDriver := AppConfig.DBDriver
	dbSchema := AppConfig.DBSchema

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

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	var searchPath string
	err = db.Get(&searchPath, "SHOW search_path")
	if err != nil {
		log.Fatalf("Failed to get search_path: %v", err)
	}
	log.Printf("Connected to database: %s with search_path: %s", dbName, searchPath)

	return db
}
