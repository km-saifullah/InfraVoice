package project

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	repository *Repository
}

type CreateInput struct {
	Name        string
	Description string
}

type UpdateInput struct {
	Name        *string
	Description *string
	Status      *string
}

func NewService(
	repository *Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(
	ctx context.Context,
	ownerID bson.ObjectID,
	input CreateInput,
) (*Project, error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)

	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	if name == "" {
		return nil, fmt.Errorf(
			"%w: project name is required",
			ErrInvalidInput,
		)
	}

	if len(name) > 100 {
		return nil, fmt.Errorf(
			"%w: project name must not exceed 100 characters",
			ErrInvalidInput,
		)
	}

	if len(description) > 500 {
		return nil, fmt.Errorf(
			"%w: project description must not exceed 500 characters",
			ErrInvalidInput,
		)
	}

	now := time.Now()

	newProject := &Project{
		ID:          bson.NewObjectID(),
		OwnerID:     ownerID,
		Name:        name,
		Description: description,
		Status:      StatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repository.Create(
		ctx,
		newProject,
	); err != nil {
		return nil, err
	}

	return newProject, nil
}

func (s *Service) List(
	ctx context.Context,
	ownerID bson.ObjectID,
) ([]Project, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	return s.repository.ListByOwner(
		ctx,
		ownerID,
	)
}

func (s *Service) Get(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
) (*Project, error) {
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

	return s.repository.FindByIDAndOwner(
		ctx,
		projectID,
		ownerID,
	)
}

func (s *Service) Update(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	input UpdateInput,
) (*Project, error) {
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

	update := bson.D{}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)

		if name == "" {
			return nil, fmt.Errorf(
				"%w: project name cannot be empty",
				ErrInvalidInput,
			)
		}

		if len(name) > 100 {
			return nil, fmt.Errorf(
				"%w: project name must not exceed 100 characters",
				ErrInvalidInput,
			)
		}

		update = append(
			update,
			bson.E{
				Key:   "name",
				Value: name,
			},
		)
	}

	if input.Description != nil {
		description := strings.TrimSpace(
			*input.Description,
		)

		if len(description) > 500 {
			return nil, fmt.Errorf(
				"%w: project description must not exceed 500 characters",
				ErrInvalidInput,
			)
		}

		update = append(
			update,
			bson.E{
				Key:   "description",
				Value: description,
			},
		)
	}

	if input.Status != nil {
		status := strings.TrimSpace(
			*input.Status,
		)

		if status != StatusActive &&
			status != StatusArchived {
			return nil, fmt.Errorf(
				"%w: invalid project status",
				ErrInvalidInput,
			)
		}

		update = append(
			update,
			bson.E{
				Key:   "status",
				Value: status,
			},
		)
	}

	if len(update) == 0 {
		return nil, fmt.Errorf(
			"%w: no fields to update",
			ErrInvalidInput,
		)
	}

	update = append(
		update,
		bson.E{
			Key:   "updated_at",
			Value: time.Now(),
		},
	)

	return s.repository.UpdateByIDAndOwner(
		ctx,
		projectID,
		ownerID,
		update,
	)
}

func (s *Service) Delete(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
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
			ErrInvalidInput,
		)
	}

	return s.repository.DeleteByIDAndOwner(
		ctx,
		projectID,
		ownerID,
	)
}
