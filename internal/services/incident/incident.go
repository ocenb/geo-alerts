package incident

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ocenb/geo-alerts/internal/config"
	"github.com/ocenb/geo-alerts/internal/domain/errs"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/logger/logattr"
)

type IncidentRepo interface {
	Create(ctx context.Context, params *models.CreateIncidentParams) (*models.Incident, error)
	GetByID(ctx context.Context, id int64) (*models.Incident, error)
	Update(ctx context.Context, params *models.UpdateIncidentParams) (*models.Incident, error)
	Deactivate(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]models.Incident, error)
	GetStats(ctx context.Context, window time.Duration) ([]models.Stats, error)
}

type CacheRepo interface {
	InvalidateActiveIncidents(ctx context.Context) error
}

type Service struct {
	log       *slog.Logger
	cfg       config.AppConfig
	incRepo   IncidentRepo
	cacheRepo CacheRepo
}

func New(log *slog.Logger, cfg config.AppConfig, incRepo IncidentRepo, cacheRepo CacheRepo) *Service {
	return &Service{
		log:       log,
		cfg:       cfg,
		incRepo:   incRepo,
		cacheRepo: cacheRepo,
	}
}

func (s *Service) GetByID(ctx context.Context, id int64) (*models.Incident, error) {
	incident, err := s.incRepo.GetByID(ctx, id)
	if err != nil {
		if !errors.Is(err, errs.ErrIncidentNotFound) {
			s.log.Error("failed to get incident", logattr.Op("IncidentService.GetByID"), slog.Int64("id", id), logattr.Err(err))
		}
		return nil, err
	}

	return incident, nil
}

func (s *Service) Create(ctx context.Context, params *models.CreateIncidentParams) (*models.Incident, error) {
	log := s.log.With(
		logattr.Op("IncidentService.Create"),
		slog.Float64("latitude", params.Latitude),
		slog.Float64("longitude", params.Longitude),
	)

	created, err := s.incRepo.Create(ctx, params)
	if err != nil {
		if !errors.Is(err, errs.ErrIncidentExists) {
			log.Error("failed to create incident", logattr.Err(err))
		}
		return nil, err
	}

	if err := s.cacheRepo.InvalidateActiveIncidents(ctx); err != nil {
		log.Warn("failed to invalidate cache", logattr.Err(err))
	}

	return created, nil
}

func (s *Service) Update(ctx context.Context, params *models.UpdateIncidentParams) (*models.Incident, error) {
	log := s.log.With(
		logattr.Op("IncidentService.Update"),
		slog.Int64("id", params.ID),
	)

	updated, err := s.incRepo.Update(ctx, params)
	if err != nil {
		if !errors.Is(err, errs.ErrIncidentNotFound) && !errors.Is(err, errs.ErrIncidentExists) {
			log.Error("failed to update incident", logattr.Err(err))
		}
		return nil, err
	}

	if err := s.cacheRepo.InvalidateActiveIncidents(ctx); err != nil {
		log.Warn("failed to invalidate cache", logattr.Err(err))
	}

	return updated, nil
}

func (s *Service) Deactivate(ctx context.Context, id int64) error {
	log := s.log.With(
		logattr.Op("IncidentService.Deactivate"),
		slog.Int64("id", id),
	)

	err := s.incRepo.Deactivate(ctx, id)
	if err != nil {
		if !errors.Is(err, errs.ErrIncidentNotFound) {
			log.Error("failed to deactivate incident", logattr.Err(err))
		}
		return err
	}

	if err := s.cacheRepo.InvalidateActiveIncidents(ctx); err != nil {
		log.Warn("failed to invalidate cache", logattr.Err(err))
	}

	return nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]models.Incident, error) {
	list, err := s.incRepo.List(ctx, limit, offset)
	if err != nil {
		s.log.Error("failed to list incidents", logattr.Op("IncidentService.List"), logattr.Err(err))
		return nil, err
	}

	return list, nil
}

func (s *Service) GetStats(ctx context.Context) ([]models.Stats, error) {
	stats, err := s.incRepo.GetStats(ctx, s.cfg.StatsTimeWindow)
	if err != nil {
		s.log.Error("failed to get unique users stats", logattr.Err(err))
		return nil, err
	}
	return stats, nil
}
