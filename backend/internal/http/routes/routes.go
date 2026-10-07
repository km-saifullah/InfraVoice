package routes

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/ai"
	"github.com/km-saifullah/infra-voice/backend/internal/ai/ollama"
	"github.com/km-saifullah/infra-voice/backend/internal/auth"
	"github.com/km-saifullah/infra-voice/backend/internal/command"
	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
	redisdb "github.com/km-saifullah/infra-voice/backend/internal/database/redis"
	"github.com/km-saifullah/infra-voice/backend/internal/http/handler"
	"github.com/km-saifullah/infra-voice/backend/internal/http/middleware"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/overview"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
	"github.com/km-saifullah/infra-voice/backend/internal/speech"
	"github.com/km-saifullah/infra-voice/backend/internal/speech/whisper"
	"github.com/km-saifullah/infra-voice/backend/internal/terraform"
	"github.com/km-saifullah/infra-voice/backend/internal/user"
)

func Setup(
	router *gin.Engine,
	cfg config.Config,
	logger *slog.Logger,
	mongoDatabase *mongodb.Database,
	redisClient *redisdb.Client,
) *terraform.ExecutionService {
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

	jwtService := auth.NewJWTService(
		cfg.JWT,
	)

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

	commandRepository := command.NewRepository(
		mongoDatabase,
	)

	commandService := command.NewService(
		commandRepository,
		projectService,
	)

	commandHandler := handler.NewCommandHandler(
		commandService,
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

	speechService := newSpeechService(
		cfg,
		logger,
	)

	speechHandler := handler.NewSpeechHandler(
		speechService,
		commandService,
		aiService,
		projectService,
	)

	terraformGenerator := terraform.NewGenerator()

	terraformService := terraform.NewService(
		terraformGenerator,
		infrastructureService,
	)

	terraformHandler := handler.NewTerraformHandler(
		terraformService,
	)

	terraformRunRepository := terraform.NewRunRepository(
		mongoDatabase,
	)

	terraformExecutor := terraform.NewCLIExecutor(
		cfg.Terraform.BinaryPath,
	)

	terraformExecutionService := terraform.NewExecutionService(
		cfg.Terraform,
		terraformGenerator,
		infrastructureService,
		projectService,
		terraformRunRepository,
		terraformExecutor,
		logger,
	)

	terraformExecutionHandler := handler.NewTerraformExecutionHandler(
		terraformExecutionService,
	)

	overviewService := overview.NewService(
		projectRepository,
		commandRepository,
		infrastructureRepository,
		terraformRunRepository,
	)

	overviewHandler := handler.NewOverviewHandler(
		overviewService,
	)

	api := router.Group(
		"/api/v1",
	)

	authRoutes := api.Group(
		"/auth",
	)

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

	authenticatedRoutes.GET(
		"/overview",
		overviewHandler.Get,
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

	speechRoutes := projectRoutes.Group(
		"/:id/speech",
	)

	speechRoutes.POST(
		"/transcribe",
		speechHandler.Transcribe,
	)

	commandRoutes := projectRoutes.Group(
		"/:id/commands",
	)

	commandRoutes.POST(
		"",
		commandHandler.Create,
	)

	commandRoutes.GET(
		"",
		commandHandler.List,
	)

	commandRoutes.POST(
		"/voice",
		speechHandler.CreateVoiceCommand,
	)

	commandByIDRoutes := authenticatedRoutes.Group(
		"/commands",
	)

	commandByIDRoutes.GET(
		"/:command_id",
		commandHandler.Get,
	)

	infrastructureRoutes := projectRoutes.Group(
		"/:id/infrastructure",
	)

	infrastructureRoutes.POST(
		"/parse",
		aiHandler.ParseInfrastructure,
	)

	infrastructureRoutes.POST(
		"/parse/voice",
		speechHandler.ParseVoiceInfrastructure,
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

	terraformRoutes := projectRoutes.Group(
		"/:id/terraform",
	)

	terraformRoutes.POST(
		"/generate",
		terraformHandler.Generate,
	)

	terraformRunRoutes := terraformRoutes.Group(
		"/runs",
	)

	terraformRunRoutes.POST(
		"",
		terraformExecutionHandler.Start,
	)

	terraformRunRoutes.GET(
		"",
		terraformExecutionHandler.List,
	)

	terraformRunRoutes.GET(
		"/:run_id",
		terraformExecutionHandler.Get,
	)

	return terraformExecutionService
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

		return ai.NewService(
			provider,
		)

	default:
		logger.Error(
			"unsupported AI provider configured",
			"provider",
			cfg.AI.Provider,
		)

		return ai.NewService(
			nil,
		)
	}
}

func newSpeechService(
	cfg config.Config,
	logger *slog.Logger,
) *speech.Service {
	maxUploadBytes := int64(
		cfg.Speech.MaxUploadMB,
	) * 1024 * 1024

	switch strings.ToLower(
		strings.TrimSpace(
			cfg.Speech.Provider,
		),
	) {
	case "whisper":
		provider := whisper.NewProvider(
			cfg.Speech.WhisperURL,
			cfg.Speech.Model,
			cfg.Speech.Language,
			nil,
			cfg.Speech.WhisperAPIKey,
		)

		return speech.NewService(
			provider,
			maxUploadBytes,
		)

	default:
		logger.Error(
			"unsupported speech provider configured",
			"provider",
			cfg.Speech.Provider,
		)

		return speech.NewService(
			nil,
			maxUploadBytes,
		)
	}
}
