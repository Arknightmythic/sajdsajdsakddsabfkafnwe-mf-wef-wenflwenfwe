package main

import (
	"context"
	"dokuprime-be/chat"
	"dokuprime-be/config"
	"dokuprime-be/document"
	"dokuprime-be/grafana"
	"dokuprime-be/migrate"
	"dokuprime-be/permission"
	"dokuprime-be/role"
	"dokuprime-be/seeder"
	"dokuprime-be/team"
	"dokuprime-be/user"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	grafana.RegisterRoutes(r, redisClient)
	chat.RegisterRoutes(r, db)
	asyncProcessor := document.RegisterRoutesWithProcessor(r, db, redisClient)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8000"
	}

	srv := &http.Server{
		Addr:    "0.0.0.0:8000",
		Handler: r,
	}

	go func() {
		log.Printf("Server running at http://0.0.0.0:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	if asyncProcessor != nil {
		asyncProcessor.Shutdown()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited successfully")
}
