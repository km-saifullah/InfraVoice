package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/http/middleware"
	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

type ProjectHandler struct {
	service *project.Service
}

func NewProjectHandler(
	service *project.Service,
) *ProjectHandler {
	return &ProjectHandler{
		service: service,
	}
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

func (h *ProjectHandler) Create(c *gin.Context) {
	ownerID, ok := authenticatedUserID(c)

	if !ok {
		return
	}

	var request createProjectRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid request body",
		)
		return
	}

	newProject, err := h.service.Create(
		c.Request.Context(),
		ownerID,
		project.CreateInput{
			Name:        request.Name,
			Description: request.Description,
		},
	)

	if err != nil {
		if errors.Is(err, project.ErrInvalidInput) {
			response.BadRequest(
				c,
				err.Error(),
			)
			return
		}

		response.InternalServerError(
			c,
			"failed to create project",
		)
		return
	}

	response.Created(
		c,
		gin.H{
			"project": newProject,
		},
	)
}

func (h *ProjectHandler) List(c *gin.Context) {
	ownerID, ok := authenticatedUserID(c)

	if !ok {
		return
	}

	projects, err := h.service.List(
		c.Request.Context(),
		ownerID,
	)

	if err != nil {
		if errors.Is(err, project.ErrInvalidInput) {
			response.BadRequest(
				c,
				err.Error(),
			)
			return
		}

		response.InternalServerError(
			c,
			"failed to load projects",
		)
		return
	}

	response.OK(
		c,
		gin.H{
			"projects": projects,
		},
	)
}

func (h *ProjectHandler) Get(c *gin.Context) {
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

	currentProject, err := h.service.Get(
		c.Request.Context(),
		ownerID,
		projectID,
	)

	if err != nil {
		if errors.Is(err, project.ErrInvalidInput) {
			response.BadRequest(
				c,
				err.Error(),
			)
			return
		}

		if errors.Is(err, project.ErrProjectNotFound) {
			response.NotFound(
				c,
				"project not found",
			)
			return
		}

		response.InternalServerError(
			c,
			"failed to load project",
		)
		return
	}

	response.OK(
		c,
		gin.H{
			"project": currentProject,
		},
	)
}

func (h *ProjectHandler) Update(c *gin.Context) {
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

	var request updateProjectRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid request body",
		)
		return
	}

	updatedProject, err := h.service.Update(
		c.Request.Context(),
		ownerID,
		projectID,
		project.UpdateInput{
			Name:        request.Name,
			Description: request.Description,
			Status:      request.Status,
		},
	)

	if err != nil {
		switch {
		case errors.Is(err, project.ErrInvalidInput):
			response.BadRequest(
				c,
				err.Error(),
			)

		case errors.Is(err, project.ErrProjectNotFound):
			response.NotFound(
				c,
				"project not found",
			)

		default:
			response.InternalServerError(
				c,
				"failed to update project",
			)
		}

		return
	}

	response.OK(
		c,
		gin.H{
			"project": updatedProject,
		},
	)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
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

	if err := h.service.Delete(
		c.Request.Context(),
		ownerID,
		projectID,
	); err != nil {
		switch {
		case errors.Is(err, project.ErrInvalidInput):
			response.BadRequest(
				c,
				err.Error(),
			)

		case errors.Is(err, project.ErrProjectNotFound):
			response.NotFound(
				c,
				"project not found",
			)

		default:
			response.InternalServerError(
				c,
				"failed to delete project",
			)
		}

		return
	}

	response.JSON(
		c,
		http.StatusOK,
		gin.H{
			"message": "project deleted successfully",
		},
	)
}

func authenticatedUserID(
	c *gin.Context,
) (bson.ObjectID, bool) {
	userIDString, exists := middleware.UserID(c)

	if !exists {
		response.Unauthorized(
			c,
			"authenticated user not found",
		)
		return bson.NilObjectID, false
	}

	userID, err := bson.ObjectIDFromHex(
		userIDString,
	)

	if err != nil {
		response.Unauthorized(
			c,
			"invalid authenticated user",
		)
		return bson.NilObjectID, false
	}

	return userID, true
}

func parseProjectID(
	c *gin.Context,
) (bson.ObjectID, error) {
	projectID := c.Param("id")

	return bson.ObjectIDFromHex(projectID)
}
