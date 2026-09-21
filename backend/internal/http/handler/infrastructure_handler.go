package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

type InfrastructureHandler struct {
	service *infrastructure.Service
}

func NewInfrastructureHandler(
	service *infrastructure.Service,
) *InfrastructureHandler {
	return &InfrastructureHandler{
		service: service,
	}
}

func (h *InfrastructureHandler) Create(c *gin.Context) {
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

	var spec infrastructure.InfrastructureSpec

	if err := c.ShouldBindJSON(&spec); err != nil {
		response.BadRequest(
			c,
			"invalid infrastructure specification",
		)
		return
	}

	createdSpec, err := h.service.Create(
		c.Request.Context(),
		ownerID,
		projectID,
		spec,
	)

	if err != nil {
		handleInfrastructureError(c, err)
		return
	}

	response.Created(
		c,
		gin.H{
			"specification": createdSpec,
		},
	)
}

func (h *InfrastructureHandler) List(c *gin.Context) {
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

	specifications, err := h.service.List(
		c.Request.Context(),
		ownerID,
		projectID,
	)

	if err != nil {
		handleInfrastructureError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"specifications": specifications,
		},
	)
}

func (h *InfrastructureHandler) Get(c *gin.Context) {
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

	specID, err := parseSpecificationID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid specification id",
		)
		return
	}

	spec, err := h.service.Get(
		c.Request.Context(),
		ownerID,
		projectID,
		specID,
	)

	if err != nil {
		handleInfrastructureError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"specification": spec,
		},
	)
}

func (h *InfrastructureHandler) Update(c *gin.Context) {
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

	specID, err := parseSpecificationID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid specification id",
		)
		return
	}

	var spec infrastructure.InfrastructureSpec

	if err := c.ShouldBindJSON(&spec); err != nil {
		response.BadRequest(
			c,
			"invalid infrastructure specification",
		)
		return
	}

	updatedSpec, err := h.service.Update(
		c.Request.Context(),
		ownerID,
		projectID,
		specID,
		spec,
	)

	if err != nil {
		handleInfrastructureError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"specification": updatedSpec,
		},
	)
}

func (h *InfrastructureHandler) Delete(c *gin.Context) {
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

	specID, err := parseSpecificationID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid specification id",
		)
		return
	}

	if err := h.service.Delete(
		c.Request.Context(),
		ownerID,
		projectID,
		specID,
	); err != nil {
		handleInfrastructureError(c, err)
		return
	}

	response.JSON(
		c,
		http.StatusOK,
		gin.H{
			"message": "infrastructure specification deleted successfully",
		},
	)
}

func (h *InfrastructureHandler) Validate(c *gin.Context) {
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

	specID, err := parseSpecificationID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid specification id",
		)
		return
	}

	spec, err := h.service.Validate(
		c.Request.Context(),
		ownerID,
		projectID,
		specID,
	)

	if err != nil {
		handleInfrastructureError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"valid":         true,
			"specification": spec,
		},
	)
}

func parseSpecificationID(
	c *gin.Context,
) (bson.ObjectID, error) {
	specificationID := c.Param("spec_id")

	return bson.ObjectIDFromHex(
		specificationID,
	)
}

func handleInfrastructureError(
	c *gin.Context,
	err error,
) {
	var validationErrors *infrastructure.ValidationErrors
	var policyErrors *infrastructure.PolicyErrors

	switch {
	case errors.As(err, &validationErrors):
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			response.CodeBadRequest,
			"infrastructure specification validation failed",
			validationErrors.Errors,
		)

	case errors.As(err, &policyErrors):
		response.ErrorWithDetails(
			c,
			http.StatusBadRequest,
			response.CodeBadRequest,
			"infrastructure policy validation failed",
			policyErrors.Errors,
		)

	case errors.Is(
		err,
		infrastructure.ErrInvalidInput,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		infrastructure.ErrSpecificationNotFound,
	):
		response.NotFound(
			c,
			"infrastructure specification not found",
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
		infrastructure.ErrInvalidProject,
	):
		response.BadRequest(
			c,
			err.Error(),
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
			"failed to process infrastructure specification",
		)
	}
}
