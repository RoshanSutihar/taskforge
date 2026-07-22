package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"taskforge/internal/db"
	"taskforge/internal/domain"
	"taskforge/internal/queue"
)

type JobExecutor struct {
	db    *db.PostgresDB
	queue *queue.RedisQueue
}

func NewJobExecutor(db *db.PostgresDB, queue *queue.RedisQueue) *JobExecutor {
	return &JobExecutor{
		db:    db,
		queue: queue,
	}
}

func (e *JobExecutor) Execute(ctx context.Context, jobID string, workerID string) error {
	// Get job from database
	job, err := e.db.GetJobByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Update job status to processing
	if err := e.db.UpdateJobProcessing(ctx, jobID, workerID); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	log.Printf("Processing job %s of type %s", jobID, job.Type)

	// Execute the job based on its type
	err = e.executeJob(ctx, job)

	if err != nil {
		log.Printf("Job %s failed: %v", jobID, err)
		return e.handleJobFailure(ctx, job, err.Error())
	}

	// Job completed successfully
	log.Printf("Job %s completed successfully", jobID)
	return e.db.CompleteJob(ctx, jobID)
}

func (e *JobExecutor) executeJob(ctx context.Context, job *domain.Job) error {
	var delay time.Duration
	switch job.Priority {
	case domain.PriorityHigh:
		delay = 500 * time.Millisecond
	case domain.PriorityDefault:
		delay = 1 * time.Second
	case domain.PriorityLow:
		delay = 2 * time.Second
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func (e *JobExecutor) handleJobFailure(ctx context.Context, job *domain.Job, errorMsg string) error {
	job.CurrentRetry++

	if job.CurrentRetry < job.MaxRetries {
		// Calculate exponential backoff: 2^retry * 5 seconds
		backoffSeconds := int(math.Pow(2, float64(job.CurrentRetry))) * 5
		runAt := time.Now().Add(time.Duration(backoffSeconds) * time.Second)

		// Update job in database
		if err := e.db.UpdateJobStatus(ctx, job.ID.String(), domain.StatusPending, nil, &errorMsg); err != nil {
			return fmt.Errorf("failed to update job status: %w", err)
		}

		// Add to delayed queue
		if err := e.queue.EnqueueDelayed(ctx, job.ID.String(), runAt); err != nil {
			return fmt.Errorf("failed to enqueue delayed job: %w", err)
		}

		log.Printf("Job %s will retry in %d seconds (attempt %d/%d)",
			job.ID, backoffSeconds, job.CurrentRetry, job.MaxRetries)
		return nil
	}

	// Max retries exceeded - move to DLQ
	log.Printf("Job %s moved to DLQ after %d retries", job.ID, job.MaxRetries)
	return e.db.MoveToDLQ(ctx, job.ID.String(), errorMsg)
}
