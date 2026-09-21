package routes

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/ai"
	"github.com/km-saifullah/infra-voice/backend/internal/ai/ollama"
	"github.com/km-saifullah/infra-voice/backend/internal/auth"
	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
	redisdb "github.com/km-saifullah/infra-voice/backend/internal/database/redis"
	"github.com/km-saifullah/infra-voice/backend/internal/http/handler"
	"github.com/km-saifullah/infra-voice/backend/internal/http/middleware"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
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

	projectRepository := project.NewRepository(
		mongoDatabase,
	)

	projectService := project.NewService(
		projectRepository,
	)

	projectHandler := handler.NewProjectHandler(
		projectService,
	)

	infrastructureRepository :=
		infrastructure.NewRepository(
			mongoDatabase,
		)

	infrastructureService :=
		infrastructure.NewService(
			infrastructureRepository,
			projectService,
		)

	infrastructureHandler :=
		handler.NewInfrastructureHandler(
			infrastructureService,
		)

	aiService := newAIService(
		cfg,
		logger,
	)

	aiHandler := handler.NewAIHandler(
		aiService,
		projectService,
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

	projectRoutes := authenticatedRoutes.Group(
		"/projects",
	)

	projectRoutes.POST(
		"",
		projectHandler.Create,
	)

	projectRoutes.GET(
		"",
		projectHandler.List,
	)

	projectRoutes.GET(
		"/:id",
		projectHandler.Get,
	)

	projectRoutes.PATCH(
		"/:id",
		projectHandler.Update,
	)

	projectRoutes.DELETE(
		"/:id",
		projectHandler.Delete,
	)

	infrastructureRoutes := projectRoutes.Group(
		"/:id/infrastructure",
	)

	infrastructureRoutes.POST(
		"/parse",
		aiHandler.ParseInfrastructure,
	)

	infrastructureRoutes.POST(
		"",
		infrastructureHandler.Create,
	)

	infrastructureRoutes.GET(
		"",
		infrastructureHandler.List,
	)

	infrastructureRoutes.GET(
		"/:spec_id",
		infrastructureHandler.Get,
	)

	infrastructureRoutes.PATCH(
		"/:spec_id",
		infrastructureHandler.Update,
	)

	infrastructureRoutes.DELETE(
		"/:spec_id",
		infrastructureHandler.Delete,
	)

	infrastructureRoutes.POST(
		"/:spec_id/validate",
		infrastructureHandler.Validate,
	)
}

func newAIService(
	cfg config.Config,
	logger *slog.Logger,
) *ai.Service {
	switch strings.ToLower(
		strings.TrimSpace(
			cfg.AI.Provider,
		),
	) {
	case "ollama":
		provider := ollama.NewProvider(
			cfg.AI.OllamaURL,
			cfg.AI.Model,
			nil,
		)

		return ai.NewService(provider)

	default:
		logger.Error(
			"unsupported AI provider configured",
			"provider",
			cfg.AI.Provider,
		)

		return ai.NewService(nil)
	}
}
