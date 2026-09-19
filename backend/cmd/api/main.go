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

	defer func() {
		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := mongoClient.Disconnect(shutdownContext); err != nil {
			log.Printf("mongodb disconnect error: %v", err)
		}
	}()

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"database": func() string {
				if mongoDatabase.Database() != nil {
					return "connected"
				}

				return "disconnected"
			}(),
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
			log.Fatalf("server error: %v", err)
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

	if err := mongoClient.Disconnect(shutdownContext); err != nil {
		log.Printf("mongodb disconnect error: %v", err)
	}

	log.Println("application shutdown completed")
}
