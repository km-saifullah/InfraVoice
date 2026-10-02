package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/terraform"
)

type TerraformExecutionHandler struct {
	service *terraform.ExecutionService
}

func NewTerraformExecutionHandler(
	service *terraform.ExecutionService,
) *TerraformExecutionHandler {
	return &TerraformExecutionHandler{
		service: service,
	}
}

type startTerraformRunRequest struct {
	SpecificationID string `json:"specification_id"`
	Region          string `json:"region"`
	Type            string `json:"type"`
	PlanRunID       string `json:"plan_run_id"`
	Confirm         bool   `json:"confirm"`
}

// Start begins a terraform run (validate, plan, apply or destroy) in
// the background and returns the initial, queued snapshot of it.
// Callers poll Get for progress and the final result.
func (h *TerraformExecutionHandler) Start(c *gin.Context) {
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

	var request startTerraformRunRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(
			c,
			"invalid terraform run request",
		)
		return
	}

	input := terraform.StartInput{
		Region:  request.Region,
		Type:    request.Type,
		Confirm: request.Confirm,
	}

	if request.SpecificationID != "" {
		specificationID, err := bson.ObjectIDFromHex(
			request.SpecificationID,
		)

		if err != nil {
			response.BadRequest(
				c,
				"invalid specification id",
			)
			return
		}

		input.SpecificationID = specificationID
	}

	if request.PlanRunID != "" {
		planRunID, err := bson.ObjectIDFromHex(
			request.PlanRunID,
		)

		if err != nil {
			response.BadRequest(
				c,
				"invalid plan_run_id",
			)
			return
		}

		input.PlanRunID = planRunID
	}

	run, err := h.service.Start(
		c.Request.Context(),
		ownerID,
		projectID,
		input,
	)

	if err != nil {
		handleTerraformExecutionError(c, err)
		return
	}

	response.Created(
		c,
		gin.H{
			"run": run,
		},
	)
}

// Get returns the current state of a terraform run, including its
// per-step output. Poll this endpoint while a run's status is
// "queued" or "running".
func (h *TerraformExecutionHandler) Get(c *gin.Context) {
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

	runID, err := parseTerraformRunID(c)

	if err != nil {
		response.BadRequest(
			c,
			"invalid run id",
		)
		return
	}

	run, err := h.service.Get(
		c.Request.Context(),
		ownerID,
		projectID,
		runID,
	)

	if err != nil {
		handleTerraformExecutionError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"run": run,
		},
	)
}

// List returns the most recent terraform runs for a project, newest
// first.
func (h *TerraformExecutionHandler) List(c *gin.Context) {
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

	runs, err := h.service.List(
		c.Request.Context(),
		ownerID,
		projectID,
	)

	if err != nil {
		handleTerraformExecutionError(c, err)
		return
	}

	response.OK(
		c,
		gin.H{
			"runs": runs,
		},
	)
}

func parseTerraformRunID(
	c *gin.Context,
) (bson.ObjectID, error) {
	return bson.ObjectIDFromHex(
		c.Param("run_id"),
	)
}

func handleTerraformExecutionError(
	c *gin.Context,
	err error,
) {
	switch {
	case errors.Is(
		err,
		terraform.ErrConfirmationRequired,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		terraform.ErrInvalidPlan,
	):
		response.BadRequest(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		terraform.ErrRunInProgress,
	):
		response.Conflict(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		terraform.ErrPlanStale,
	):
		response.Conflict(
			c,
			err.Error(),
		)

	case errors.Is(
		err,
		terraform.ErrRunNotFound,
	):
		response.NotFound(
			c,
			"terraform run not found",
		)

	case errors.Is(
		err,
		terraform.ErrTerraformUnavailable,
	):
		response.ServiceUnavailable(
			c,
			"terraform is not available on the server",
		)

	default:
		handleTerraformError(c, err)
	}
}
