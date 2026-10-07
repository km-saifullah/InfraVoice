package overview

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/command"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
	"github.com/km-saifullah/infra-voice/backend/internal/terraform"
)

type fakeProjectCounter struct {
	statusCounts project.StatusCounts
	statusErr    error
	ids          []bson.ObjectID
	idsErr       error

	statusCalledWith bson.ObjectID
	idsCalledWith    bson.ObjectID
	idsCalled        bool
}

func (f *fakeProjectCounter) CountStatusesByOwner(
	ctx context.Context,
	ownerID bson.ObjectID,
) (project.StatusCounts, error) {
	f.statusCalledWith = ownerID

	if f.statusErr != nil {
		return project.StatusCounts{}, f.statusErr
	}

	return f.statusCounts, nil
}

func (f *fakeProjectCounter) ListIDsByOwner(
	ctx context.Context,
	ownerID bson.ObjectID,
) ([]bson.ObjectID, error) {
	f.idsCalled = true
	f.idsCalledWith = ownerID

	if f.idsErr != nil {
		return nil, f.idsErr
	}

	return f.ids, nil
}

type fakeCommandCounter struct {
	counts     command.Counts
	err        error
	calledWith bson.ObjectID
	called     bool
}

func (f *fakeCommandCounter) CountByUser(
	ctx context.Context,
	userID bson.ObjectID,
) (command.Counts, error) {
	f.called = true
	f.calledWith = userID

	if f.err != nil {
		return command.Counts{}, f.err
	}

	return f.counts, nil
}

type fakeInfrastructureCounter struct {
	counts     infrastructure.ResourceCounts
	err        error
	calledWith []bson.ObjectID
	called     bool
}

func (f *fakeInfrastructureCounter) AggregateResourceCounts(
	ctx context.Context,
	projectIDs []bson.ObjectID,
) (infrastructure.ResourceCounts, error) {
	f.called = true
	f.calledWith = projectIDs

	if f.err != nil {
		return infrastructure.ResourceCounts{}, f.err
	}

	return f.counts, nil
}

type fakeTerraformRunCounter struct {
	counts     terraform.RunCounts
	err        error
	calledWith bson.ObjectID
	called     bool
}

func (f *fakeTerraformRunCounter) CountByOwner(
	ctx context.Context,
	ownerID bson.ObjectID,
) (terraform.RunCounts, error) {
	f.called = true
	f.calledWith = ownerID

	if f.err != nil {
		return terraform.RunCounts{}, f.err
	}

	return f.counts, nil
}

type fixture struct {
	projects       *fakeProjectCounter
	commands       *fakeCommandCounter
	infrastructure *fakeInfrastructureCounter
	terraformRuns  *fakeTerraformRunCounter
	service        *Service
}

func newFixture() *fixture {
	f := &fixture{
		projects:       &fakeProjectCounter{},
		commands:       &fakeCommandCounter{},
		infrastructure: &fakeInfrastructureCounter{},
		terraformRuns:  &fakeTerraformRunCounter{},
	}

	f.service = NewService(
		f.projects,
		f.commands,
		f.infrastructure,
		f.terraformRuns,
	)

	return f
}

func TestGetRejectsZeroOwner(t *testing.T) {
	f := newFixture()

	_, err := f.service.Get(
		context.Background(),
		bson.ObjectID{},
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestGetComputesCountsAndArchivedBySubtraction(t *testing.T) {
	f := newFixture()
	ownerID := bson.NewObjectID()
	projectA := bson.NewObjectID()
	projectB := bson.NewObjectID()

	f.projects.statusCounts = project.StatusCounts{
		Total:    5,
		Active:   3,
		Archived: 2,
	}
	f.projects.ids = []bson.ObjectID{projectA, projectB}

	f.infrastructure.counts = infrastructure.ResourceCounts{
		Specifications: 4,
		VPC:            2,
		EC2:            6,
		S3:             1,
		SNS:            0,
	}

	f.commands.counts = command.Counts{
		Total: 10,
		ByStatus: map[string]int64{
			"received": 10,
		},
		BySource: map[string]int64{
			"text":  7,
			"voice": 3,
		},
	}

	f.terraformRuns.counts = terraform.RunCounts{
		Total: 6,
		ByStatus: map[string]int64{
			"succeeded": 4,
			"failed":    2,
		},
		ByType: map[string]int64{
			"plan":  3,
			"apply": 3,
		},
		ByRegion: map[string]int64{
			"us-east-1": 6,
		},
	}

	result, err := f.service.Get(
		context.Background(),
		ownerID,
	)

	if err != nil {
		t.Fatalf(
			"Get() returned unexpected error: %v",
			err,
		)
	}

	if result.Projects.Total != 5 ||
		result.Projects.Active != 3 ||
		result.Projects.Archived != 2 {
		t.Fatalf(
			"unexpected project counts: %+v",
			result.Projects,
		)
	}

	if result.Infrastructure.Specifications != 4 ||
		result.Infrastructure.Resources.EC2 != 6 {
		t.Fatalf(
			"unexpected infrastructure counts: %+v",
			result.Infrastructure,
		)
	}

	if result.Commands.Total != 10 ||
		result.Commands.ByStatus["received"] != 10 ||
		result.Commands.BySource["voice"] != 3 {
		t.Fatalf(
			"unexpected command counts: %+v",
			result.Commands,
		)
	}

	if result.Terraform.Total != 6 ||
		result.Terraform.ByStatus["succeeded"] != 4 ||
		result.Terraform.ByType["apply"] != 3 ||
		result.Terraform.ByRegion["us-east-1"] != 6 {
		t.Fatalf(
			"unexpected terraform counts: %+v",
			result.Terraform,
		)
	}
}

func TestGetThreadsOwnerIDToEveryDependencyForIsolation(t *testing.T) {
	f := newFixture()
	ownerID := bson.NewObjectID()
	projectIDs := []bson.ObjectID{bson.NewObjectID()}
	f.projects.ids = projectIDs

	if _, err := f.service.Get(
		context.Background(),
		ownerID,
	); err != nil {
		t.Fatalf(
			"Get() returned unexpected error: %v",
			err,
		)
	}

	if f.projects.statusCalledWith != ownerID {
		t.Fatalf(
			"project status count was not scoped to the caller: got %s",
			f.projects.statusCalledWith.Hex(),
		)
	}

	if f.projects.idsCalledWith != ownerID {
		t.Fatalf(
			"project id listing was not scoped to the caller: got %s",
			f.projects.idsCalledWith.Hex(),
		)
	}

	if f.commands.calledWith != ownerID {
		t.Fatalf(
			"command count was not scoped to the caller: got %s",
			f.commands.calledWith.Hex(),
		)
	}

	if f.terraformRuns.calledWith != ownerID {
		t.Fatalf(
			"terraform run count was not scoped to the caller: got %s",
			f.terraformRuns.calledWith.Hex(),
		)
	}

	if len(f.infrastructure.calledWith) != 1 ||
		f.infrastructure.calledWith[0] != projectIDs[0] {
		t.Fatalf(
			"infrastructure aggregation was not scoped to the caller's own project ids: got %v",
			f.infrastructure.calledWith,
		)
	}
}

func TestGetReturnsZeroedResponseForUserWithNoData(t *testing.T) {
	f := newFixture()

	result, err := f.service.Get(
		context.Background(),
		bson.NewObjectID(),
	)

	if err != nil {
		t.Fatalf(
			"Get() returned unexpected error: %v",
			err,
		)
	}

	if result.Projects.Total != 0 ||
		result.Infrastructure.Specifications != 0 ||
		result.Commands.Total != 0 ||
		result.Terraform.Total != 0 {
		t.Fatalf(
			"expected an all-zero response, got %+v",
			result,
		)
	}

	if result.Commands.ByStatus == nil ||
		result.Commands.BySource == nil ||
		result.Terraform.ByStatus == nil ||
		result.Terraform.ByType == nil ||
		result.Terraform.ByRegion == nil {
		t.Fatal("empty breakdowns must be non-nil maps")
	}

	if !f.infrastructure.called {
		t.Fatal("infrastructure aggregation should still be called, with an empty project id list")
	}

	if len(f.infrastructure.calledWith) != 0 {
		t.Fatalf(
			"expected an empty project id list, got %v",
			f.infrastructure.calledWith,
		)
	}
}

func TestGetPropagatesProjectStatusCountError(t *testing.T) {
	f := newFixture()
	sentinel := errors.New("mongo: no reachable servers")
	f.projects.statusErr = sentinel

	_, err := f.service.Get(
		context.Background(),
		bson.NewObjectID(),
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf(
			"expected the underlying error to propagate, got %v",
			err,
		)
	}

	if f.commands.called || f.terraformRuns.called || f.infrastructure.called {
		t.Fatal("later dependencies must not be queried after an earlier failure")
	}
}

func TestGetPropagatesProjectIDListError(t *testing.T) {
	f := newFixture()
	sentinel := errors.New("mongo: cursor closed")
	f.projects.idsErr = sentinel

	_, err := f.service.Get(
		context.Background(),
		bson.NewObjectID(),
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf(
			"expected the underlying error to propagate, got %v",
			err,
		)
	}

	if f.infrastructure.called || f.commands.called || f.terraformRuns.called {
		t.Fatal("later dependencies must not be queried after an earlier failure")
	}
}

func TestGetPropagatesInfrastructureAggregationError(t *testing.T) {
	f := newFixture()
	f.projects.ids = []bson.ObjectID{bson.NewObjectID()}
	sentinel := errors.New("mongo: aggregation failed")
	f.infrastructure.err = sentinel

	_, err := f.service.Get(
		context.Background(),
		bson.NewObjectID(),
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf(
			"expected the underlying error to propagate, got %v",
			err,
		)
	}

	if f.commands.called || f.terraformRuns.called {
		t.Fatal("later dependencies must not be queried after an earlier failure")
	}
}

func TestGetPropagatesCommandCountError(t *testing.T) {
	f := newFixture()
	sentinel := errors.New("mongo: write conflict")
	f.commands.err = sentinel

	_, err := f.service.Get(
		context.Background(),
		bson.NewObjectID(),
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf(
			"expected the underlying error to propagate, got %v",
			err,
		)
	}

	if f.terraformRuns.called {
		t.Fatal("later dependencies must not be queried after an earlier failure")
	}
}

func TestGetPropagatesTerraformRunCountError(t *testing.T) {
	f := newFixture()
	sentinel := errors.New("mongo: connection reset")
	f.terraformRuns.err = sentinel

	_, err := f.service.Get(
		context.Background(),
		bson.NewObjectID(),
	)

	if !errors.Is(err, sentinel) {
		t.Fatalf(
			"expected the underlying error to propagate, got %v",
			err,
		)
	}
}
