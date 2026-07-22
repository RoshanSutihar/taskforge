package heartbeat

import (
	"context"
	"log"
	"time"

	"taskforge/internal/db"
	"taskforge/internal/queue"
)

type HeartbeatEmitter struct {
	db       *db.PostgresDB
	queue    *queue.RedisQueue
	workerID string
	hostname string
	interval time.Duration
}

func NewHeartbeatEmitter(db *db.PostgresDB, queue *queue.RedisQueue, workerID, hostname string, interval time.Duration) *HeartbeatEmitter {
	return &HeartbeatEmitter{
		db:       db,
		queue:    queue,
		workerID: workerID,
		hostname: hostname,
		interval: interval,
	}
}

func (h *HeartbeatEmitter) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	log.Printf("Heartbeat emitter started for worker %s", h.workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Heartbeat emitter stopped for worker %s", h.workerID)
			return
		case <-ticker.C:
			h.sendHeartbeat(ctx)
		}
	}
}

func (h *HeartbeatEmitter) sendHeartbeat(ctx context.Context) {
	// Update Redis heartbeat
	if err := h.queue.UpdateWorkerHeartbeat(ctx, h.workerID); err != nil {
		log.Printf("Failed to update Redis heartbeat: %v", err)
	}

	// Update PostgreSQL heartbeat
	if err := h.db.UpdateWorkerHeartbeat(ctx, h.workerID); err != nil {
		log.Printf("Failed to update PostgreSQL heartbeat: %v", err)
	}
}
