package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/ai"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

type AIHandler struct {
	service        *ai.Service
	projectService *project.Service
}

func NewAIHandler(
	service *ai.Service,
	projectService *project.Service,
) *AIHandler {
	return &AIHandler{
		service:        service,
		projectService: projectService,
	}
}

type parseInfrastructureRequest struct {
	Command string `json:"command"`
}

func (h *AIHandler) ParseInfrastructure(
	c *gin.Context,
) {
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

	if h.projectService == nil {
		response.InternalServerError(
			c,
			"project service is not initialized",
		)
		return
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

		return
	}

	var request parseInfrastructureRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid AI parse request",
		)
		return
	}

	result, err := h.service.ParseInfrastructure(
		c.Request.Context(),
		ai.ParseRequest{
			Command: request.Command,
		},
	)

	if err != nil {
		handleAIError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"needs_clarification": result.NeedsClarification,
			"clarification":       result.Clarification,
			"specification":       result.Specification,
		},
	)
}

func handleAIError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		ai.ErrInvalidInput,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		ai.ErrProviderNotConfigured,
	):
		response.ServiceUnavailable(
			c,
			"AI provider is not configured",
		)

	case errors.Is(
		err,
		ai.ErrProviderUnavailable,
	):
		response.ServiceUnavailable(
			c,
			"AI provider is unavailable",
		)

	case errors.Is(
		err,
		ai.ErrInvalidProviderResponse,
	):
		response.JSON(
			c,
			http.StatusUnprocessableEntity,
			gin.H{
				"message": "AI provider returned an invalid infrastructure response",
			},
		)

	default:
		response.InternalServerError(
			c,
			"failed to parse infrastructure request",
		)
	}
}
