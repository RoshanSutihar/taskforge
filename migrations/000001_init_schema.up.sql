CREATE TYPE job_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'dlq');
CREATE TYPE job_priority AS ENUM ('high', 'default', 'low');

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status job_status NOT NULL DEFAULT 'pending',
    priority job_priority NOT NULL DEFAULT 'default',
    max_retries INT NOT NULL DEFAULT 3,
    current_retry INT NOT NULL DEFAULT 0,
    error_message TEXT,
    run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    worker_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workers (
    id VARCHAR(255) PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    active_jobs INT NOT NULL DEFAULT 0,
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_jobs_status_priority ON jobs(status, priority);
CREATE INDEX idx_jobs_run_at ON jobs(run_at) WHERE status = 'pending';
CREATE INDEX idx_jobs_worker_id ON jobs(worker_id) WHERE status = 'processing';