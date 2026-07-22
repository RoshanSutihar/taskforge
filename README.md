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

```bash
TEST RESULTS WITH THREE(3) WORKERS
============================================================
TASKFORGE MASS TEST - 10,000+ JOBS
============================================================

[INFO] Checking connection...
[INFO] Connected to http://192.168.0.103:8188

----------------------------------------
QUEUE STATISTICS
----------------------------------------
High Priority:    0
Default Priority: 1952
Low Priority:     6000
Delayed:          0
Dead Letter Queue: 0
----------------------------------------
TOTAL JOBS:       7952
----------------------------------------


============================================================
MASS TEST: 10,000+ Jobs
============================================================

[INFO] Creating 3000 high priority jobs...

[=---------------------------------------] 3.3% (100/3000) Created: 100, Failed: 0
[==--------------------------------------] 6.7% (200/3000) Created: 200, Failed: 0
[====------------------------------------] 10.0% (300/3000) Created: 300, Failed: 0
[=====-----------------------------------] 13.3% (400/3000) Created: 400, Failed: 0
[======----------------------------------] 16.7% (500/3000) Created: 500, Failed: 0
[========--------------------------------] 20.0% (600/3000) Created: 600, Failed: 0
[=========-------------------------------] 23.3% (700/3000) Created: 700, Failed: 0
[==========------------------------------] 26.7% (800/3000) Created: 800, Failed: 0
[============----------------------------] 30.0% (900/3000) Created: 900, Failed: 0
[=============---------------------------] 33.3% (1000/3000) Created: 1000, Failed: 0
[==============--------------------------] 36.7% (1100/3000) Created: 1100, Failed: 0
[================------------------------] 40.0% (1200/3000) Created: 1200, Failed: 0
[=================-----------------------] 43.3% (1300/3000) Created: 1300, Failed: 0
[==================----------------------] 46.7% (1400/3000) Created: 1400, Failed: 0
[====================--------------------] 50.0% (1500/3000) Created: 1500, Failed: 0
[=====================-------------------] 53.3% (1600/3000) Created: 1600, Failed: 0
[======================------------------] 56.7% (1700/3000) Created: 1700, Failed: 0
[========================----------------] 60.0% (1800/3000) Created: 1800, Failed: 0
[=========================---------------] 63.3% (1900/3000) Created: 1900, Failed: 0
[==========================--------------] 66.7% (2000/3000) Created: 2000, Failed: 0
[============================------------] 70.0% (2100/3000) Created: 2100, Failed: 0
[=============================-----------] 73.3% (2200/3000) Created: 2200, Failed: 0
[==============================----------] 76.7% (2300/3000) Created: 2300, Failed: 0
[================================--------] 80.0% (2400/3000) Created: 2400, Failed: 0
[=================================-------] 83.3% (2500/3000) Created: 2500, Failed: 0
[==================================------] 86.7% (2600/3000) Created: 2600, Failed: 0
[====================================----] 90.0% (2700/3000) Created: 2700, Failed: 0
[=====================================---] 93.3% (2800/3000) Created: 2800, Failed: 0
[======================================--] 96.7% (2900/3000) Created: 2900, Failed: 0
[========================================] 100.0% (3000/3000) Created: 3000, Failed: 0
[INFO] Created 3000 jobs, Failed 0 jobs
[INFO] Creating 4000 default priority jobs...

[=---------------------------------------] 2.5% (100/4000) Created: 100, Failed: 0
[==--------------------------------------] 5.0% (200/4000) Created: 200, Failed: 0
[===-------------------------------------] 7.5% (300/4000) Created: 300, Failed: 0
[====------------------------------------] 10.0% (400/4000) Created: 400, Failed: 0
[=====-----------------------------------] 12.5% (500/4000) Created: 500, Failed: 0
[======----------------------------------] 15.0% (600/4000) Created: 600, Failed: 0
[=======---------------------------------] 17.5% (700/4000) Created: 700, Failed: 0
[========--------------------------------] 20.0% (800/4000) Created: 800, Failed: 0
[=========-------------------------------] 22.5% (900/4000) Created: 900, Failed: 0
[==========------------------------------] 25.0% (1000/4000) Created: 1000, Failed: 0
[===========-----------------------------] 27.5% (1100/4000) Created: 1100, Failed: 0
[============----------------------------] 30.0% (1200/4000) Created: 1200, Failed: 0
[=============---------------------------] 32.5% (1300/4000) Created: 1300, Failed: 0
[==============--------------------------] 35.0% (1400/4000) Created: 1400, Failed: 0
[===============-------------------------] 37.5% (1500/4000) Created: 1500, Failed: 0
[================------------------------] 40.0% (1600/4000) Created: 1600, Failed: 0
[=================-----------------------] 42.5% (1700/4000) Created: 1700, Failed: 0
[==================----------------------] 45.0% (1800/4000) Created: 1800, Failed: 0
[===================---------------------] 47.5% (1900/4000) Created: 1900, Failed: 0
[====================--------------------] 50.0% (2000/4000) Created: 2000, Failed: 0
[=====================-------------------] 52.5% (2100/4000) Created: 2100, Failed: 0
[======================------------------] 55.0% (2200/4000) Created: 2200, Failed: 0
[=======================-----------------] 57.5% (2300/4000) Created: 2300, Failed: 0
[========================----------------] 60.0% (2400/4000) Created: 2400, Failed: 0
[=========================---------------] 62.5% (2500/4000) Created: 2500, Failed: 0
[==========================--------------] 65.0% (2600/4000) Created: 2600, Failed: 0
[===========================-------------] 67.5% (2700/4000) Created: 2700, Failed: 0
[============================------------] 70.0% (2800/4000) Created: 2800, Failed: 0
[=============================-----------] 72.5% (2900/4000) Created: 2900, Failed: 0
[==============================----------] 75.0% (3000/4000) Created: 3000, Failed: 0
[===============================---------] 77.5% (3100/4000) Created: 3100, Failed: 0
[================================--------] 80.0% (3200/4000) Created: 3200, Failed: 0
[=================================-------] 82.5% (3300/4000) Created: 3300, Failed: 0
[==================================------] 85.0% (3400/4000) Created: 3400, Failed: 0
[===================================-----] 87.5% (3500/4000) Created: 3500, Failed: 0
[====================================----] 90.0% (3600/4000) Created: 3600, Failed: 0
[=====================================---] 92.5% (3700/4000) Created: 3700, Failed: 0
[======================================--] 95.0% (3800/4000) Created: 3800, Failed: 0
[=======================================-] 97.5% (3900/4000) Created: 3900, Failed: 0
[========================================] 100.0% (4000/4000) Created: 4000, Failed: 0
[INFO] Created 4000 jobs, Failed 0 jobs
[INFO] Creating 3000 low priority jobs...

[=---------------------------------------] 3.3% (100/3000) Created: 100, Failed: 0
[==--------------------------------------] 6.7% (200/3000) Created: 200, Failed: 0
[====------------------------------------] 10.0% (300/3000) Created: 300, Failed: 0
[=====-----------------------------------] 13.3% (400/3000) Created: 400, Failed: 0
[======----------------------------------] 16.7% (500/3000) Created: 500, Failed: 0
[========--------------------------------] 20.0% (600/3000) Created: 600, Failed: 0
[=========-------------------------------] 23.3% (700/3000) Created: 700, Failed: 0
[==========------------------------------] 26.7% (800/3000) Created: 800, Failed: 0
[============----------------------------] 30.0% (900/3000) Created: 900, Failed: 0
[=============---------------------------] 33.3% (1000/3000) Created: 1000, Failed: 0
[==============--------------------------] 36.7% (1100/3000) Created: 1100, Failed: 0
[================------------------------] 40.0% (1200/3000) Created: 1200, Failed: 0
[=================-----------------------] 43.3% (1300/3000) Created: 1300, Failed: 0
[==================----------------------] 46.7% (1400/3000) Created: 1400, Failed: 0
[====================--------------------] 50.0% (1500/3000) Created: 1500, Failed: 0
[=====================-------------------] 53.3% (1600/3000) Created: 1600, Failed: 0
[======================------------------] 56.7% (1700/3000) Created: 1700, Failed: 0
[========================----------------] 60.0% (1800/3000) Created: 1800, Failed: 0
[=========================---------------] 63.3% (1900/3000) Created: 1900, Failed: 0
[==========================--------------] 66.7% (2000/3000) Created: 2000, Failed: 0
[============================------------] 70.0% (2100/3000) Created: 2100, Failed: 0
[=============================-----------] 73.3% (2200/3000) Created: 2200, Failed: 0
[==============================----------] 76.7% (2300/3000) Created: 2300, Failed: 0
[================================--------] 80.0% (2400/3000) Created: 2400, Failed: 0
[=================================-------] 83.3% (2500/3000) Created: 2500, Failed: 0
[==================================------] 86.7% (2600/3000) Created: 2600, Failed: 0
[====================================----] 90.0% (2700/3000) Created: 2700, Failed: 0
[=====================================---] 93.3% (2800/3000) Created: 2800, Failed: 0
[======================================--] 96.7% (2900/3000) Created: 2900, Failed: 0
[========================================] 100.0% (3000/3000) Created: 3000, Failed: 0
[INFO] Created 3000 jobs, Failed 0 jobs

----------------------------------------
QUEUE STATISTICS
----------------------------------------
High Priority:    1741
Default Priority: 9952
Low Priority:     12000
Delayed:          0
Dead Letter Queue: 0
----------------------------------------
TOTAL JOBS:       23693
----------------------------------------

[INFO] Waiting for workers to start processing...

============================================================
MONITORING STATS (every 2s for 20s)
============================================================

[0s] High:1140 Default:9952 Low:12000 Delayed:   0 DLQ:   0 Total:23092
[2s] High: 900 Default:9952 Low:12000 Delayed:   0 DLQ:   0 Total:22852
[4s] High: 656 Default:9952 Low:12000 Delayed:   0 DLQ:   0 Total:22608
[6s] High: 416 Default:9952 Low:12000 Delayed:   0 DLQ:   0 Total:22368
[8s] High: 176 Default:9952 Low:12000 Delayed:   0 DLQ:   0 Total:22128
[10s] High:   0 Default:9892 Low:12000 Delayed:   0 DLQ:   0 Total:21892
[12s] High:   0 Default:9772 Low:12000 Delayed:   0 DLQ:   0 Total:21772
[14s] High:   0 Default:9652 Low:12000 Delayed:   0 DLQ:   0 Total:21652
[16s] High:   0 Default:9532 Low:12000 Delayed:   0 DLQ:   0 Total:21532
[18s] High:   0 Default:9412 Low:12000 Delayed:   0 DLQ:   0 Total:21412

============================================================
DELAYED JOBS TEST
============================================================

[INFO] Creating 1000 delayed jobs (30 second delay)...
[INFO] Creating 1000 default priority jobs...

[====------------------------------------] 10.0% (100/1000) Created: 100, Failed: 0
[========--------------------------------] 20.0% (200/1000) Created: 200, Failed: 0
[============----------------------------] 30.0% (300/1000) Created: 300, Failed: 0
[================------------------------] 40.0% (400/1000) Created: 400, Failed: 0
[====================--------------------] 50.0% (500/1000) Created: 500, Failed: 0
[========================----------------] 60.0% (600/1000) Created: 600, Failed: 0
[============================------------] 70.0% (700/1000) Created: 700, Failed: 0
[================================--------] 80.0% (800/1000) Created: 800, Failed: 0
[====================================----] 90.0% (900/1000) Created: 900, Failed: 0
[========================================] 100.0% (1000/1000) Created: 1000, Failed: 0
[INFO] Created 1000 jobs, Failed 0 jobs

----------------------------------------
QUEUE STATISTICS
----------------------------------------
High Priority:    0
Default Priority: 10996
Low Priority:     12000
Delayed:          0
Dead Letter Queue: 0
----------------------------------------
TOTAL JOBS:       22996
----------------------------------------


============================================================
TEST SUMMARY
============================================================

Total jobs created: 11000
Successful creations: 11000
Failed creations: 0
Success rate: 100.00%

Current queue totals:
  High: 0
  Default: 10996
  Low: 12000
  Delayed: 0
  DLQ: 0
  Total: 22996

============================================================
MASS TEST COMPLETE
============================================================

```

## 📜 License

Distributed under the MIT License. See LICENSE for more information.
