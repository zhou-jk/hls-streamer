package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Heartbeat struct {
	rdb      *redis.Client
	workerID string
	interval time.Duration
}

func NewHeartbeat(rdb *redis.Client, workerID string, interval time.Duration) *Heartbeat {
	return &Heartbeat{
		rdb:      rdb,
		workerID: workerID,
		interval: interval,
	}
}

func (h *Heartbeat) Run(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			key := fmt.Sprintf("worker:%s:heartbeat", h.workerID)
			err := h.rdb.Set(ctx, key, time.Now().Unix(), h.interval*3).Err()
			if err != nil {
				slog.Error("heartbeat failed", "worker_id", h.workerID, "error", err)
			}
		}
	}
}
