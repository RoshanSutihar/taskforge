# TaskForge - Distributed Task Queue System

TaskForge is a production-ready, high-performance distributed task queue system built in Go. It provides reliable asynchronous job processing with priority queues, retry logic, and real-time observability.

## Features

- **At-Least-Once Delivery**: Automatic task re-queuing if a worker fails mid-execution
- **Priority Queues**: High, default, and low priority queues with weighted processing
- **Delayed & Cron Jobs**: Schedule tasks for future execution using Redis Sorted Sets
- **Dead-Letter Queue**: Failed jobs exceeding max retries are stored for manual inspection
- **Real-time Dashboard**: Web UI showing queue depths, worker status, and execution metrics
- **Horizontal Scaling**: Multiple workers can be deployed for parallel processing
- **Graceful Shutdown**: Workers complete in-flight jobs before termination

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local development)

### Running with Docker Compose

```bash
# Clone the repository
git clone https://github.com/yourusername/taskforge.git
cd taskforge

# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f orchestrator
```
