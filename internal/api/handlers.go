package api

import (
	"encoding/json"
	"net/http"
	"time"

	"taskforge/internal/db"
	"taskforge/internal/domain"
	"taskforge/internal/queue"

	"github.com/google/uuid"
)

type JobHandler struct {
	db    *db.PostgresDB
	queue *queue.RedisQueue
}

func NewJobHandler(db *db.PostgresDB, queue *queue.RedisQueue) *JobHandler {
	return &JobHandler{
		db:    db,
		queue: queue,
	}
}

type CreateJobResponse struct {
	ID string `json:"id"`
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Type == "" {
		http.Error(w, "Job type is required", http.StatusBadRequest)
		return
	}

	if req.Priority == "" {
		req.Priority = domain.PriorityDefault
	}

	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}

	// Create job
	job := &domain.Job{
		ID:           uuid.New(),
		Type:         req.Type,
		Payload:      req.Payload,
		Status:       domain.StatusPending,
		Priority:     req.Priority,
		MaxRetries:   req.MaxRetries,
		CurrentRetry: 0,
	}

	if req.RunAt != nil && req.RunAt.After(time.Now()) {
		// Delayed job
		if err := h.db.CreateJob(r.Context(), job); err != nil {
			http.Error(w, "Failed to create job", http.StatusInternalServerError)
			return
		}
		if err := h.queue.EnqueueDelayed(r.Context(), job.ID.String(), *req.RunAt); err != nil {
			http.Error(w, "Failed to enqueue delayed job", http.StatusInternalServerError)
			return
		}
	} else {
		// Immediate job
		job.RunAt = time.Now()
		if err := h.db.CreateJob(r.Context(), job); err != nil {
			http.Error(w, "Failed to create job", http.StatusInternalServerError)
			return
		}
		if err := h.queue.Enqueue(r.Context(), job.ID.String(), job.Priority); err != nil {
			http.Error(w, "Failed to enqueue job", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateJobResponse{ID: job.ID.String()})
}

func (h *JobHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting a specific job
	// Parse ID from URL path
	// Return job details
}

func (h *JobHandler) GetQueueStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetQueueStats(r.Context())
	if err != nil {
		http.Error(w, "Failed to get queue stats", http.StatusInternalServerError)
		return
	}

	// Add Redis queue lengths
	highLen, _ := h.queue.GetQueueLength(r.Context(), domain.PriorityHigh)
	defLen, _ := h.queue.GetQueueLength(r.Context(), domain.PriorityDefault)
	lowLen, _ := h.queue.GetQueueLength(r.Context(), domain.PriorityLow)
	delayedLen, _ := h.queue.GetDelayedCount(r.Context())

	stats.High += highLen
	stats.Default += defLen
	stats.Low += lowLen
	stats.Delayed = delayedLen

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *JobHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	// Implementation for listing jobs with filters
	// Parse query parameters
	// Return paginated list of jobs
}

func (h *JobHandler) RequeueDLQJob(w http.ResponseWriter, r *http.Request) {
	// Implementation for requeueing a DLQ job
	// Parse ID from URL path
	// Move job from DLQ to pending
}
