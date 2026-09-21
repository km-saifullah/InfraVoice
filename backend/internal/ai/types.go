package ai

import (
	"context"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

type ParseRequest struct {
	Command string
}

type ParseResult struct {
	Specification      *infrastructure.InfrastructureSpec `json:"specification,omitempty"`
	NeedsClarification bool                               `json:"needs_clarification"`
	Clarification      string                             `json:"clarification,omitempty"`
}

type Provider interface {
	Name() string

	ParseInfrastructure(
		ctx context.Context,
		request ParseRequest,
	) (ParseResult, error)
}
