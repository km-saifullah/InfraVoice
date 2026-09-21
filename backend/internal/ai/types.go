package ai

import (
	"context"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

type ParseRequest struct {
	Command string
}

type ParseResult struct {
	Specification      *infrastructure.InfrastructureSpec
	NeedsClarification bool
	Clarification      string
}

type Provider interface {
	Name() string

	ParseInfrastructure(
		ctx context.Context,
		request ParseRequest,
	) (ParseResult, error)
}
