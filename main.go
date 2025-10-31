package main

import (
	"dokuprime-be/config"
	"dokuprime-be/document"
	"dokuprime-be/migrate"
	"dokuprime-be/permission"
	"dokuprime-be/role"
	"dokuprime-be/seeder"
	"dokuprime-be/team"
	"dokuprime-be/user"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables.")
	}

	args := os.Args
	db := config.InitDB()
	defer db.Close()

	if len(args) > 1 && args[1] == "--migrate" {
		migrate.RunMigrations(db)
		return
	}

	if len(args) > 1 && args[1] == "--seed" {
		seeder.RunSeeder(db)
		return
	}

	redisClient := config.InitRedis()
	defer redisClient.Close()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("ALLOWED_ORIGINS")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	user.RegisterRoutes(r, db, redisClient)
	role.RegisterRoutes(r, db)
	team.RegisterRoutes(r, db)
	permission.RegisterRoutes(r, db)
	document.RegisterRoutes(r, db)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running at http://localhost:%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
