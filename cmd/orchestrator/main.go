package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"taskforge/internal/api"
	"taskforge/internal/config"
	"taskforge/internal/db"
	"taskforge/internal/heartbeat"
	"taskforge/internal/queue"
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

	// Initialize job handler
	jobHandler := api.NewJobHandler(postgres, redisQueue)

	// Start scheduler
	scheduler := queue.NewScheduler(redisQueue, postgres,
		time.Duration(cfg.SchedulerInterval)*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go scheduler.Start(ctx)

	// Start reaper
	reaper := heartbeat.NewReaper(postgres, redisQueue,
		time.Duration(cfg.HeartbeatTimeout)*time.Second,
		time.Duration(cfg.ReaperInterval)*time.Second)
	go reaper.Start(ctx)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/jobs", jobHandler.CreateJob)
	mux.HandleFunc("GET /v1/jobs/{id}", jobHandler.GetJob)
	mux.HandleFunc("GET /v1/stats", jobHandler.GetQueueStats)
	mux.HandleFunc("GET /v1/jobs", jobHandler.ListJobs)
	mux.HandleFunc("POST /v1/jobs/{id}/requeue", jobHandler.RequeueDLQJob)

	// Dashboard routes will be added separately

	// Start HTTP server
	server := &http.Server{
		Addr:    cfg.OrchestratorPort,
		Handler: mux,
	}

	go func() {
		log.Printf("Orchestrator API server starting on %s", cfg.OrchestratorPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down orchestrator...")

	// Cancel scheduler and reaper
	cancel()

	// Shutdown HTTP server with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Orchestrator stopped")
}
