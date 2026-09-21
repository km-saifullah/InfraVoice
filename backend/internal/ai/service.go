package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

type Service struct {
	provider Provider
}

func NewService(provider Provider) *Service {
	return &Service{
		provider: provider,
	}
}

func (s *Service) ParseInfrastructure(
	ctx context.Context,
	request ParseRequest,
) (ParseResult, error) {
	if s == nil || s.provider == nil {
		return ParseResult{}, ErrProviderNotConfigured
	}

	request.Command = strings.TrimSpace(request.Command)

	if request.Command == "" {
		return ParseResult{}, fmt.Errorf(
			"%w: command cannot be empty",
			ErrInvalidInput,
		)
	}

	result, err := s.provider.ParseInfrastructure(
		ctx,
		request,
	)
	if err != nil {
		return ParseResult{}, err
	}

	result.Clarification = strings.TrimSpace(
		result.Clarification,
	)

	if result.NeedsClarification {
		if result.Clarification == "" {
			return ParseResult{}, fmt.Errorf(
				"%w: provider did not specify what information is required",
				ErrInvalidProviderResponse,
			)
		}

		result.Specification = nil

		return result, nil
	}

	if result.Specification == nil {
		return ParseResult{}, fmt.Errorf(
			"%w: provider returned no infrastructure specification",
			ErrInvalidProviderResponse,
		)
	}

	result.Specification.Normalize()

	if err := infrastructure.Validate(
		result.Specification,
	); err != nil {
		return ParseResult{}, fmt.Errorf(
			"%w: %w",
			ErrInvalidProviderResponse,
			err,
		)
	}

	if err := infrastructure.ValidatePolicy(
		result.Specification,
	); err != nil {
		return ParseResult{}, fmt.Errorf(
			"%w: %w",
			ErrInvalidProviderResponse,
			err,
		)
	}

	return result, nil
}
