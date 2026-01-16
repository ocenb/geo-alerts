package incident

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/ocenb/geo-alerts/internal/config"
	"github.com/ocenb/geo-alerts/internal/domain/errs"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/logger"
)

type IncidentServiceSuite struct {
	suite.Suite
	mockInc   *MockIncidentRepo
	mockCache *MockCacheRepo
	service   *Service
}

func (s *IncidentServiceSuite) SetupTest() {
	s.mockInc = NewMockIncidentRepo(s.T())
	s.mockCache = NewMockCacheRepo(s.T())

	cfg := config.AppConfig{
		StatsTimeWindow: 15 * time.Minute,
	}

	s.service = New(
		logger.NewDiscard(),
		cfg,
		s.mockInc,
		s.mockCache,
	)
}

func TestIncidentServiceSuite(t *testing.T) {
	suite.Run(t, new(IncidentServiceSuite))
}

// --- Tests for GetByID ---

func (s *IncidentServiceSuite) TestGetByID_Success() {
	ctx := context.Background()
	expected := &models.Incident{ID: 1, IsActive: true}

	s.mockInc.On("GetByID", mock.Anything, int64(1)).Return(expected, nil)

	res, err := s.service.GetByID(ctx, 1)

	s.NoError(err)
	s.Equal(expected, res)
}

func (s *IncidentServiceSuite) TestGetByID_NotFound() {
	ctx := context.Background()

	s.mockInc.On("GetByID", mock.Anything, int64(1)).Return(nil, errs.ErrIncidentNotFound)

	res, err := s.service.GetByID(ctx, 1)

	s.ErrorIs(err, errs.ErrIncidentNotFound)
	s.Nil(res)
}

// --- Tests for Create ---

func (s *IncidentServiceSuite) TestCreate_Success() {
	ctx := context.Background()
	params := &models.CreateIncidentParams{Latitude: 10, Longitude: 10, Radius: 100}
	created := &models.Incident{ID: 1, Latitude: 10, Longitude: 10, Radius: 100, IsActive: true}

	s.mockInc.On("Create", mock.Anything, params).Return(created, nil)

	s.mockCache.On("InvalidateActiveIncidents", mock.Anything).Return(nil)

	res, err := s.service.Create(ctx, params)

	s.NoError(err)
	s.Equal(created, res)
}

func (s *IncidentServiceSuite) TestCreate_RepoError() {
	ctx := context.Background()
	params := &models.CreateIncidentParams{Latitude: 10, Longitude: 10}

	s.mockInc.On("Create", mock.Anything, params).Return(nil, errors.New("db error"))

	res, err := s.service.Create(ctx, params)

	s.Error(err)
	s.Nil(res)
}

func (s *IncidentServiceSuite) TestCreate_CacheInvalidationError() {
	ctx := context.Background()
	params := &models.CreateIncidentParams{Latitude: 10}
	created := &models.Incident{ID: 1}

	s.mockInc.On("Create", mock.Anything, params).Return(created, nil)
	s.mockCache.On("InvalidateActiveIncidents", mock.Anything).Return(errors.New("redis down"))

	res, err := s.service.Create(ctx, params)

	s.NoError(err)
	s.Equal(created, res)
}

// --- Tests for Update ---

func (s *IncidentServiceSuite) TestUpdate_Success() {
	ctx := context.Background()
	params := &models.UpdateIncidentParams{ID: 1}
	updated := &models.Incident{ID: 1}

	s.mockInc.On("Update", mock.Anything, params).Return(updated, nil)
	s.mockCache.On("InvalidateActiveIncidents", mock.Anything).Return(nil)

	res, err := s.service.Update(ctx, params)

	s.NoError(err)
	s.Equal(updated, res)
}

func (s *IncidentServiceSuite) TestUpdate_NotFound() {
	ctx := context.Background()
	params := &models.UpdateIncidentParams{ID: 1}

	s.mockInc.On("Update", mock.Anything, params).Return(nil, errs.ErrIncidentNotFound)

	res, err := s.service.Update(ctx, params)

	s.ErrorIs(err, errs.ErrIncidentNotFound)
	s.Nil(res)
}

// --- Tests for Deactivate ---

func (s *IncidentServiceSuite) TestDeactivate_Success() {
	ctx := context.Background()

	s.mockInc.On("Deactivate", mock.Anything, int64(1)).Return(nil)
	s.mockCache.On("InvalidateActiveIncidents", mock.Anything).Return(nil)

	err := s.service.Deactivate(ctx, 1)

	s.NoError(err)
}

func (s *IncidentServiceSuite) TestDeactivate_NotFound() {
	ctx := context.Background()

	s.mockInc.On("Deactivate", mock.Anything, int64(1)).Return(errs.ErrIncidentNotFound)

	err := s.service.Deactivate(ctx, 1)

	s.ErrorIs(err, errs.ErrIncidentNotFound)
}

// --- Tests for GetStats ---

func (s *IncidentServiceSuite) TestGetStats_Success() {
	ctx := context.Background()
	window := 15 * time.Minute
	expected := []models.Stats{{IncidentID: 1, UserCount: 100}}

	s.mockInc.On("GetStats", mock.Anything, window).Return(expected, nil)

	res, err := s.service.GetStats(ctx)

	s.NoError(err)
	s.Equal(expected, res)
}
