package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"taskforge/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	// If URL is empty, use default
	if redisURL == "" {
		redisURL = "redis://redis:6379/0"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &RedisQueue{client: client}, nil
}

func (r *RedisQueue) Close() error {
	return r.client.Close()
}

func (r *RedisQueue) GetClient() *redis.Client {
	return r.client
}

func (r *RedisQueue) Enqueue(ctx context.Context, jobID string, priority domain.JobPriority) error {
	queueKey := fmt.Sprintf("queue:%s", priority)
	return r.client.RPush(ctx, queueKey, jobID).Err()
}

func (r *RedisQueue) EnqueueDelayed(ctx context.Context, jobID string, runAt time.Time) error {
	score := float64(runAt.Unix())
	return r.client.ZAdd(ctx, "queue:delayed", redis.Z{
		Score:  score,
		Member: jobID,
	}).Err()
}

func (r *RedisQueue) Dequeue(ctx context.Context, priority domain.JobPriority, timeout time.Duration) (string, error) {
	queueKey := fmt.Sprintf("queue:%s", priority)
	result, err := r.client.BLPop(ctx, timeout, queueKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if len(result) < 2 {
		return "", nil
	}
	return result[1], nil
}

func (r *RedisQueue) DequeueWithPriority(ctx context.Context, timeout time.Duration) (string, domain.JobPriority, error) {
	highKey := "queue:high"
	defaultKey := "queue:default"
	lowKey := "queue:low"

	result, err := r.client.BLPop(ctx, timeout, highKey, defaultKey, lowKey).Result()
	if err == redis.Nil {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	if len(result) < 2 {
		return "", "", nil
	}

	var priority domain.JobPriority
	switch result[0] {
	case highKey:
		priority = domain.PriorityHigh
	case defaultKey:
		priority = domain.PriorityDefault
	case lowKey:
		priority = domain.PriorityLow
	default:
		return "", "", fmt.Errorf("unknown queue: %s", result[0])
	}

	return result[1], priority, nil
}

func (r *RedisQueue) GetQueueLength(ctx context.Context, priority domain.JobPriority) (int64, error) {
	queueKey := fmt.Sprintf("queue:%s", priority)
	return r.client.LLen(ctx, queueKey).Result()
}

func (r *RedisQueue) GetDelayedCount(ctx context.Context) (int64, error) {
	return r.client.ZCard(ctx, "queue:delayed").Result()
}

func (r *RedisQueue) GetReadyDelayed(ctx context.Context, now time.Time) ([]string, error) {
	maxScore := fmt.Sprintf("%d", now.Unix())
	result, err := r.client.ZRangeByScore(ctx, "queue:delayed", &redis.ZRangeBy{
		Min: "0",
		Max: maxScore,
	}).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RedisQueue) RemoveDelayed(ctx context.Context, jobID string) error {
	return r.client.ZRem(ctx, "queue:delayed", jobID).Err()
}

func (r *RedisQueue) UpdateWorkerHeartbeat(ctx context.Context, workerID string) error {
	return r.client.HSet(ctx, "worker:heartbeats", workerID, time.Now().Unix()).Err()
}

func (r *RedisQueue) GetWorkerHeartbeat(ctx context.Context, workerID string) (int64, error) {
	return r.client.HGet(ctx, "worker:heartbeats", workerID).Int64()
}

func (r *RedisQueue) RemoveWorkerHeartbeat(ctx context.Context, workerID string) error {
	return r.client.HDel(ctx, "worker:heartbeats", workerID).Err()
}

func (r *RedisQueue) GetAllHeartbeats(ctx context.Context) (map[string]int64, error) {
	result, err := r.client.HGetAll(ctx, "worker:heartbeats").Result()
	if err != nil {
		return nil, err
	}

	heartbeats := make(map[string]int64)
	for k, v := range result {
		var ts int64
		if err := json.Unmarshal([]byte(v), &ts); err == nil {
			heartbeats[k] = ts
		}
	}
	return heartbeats, nil
}
