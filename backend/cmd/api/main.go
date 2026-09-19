package main

import (
	"context"
	"log"
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

	router.GET("/health", healthHandler)

	go func() {
		if err := router.Run(cfg.Address()); err != nil {
			log.Printf("server stopped: %v", err)
			stop()
		}
	}()

	<-appContext.Done()

	log.Println("shutdown signal received")
}

func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "ok",
		"service": "infra-voice-api",
	})
}
