package queue

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/ocenb/geo-alerts/internal/config"
	"github.com/ocenb/geo-alerts/internal/logger/logattr"
)

const TypeDangerWebhook = "webhook:danger"

type Server struct {
	log    *slog.Logger
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewServer(log *slog.Logger, asynqlog asynq.Logger, redisCfg config.RedisConfig, queueCfg config.QueueConfig, webhookWorker asynq.Handler) *Server {
	server := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:         redisCfg.Addr,
			Password:     redisCfg.Password,
			DB:           redisCfg.DBQueue,
			DialTimeout:  redisCfg.DialTimeout,
			WriteTimeout: redisCfg.WriteTimeout,
			ReadTimeout:  -1,
		},
		asynq.Config{
			Concurrency: queueCfg.Concurrency,
			Logger:      asynqlog,
			Queues: map[string]int{
				"default": 1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Error("process task failed",
					slog.String("type", task.Type()),
					logattr.Err(err),
				)
			}),
		},
	)

	mux := asynq.NewServeMux()
	mux.Handle(TypeDangerWebhook, webhookWorker)

	return &Server{
		log:    log,
		server: server,
		mux:    mux,
	}
}

func (s *Server) Run() error {
	s.log.Info("starting queue server")
	if err := s.server.Run(s.mux); err != nil && err != asynq.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Stop() {
	s.log.Info("stopping queue server")
	s.server.Stop()
	s.server.Shutdown()
}
