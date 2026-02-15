package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	StreamTranscode = "tasks:transcode"
	StreamThumbnail = "tasks:thumbnail"
	StreamDRM       = "tasks:drm"
	StreamProbe     = "tasks:probe"
	ConsumerGroup   = "workers"
)

type TaskMessage struct {
	TaskUUID string          `json:"task_uuid"`
	VideoID  uint            `json:"video_id"`
	Type     string          `json:"type"`
	Params   json.RawMessage `json:"params"`
}

type Producer struct {
	rdb *redis.Client
}

func NewProducer(rdb *redis.Client) *Producer {
	return &Producer{rdb: rdb}
}

// InitStreams creates the streams and consumer groups if they don't exist.
func (p *Producer) InitStreams(ctx context.Context) error {
	streams := []string{StreamTranscode, StreamThumbnail, StreamDRM, StreamProbe}
	for _, stream := range streams {
		err := p.rdb.XGroupCreateMkStream(ctx, stream, ConsumerGroup, "0").Err()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return fmt.Errorf("create consumer group for %s: %w", stream, err)
		}
	}
	return nil
}

// Publish sends a task message to the appropriate stream.
func (p *Producer) Publish(ctx context.Context, msg TaskMessage) error {
	stream := streamForType(msg.Type)
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal task message: %w", err)
	}

	return p.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Err()
}

func streamForType(taskType string) string {
	switch taskType {
	case "transcode":
		return StreamTranscode
	case "thumbnail":
		return StreamThumbnail
	case "drm_package":
		return StreamDRM
	case "probe":
		return StreamProbe
	default:
		return StreamTranscode
	}
}
