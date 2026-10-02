package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/ai"
	"github.com/km-saifullah/infra-voice/backend/internal/command"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
	"github.com/km-saifullah/infra-voice/backend/internal/speech"
)

const audioFormFieldName = "audio"

// SpeechHandler exposes voice input on top of the existing text
// pipeline. It never changes what command.Service or ai.Service do —
// it only transcribes audio to text and hands that text to them
// exactly as a text client already would.
type SpeechHandler struct {
	speechService  *speech.Service
	commandService *command.Service
	aiService      *ai.Service
	projectService *project.Service
}

func NewSpeechHandler(
	speechService *speech.Service,
	commandService *command.Service,
	aiService *ai.Service,
	projectService *project.Service,
) *SpeechHandler {
	return &SpeechHandler{
		speechService:  speechService,
		commandService: commandService,
		aiService:      aiService,
		projectService: projectService,
	}
}

// Transcribe converts an uploaded audio clip to text only. It is
// useful for a "preview before you submit" UI, and for verifying the
// speech provider independently of anything else.
func (h *SpeechHandler) Transcribe(c *gin.Context) {
	ownerID, ok := authenticatedUserID(c)

	if !ok {
		return
	}

	projectID, err := parseProjectID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid project id",
		)
		return
	}

	if !h.ensureProjectOwnership(
		c,
		ownerID,
		projectID,
	) {
		return
	}

	result, ok := h.transcribeUpload(c)

	if !ok {
		return
	}

	response.OK(
		c,
		gin.H{
			"text":     result.Text,
			"language": result.Language,
		},
	)
}

// CreateVoiceCommand transcribes an uploaded audio clip and logs it
// through the existing command.Service, exactly as
// POST /commands {source:"voice"} already does for typed text.
func (h *SpeechHandler) CreateVoiceCommand(c *gin.Context) {
	userID, ok := authenticatedUserID(c)

	if !ok {
		return
	}

	projectID, err := parseProjectID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid project id",
		)
		return
	}

	if !h.ensureProjectOwnership(
		c,
		userID,
		projectID,
	) {
		return
	}

	if h.commandService == nil {
		response.InternalServerError(
			c,
			"command service is not initialized",
		)
		return
	}

	result, ok := h.transcribeUpload(c)

	if !ok {
		return
	}

	createdCommand, err := h.commandService.Create(
		c.Request.Context(),
		userID,
		projectID,
		command.CreateInput{
			Input:  result.Text,
			Source: command.SourceVoice,
		},
	)

	if err != nil {
		handleCommandError(c, err)
		return
	}

	response.Created(
		c,
		gin.H{
			"command":    createdCommand,
			"transcript": result.Text,
		},
	)
}

// ParseVoiceInfrastructure transcribes an uploaded audio clip and
// runs it through the existing ai.Service, exactly as
// POST /infrastructure/parse {command:"..."} already does for typed
// text. The result is not persisted; save it via
// POST /infrastructure once the caller accepts it.
func (h *SpeechHandler) ParseVoiceInfrastructure(c *gin.Context) {
	ownerID, ok := authenticatedUserID(c)

	if !ok {
		return
	}

	projectID, err := parseProjectID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid project id",
		)
		return
	}

	if !h.ensureProjectOwnership(
		c,
		ownerID,
		projectID,
	) {
		return
	}

	if h.aiService == nil {
		response.ServiceUnavailable(
			c,
			"AI provider is not configured",
		)
		return
	}

	result, ok := h.transcribeUpload(c)

	if !ok {
		return
	}

	parseResult, err := h.aiService.ParseInfrastructure(
		c.Request.Context(),
		ai.ParseRequest{
			Command: result.Text,
		},
	)

	if err != nil {
		handleAIError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"transcript":          result.Text,
			"needs_clarification": parseResult.NeedsClarification,
			"clarification":       parseResult.Clarification,
			"specification":       parseResult.Specification,
		},
	)
}

// transcribeUpload reads the "audio" multipart field, enforces the
// configured upload size limit, and transcribes it. On failure it
// writes the HTTP response itself and returns ok=false.
func (h *SpeechHandler) transcribeUpload(
	c *gin.Context,
) (speech.TranscribeResult, bool) {
	if h.speechService == nil {
		response.ServiceUnavailable(
			c,
			"speech-to-text is not configured on this server",
		)
		return speech.TranscribeResult{}, false
	}

	maxUploadBytes := h.speechService.MaxUploadBytes()

	// A little headroom above the audio limit itself for the other
	// multipart fields and boundaries in the same request body.
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxUploadBytes+64*1024,
	)

	fileHeader, err := c.FormFile(audioFormFieldName)

	if err != nil {
		if isRequestTooLargeError(err) {
			response.PayloadTooLarge(
				c,
				fmt.Sprintf(
					"audio file exceeds the %d MB limit",
					maxUploadBytes/(1024*1024),
				),
			)
		} else {
			response.BadRequest(
				c,
				`audio file is required as multipart form field "audio"`,
			)
		}

		return speech.TranscribeResult{}, false
	}

	file, err := fileHeader.Open()

	if err != nil {
		response.BadRequest(
			c,
			"failed to read uploaded audio file",
		)
		return speech.TranscribeResult{}, false
	}

	defer file.Close()

	result, err := h.speechService.Transcribe(
		c.Request.Context(),
		speech.TranscribeRequest{
			Reader:      file,
			Filename:    fileHeader.Filename,
			ContentType: fileHeader.Header.Get("Content-Type"),
			SizeBytes:   fileHeader.Size,
			Language: strings.TrimSpace(
				c.PostForm("language"),
			),
		},
	)

	if err != nil {
		handleSpeechError(c, err)
		return speech.TranscribeResult{}, false
	}

	return result, true
}

// ensureProjectOwnership is checked before transcription (not just
// after, inside the downstream service) so a request for a project
// the caller does not own fails fast, without spending a speech
// provider call on audio that will be rejected anyway.
func (h *SpeechHandler) ensureProjectOwnership(
	c *gin.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
) bool {
	if h.projectService == nil {
		response.InternalServerError(
			c,
			"project service is not initialized",
		)
		return false
	}

	if _, err := h.projectService.Get(
		c.Request.Context(),
		ownerID,
		projectID,
	); err != nil {
		switch {
		case errors.Is(
			err,
			project.ErrProjectNotFound,
		):
			response.NotFound(
				c,
				"project not found",
			)

		case errors.Is(
			err,
			project.ErrInvalidInput,
		):
			response.BadRequest(
				c,
				err.Error(),
			)

		default:
			response.InternalServerError(
				c,
				"failed to load project",
			)
		}

		return false
	}

	return true
}

func isRequestTooLargeError(
	err error,
) bool {
	if err == nil {
		return false
	}

	var maxBytesError *http.MaxBytesError

	if errors.As(err, &maxBytesError) {
		return true
	}

	return strings.Contains(
		err.Error(),
		"request body too large",
	)
}

func handleSpeechError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		speech.ErrInvalidInput,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		speech.ErrUnsupportedFormat,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		speech.ErrFileTooLarge,
	):
		response.PayloadTooLarge(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		speech.ErrProviderNotConfigured,
	):
		response.ServiceUnavailable(
			c,
			"speech-to-text provider is not configured",
		)

	case errors.Is(
		err,
		speech.ErrProviderUnavailable,
	):
		response.ServiceUnavailable(
			c,
			"speech-to-text provider is unavailable",
		)

	case errors.Is(
		err,
		speech.ErrInvalidProviderResponse,
	):
		response.JSON(
			c,
			http.StatusUnprocessableEntity,
			gin.H{
				"message": err.Error(),
			},
		)

	default:
		response.InternalServerError(
			c,
			"failed to transcribe audio",
		)
	}
}
