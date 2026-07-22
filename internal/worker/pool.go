package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"taskforge/internal/db"
	"taskforge/internal/domain"
	"taskforge/internal/heartbeat"
	"taskforge/internal/queue"
)

type WorkerPool struct {
	db               *db.PostgresDB
	queue            *queue.RedisQueue
	concurrency      int
	workerID         string
	hostname         string
	heartbeatEmitter *heartbeat.HeartbeatEmitter
	executor         *JobExecutor
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	stopCh           chan struct{}
	activeJobs       int32
}

func NewWorkerPool(db *db.PostgresDB, queue *queue.RedisQueue, concurrency int) *WorkerPool {
	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		db:          db,
		queue:       queue,
		concurrency: concurrency,
		workerID:    workerID,
		hostname:    hostname,
		ctx:         ctx,
		cancel:      cancel,
		stopCh:      make(chan struct{}),
	}

	pool.executor = NewJobExecutor(db, queue)
	pool.heartbeatEmitter = heartbeat.NewHeartbeatEmitter(db, queue, workerID, hostname, 5*time.Second)

	return pool
}

func (p *WorkerPool) Start() error {
	// Register worker
	worker := &domain.Worker{
		ID:         p.workerID,
		Hostname:   p.hostname,
		Status:     "active",
		ActiveJobs: 0,
	}

	if err := p.db.RegisterWorker(p.ctx, worker); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	log.Printf("Worker %s started with concurrency %d", p.workerID, p.concurrency)

	// Start heartbeat emitter
	go p.heartbeatEmitter.Start(p.ctx)

	// Start worker goroutines
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.workerLoop(i)
	}

	return nil
}

func (p *WorkerPool) workerLoop(id int) {
	defer p.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		default:
			// Try to get a job with priority
			jobID, priority, err := p.queue.DequeueWithPriority(p.ctx, 1*time.Second)
			if err != nil {
				log.Printf("Worker %d dequeue error: %v", id, err)
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if jobID == "" {
				// No jobs available, continue
				continue
			}

			log.Printf("Worker %d received job %s from %s queue", id, jobID, priority)

			// Execute job
			if err := p.executor.Execute(p.ctx, jobID, p.workerID); err != nil {
				log.Printf("Worker %d failed to execute job %s: %v", id, jobID, err)
			}
		}
	}
}

func (p *WorkerPool) Stop() {
	log.Printf("Stopping worker pool %s", p.workerID)

	// Signal shutdown
	p.cancel()

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("Worker shutdown timeout - forcing exit")
	}

	// Unregister worker
	if err := p.db.MarkWorkerOffline(context.Background(), p.workerID); err != nil {
		log.Printf("Failed to mark worker offline: %v", err)
	}
}
