package handler

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
	redisdb "github.com/km-saifullah/infra-voice/backend/internal/database/redis"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
)

type HealthHandler struct {
	config        config.Config
	mongoDatabase *mongodb.Database
	redisClient   *redisdb.Client
	httpClient    *http.Client
}

func NewHealthHandler(
	cfg config.Config,
	mongoDatabase *mongodb.Database,
	redisClient *redisdb.Client,
) *HealthHandler {
	return &HealthHandler{
		config:        cfg,
		mongoDatabase: mongoDatabase,
		redisClient:   redisClient,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(
		c.Request.Context(),
		2*time.Second,
	)
	defer cancel()

	mongoStatus := "connected"

	if err := h.mongoDatabase.Client().Ping(ctx); err != nil {
		mongoStatus = "disconnected"
	}

	redisStatus := "connected"

	if err := h.redisClient.Ping(ctx); err != nil {
		redisStatus = "disconnected"
	}

	// Ollama and Whisper are checked too, so "is my voice pipeline
	// even reachable" is a single GET /health instead of trial and
	// error through the UI. They are informational only: an
	// unreachable AI/speech provider degrades those two features but
	// not the rest of the API, so it never flips the overall status.
	var waitGroup sync.WaitGroup

	var aiStatus, speechStatus string

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		aiStatus = h.pingProvider(ctx, h.config.AI.OllamaURL)
	}()

	go func() {
		defer waitGroup.Done()
		speechStatus = h.pingProvider(ctx, h.config.Speech.WhisperURL)
	}()

	waitGroup.Wait()

	statusCode := http.StatusOK
	overallStatus := "ok"

	if mongoStatus != "connected" ||
		redisStatus != "connected" {
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
				"ollama":  aiStatus,
				"whisper": speechStatus,
			},
		},
	)
}

// pingProvider reports whether something is listening at baseURL.
// Any HTTP response at all (even a 404) counts as "reachable" -- the
// point is distinguishing "nothing is running there" from "something
// is running there, and the rest of the pipeline can take it from
// here"; it does not validate the provider's actual API contract.
func (h *HealthHandler) pingProvider(
	ctx context.Context,
	baseURL string,
) string {
	baseURL = strings.TrimSpace(baseURL)

	if baseURL == "" {
		return "not configured"
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		baseURL,
		nil,
	)

	if err != nil {
		return "unreachable"
	}

	response, err := h.httpClient.Do(request)

	if err != nil {
		return "unreachable"
	}

	defer response.Body.Close()

	return "reachable"
}
