package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	serverAddress = ":8000"
)

func main() {
	router := gin.Default()

	router.GET("/health", healthHandler)

	if err := router.Run(serverAddress); err != nil {
		panic(err)
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "infra-voice-api",
	})
}
