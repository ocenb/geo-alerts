package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ocenb/geo-alerts/internal/config"
	"github.com/ocenb/geo-alerts/internal/domain/errs"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/redis/go-redis/v9"
)

type Repo struct {
	cfg    config.CacheConfig
	client *redis.Client
}

func New(cfg config.CacheConfig, client *redis.Client) *Repo {
	return &Repo{cfg, client}
}

const KeyActiveIncidents = "incidents:active"

func (r *Repo) InvalidateActiveIncidents(ctx context.Context) error {
	return r.client.Del(ctx, KeyActiveIncidents).Err()
}

func (r *Repo) GetActiveIncidents(ctx context.Context) ([]models.IncidentShort, error) {
	val, err := r.client.Get(ctx, KeyActiveIncidents).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errs.ErrCacheMiss
		}
		return nil, err
	}

	var res []models.IncidentShort
	if err := json.Unmarshal(val, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}
	return res, nil
}

func (r *Repo) SetActiveIncidents(ctx context.Context, incidents []models.IncidentShort) error {
	data, err := json.Marshal(incidents)
	if err != nil {
		return fmt.Errorf("failed to marshal incidents: %w", err)
	}
	return r.client.Set(ctx, KeyActiveIncidents, data, r.cfg.IncidentsTTL).Err()
}
