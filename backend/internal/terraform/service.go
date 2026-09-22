package terraform

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
)

type Service struct {
	generator             *Generator
	infrastructureService *infrastructure.Service
}

func NewService(
	generator *Generator,
	infrastructureService *infrastructure.Service,
) *Service {
	return &Service{
		generator:             generator,
		infrastructureService: infrastructureService,
	}
}

func (s *Service) Generate(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	specificationID bson.ObjectID,
	request GenerateRequest,
) (Files, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf(
			"%w: project is required",
			ErrInvalidInput,
		)
	}

	if specificationID.IsZero() {
		return nil, fmt.Errorf(
			"%w: specification is required",
			ErrInvalidInput,
		)
	}

	if s.generator == nil {
		return nil, errors.New(
			"terraform generator is not initialized",
		)
	}

	if s.infrastructureService == nil {
		return nil, errors.New(
			"infrastructure service is not initialized",
		)
	}

	specification, err := s.infrastructureService.Get(
		ctx,
		ownerID,
		projectID,
		specificationID,
	)
	if err != nil {
		return nil, err
	}

	files, err := s.generator.Generate(
		*specification,
		request,
	)
	if err != nil {
		return nil, err
	}

	return files, nil
}
