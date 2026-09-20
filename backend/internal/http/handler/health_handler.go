package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
	redisdb "github.com/km-saifullah/infra-voice/backend/internal/database/redis"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
)

type HealthHandler struct {
	config      config.Config
	mongoClient *mongodb.Client
	redisClient *redisdb.Client
}

func NewHealthHandler(
	cfg config.Config,
	mongoClient *mongodb.Client,
	redisClient *redisdb.Client,
) *HealthHandler {
	return &HealthHandler{
		config:      cfg,
		mongoClient: mongoClient,
		redisClient: redisClient,
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		2*time.Second,
	)
	defer cancel()

	mongoStatus := "connected"

	if err := h.mongoClient.Ping(ctx); err != nil {
		mongoStatus = "disconnected"
	}

	redisStatus := "connected"

	if err := h.redisClient.Ping(ctx); err != nil {
		redisStatus = "disconnected"
	}

	statusCode := http.StatusOK
	overallStatus := "ok"

	if mongoStatus != "connected" || redisStatus != "connected" {
		statusCode = http.StatusServiceUnavailable
		overallStatus = "degraded"
	}

	response.JSON(
		c,
		statusCode,
		gin.H{
			"status":  overallStatus,
			"service": h.config.App.Name,
			"dependencies": gin.H{
				"mongodb": mongoStatus,
				"redis":   redisStatus,
			},
		},
	)
}
