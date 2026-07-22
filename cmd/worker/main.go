package main

import (
	"log"
	"os"
	"os/signal"

	"taskforge/internal/config"
	"taskforge/internal/db"
	"taskforge/internal/queue"
	"taskforge/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize PostgreSQL
	postgres, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgres.Close()

	// Initialize Redis
	redisQueue, err := queue.NewRedisQueue(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisQueue.Close()

	// Create worker pool
	workerPool := worker.NewWorkerPool(postgres, redisQueue, cfg.WorkerConcurrency)

	if err := workerPool.Start(); err != nil {
		log.Fatalf("Failed to start worker pool: %v", err)
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down worker...")
	workerPool.Stop()
	log.Println("Worker stopped")
}
