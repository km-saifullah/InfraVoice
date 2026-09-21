package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/command"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

type CommandHandler struct {
	service *command.Service
}

func NewCommandHandler(
	service *command.Service,
) *CommandHandler {
	return &CommandHandler{
		service: service,
	}
}

type createCommandRequest struct {
	Input  string `json:"input"`
	Source string `json:"source"`
}

func (h *CommandHandler) Create(c *gin.Context) {
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

	var request createCommandRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid request body",
		)
		return
	}

	createdCommand, err := h.service.Create(
		c.Request.Context(),
		userID,
		projectID,
		command.CreateInput{
			Input:  request.Input,
			Source: request.Source,
		},
	)

	if err != nil {
		handleCommandError(c, err)
		return
	}

	response.Created(
		c,
		gin.H{
			"command": createdCommand,
		},
	)
}

func (h *CommandHandler) List(c *gin.Context) {
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

	commands, err := h.service.List(
		c.Request.Context(),
		userID,
		projectID,
	)

	if err != nil {
		handleCommandError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"commands": commands,
		},
	)
}

func (h *CommandHandler) Get(c *gin.Context) {
	userID, ok := authenticatedUserID(c)

	if !ok {
		return
	}

	commandID, err := parseCommandID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid command id",
		)
		return
	}

	currentCommand, err := h.service.Get(
		c.Request.Context(),
		userID,
		commandID,
	)

	if err != nil {
		handleCommandError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"command": currentCommand,
		},
	)
}

func parseCommandID(
	c *gin.Context,
) (bson.ObjectID, error) {
	commandID := c.Param("command_id")

	return bson.ObjectIDFromHex(commandID)
}

func handleCommandError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		command.ErrInvalidInput,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		command.ErrCommandNotFound,
	):
		response.NotFound(
			c,
			"command not found",
		)

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
			"failed to process command",
		)
	}
}
