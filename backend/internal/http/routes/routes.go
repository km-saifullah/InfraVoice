package routes

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/auth"
	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
	redisdb "github.com/km-saifullah/infra-voice/backend/internal/database/redis"
	"github.com/km-saifullah/infra-voice/backend/internal/http/handler"
	"github.com/km-saifullah/infra-voice/backend/internal/http/middleware"
	"github.com/km-saifullah/infra-voice/backend/internal/user"
)

func Setup(
	router *gin.Engine,
	cfg config.Config,
	logger *slog.Logger,
	mongoDatabase *mongodb.Database,
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
		mongoDatabase,
		redisClient,
	)

	router.GET(
		"/health",
		healthHandler.Health,
	)

	jwtService := auth.NewJWTService(cfg.JWT)

	sessionStore := auth.NewSessionStore(
		redisClient.Client(),
		cfg.JWT,
	)

	userRepository := user.NewRepository(
		mongoDatabase,
	)

	authService := auth.NewService(
		userRepository,
		jwtService,
		sessionStore,
	)

	authHandler := handler.NewAuthHandler(
		authService,
	)

	api := router.Group("/api/v1")

	authRoutes := api.Group("/auth")

	authRoutes.POST(
		"/register",
		authHandler.Register,
	)

	authRoutes.POST(
		"/login",
		authHandler.Login,
	)

	authRoutes.POST(
		"/refresh",
		authHandler.Refresh,
	)

	authRoutes.POST(
		"/logout",
		authHandler.Logout,
	)

	authenticatedRoutes := api.Group("")
	authenticatedRoutes.Use(
		middleware.Auth(jwtService),
	)

	authenticatedRoutes.GET(
		"/auth/me",
		authHandler.Me,
	)
}
