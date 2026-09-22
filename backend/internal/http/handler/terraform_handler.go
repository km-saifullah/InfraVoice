package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
	"github.com/km-saifullah/infra-voice/backend/internal/terraform"
)

type TerraformHandler struct {
	service *terraform.Service
}

func NewTerraformHandler(
	service *terraform.Service,
) *TerraformHandler {
	return &TerraformHandler{
		service: service,
	}
}

type generateTerraformRequest struct {
	SpecificationID string `json:"specification_id"`
	Region          string `json:"region"`
}

func (h *TerraformHandler) Generate(c *gin.Context) {
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

	var request generateTerraformRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid terraform generation request",
		)
		return
	}

	specificationID, err := parseTerraformSpecificationID(
		request.SpecificationID,
	)

	if err != nil {
		response.BadRequest(
			c,
			"invalid specification id",
		)
		return
	}

	files, err := h.service.Generate(
		c.Request.Context(),
		ownerID,
		projectID,
		specificationID,
		terraform.GenerateRequest{
			Region: request.Region,
		},
	)

	if err != nil {
		handleTerraformError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"files": files,
		},
	)
}

func parseTerraformSpecificationID(
	value string,
) (bson.ObjectID, error) {
	return bson.ObjectIDFromHex(value)
}

func handleTerraformError(
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
		terraform.ErrInvalidInput,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		terraform.ErrUnsupportedProvider,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		terraform.ErrAmbiguousSubnet,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		terraform.ErrSubnetNotFound,
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
			"failed to generate terraform configuration",
		)
	}
}
