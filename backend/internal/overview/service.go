package overview

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/command"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
	"github.com/km-saifullah/infra-voice/backend/internal/terraform"
)

var ErrInvalidInput = errors.New("invalid overview input")

type ProjectCounter interface {
	CountStatusesByOwner(
		ctx context.Context,
		ownerID bson.ObjectID,
	) (project.StatusCounts, error)

	ListIDsByOwner(
		ctx context.Context,
		ownerID bson.ObjectID,
	) ([]bson.ObjectID, error)
}

// CommandCounter is satisfied by *command.Repository.
type CommandCounter interface {
	CountByUser(
		ctx context.Context,
		userID bson.ObjectID,
	) (command.Counts, error)
}

// InfrastructureCounter is satisfied by *infrastructure.Repository.
type InfrastructureCounter interface {
	AggregateResourceCounts(
		ctx context.Context,
		projectIDs []bson.ObjectID,
	) (infrastructure.ResourceCounts, error)
}

// TerraformRunCounter is satisfied by *terraform.RunRepository.
type TerraformRunCounter interface {
	CountByOwner(
		ctx context.Context,
		ownerID bson.ObjectID,
	) (terraform.RunCounts, error)
}

type Service struct {
	projects       ProjectCounter
	commands       CommandCounter
	infrastructure InfrastructureCounter
	terraformRuns  TerraformRunCounter
}

func NewService(
	projects ProjectCounter,
	commands CommandCounter,
	infrastructureCounter InfrastructureCounter,
	terraformRuns TerraformRunCounter,
) *Service {
	return &Service{
		projects:       projects,
		commands:       commands,
		infrastructure: infrastructureCounter,
		terraformRuns:  terraformRuns,
	}
}

func (s *Service) Get(
	ctx context.Context,
	ownerID bson.ObjectID,
) (*Response, error) {
	if ownerID.IsZero() {
		return nil, fmt.Errorf(
			"%w: owner is required",
			ErrInvalidInput,
		)
	}

	projectCounts, err := s.projects.CountStatusesByOwner(
		ctx,
		ownerID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to count projects: %w",
			err,
		)
	}

	projectIDs, err := s.projects.ListIDsByOwner(
		ctx,
		ownerID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to list project ids: %w",
			err,
		)
	}

	resourceCounts, err := s.infrastructure.AggregateResourceCounts(
		ctx,
		projectIDs,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to aggregate infrastructure resources: %w",
			err,
		)
	}

	commandCounts, err := s.commands.CountByUser(
		ctx,
		ownerID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to count commands: %w",
			err,
		)
	}

	runCounts, err := s.terraformRuns.CountByOwner(
		ctx,
		ownerID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to count terraform runs: %w",
			err,
		)
	}

	return &Response{
		Projects: ProjectsOverview{
			Total:    projectCounts.Total,
			Active:   projectCounts.Active,
			Archived: projectCounts.Archived,
		},
		Infrastructure: InfrastructureOverview{
			Specifications: resourceCounts.Specifications,
			Resources: ResourcesByType{
				VPC: resourceCounts.VPC,
				EC2: resourceCounts.EC2,
				S3:  resourceCounts.S3,
				SNS: resourceCounts.SNS,
			},
		},
		Commands: CommandsOverview{
			Total:    commandCounts.Total,
			ByStatus: nonNilCounts(commandCounts.ByStatus),
			BySource: nonNilCounts(commandCounts.BySource),
		},
		Terraform: TerraformOverview{
			Total:    runCounts.Total,
			ByStatus: nonNilCounts(runCounts.ByStatus),
			ByType:   nonNilCounts(runCounts.ByType),
			ByRegion: nonNilCounts(runCounts.ByRegion),
		},
	}, nil
}

func nonNilCounts(
	counts map[string]int64,
) map[string]int64 {
	if counts == nil {
		return map[string]int64{}
	}

	return counts
}
