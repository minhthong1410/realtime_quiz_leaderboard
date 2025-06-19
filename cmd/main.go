package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
	"realtime_leaderboard/internal/database"
	"realtime_leaderboard/internal/server"
	"realtime_leaderboard/internal/services"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	db, err := database.NewDB(dbURL)
	if err != nil {
		log.Fatal(err)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("REDIS_ADDR environment variable not set")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	quizService := services.NewQuizService(db, redisClient)
	ser := server.NewServer(quizService)

	log.Println("Starting ser on :8080")
	log.Fatal(http.ListenAndServe(":8080", ser.Router))
}
