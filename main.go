package main

import (
	"dokuprime-be/config"
	"dokuprime-be/migrate"
	"dokuprime-be/user"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Warning: .env file not found, using system env variables.")
	}

	r := gin.Default()
	db := config.InitDB()
	defer db.Close()

	redisClient := config.InitRedis()
	defer redisClient.Close()

	migrate.RunMigrations(db)
	user.RegisterRoutes(r, db, redisClient)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running at http://localhost:%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}