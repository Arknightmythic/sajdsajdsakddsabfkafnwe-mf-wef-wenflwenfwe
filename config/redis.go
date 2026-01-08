package config

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func InitRedis() *redis.Client {
	host := AppConfig.RedisHost
	port := AppConfig.RedisPort
	password := AppConfig.RedisPassword
	dbStr := AppConfig.RedisDB

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "6379"
	}
	if dbStr == "" {
		dbStr = "0"
	}

	db, err := strconv.Atoi(dbStr)
	if err != nil {
		db = 0
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis connected successfully")
	return client
}
