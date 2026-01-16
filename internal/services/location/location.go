package location

import (
	"context"
	"log/slog"
	"time"

	"github.com/ocenb/geo-alerts/internal/domain/errs"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/logger/logattr"
	"github.com/ocenb/geo-alerts/internal/utils/geo"
)

type LocationRepo interface {
	SaveCheckLog(ctx context.Context, check *models.CheckLocationResult) error
}

type IncidentRepo interface {
	GetActive(ctx context.Context) ([]models.IncidentShort, error)
}

type CacheRepo interface {
	GetActiveIncidents(ctx context.Context) ([]models.IncidentShort, error)
	SetActiveIncidents(ctx context.Context, incidents []models.IncidentShort) error
}

type QueueProducer interface {
	EnqueueDangerAlert(ctx context.Context, userID string, latitude, longitude float64) error
}

type Service struct {
	log             *slog.Logger
	asyncJobTimeout time.Duration
	locationRepo    LocationRepo
	incRepo         IncidentRepo
	cacheRepo       CacheRepo
	queue           QueueProducer
}

func New(log *slog.Logger, asyncJobTimeout time.Duration, locationRepo LocationRepo, incRepo IncidentRepo, cacheRepo CacheRepo, queue QueueProducer) *Service {
	return &Service{
		log:             log,
		asyncJobTimeout: asyncJobTimeout,
		locationRepo:    locationRepo,
		incRepo:         incRepo,
		cacheRepo:       cacheRepo,
		queue:           queue,
	}
}

func (s *Service) Check(ctx context.Context, params *models.CheckLocationParams) (*models.CheckLocationResult, error) {
	log := s.log.With(
		logattr.Op("LocationService.Check"),
		slog.String("user_id", params.UserID),
		slog.Float64("latitude", params.Latitude),
		slog.Float64("longitude", params.Longitude),
	)

	incidents, err := s.getActiveIncidents(ctx)
	if err != nil {
		log.Error("failed to get active incidents", logattr.Err(err))
		return nil, err
	}

	foundDangers := make([]models.IncidentShort, 0)
	for _, inc := range incidents {
		dist := geo.Distance(params.Latitude, params.Longitude, inc.Latitude, inc.Longitude)
		if dist <= float64(inc.Radius) {
			foundDangers = append(foundDangers, inc)
		}
	}

	result := &models.CheckLocationResult{
		UserID:    params.UserID,
		Latitude:  params.Latitude,
		Longitude: params.Longitude,
		HasDanger: len(foundDangers) > 0,
		Dangers:   foundDangers,
		CreatedAt: time.Now(),
	}

	go s.processPostCheck(result, log)

	return result, nil
}

func (s *Service) getActiveIncidents(ctx context.Context) ([]models.IncidentShort, error) {
	incidents, err := s.cacheRepo.GetActiveIncidents(ctx)
	if err == nil {
		return incidents, nil
	}

	if err != errs.ErrCacheMiss {
		s.log.Warn("failed to read from cache", logattr.Err(err))
	} else {
		s.log.Debug("cache miss, reading from DB")
	}

	incidents, err = s.incRepo.GetActive(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.cacheRepo.SetActiveIncidents(ctx, incidents); err != nil {
		s.log.Warn("failed to update cache", logattr.Err(err))
	}

	return incidents, nil
}

func (s *Service) processPostCheck(check *models.CheckLocationResult, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), s.asyncJobTimeout)
	defer cancel()

	if err := s.locationRepo.SaveCheckLog(ctx, check); err != nil {
		log.Error("failed to save log for user", logattr.Err(err))
	}

	if check.HasDanger {
		if err := s.queue.EnqueueDangerAlert(ctx, check.UserID, check.Latitude, check.Longitude); err != nil {
			log.Error("failed to enqueue webhook for user", logattr.Err(err))
		}
	}
}
