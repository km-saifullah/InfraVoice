package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/km-saifullah/infra-voice/backend/internal/http/response"
	"github.com/km-saifullah/infra-voice/backend/internal/overview"
)

type OverviewHandler struct {
	service *overview.Service
}

func NewOverviewHandler(
	service *overview.Service,
) *OverviewHandler {
	return &OverviewHandler{
		service: service,
	}
}

func (h *OverviewHandler) Get(c *gin.Context) {
	ownerID, ok := authenticatedUserID(c)

	if !ok {
		return
	}

	result, err := h.service.Get(
		c.Request.Context(),
		ownerID,
	)

	if err != nil {
		response.InternalServerError(
			c,
			"failed to load overview",
		)
		return
	}

	response.OK(
		c,
		gin.H{
			"overview": result,
		},
	)
}
