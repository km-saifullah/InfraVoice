package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/km-saifullah/infra-voice/backend/internal/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found; using environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	router := gin.Default()

	router.GET("/health", healthHandler)

	if err := router.Run(cfg.Address()); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "infra-voice-api",
	})
}
