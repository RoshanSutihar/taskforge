# TaskForge 🚀

> A high-performance, resilient, distributed background task execution platform built with **Go**, **PostgreSQL**, and **Redis**.

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker Compose](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![Coolify Ready](https://img.shields.io/badge/Deploy-Coolify-6366F1?style=flat)](https://coolify.io/)

TaskForge is an asynchronous background processing system designed for high availability and low latency. It handles task scheduling, priority queue management, exponential backoff retries, dead-letter queue (DLQ) processing, and worker node heartbeat tracking.

## ![alt text](image.png)

## 🏗 System Architecture

```
                   ┌─────────────────────────┐
                   │   Client / Ingestion    │
                   │        REST API         │
                   └────────────┬────────────┘
                                │
                                v
                   ┌─────────────────────────┐
                   │  TaskForge Orchestrator │
                   │    (Scheduler Loop)     │
                   └────────────┬────────────┘
                                │
       ┌────────────────────────┼────────────────────────┐
       │                        │                        │
       v                        v                        v
┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│ PostgreSQL 16     │    │ Redis 7           │    │ HTMX Dashboard    │
│ (Persistent State │    │ (Priority Queues &│    │ (Live Observability│
│  & Worker Logs)   │    │  Heartbeat Hash)  │    │  & DLQ Control)   │
└───────────────────┘    └─────────┬─────────┘    └───────────────────┘
                                    │
                    ┌────────────────────────┼────────────────────────┐
                    │                        │                        │
                    v                        v                        v
             ┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
             │     Worker 1      │    │     Worker 2      │    │     Worker 3      │
             │ (Go Worker Pool)  │    │ (Go Worker Pool)  │    │ (Go Worker Pool)  │
             └───────────────────┘    └───────────────────┘    └───────────────────┘
```

---

## ✨ Key Features

- **Priority Queuing:** Multi-channel priority routing (`high`, `default`, `low`) ensuring critical tasks are dispatched first.
- **At-Least-Once Execution & Heartbeats:** Worker nodes emit periodic heartbeats. If a worker node crashes mid-execution, an automated reaper reclaims and re-queues orphaned tasks.
- **Delayed & Scheduled Jobs:** Precision task execution scheduling backed by Redis Sorted Sets (ZSET) with O(log N) extraction performance.
- **Dead-Letter Queue (DLQ):** Failed jobs exceeding `max_retries` are isolated into a DLQ with exponential backoff calculation (2^retry × 5s) and manual re-trigger capabilities.
- **Live HTMX Dashboard:** Real-time observability dashboard built with **Go Templ** and **HTMX** for tracking queue depths, worker status, and execution metrics.
- **Self-Hosted & Containerized:** Ships with a multi-stage `docker-compose.yaml` optimized for **Coolify** and custom homelabs.

---

## 🛠 Tech Stack

| Domain                   | Technology                               |
| :----------------------- | :--------------------------------------- |
| **Language**             | Go 1.22+ (`net/http`, `sync`, `context`) |
| **Primary Storage**      | PostgreSQL 16 (`pgx/v5`)                 |
| **Queue / Cache**        | Redis 7 (`go-redis/v9`)                  |
| **Frontend / Dashboard** | Go `templ` + `htmx` + Tailwind CSS       |
| **Orchestration**        | Docker, Docker Compose, Coolify          |

---

## 📁 Repository Structure

```text
taskforge/
├── cmd/
│   ├── orchestrator/      # Ingestion API, Scheduler, and Web Dashboard entrypoint
│   └── worker/            # Asynchronous Worker Node entrypoint
├── internal/
│   ├── config/            # Environment variable parsing and fallback handlers
│   ├── db/                # PostgreSQL migration and connection setup
│   ├── queue/             # Redis queue logic (Pub/Sub, ZSET scheduler, DLQ)
│   ├── worker/            # Dynamic worker pool implementation
│   ├── heartbeat/         # Worker heartbeat emitter & reaper process
│   ├── dashboard/         # HTMX/Templ dashboard handlers
│   └── domain/            # Core structs, models, and interfaces
├── templates/             # Go Templ template files (.templ)
├── migrations/             # SQL migration scripts
├── Dockerfile.orchestrator # Multi-stage Dockerfile for orchestrator binary
├── Dockerfile.worker       # Multi-stage Dockerfile for worker binary
└── docker-compose.yaml     # Production deployment topology
```

## 🚀 Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local binary execution)

### Running Locally with Docker Compose

Clone the repository:

```bash
git clone https://github.com/RoshanSutihar/taskforge.git
cd taskforge
```

Launch infrastructure and microservices:

```bash
docker compose up -d --build
```

Access services:

- Web Dashboard: http://localhost:8188/stats
- PostgreSQL: localhost:5432 (Internal Docker network)
- Redis: localhost:6379 (Internal Docker network)

## ⚡ API Endpoints

Routes are registered in the orchestrator's HTTP mux:

```go
mux.HandleFunc("POST /v1/jobs", jobHandler.CreateJob)
mux.HandleFunc("GET /v1/jobs/{id}", jobHandler.GetJob)
mux.HandleFunc("GET /v1/stats", jobHandler.GetQueueStats)
mux.HandleFunc("GET /v1/jobs", jobHandler.ListJobs)
mux.HandleFunc("POST /v1/jobs/{id}/requeue", jobHandler.RequeueDLQJob)
```

### 1. Enqueue a Job

```http
POST /v1/jobs
Content-Type: application/json

{
  "type": "email:send",
  "priority": "high",
  "payload": {
    "recipient": "user@example.com",
    "template_id": "welcome_email"
  },
  "max_retries": 3,
  "delay_seconds": 0
}
```

Response (202 Accepted):

```json
{
  "id": "d6302597-6dea-468b-b31f-db8db9027322"
}
```

### 2. Fetch Job Status

```http
GET /v1/jobs/c9bf9e57-1685-4c89-bafb-ff5af830be8a
```

## 🌐 Deploying with Coolify

TaskForge is structured specifically for one-click deployment via Coolify:

1. Create a new Docker Compose resource in your Coolify project.
2. Connect your Git repository (`RoshanSutihar/taskforge`).
3. Set the following environment variables in Coolify's Environment Variables tab:

```
DATABASE_URL=postgres://taskforge:taskforge@postgres:5432/taskforge?sslmode=disable
REDIS_URL=redis://redis:6379/0
ORCHESTRATOR_PORT=:8080
WORKER_CONCURRENCY=10
HEARTBEAT_INTERVAL=5
HEARTBEAT_TIMEOUT=15
```

4. Click **Deploy**.

## 🧪 Testing & Benchmarks

To run unit tests and stress-test local worker queue performance:

```bash
# Run internal package unit tests
go test ./... -v

# Run concurrent execution benchmark
go test -bench=. ./internal/worker/
```

## 📜 License

Distributed under the MIT License. See LICENSE for more information.
