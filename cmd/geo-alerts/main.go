package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	_ "github.com/ocenb/geo-alerts/docs"
	"github.com/ocenb/geo-alerts/internal/config"
	incidenthandler "github.com/ocenb/geo-alerts/internal/handlers/incident"
	locationhandler "github.com/ocenb/geo-alerts/internal/handlers/location"
	systemhandler "github.com/ocenb/geo-alerts/internal/handlers/system"
	"github.com/ocenb/geo-alerts/internal/http/server"
	"github.com/ocenb/geo-alerts/internal/logger"
	"github.com/ocenb/geo-alerts/internal/logger/logattr"
	"github.com/ocenb/geo-alerts/internal/middlewares"
	"github.com/ocenb/geo-alerts/internal/queue"
	cacherepo "github.com/ocenb/geo-alerts/internal/repos/cache"
	incidentrepo "github.com/ocenb/geo-alerts/internal/repos/incident"
	locationrepo "github.com/ocenb/geo-alerts/internal/repos/location"
	incidentsvc "github.com/ocenb/geo-alerts/internal/services/incident"
	locationsvc "github.com/ocenb/geo-alerts/internal/services/location"
	"github.com/ocenb/geo-alerts/internal/storage/cache"
	"github.com/ocenb/geo-alerts/internal/storage/migrator"
	"github.com/ocenb/geo-alerts/internal/storage/postgres"
	"github.com/ocenb/geo-alerts/internal/storage/transactor"
	"github.com/ocenb/geo-alerts/internal/workers/webhook"
	"github.com/ocenb/geo-alerts/migrations"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Geo Alerts API
// @version         1.0
// @description     API service for tracking dangerous incidents and checking user locations.
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {
	os.Exit(run())
}

func run() int {
	cfg := config.MustLoad()
	log := logger.New(cfg.Log, cfg.Environment)

	defer log.Info("app stopped")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	err := func() error {
		log.Info("connecting to database for migrations",
			slog.String("host", cfg.Postgres.Host),
			slog.String("port", cfg.Postgres.Port),
			slog.String("database", cfg.Postgres.Name),
		)
		migrateCtx, migrateCancel := context.WithTimeout(ctx, cfg.DBConnectTimeout)
		defer migrateCancel()
		migrate, err := migrator.New(migrateCtx, cfg.Postgres, migrations.FS)
		if err != nil {
			return err
		}
		defer func() {
			if err := migrate.Close(); err != nil {
				log.Error("failed to close migrate connection", logattr.Err(err))
			}
		}()

		log.Info("running migrations")
		if err := migrate.Up(); err != nil {
			return err
		}
		log.Info("migrations completed successfully")
		return nil
	}()
	if err != nil {
		log.Error("initialization failed", logattr.Err(err))
		return 1
	}

	log.Info("connecting to database",
		slog.String("host", cfg.Postgres.Host),
		slog.String("port", cfg.Postgres.Port),
		slog.String("database", cfg.Postgres.Name),
	)
	connectCtx, connectCancel := context.WithTimeout(ctx, cfg.DBConnectTimeout)
	defer connectCancel()
	pool, err := postgres.NewPool(connectCtx, cfg.Postgres)
	if err != nil {
		log.Error("initialization failed", logattr.Err(err))
		return 1
	}
	defer pool.Close()

	tm := transactor.New(pool)

	cacheClient, err := cache.NewClient(cfg.Redis)
	if err != nil {
		log.Error("initialization failed", logattr.Err(err))
		return 1
	}
	defer func() {
		if err := cacheClient.Close(); err != nil {
			log.Error("failed to close redis cache client", logattr.Err(err))
		}
	}()

	queueClient, err := queue.NewClient(cfg.Redis, cfg.Queue)
	if err != nil {
		log.Error("initialization failed", logattr.Err(err))
		return 1
	}
	defer func() {
		if err := queueClient.Close(); err != nil {
			log.Error("failed to close redis queue client", logattr.Err(err))
		}
	}()

	cacheRepo := cacherepo.New(cfg.Cache, cacheClient)
	incRepo := incidentrepo.New(tm)
	locationRepo := locationrepo.New(tm)

	incService := incidentsvc.New(log, cfg.App, incRepo, cacheRepo)
	locationService := locationsvc.New(log, cfg.AsyncJobTimeout, locationRepo, incRepo, cacheRepo, queueClient)

	incHandler := incidenthandler.New(incService)
	locationHandler := locationhandler.New(locationService)
	systemHandler := systemhandler.New()

	if cfg.Environment == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(middlewares.RequestID())
	router.Use(middlewares.Logging(log))
	router.Use(gin.Recovery())

	api := router.Group("/api/v1")
	apiWithAuth := router.Group("/api/v1")
	apiWithAuth.Use(middlewares.Auth(cfg.App.APIKey))

	incHandler.RegisterRoutes(apiWithAuth)
	locationHandler.RegisterRoutes(api)
	systemHandler.RegisterRoutes(api)
	api.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	webhookWorker := webhook.New(log, cfg.Webhook)
	queueServer := queue.NewServer(log, logger.NewAsynqAdapter(log), cfg.Redis, cfg.Queue, webhookWorker)
	queueServerErrors := make(chan error, 1)
	go func() {
		queueServerErrors <- queueServer.Run()
	}()

	httpServer := server.New(log, cfg.Server, router)
	httpServerErrors := make(chan error, 1)
	go func() {
		httpServerErrors <- httpServer.Start()
	}()

	var queueServerErr, httpServerErr error
	select {
	case queueServerErr = <-queueServerErrors:
		if queueServerErr != nil {
			log.Error("queue server crashed", logattr.Err(queueServerErr))
		} else {
			log.Error("queue server stopped unexpectedly")
		}
	case httpServerErr = <-httpServerErrors:
		if httpServerErr != nil {
			log.Error("HTTP server crashed", logattr.Err(httpServerErr))
		} else {
			log.Error("HTTP server stopped unexpectedly")
		}
	case <-ctx.Done():
		log.Info("received shutdown signal")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	shutdownErr := httpServer.Stop(shutdownCtx)
	if shutdownErr != nil {
		log.Error("HTTP server shutdown error", logattr.Err(shutdownErr))
	}

	queueServer.Stop()

	if shutdownErr != nil || queueServerErr != nil || httpServerErr != nil {
		return 1
	}
	return 0
}
