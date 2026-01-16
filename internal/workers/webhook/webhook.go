package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/hibiken/asynq"
	"github.com/ocenb/geo-alerts/internal/config"
	"github.com/ocenb/geo-alerts/internal/logger/logattr"
	"github.com/ocenb/geo-alerts/internal/queue"
)

type TaskHandler struct {
	log        *slog.Logger
	webhookURL string
	httpClient *http.Client
}

func New(log *slog.Logger, cfg config.WebhookConfig) *TaskHandler {
	return &TaskHandler{
		log:        log,
		webhookURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        cfg.ClientMaxIdleConns,
				MaxIdleConnsPerHost: cfg.ClientMaxIdleConnsPerHost,
				IdleConnTimeout:     cfg.ClientIdleConnTimeout,
			},
		},
	}
}

func (h *TaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload queue.WebhookPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		h.log.Error("failed to unmarshal task payload", logattr.Err(err))
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	log := h.log.With(
		slog.String("user_id", payload.UserID),
		slog.String("task_type", t.Type()),
	)

	log.Debug("processing webhook task")

	reqBody, err := json.Marshal(payload)
	if err != nil {
		log.Error("failed to marshal request body", logattr.Err(err))
		return fmt.Errorf("json.Marshal failed: %v: %w", err, asynq.SkipRetry)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.webhookURL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Error("failed to create http request", logattr.Err(err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Warn("failed to send webhook", logattr.Err(err))
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warn("failed to close response body", logattr.Err(err))
		}
	}()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn("webhook returned non-success status", slog.Int("status", resp.StatusCode))
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	log.Info("webhook sent successfully")
	return nil
}
