package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

var (
	ErrInvalidInput = errors.New(
		"invalid infrastructure input",
	)

	ErrInvalidProject = errors.New(
		"invalid project",
	)
)

type Service struct {
	repository     *Repository
	projectService *project.Service
}

func NewService(
	repository *Repository,
	projectService *project.Service,
) *Service {
	return &Service{
		repository:     repository,
		projectService: projectService,
	}
}

func (s *Service) Create(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	spec InfrastructureSpec,
) (*InfrastructureSpec, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf(
			"%w: project is required",
			ErrInvalidProject,
		)
	}

	if _, err := s.projectService.Get(
		ctx,
		ownerID,
		projectID,
	); err != nil {
		return nil, err
	}

	spec.ProjectID = projectID

	if spec.ID.IsZero() {
		spec.ID = bson.NewObjectID()
	}

	if spec.Version == 0 {
		spec.Version = 1
	}

	spec.Normalize()

	if err := Validate(&spec); err != nil {
		return nil, err
	}

	if err := ValidatePolicy(&spec); err != nil {
		return nil, err
	}

	now := time.Now()

	spec.CreatedAt = now
	spec.UpdatedAt = now

	if err := s.repository.Create(
		ctx,
		&spec,
	); err != nil {
		return nil, err
	}

	return &spec, nil
}

func (s *Service) List(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
) ([]InfrastructureSpec, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf(
			"%w: project is required",
			ErrInvalidProject,
		)
	}

	if _, err := s.projectService.Get(
		ctx,
		ownerID,
		projectID,
	); err != nil {
		return nil, err
	}

	return s.repository.ListByProject(
		ctx,
		projectID,
	)
}

func (s *Service) Get(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	specID bson.ObjectID,
) (*InfrastructureSpec, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf(
			"%w: project is required",
			ErrInvalidProject,
		)
	}

	if specID.IsZero() {
		return nil, fmt.Errorf(
			"%w: specification is required",
			ErrInvalidInput,
		)
	}

	if _, err := s.projectService.Get(
		ctx,
		ownerID,
		projectID,
	); err != nil {
		return nil, err
	}

	return s.repository.FindByIDAndProject(
		ctx,
		specID,
		projectID,
	)
}

func (s *Service) Update(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	specID bson.ObjectID,
	spec InfrastructureSpec,
) (*InfrastructureSpec, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf(
			"%w: project is required",
			ErrInvalidProject,
		)
	}

	if specID.IsZero() {
		return nil, fmt.Errorf(
			"%w: specification is required",
			ErrInvalidInput,
		)
	}

	if _, err := s.projectService.Get(
		ctx,
		ownerID,
		projectID,
	); err != nil {
		return nil, err
	}

	spec.ID = specID
	spec.ProjectID = projectID

	if spec.Version == 0 {
		spec.Version = 1
	}

	spec.Normalize()

	if err := Validate(&spec); err != nil {
		return nil, err
	}

	if err := ValidatePolicy(&spec); err != nil {
		return nil, err
	}

	update := bson.D{
		bson.E{
			Key:   "provider",
			Value: spec.Provider,
		},
		bson.E{
			Key:   "version",
			Value: spec.Version,
		},
		bson.E{
			Key:   "vpcs",
			Value: spec.VPCs,
		},
		bson.E{
			Key:   "s3",
			Value: spec.S3,
		},
		bson.E{
			Key:   "ec2",
			Value: spec.EC2,
		},
		bson.E{
			Key:   "sns",
			Value: spec.SNS,
		},
		bson.E{
			Key:   "updated_at",
			Value: time.Now(),
		},
	}

	return s.repository.UpdateByIDAndProject(
		ctx,
		specID,
		projectID,
		update,
	)
}

func (s *Service) Delete(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	specID bson.ObjectID,
) error {
	if ownerID.IsZero() {
		return fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return fmt.Errorf(
			"%w: project is required",
			ErrInvalidProject,
		)
	}

	if specID.IsZero() {
		return fmt.Errorf(
			"%w: specification is required",
			ErrInvalidInput,
		)
	}

	if _, err := s.projectService.Get(
		ctx,
		ownerID,
		projectID,
	); err != nil {
		return err
	}

	return s.repository.DeleteByIDAndProject(
		ctx,
		specID,
		projectID,
	)
}

func (s *Service) Validate(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	specID bson.ObjectID,
) (*InfrastructureSpec, error) {
	spec, err := s.Get(
		ctx,
		ownerID,
		projectID,
		specID,
	)
	if err != nil {
		return nil, err
	}

	if err := Validate(spec); err != nil {
		return nil, err
	}

	if err := ValidatePolicy(spec); err != nil {
		return nil, err
	}

	return spec, nil
}
