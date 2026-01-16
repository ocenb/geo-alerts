package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/ocenb/geo-alerts/internal/config"
)

type WebhookPayload struct {
	UserID    string  `json:"user_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Client struct {
	client     *asynq.Client
	maxRetries int
	timeout    time.Duration
}

func NewClient(redisCfg config.RedisConfig, queueCfg config.QueueConfig) (*Client, error) {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:         redisCfg.Addr,
		Password:     redisCfg.Password,
		DB:           redisCfg.DBQueue,
		DialTimeout:  redisCfg.DialTimeout,
		ReadTimeout:  redisCfg.ReadTimeout,
		WriteTimeout: redisCfg.WriteTimeout,
	})

	if err := client.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping redis (queue client): %w", err)
	}

	return &Client{
		client:     client,
		maxRetries: queueCfg.MaxRetries,
		timeout:    queueCfg.Timeout,
	}, nil
}

func (q *Client) EnqueueDangerAlert(ctx context.Context, userID string, latitude, longitude float64) error {
	payload, err := json.Marshal(WebhookPayload{UserID: userID, Latitude: latitude, Longitude: longitude})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	task := asynq.NewTask(TypeDangerWebhook, payload)

	_, err = q.client.EnqueueContext(ctx, task,
		asynq.MaxRetry(q.maxRetries),
		asynq.Timeout(q.timeout),
	)

	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}
	return nil
}

func (q *Client) Close() error {
	return q.client.Close()
}
