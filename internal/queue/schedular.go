package queue

import (
	"context"
	"log"
	"taskforge/internal/db"
	"time"
)

type Scheduler struct {
	queue    *RedisQueue
	db       *db.PostgresDB
	interval time.Duration
	stopCh   chan struct{}
}

func NewScheduler(queue *RedisQueue, db *db.PostgresDB, interval time.Duration) *Scheduler {
	return &Scheduler{
		queue:    queue,
		db:       db,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	log.Println("Scheduler started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopping due to context cancellation")
			return
		case <-s.stopCh:
			log.Println("Scheduler stopped")
			return
		case <-ticker.C:
			s.processDelayedJobs(ctx)
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
}

func (s *Scheduler) processDelayedJobs(ctx context.Context) {
	now := time.Now()

	// Get all jobs ready to be executed
	jobIDs, err := s.queue.GetReadyDelayed(ctx, now)
	if err != nil {
		log.Printf("Failed to get ready delayed jobs: %v", err)
		return
	}

	if len(jobIDs) == 0 {
		return
	}

	log.Printf("Processing %d delayed jobs", len(jobIDs))

	for _, jobID := range jobIDs {
		// Get job details from PostgreSQL
		job, err := s.db.GetJobByID(ctx, jobID)
		if err != nil {
			log.Printf("Failed to get job %s: %v", jobID, err)
			continue
		}

		// Remove from delayed queue
		if err := s.queue.RemoveDelayed(ctx, jobID); err != nil {
			log.Printf("Failed to remove job %s from delayed queue: %v", jobID, err)
			continue
		}

		// Enqueue to appropriate priority queue
		if err := s.queue.Enqueue(ctx, jobID, job.Priority); err != nil {
			log.Printf("Failed to enqueue job %s: %v", jobID, err)
			// Re-add to delayed queue if enqueue fails
			s.queue.EnqueueDelayed(ctx, jobID, now.Add(5*time.Second))
			continue
		}

		log.Printf("Moved job %s from delayed to %s queue", jobID, job.Priority)
	}
}
