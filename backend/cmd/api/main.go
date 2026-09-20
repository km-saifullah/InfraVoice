package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
	redisdb "github.com/km-saifullah/infra-voice/backend/internal/database/redis"
	"github.com/km-saifullah/infra-voice/backend/internal/http/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found; using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		),
	)

	appContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	mongoClient, err := mongodb.New(
		appContext,
		cfg.MongoDB,
	)
	if err != nil {
		logger.Error(
			"failed to initialize mongodb",
			"error",
			err,
		)

		os.Exit(1)
	}

	mongoDatabase, err := mongodb.NewDatabase(
		mongoClient,
	)
	if err != nil {
		_ = mongoClient.Disconnect(
			context.Background(),
		)

		logger.Error(
			"failed to initialize mongodb database",
			"error",
			err,
		)

		os.Exit(1)
	}

	indexContext, indexCancel := context.WithTimeout(
		appContext,
		10*time.Second,
	)

	if err := mongodb.EnsureIndexes(
		indexContext,
		mongoDatabase,
	); err != nil {
		indexCancel()
		_ = mongoClient.Disconnect(
			context.Background(),
		)

		logger.Error(
			"failed to initialize mongodb indexes",
			"error",
			err,
		)

		os.Exit(1)
	}

	indexCancel()

	redisClient, err := redisdb.New(
		appContext,
		cfg.Redis,
	)
	if err != nil {
		_ = mongoClient.Disconnect(
			context.Background(),
		)

		logger.Error(
			"failed to initialize redis",
			"error",
			err,
		)

		os.Exit(1)
	}

	router := gin.New()

	routes.Setup(
		router,
		cfg,
		logger,
		mongoDatabase,
		redisClient,
	)

	server := &http.Server{
		Addr:    cfg.Address(),
		Handler: router,

		ReadTimeout: time.Duration(
			cfg.Server.ReadTimeoutSec,
		) * time.Second,

		WriteTimeout: time.Duration(
			cfg.Server.WriteTimeoutSec,
		) * time.Second,

		IdleTimeout: time.Duration(
			cfg.Server.IdleTimeoutSec,
		) * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info(
			"starting application",
			"service",
			cfg.App.Name,
			"address",
			cfg.Address(),
			"environment",
			cfg.App.Env,
		)

		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil &&
			err != http.ErrServerClosed {
			logger.Error(
				"http server error",
				"error",
				err,
			)
		}

	case <-appContext.Done():
		logger.Info(
			"shutdown signal received",
		)
	}

	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(
		shutdownContext,
	); err != nil {
		logger.Error(
			"http server shutdown error",
			"error",
			err,
		)
	}

	if err := redisClient.Close(); err != nil {
		logger.Error(
			"redis shutdown error",
			"error",
			err,
		)
	}

	if err := mongoClient.Disconnect(
		shutdownContext,
	); err != nil {
		logger.Error(
			"mongodb shutdown error",
			"error",
			err,
		)
	}

	logger.Info(
		"application shutdown completed",
	)
}
