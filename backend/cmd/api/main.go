package main

import (
	"context"
	"log"
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
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found; using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	appContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	mongoClient, err := mongodb.New(appContext, cfg.MongoDB)
	if err != nil {
		log.Fatalf("failed to initialize mongodb: %v", err)
	}

	mongoDatabase, err := mongodb.NewDatabase(mongoClient)
	if err != nil {
		log.Fatalf("failed to initialize mongodb database: %v", err)
	}

	redisClient, err := redisdb.New(appContext, cfg.Redis)
	if err != nil {
		_ = mongoClient.Disconnect(context.Background())

		log.Fatalf("failed to initialize redis: %v", err)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		databaseStatus := "connected"

		if mongoDatabase.Database() == nil {
			databaseStatus = "disconnected"
		}

		redisStatus := "connected"

		healthContext, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := redisClient.Ping(healthContext); err != nil {
			redisStatus = "disconnected"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"dependencies": gin.H{
				"mongodb": databaseStatus,
				"redis":   redisStatus,
			},
		})
	})

	server := &http.Server{
		Addr:         cfg.Address(),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeoutSec) * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf(
			"starting %s on %s",
			cfg.App.Name,
			cfg.Address(),
		)

		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}

	case <-appContext.Done():
		log.Println("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("http server shutdown error: %v", err)
	}

	if err := redisClient.Close(); err != nil {
		log.Printf("redis shutdown error: %v", err)
	}

	if err := mongoClient.Disconnect(shutdownContext); err != nil {
		log.Printf("mongodb shutdown error: %v", err)
	}

	log.Println("application shutdown completed")
}
