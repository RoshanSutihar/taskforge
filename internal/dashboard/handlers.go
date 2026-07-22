package dashboard

import (
	"encoding/json"
	"net/http"
	"taskforge/internal/db"
	"taskforge/internal/domain"
	"taskforge/internal/queue"
)

type DashboardHandler struct {
	db    *db.PostgresDB
	queue *queue.RedisQueue
}

func NewDashboardHandler(db *db.PostgresDB, queue *queue.RedisQueue) *DashboardHandler {
	return &DashboardHandler{
		db:    db,
		queue: queue,
	}
}

func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	// Get queue stats from PostgreSQL
	stats, err := h.db.GetQueueStats(r.Context())
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	// Get Redis queue lengths
	highLen, _ := h.queue.GetQueueLength(r.Context(), domain.PriorityHigh)
	defLen, _ := h.queue.GetQueueLength(r.Context(), domain.PriorityDefault)
	lowLen, _ := h.queue.GetQueueLength(r.Context(), domain.PriorityLow)
	delayedLen, _ := h.queue.GetDelayedCount(r.Context())

	stats.High += highLen
	stats.Default += defLen
	stats.Low += lowLen
	stats.Delayed = delayedLen

	// Get active workers
	workers, _ := h.db.GetActiveWorkers(r.Context())

	// Get recent jobs
	recentJobs, _ := h.db.GetJobsByStatus(r.Context(), domain.StatusPending, 10)
	completedJobs, _ := h.db.GetJobsByStatus(r.Context(), domain.StatusCompleted, 10)
	failedJobs, _ := h.db.GetJobsByStatus(r.Context(), domain.StatusFailed, 10)
	dlqJobs, _ := h.db.GetJobsByStatus(r.Context(), domain.StatusDLQ, 10)

	response := map[string]interface{}{
		"queue_stats": stats,
		"workers":     workers,
		"recent_jobs": map[string]interface{}{
			"pending":   recentJobs,
			"completed": completedJobs,
			"failed":    failedJobs,
			"dlq":       dlqJobs,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
