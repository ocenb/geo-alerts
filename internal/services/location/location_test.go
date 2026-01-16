package location

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/ocenb/geo-alerts/internal/domain/errs"
	"github.com/ocenb/geo-alerts/internal/domain/models"
	"github.com/ocenb/geo-alerts/internal/logger"
)

type LocationServiceSuite struct {
	suite.Suite
	mockCache *MockCacheRepo
	mockInc   *MockIncidentRepo
	mockLoc   *MockLocationRepo
	mockQueue *MockQueueProducer
	service   *Service
}

func (s *LocationServiceSuite) SetupTest() {
	s.mockCache = NewMockCacheRepo(s.T())
	s.mockInc = NewMockIncidentRepo(s.T())
	s.mockLoc = NewMockLocationRepo(s.T())
	s.mockQueue = NewMockQueueProducer(s.T())

	s.service = New(
		logger.NewDiscard(),
		time.Second,
		s.mockLoc,
		s.mockInc,
		s.mockCache,
		s.mockQueue,
	)
}

func TestLocationServiceSuite(t *testing.T) {
	suite.Run(t, new(LocationServiceSuite))
}

func (s *LocationServiceSuite) TestCheck_CacheHit_Danger() {
	ctx := context.Background()
	incident := models.IncidentShort{ID: 1, Latitude: 10.0, Longitude: 10.0, Radius: 1000}

	s.mockCache.On("GetActiveIncidents", mock.Anything).
		Return([]models.IncidentShort{incident}, nil)

	// Async calls
	s.mockLoc.On("SaveCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	s.mockQueue.On("EnqueueDangerAlert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	res, err := s.service.Check(ctx, &models.CheckLocationParams{
		UserID: "u1", Latitude: 10.0, Longitude: 10.0,
	})

	s.NoError(err)
	s.True(res.HasDanger)
	s.Len(res.Dangers, 1)
}

func (s *LocationServiceSuite) TestCheck_CacheMiss_DBSuccess() {
	ctx := context.Background()
	incident := models.IncidentShort{ID: 1, Latitude: 10.0, Longitude: 10.0, Radius: 1000}

	s.mockCache.On("GetActiveIncidents", mock.Anything).Return(nil, errs.ErrCacheMiss)

	s.mockInc.On("GetActive", mock.Anything).Return([]models.IncidentShort{incident}, nil)

	s.mockCache.On("SetActiveIncidents", mock.Anything, mock.Anything).Return(nil)

	// Async calls
	s.mockLoc.On("SaveCheckLog", mock.Anything, mock.Anything).Return(nil).Maybe()
	s.mockQueue.On("EnqueueDangerAlert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	res, err := s.service.Check(ctx, &models.CheckLocationParams{UserID: "u1", Latitude: 10.0, Longitude: 10.0})

	s.NoError(err)
	s.True(res.HasDanger)
}

func (s *LocationServiceSuite) TestCheck_DBError() {
	ctx := context.Background()

	s.mockCache.On("GetActiveIncidents", mock.Anything).Return(nil, errs.ErrCacheMiss)
	s.mockInc.On("GetActive", mock.Anything).Return(nil, errors.New("db error"))

	res, err := s.service.Check(ctx, &models.CheckLocationParams{UserID: "u1"})

	s.Error(err)
	s.Nil(res)
}

func (s *LocationServiceSuite) TestProcessPostCheck_Danger() {
	s.mockLoc.On("SaveCheckLog", mock.Anything, mock.Anything).Return(nil).Once()
	s.mockQueue.On("EnqueueDangerAlert", mock.Anything, "u1", 10.0, 10.0).Return(nil).Once()

	check := &models.CheckLocationResult{UserID: "u1", Latitude: 10.0, Longitude: 10.0, HasDanger: true}

	s.service.processPostCheck(check, logger.NewDiscard())
}

func (s *LocationServiceSuite) TestProcessPostCheck_Safe() {
	s.mockLoc.On("SaveCheckLog", mock.Anything, mock.Anything).Return(nil).Once()

	check := &models.CheckLocationResult{UserID: "u1", HasDanger: false}

	s.service.processPostCheck(check, logger.NewDiscard())
}
