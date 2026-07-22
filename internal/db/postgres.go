package db

import (
	"context"
	"fmt"

	"taskforge/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type PostgresDB struct {
	db *sqlx.DB
}

func NewPostgresDB(databaseURL string) (*PostgresDB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	conn := stdlib.OpenDB(*config.ConnConfig)
	db := sqlx.NewDb(conn, "pgx")

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{db: db}, nil
}

func (p *PostgresDB) Close() error {
	return p.db.Close()
}

func (p *PostgresDB) CreateJob(ctx context.Context, job *domain.Job) error {
	query := `
        INSERT INTO jobs (id, type, payload, status, priority, max_retries, run_at, worker_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING created_at, updated_at
    `
	return p.db.QueryRowxContext(ctx, query,
		job.ID, job.Type, job.Payload, job.Status, job.Priority,
		job.MaxRetries, job.RunAt, job.WorkerID,
	).Scan(&job.CreatedAt, &job.UpdatedAt)
}

func (p *PostgresDB) GetJobByID(ctx context.Context, id string) (*domain.Job, error) {
	var job domain.Job
	query := `SELECT * FROM jobs WHERE id = $1`
	err := p.db.GetContext(ctx, &job, query, id)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (p *PostgresDB) UpdateJobStatus(ctx context.Context, id string, status domain.JobStatus, workerID *string, errorMsg *string) error {
	query := `
        UPDATE jobs 
        SET status = $1, worker_id = $2, error_message = $3, updated_at = NOW()
        WHERE id = $4
    `
	_, err := p.db.ExecContext(ctx, query, status, workerID, errorMsg, id)
	return err
}

func (p *PostgresDB) UpdateJobProcessing(ctx context.Context, id string, workerID string) error {
	query := `
        UPDATE jobs 
        SET status = 'processing', worker_id = $1, started_at = NOW(), updated_at = NOW()
        WHERE id = $2 AND status = 'pending'
    `
	_, err := p.db.ExecContext(ctx, query, workerID, id)
	return err
}

func (p *PostgresDB) CompleteJob(ctx context.Context, id string) error {
	query := `
        UPDATE jobs 
        SET status = 'completed', completed_at = NOW(), updated_at = NOW()
        WHERE id = $1
    `
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresDB) FailJob(ctx context.Context, id string, errorMsg string) error {
	query := `
        UPDATE jobs 
        SET status = 'failed', error_message = $1, updated_at = NOW()
        WHERE id = $2
    `
	_, err := p.db.ExecContext(ctx, query, errorMsg, id)
	return err
}

func (p *PostgresDB) MoveToDLQ(ctx context.Context, id string, errorMsg string) error {
	query := `
        UPDATE jobs 
        SET status = 'dlq', error_message = $1, updated_at = NOW()
        WHERE id = $2
    `
	_, err := p.db.ExecContext(ctx, query, errorMsg, id)
	return err
}

func (p *PostgresDB) RegisterWorker(ctx context.Context, worker *domain.Worker) error {
	query := `
        INSERT INTO workers (id, hostname, status, active_jobs, joined_at)
        VALUES ($1, $2, 'active', $3, NOW())
        ON CONFLICT (id) DO UPDATE SET
            status = 'active',
            last_heartbeat = NOW(),
            updated_at = NOW()
        RETURNING joined_at, last_heartbeat
    `
	return p.db.QueryRowxContext(ctx, query, worker.ID, worker.Hostname, worker.ActiveJobs).
		Scan(&worker.JoinedAt, &worker.LastHeartbeat)
}

func (p *PostgresDB) UpdateWorkerHeartbeat(ctx context.Context, id string) error {
	query := `
        UPDATE workers 
        SET last_heartbeat = NOW(), status = 'active'
        WHERE id = $1
    `
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresDB) GetActiveWorkers(ctx context.Context) ([]domain.Worker, error) {
	var workers []domain.Worker
	query := `SELECT * FROM workers WHERE status = 'active' ORDER BY joined_at DESC`
	err := p.db.SelectContext(ctx, &workers, query)
	return workers, err
}

func (p *PostgresDB) MarkWorkerOffline(ctx context.Context, id string) error {
	query := `
        UPDATE workers 
        SET status = 'offline', updated_at = NOW()
        WHERE id = $1
    `
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgresDB) GetJobsByStatus(ctx context.Context, status domain.JobStatus, limit int) ([]domain.Job, error) {
	var jobs []domain.Job
	query := `SELECT * FROM jobs WHERE status = $1 ORDER BY created_at DESC LIMIT $2`
	err := p.db.SelectContext(ctx, &jobs, query, status, limit)
	return jobs, err
}

func (p *PostgresDB) GetQueueStats(ctx context.Context) (*domain.QueueStats, error) {
	stats := &domain.QueueStats{}

	// Get pending jobs count by priority
	var high, def, low int64
	err := p.db.GetContext(ctx, &high,
		`SELECT COUNT(*) FROM jobs WHERE status = 'pending' AND priority = 'high'`)
	if err != nil {
		return nil, err
	}
	err = p.db.GetContext(ctx, &def,
		`SELECT COUNT(*) FROM jobs WHERE status = 'pending' AND priority = 'default'`)
	if err != nil {
		return nil, err
	}
	err = p.db.GetContext(ctx, &low,
		`SELECT COUNT(*) FROM jobs WHERE status = 'pending' AND priority = 'low'`)
	if err != nil {
		return nil, err
	}

	var dlq int64
	err = p.db.GetContext(ctx, &dlq,
		`SELECT COUNT(*) FROM jobs WHERE status = 'dlq'`)
	if err != nil {
		return nil, err
	}

	stats.High = high
	stats.Default = def
	stats.Low = low
	stats.DLQ = dlq

	return stats, nil
}

func (p *PostgresDB) RequeueWorkerJobs(ctx context.Context, workerID string) error {
	query := `
        UPDATE jobs 
        SET status = 'pending', worker_id = NULL, started_at = NULL
        WHERE worker_id = $1 AND status = 'processing'
    `
	_, err := p.db.ExecContext(ctx, query, workerID)
	return err
}
