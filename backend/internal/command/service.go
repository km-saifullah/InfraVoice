package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

var ErrInvalidInput = errors.New("invalid command input")

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
	userID bson.ObjectID,
	projectID bson.ObjectID,
	input CreateInput,
) (*Command, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf(
			"%w: user is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf(
			"%w: project is required",
			ErrInvalidInput,
		)
	}

	if _, err := s.projectService.Get(
		ctx,
		userID,
		projectID,
	); err != nil {
		return nil, err
	}

	source := input.Source

	if source == "" {
		source = SourceText
	}

	now := time.Now()

	newCommand := &Command{
		ID:        bson.NewObjectID(),
		ProjectID: projectID,
		UserID:    userID,
		Input:     input.Input,
		Source:    source,
		Status:    StatusReceived,
		CreatedAt: now,
		UpdatedAt: now,
	}

	newCommand.Normalize()

	if err := newCommand.Validate(); err != nil {
		return nil, fmt.Errorf(
			"%w: %s",
			ErrInvalidInput,
			err,
		)
	}

	if err := s.repository.Create(
		ctx,
		newCommand,
	); err != nil {
		return nil, err
	}

	return newCommand, nil
}

func (s *Service) List(
	ctx context.Context,
	userID bson.ObjectID,
	projectID bson.ObjectID,
) ([]Command, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf(
			"%w: user is required",
			ErrInvalidInput,
		)
	}

	if projectID.IsZero() {
		return nil, fmt.Errorf(
			"%w: project is required",
			ErrInvalidInput,
		)
	}

	if _, err := s.projectService.Get(
		ctx,
		userID,
		projectID,
	); err != nil {
		return nil, err
	}

	return s.repository.ListByProjectAndUser(
		ctx,
		projectID,
		userID,
	)
}

func (s *Service) Get(
	ctx context.Context,
	userID bson.ObjectID,
	commandID bson.ObjectID,
) (*Command, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf(
			"%w: user is required",
			ErrInvalidInput,
		)
	}

	if commandID.IsZero() {
		return nil, fmt.Errorf(
			"%w: command is required",
			ErrInvalidInput,
		)
	}

	return s.repository.FindByIDAndUser(
		ctx,
		commandID,
		userID,
	)
}
