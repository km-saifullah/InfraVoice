package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
	redisdb "github.com/km-saifullah/infra-voice/backend/internal/database/redis"
	"github.com/km-saifullah/infra-voice/backend/internal/http/handler"
	"github.com/km-saifullah/infra-voice/backend/internal/http/middleware"
)

func Setup(
	router *gin.Engine,
	cfg config.Config,
	logger *slog.Logger,
	mongoClient *mongodb.Client,
	redisClient *redisdb.Client,
) {
	router.Use(
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.Recovery(logger),
		middleware.CORS([]string{
			"http://localhost:3000",
			"http://localhost:5173",
		}),
		middleware.RateLimit(
			redisClient.Client(),
			100,
		),
	)

	healthHandler := handler.NewHealthHandler(
		cfg,
		mongoClient,
		redisClient,
	)

	router.GET("/health", healthHandler.Health)

	api := router.Group("/api/v1")

	_ = api
}
