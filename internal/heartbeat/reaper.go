package heartbeat

import (
	"context"
	"log"
	"time"

	"taskforge/internal/db"
	"taskforge/internal/queue"
)

type Reaper struct {
	db       *db.PostgresDB
	queue    *queue.RedisQueue
	timeout  time.Duration
	interval time.Duration
	stopCh   chan struct{}
}

func NewReaper(db *db.PostgresDB, queue *queue.RedisQueue, timeout, interval time.Duration) *Reaper {
	return &Reaper{
		db:       db,
		queue:    queue,
		timeout:  timeout,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (r *Reaper) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	log.Println("Reaper started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Reaper stopping due to context cancellation")
			return
		case <-r.stopCh:
			log.Println("Reaper stopped")
			return
		case <-ticker.C:
			r.reapDeadWorkers(ctx)
		}
	}
}

func (r *Reaper) Stop() {
	close(r.stopCh)
}

func (r *Reaper) reapDeadWorkers(ctx context.Context) {
	// Get all heartbeats from Redis
	heartbeats, err := r.queue.GetAllHeartbeats(ctx)
	if err != nil {
		log.Printf("Failed to get heartbeats: %v", err)
		return
	}

	now := time.Now().Unix()

	for workerID, lastHeartbeat := range heartbeats {
		if now-lastHeartbeat > int64(r.timeout.Seconds()) {
			log.Printf("Worker %s is dead (last heartbeat: %d seconds ago)",
				workerID, now-lastHeartbeat)

			// Remove from Redis
			if err := r.queue.RemoveWorkerHeartbeat(ctx, workerID); err != nil {
				log.Printf("Failed to remove worker %s heartbeat: %v", workerID, err)
			}

			// Mark worker offline in PostgreSQL
			if err := r.db.MarkWorkerOffline(ctx, workerID); err != nil {
				log.Printf("Failed to mark worker %s offline: %v", workerID, err)
			}

			// Requeue any jobs assigned to this worker
			if err := r.db.RequeueWorkerJobs(ctx, workerID); err != nil {
				log.Printf("Failed to requeue jobs for worker %s: %v", workerID, err)
			}
		}
	}
}
