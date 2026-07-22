package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusDLQ        JobStatus = "dlq"
)

type JobPriority string

const (
	PriorityHigh    JobPriority = "high"
	PriorityDefault JobPriority = "default"
	PriorityLow     JobPriority = "low"
)

type Job struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	Type         string          `db:"type" json:"type"`
	Payload      json.RawMessage `db:"payload" json:"payload"`
	Status       JobStatus       `db:"status" json:"status"`
	Priority     JobPriority     `db:"priority" json:"priority"`
	MaxRetries   int             `db:"max_retries" json:"max_retries"`
	CurrentRetry int             `db:"current_retry" json:"current_retry"`
	ErrorMessage *string         `db:"error_message" json:"error_message,omitempty"`
	RunAt        time.Time       `db:"run_at" json:"run_at"`
	StartedAt    *time.Time      `db:"started_at" json:"started_at,omitempty"`
	CompletedAt  *time.Time      `db:"completed_at" json:"completed_at,omitempty"`
	WorkerID     *string         `db:"worker_id" json:"worker_id,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

type CreateJobRequest struct {
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Priority   JobPriority     `json:"priority"`
	MaxRetries int             `json:"max_retries"`
	RunAt      *time.Time      `json:"run_at,omitempty"`
}

type Worker struct {
	ID            string    `db:"id" json:"id"`
	Hostname      string    `db:"hostname" json:"hostname"`
	Status        string    `db:"status" json:"status"`
	ActiveJobs    int       `db:"active_jobs" json:"active_jobs"`
	LastHeartbeat time.Time `db:"last_heartbeat" json:"last_heartbeat"`
	JoinedAt      time.Time `db:"joined_at" json:"joined_at"`
}

type QueueStats struct {
	High    int64 `json:"high"`
	Default int64 `json:"default"`
	Low     int64 `json:"low"`
	Delayed int64 `json:"delayed"`
	DLQ     int64 `json:"dlq"`
}
