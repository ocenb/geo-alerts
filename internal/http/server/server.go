package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ocenb/geo-alerts/internal/config"
)

type HttpServer struct {
	log        *slog.Logger
	cfg        config.ServerConfig
	httpServer *http.Server
}

func New(log *slog.Logger, cfg config.ServerConfig, handler http.Handler) *HttpServer {
	return &HttpServer{
		log: log,
		cfg: cfg,
		httpServer: &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           handler,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		},
	}
}

func (s *HttpServer) Start() error {
	s.log.Info("starting HTTP server", slog.String("port", s.cfg.Port))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *HttpServer) Stop(ctx context.Context) error {
	s.log.Info("stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}
