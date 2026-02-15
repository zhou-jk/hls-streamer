package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Consumer struct {
	rdb        *redis.Client
	consumerID string
	streams    []string
}

func NewConsumer(rdb *redis.Client, consumerID string) *Consumer {
	return &Consumer{
		rdb:        rdb,
		consumerID: consumerID,
		streams:    []string{StreamTranscode, StreamThumbnail, StreamDRM, StreamProbe},
	}
}

// ReadTask blocks until a task is available, then returns it along with the stream name and message ID.
func (c *Consumer) ReadTask(ctx context.Context, blockTimeout time.Duration) (*TaskMessage, string, string, error) {
	// Build XREADGROUP args: stream1 stream2 ... > > ...
	args := make([]string, 0, len(c.streams)*2)
	for _, s := range c.streams {
		args = append(args, s)
	}
	for range c.streams {
		args = append(args, ">")
	}

	results, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: c.consumerID,
		Streams:  args,
		Count:    1,
		Block:    blockTimeout,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, "", "", nil // timeout, no message
		}
		return nil, "", "", fmt.Errorf("xreadgroup: %w", err)
	}

	for _, stream := range results {
		for _, msg := range stream.Messages {
			data, ok := msg.Values["data"].(string)
			if !ok {
				continue
			}
			var task TaskMessage
			if err := json.Unmarshal([]byte(data), &task); err != nil {
				continue
			}
			return &task, stream.Stream, msg.ID, nil
		}
	}

	return nil, "", "", nil
}

// Ack acknowledges a message.
func (c *Consumer) Ack(ctx context.Context, stream, messageID string) error {
	return c.rdb.XAck(ctx, stream, ConsumerGroup, messageID).Err()
}

// ClaimPending reclaims messages that have been pending for longer than minIdle.
func (c *Consumer) ClaimPending(ctx context.Context, stream string, minIdle time.Duration) ([]TaskMessage, []string, error) {
	pending, err := c.rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  ConsumerGroup,
		Start:  "-",
		End:    "+",
		Count:  10,
	}).Result()
	if err != nil {
		return nil, nil, err
	}

	var tasks []TaskMessage
	var messageIDs []string

	for _, p := range pending {
		if p.Idle < minIdle {
			continue
		}

		claimed, err := c.rdb.XClaim(ctx, &redis.XClaimArgs{
			Stream:   stream,
			Group:    ConsumerGroup,
			Consumer: c.consumerID,
			MinIdle:  minIdle,
			Messages: []string{p.ID},
		}).Result()
		if err != nil {
			continue
		}

		for _, msg := range claimed {
			data, ok := msg.Values["data"].(string)
			if !ok {
				continue
			}
			var task TaskMessage
			if err := json.Unmarshal([]byte(data), &task); err != nil {
				continue
			}
			tasks = append(tasks, task)
			messageIDs = append(messageIDs, msg.ID)
		}
	}

	return tasks, messageIDs, nil
}
