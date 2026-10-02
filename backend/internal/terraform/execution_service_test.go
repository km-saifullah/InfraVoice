package terraform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

type fakeSpecificationReader struct {
	mutex         sync.Mutex
	specification *infrastructure.InfrastructureSpec
	err           error
}

func (f *fakeSpecificationReader) Get(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	specID bson.ObjectID,
) (*infrastructure.InfrastructureSpec, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.err != nil {
		return nil, f.err
	}

	copied := *f.specification

	copied.EC2 = append(
		[]infrastructure.EC2Spec(nil),
		f.specification.EC2...,
	)

	return &copied, nil
}

func (f *fakeSpecificationReader) setInstanceType(
	instanceType string,
) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.specification.EC2[0].InstanceType = instanceType
}

type fakeProjectReader struct {
	err error
}

func (f *fakeProjectReader) Get(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
) (*project.Project, error) {
	if f.err != nil {
		return nil, f.err
	}

	return &project.Project{
		ID:      projectID,
		OwnerID: ownerID,
	}, nil
}

type memoryRunStore struct {
	mutex sync.Mutex
	runs  map[bson.ObjectID]Run
}

func newMemoryRunStore() *memoryRunStore {
	return &memoryRunStore{
		runs: make(map[bson.ObjectID]Run),
	}
}

func cloneRun(run Run) Run {
	run.Steps = append(
		[]Step(nil),
		run.Steps...,
	)

	return run
}

func (m *memoryRunStore) Create(
	ctx context.Context,
	run *Run,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.runs[run.ID] = cloneRun(*run)

	return nil
}

func (m *memoryRunStore) Update(
	ctx context.Context,
	run *Run,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.runs[run.ID]; !exists {
		return ErrRunNotFound
	}

	m.runs[run.ID] = cloneRun(*run)

	return nil
}

func (m *memoryRunStore) FindByIDAndProject(
	ctx context.Context,
	runID bson.ObjectID,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
) (*Run, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	run, exists := m.runs[runID]

	if !exists ||
		run.OwnerID != ownerID ||
		run.ProjectID != projectID {
		return nil, ErrRunNotFound
	}

	copied := cloneRun(run)

	return &copied, nil
}

func (m *memoryRunStore) ListByProject(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	limit int64,
) ([]Run, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	runs := make([]Run, 0)

	for _, run := range m.runs {
		if run.OwnerID == ownerID &&
			run.ProjectID == projectID {
			runs = append(runs, cloneRun(run))
		}
	}

	sort.Slice(
		runs,
		func(i int, j int) bool {
			return runs[i].CreatedAt.After(
				runs[j].CreatedAt,
			)
		},
	)

	if int64(len(runs)) > limit {
		runs = runs[:limit]
	}

	return runs, nil
}

func (m *memoryRunStore) MarkInterrupted(
	ctx context.Context,
	message string,
	at time.Time,
) (int64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var count int64

	for id, run := range m.runs {
		if run.Status != RunStatusQueued &&
			run.Status != RunStatusRunning {
			continue
		}

		run = cloneRun(run)
		run.Status = RunStatusFailed
		run.Error = message
		run.FinishedAt = &at

		m.runs[id] = run
		count++
	}

	return count, nil
}

func (m *memoryRunStore) seed(run Run) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.runs[run.ID] = cloneRun(run)
}

type scriptedExecutor struct {
	mutex    sync.Mutex
	calls    []Command
	checkErr error
	handler  func(command Command) (Result, error)
}

func (e *scriptedExecutor) Check() error {
	return e.checkErr
}

func (e *scriptedExecutor) Execute(
	ctx context.Context,
	command Command,
) (Result, error) {
	e.mutex.Lock()
	e.calls = append(e.calls, command)
	handler := e.handler
	e.mutex.Unlock()

	if handler == nil {
		handler = successfulTerraform
	}

	return handler(command)
}

func (e *scriptedExecutor) commandNames() []string {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	names := make([]string, 0, len(e.calls))

	for _, call := range e.calls {
		names = append(names, call.Args[0])
	}

	return names
}

// successfulTerraform emulates the parts of terraform that the
// service depends on: it writes a plan file for `plan -out` and
// prints the summary lines terraform prints.
func successfulTerraform(
	command Command,
) (Result, error) {
	switch command.Args[0] {
	case "plan":
		for _, argument := range command.Args {
			if name, found := strings.CutPrefix(
				argument,
				"-out=",
			); found {
				if err := os.WriteFile(
					filepath.Join(command.Dir, name),
					[]byte("plan-contents-"+time.Now().String()),
					0o600,
				); err != nil {
					return Result{ExitCode: 1}, err
				}
			}
		}

		return Result{
			Output: "Plan: 3 to add, 1 to change, 0 to destroy.",
		}, nil

	case "apply":
		return Result{
			Output: "Apply complete! Resources: 3 added, 1 changed, 0 destroyed.",
		}, nil

	case "destroy":
		return Result{
			Output: "Destroy complete! Resources: 4 destroyed.",
		}, nil

	default:
		return Result{
			Output: "ok",
		}, nil
	}
}

type serviceFixture struct {
	service   *ExecutionService
	executor  *scriptedExecutor
	store     *memoryRunStore
	specs     *fakeSpecificationReader
	projects  *fakeProjectReader
	workspace string
	ownerID   bson.ObjectID
	projectID bson.ObjectID
	specID    bson.ObjectID
}

func newServiceFixture(
	t *testing.T,
) *serviceFixture {
	t.Helper()

	specification := &infrastructure.InfrastructureSpec{
		Provider: infrastructure.ProviderAWS,
		Version:  1,
		VPCs: []infrastructure.VPCSpec{
			{
				Name:               "main-vpc",
				CIDR:               "10.0.0.0/16",
				EnableDNS:          true,
				EnableDNSHostnames: true,
				Subnets: []infrastructure.Subnet{
					{
						Name: "private-app",
						CIDR: "10.0.1.0/24",
						Type: "private",
					},
				},
			},
		},
		EC2: []infrastructure.EC2Spec{
			{
				Name:         "app-server",
				InstanceType: "t4g.micro",
				AMI:          "ami-1234567890abcdef0",
				Subnet:       "private-app",
				RootVolumeGB: 20,
				Count:        1,
			},
		},
	}

	fixture := &serviceFixture{
		executor:  &scriptedExecutor{},
		store:     newMemoryRunStore(),
		specs:     &fakeSpecificationReader{specification: specification},
		projects:  &fakeProjectReader{},
		workspace: t.TempDir(),
		ownerID:   bson.NewObjectID(),
		projectID: bson.NewObjectID(),
		specID:    bson.NewObjectID(),
	}

	fixture.service = NewExecutionService(
		config.TerraformConfig{
			BinaryPath:          "terraform",
			WorkspaceRoot:       fixture.workspace,
			ExecutionTimeoutSec: 30,
		},
		NewGenerator(),
		fixture.specs,
		fixture.projects,
		fixture.store,
		fixture.executor,
		nil,
	)

	return fixture
}

func (f *serviceFixture) start(
	t *testing.T,
	input StartInput,
) (*Run, error) {
	t.Helper()

	return f.service.Start(
		context.Background(),
		f.ownerID,
		f.projectID,
		input,
	)
}

func (f *serviceFixture) wait(
	t *testing.T,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	if err := f.service.Wait(ctx); err != nil {
		t.Fatalf(
			"runs did not finish: %v",
			err,
		)
	}
}

func (f *serviceFixture) finishedRun(
	t *testing.T,
	runID bson.ObjectID,
) *Run {
	t.Helper()

	f.wait(t)

	run, err := f.store.FindByIDAndProject(
		context.Background(),
		runID,
		f.ownerID,
		f.projectID,
	)

	if err != nil {
		t.Fatalf(
			"failed to load run: %v",
			err,
		)
	}

	return run
}

func (f *serviceFixture) planInput() StartInput {
	return StartInput{
		SpecificationID: f.specID,
		Region:          "ap-south-1",
		Type:            RunTypePlan,
	}
}

func (f *serviceFixture) workspaceDir() string {
	return filepath.Join(
		f.workspace,
		f.projectID.Hex(),
		f.specID.Hex(),
	)
}

func (f *serviceFixture) successfulPlan(
	t *testing.T,
) *Run {
	t.Helper()

	started, err := f.start(
		t,
		f.planInput(),
	)

	if err != nil {
		t.Fatalf(
			"plan Start() returned unexpected error: %v",
			err,
		)
	}

	plan := f.finishedRun(
		t,
		started.ID,
	)

	if plan.Status != RunStatusSucceeded {
		t.Fatalf(
			"plan run did not succeed: %s (%s)",
			plan.Status,
			plan.Error,
		)
	}

	return plan
}

func TestStartPlanRunsInitValidatePlan(t *testing.T) {
	fixture := newServiceFixture(t)

	started, err := fixture.start(
		t,
		fixture.planInput(),
	)

	if err != nil {
		t.Fatalf(
			"Start() returned unexpected error: %v",
			err,
		)
	}

	if started.Status != RunStatusQueued {
		t.Fatalf(
			"expected a queued snapshot, got %q",
			started.Status,
		)
	}

	run := fixture.finishedRun(
		t,
		started.ID,
	)

	if run.Status != RunStatusSucceeded {
		t.Fatalf(
			"expected succeeded, got %q (%s)",
			run.Status,
			run.Error,
		)
	}

	names := strings.Join(
		fixture.executor.commandNames(),
		",",
	)

	if names != "init,validate,plan" {
		t.Fatalf(
			"unexpected terraform command order: %s",
			names,
		)
	}

	if len(run.Steps) != 3 {
		t.Fatalf(
			"expected 3 recorded steps, got %d",
			len(run.Steps),
		)
	}

	for _, step := range run.Steps {
		if step.Status != StepStatusSucceeded ||
			step.FinishedAt == nil {
			t.Fatalf(
				"step %s is not finished: %+v",
				step.Name,
				step,
			)
		}
	}

	if run.Summary == nil ||
		run.Summary.Add != 3 ||
		run.Summary.Change != 1 ||
		run.Summary.Destroy != 0 {
		t.Fatalf(
			"unexpected change summary: %+v",
			run.Summary,
		)
	}

	if run.PlanChecksum == "" {
		t.Fatal("successful plan must record the plan checksum")
	}

	if run.StartedAt == nil ||
		run.FinishedAt == nil {
		t.Fatal("run timestamps were not recorded")
	}

	for _, name := range []string{
		FileProvider,
		FileVariables,
		FileVPC,
		FileEC2,
	} {
		if _, err := os.Stat(
			filepath.Join(
				fixture.workspaceDir(),
				name,
			),
		); err != nil {
			t.Fatalf(
				"workspace is missing %s: %v",
				name,
				err,
			)
		}
	}
}

func TestStartValidateRunDoesNotRequireAMIOrPlan(t *testing.T) {
	fixture := newServiceFixture(t)

	fixture.specs.specification.EC2[0].AMI = ""

	started, err := fixture.start(
		t,
		StartInput{
			SpecificationID: fixture.specID,
			Region:          "ap-south-1",
			Type:            RunTypeValidate,
		},
	)

	if err != nil {
		t.Fatalf(
			"validate Start() returned unexpected error: %v",
			err,
		)
	}

	run := fixture.finishedRun(
		t,
		started.ID,
	)

	if run.Status != RunStatusSucceeded {
		t.Fatalf(
			"validate run failed: %s",
			run.Error,
		)
	}

	if names := strings.Join(
		fixture.executor.commandNames(),
		",",
	); names != "init,validate" {
		t.Fatalf(
			"unexpected terraform command order: %s",
			names,
		)
	}

	if fixture.executor.calls[0].Args[len(
		fixture.executor.calls[0].Args,
	)-1] != "-backend=false" {
		t.Fatal("validate must initialize without a backend")
	}
}

func TestStartRejectsMissingAMIForPlan(t *testing.T) {
	fixture := newServiceFixture(t)

	fixture.specs.specification.EC2[0].AMI = ""

	_, err := fixture.start(
		t,
		fixture.planInput(),
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}

	if !strings.Contains(err.Error(), "ami") {
		t.Fatalf(
			"error should mention the missing ami: %v",
			err,
		)
	}

	if len(fixture.executor.calls) != 0 {
		t.Fatal("terraform must not run for an invalid specification")
	}
}

func TestStartRejectsInvalidInput(t *testing.T) {
	fixture := newServiceFixture(t)

	testCases := []struct {
		name  string
		input StartInput
	}{
		{
			name: "unknown type",
			input: StartInput{
				SpecificationID: fixture.specID,
				Region:          "ap-south-1",
				Type:            "refresh",
			},
		},
		{
			name: "missing specification",
			input: StartInput{
				Region: "ap-south-1",
				Type:   RunTypePlan,
			},
		},
		{
			name: "missing region",
			input: StartInput{
				SpecificationID: fixture.specID,
				Type:            RunTypePlan,
			},
		},
		{
			name: "apply without plan run",
			input: StartInput{
				Type:    RunTypeApply,
				Confirm: true,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				_, err := fixture.start(
					t,
					testCase.input,
				)

				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf(
						"expected ErrInvalidInput, got %v",
						err,
					)
				}
			},
		)
	}
}

func TestStartRequiresConfirmationForApplyAndDestroy(t *testing.T) {
	fixture := newServiceFixture(t)

	for _, runType := range []string{
		RunTypeApply,
		RunTypeDestroy,
	} {
		_, err := fixture.start(
			t,
			StartInput{
				SpecificationID: fixture.specID,
				Region:          "ap-south-1",
				Type:            runType,
				PlanRunID:       bson.NewObjectID(),
			},
		)

		if !errors.Is(err, ErrConfirmationRequired) {
			t.Fatalf(
				"%s: expected ErrConfirmationRequired, got %v",
				runType,
				err,
			)
		}
	}

	if len(fixture.executor.calls) != 0 {
		t.Fatal("terraform must not run without confirmation")
	}
}

func TestStartReportsTerraformUnavailable(t *testing.T) {
	fixture := newServiceFixture(t)

	fixture.executor.checkErr = ErrTerraformUnavailable

	_, err := fixture.start(
		t,
		fixture.planInput(),
	)

	if !errors.Is(err, ErrTerraformUnavailable) {
		t.Fatalf(
			"expected ErrTerraformUnavailable, got %v",
			err,
		)
	}
}

func TestApplyExecutesReviewedPlanOnce(t *testing.T) {
	fixture := newServiceFixture(t)

	plan := fixture.successfulPlan(t)

	started, err := fixture.start(
		t,
		StartInput{
			Type:      RunTypeApply,
			PlanRunID: plan.ID,
			Confirm:   true,
		},
	)

	if err != nil {
		t.Fatalf(
			"apply Start() returned unexpected error: %v",
			err,
		)
	}

	if started.SpecificationID != fixture.specID ||
		started.Region != "ap-south-1" {
		t.Fatal("apply must inherit specification and region from the plan")
	}

	apply := fixture.finishedRun(
		t,
		started.ID,
	)

	if apply.Status != RunStatusSucceeded {
		t.Fatalf(
			"apply failed: %s",
			apply.Error,
		)
	}

	if apply.PlanRunID == nil ||
		*apply.PlanRunID != plan.ID {
		t.Fatal("apply run must reference the plan run")
	}

	if apply.Summary == nil ||
		apply.Summary.Add != 3 {
		t.Fatalf(
			"unexpected apply summary: %+v",
			apply.Summary,
		)
	}

	names := fixture.executor.commandNames()

	if last := names[len(names)-1]; last != "apply" {
		t.Fatalf(
			"expected apply to be the last command, got %s",
			last,
		)
	}

	applyCall := fixture.executor.calls[len(names)-1]

	if applyCall.Args[len(applyCall.Args)-1] != PlanFileName {
		t.Fatal("apply must execute the saved plan file")
	}

	for _, argument := range applyCall.Args {
		if argument == "-auto-approve" {
			t.Fatal("apply of a saved plan must not use -auto-approve")
		}
	}

	if _, err := os.Stat(
		filepath.Join(
			fixture.workspaceDir(),
			PlanFileName,
		),
	); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("plan file must be removed after a successful apply")
	}

	_, err = fixture.start(
		t,
		StartInput{
			Type:      RunTypeApply,
			PlanRunID: plan.ID,
			Confirm:   true,
		},
	)

	if !errors.Is(err, ErrPlanStale) {
		t.Fatalf(
			"a plan must not be applied twice, got %v",
			err,
		)
	}
}

func TestApplyRejectsPlanAfterSpecificationChanged(t *testing.T) {
	fixture := newServiceFixture(t)

	plan := fixture.successfulPlan(t)

	fixture.specs.setInstanceType("t4g.small")

	_, err := fixture.start(
		t,
		StartInput{
			Type:      RunTypeApply,
			PlanRunID: plan.ID,
			Confirm:   true,
		},
	)

	if !errors.Is(err, ErrPlanStale) {
		t.Fatalf(
			"expected ErrPlanStale, got %v",
			err,
		)
	}
}

func TestApplyRejectsSupersededPlan(t *testing.T) {
	fixture := newServiceFixture(t)

	firstPlan := fixture.successfulPlan(t)

	// A newer plan for the same workspace overwrites the saved plan
	// file, so the older run must no longer be applicable.
	time.Sleep(2 * time.Millisecond)

	fixture.successfulPlan(t)

	_, err := fixture.start(
		t,
		StartInput{
			Type:      RunTypeApply,
			PlanRunID: firstPlan.ID,
			Confirm:   true,
		},
	)

	if !errors.Is(err, ErrPlanStale) {
		t.Fatalf(
			"expected ErrPlanStale, got %v",
			err,
		)
	}
}

func TestApplyRejectsInvalidPlanRuns(t *testing.T) {
	fixture := newServiceFixture(t)

	failedPlan := Run{
		ID:              bson.NewObjectID(),
		OwnerID:         fixture.ownerID,
		ProjectID:       fixture.projectID,
		SpecificationID: fixture.specID,
		Type:            RunTypePlan,
		Status:          RunStatusFailed,
		PlanChecksum:    "abc",
	}

	validateRun := Run{
		ID:              bson.NewObjectID(),
		OwnerID:         fixture.ownerID,
		ProjectID:       fixture.projectID,
		SpecificationID: fixture.specID,
		Type:            RunTypeValidate,
		Status:          RunStatusSucceeded,
		PlanChecksum:    "abc",
	}

	someoneElsesPlan := Run{
		ID:              bson.NewObjectID(),
		OwnerID:         bson.NewObjectID(),
		ProjectID:       fixture.projectID,
		SpecificationID: fixture.specID,
		Type:            RunTypePlan,
		Status:          RunStatusSucceeded,
		PlanChecksum:    "abc",
	}

	fixture.store.seed(failedPlan)
	fixture.store.seed(validateRun)
	fixture.store.seed(someoneElsesPlan)

	for name, planRunID := range map[string]bson.ObjectID{
		"failed plan":     failedPlan.ID,
		"not a plan":      validateRun.ID,
		"another owner":   someoneElsesPlan.ID,
		"unknown plan id": bson.NewObjectID(),
	} {
		_, err := fixture.start(
			t,
			StartInput{
				Type:      RunTypeApply,
				PlanRunID: planRunID,
				Confirm:   true,
			},
		)

		if !errors.Is(err, ErrInvalidPlan) {
			t.Fatalf(
				"%s: expected ErrInvalidPlan, got %v",
				name,
				err,
			)
		}
	}
}

func TestApplyRejectsMismatchedRegion(t *testing.T) {
	fixture := newServiceFixture(t)

	plan := fixture.successfulPlan(t)

	_, err := fixture.start(
		t,
		StartInput{
			Type:      RunTypeApply,
			PlanRunID: plan.ID,
			Region:    "us-east-1",
			Confirm:   true,
		},
	)

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf(
			"expected ErrInvalidInput, got %v",
			err,
		)
	}
}

func TestDestroyRunsInitAndDestroy(t *testing.T) {
	fixture := newServiceFixture(t)

	started, err := fixture.start(
		t,
		StartInput{
			SpecificationID: fixture.specID,
			Region:          "ap-south-1",
			Type:            RunTypeDestroy,
			Confirm:         true,
		},
	)

	if err != nil {
		t.Fatalf(
			"destroy Start() returned unexpected error: %v",
			err,
		)
	}

	run := fixture.finishedRun(
		t,
		started.ID,
	)

	if run.Status != RunStatusSucceeded {
		t.Fatalf(
			"destroy failed: %s",
			run.Error,
		)
	}

	if names := strings.Join(
		fixture.executor.commandNames(),
		",",
	); names != "init,destroy" {
		t.Fatalf(
			"unexpected terraform command order: %s",
			names,
		)
	}

	if run.Summary == nil ||
		run.Summary.Destroy != 4 {
		t.Fatalf(
			"unexpected destroy summary: %+v",
			run.Summary,
		)
	}
}

func TestFailedStepStopsRunAndKeepsOutput(t *testing.T) {
	fixture := newServiceFixture(t)

	fixture.executor.handler = func(
		command Command,
	) (Result, error) {
		if command.Args[0] == "validate" {
			return Result{
				Output:   "Error: Invalid resource type",
				ExitCode: 1,
			}, ErrExecutionFailed
		}

		return successfulTerraform(command)
	}

	started, err := fixture.start(
		t,
		fixture.planInput(),
	)

	if err != nil {
		t.Fatalf(
			"Start() returned unexpected error: %v",
			err,
		)
	}

	run := fixture.finishedRun(
		t,
		started.ID,
	)

	if run.Status != RunStatusFailed {
		t.Fatalf(
			"expected failed, got %q",
			run.Status,
		)
	}

	if names := strings.Join(
		fixture.executor.commandNames(),
		",",
	); names != "init,validate" {
		t.Fatalf(
			"plan must not run after a failed validate: %s",
			names,
		)
	}

	failedStep := run.Steps[len(run.Steps)-1]

	if failedStep.Status != StepStatusFailed ||
		!strings.Contains(
			failedStep.Output,
			"Invalid resource type",
		) {
		t.Fatalf(
			"failed step lost its output: %+v",
			failedStep,
		)
	}

	if !strings.Contains(run.Error, "validate") {
		t.Fatalf(
			"run error should name the failed step: %q",
			run.Error,
		)
	}

	if run.PlanChecksum != "" {
		t.Fatal("a failed plan must not be applicable")
	}
}

func TestFailureMessageDoesNotLeakUnexpectedErrors(t *testing.T) {
	fixture := newServiceFixture(t)

	message := fixture.service.failureMessage(
		StepPlan,
		errors.New("open /srv/secret/path: permission denied"),
	)

	if strings.Contains(message, "/srv/secret/path") {
		t.Fatalf(
			"internal error details leaked: %q",
			message,
		)
	}
}

func TestStartRejectsConcurrentRunsInSameWorkspace(t *testing.T) {
	fixture := newServiceFixture(t)

	release := make(chan struct{})
	entered := make(chan struct{})

	var once sync.Once

	fixture.executor.handler = func(
		command Command,
	) (Result, error) {
		once.Do(
			func() {
				close(entered)
			},
		)

		<-release

		return successfulTerraform(command)
	}

	if _, err := fixture.start(
		t,
		fixture.planInput(),
	); err != nil {
		t.Fatalf(
			"first Start() returned unexpected error: %v",
			err,
		)
	}

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first run never started executing")
	}

	_, err := fixture.start(
		t,
		fixture.planInput(),
	)

	if !errors.Is(err, ErrRunInProgress) {
		t.Fatalf(
			"expected ErrRunInProgress, got %v",
			err,
		)
	}

	close(release)

	fixture.wait(t)

	// The workspace lock must be released once the run is done.
	if _, err := fixture.start(
		t,
		fixture.planInput(),
	); err != nil {
		t.Fatalf(
			"Start() after completion returned unexpected error: %v",
			err,
		)
	}

	fixture.wait(t)
}

func TestPrepareWorkspaceRemovesStaleTerraformFiles(t *testing.T) {
	directory := t.TempDir()

	for name, content := range map[string]string{
		"old.tf":                "stale",
		"terraform.tfstate":     "state",
		".terraform.lock.hcl":   "lock",
		"notes.txt":             "keep",
		FileProvider:            "old provider",
		"terraform.tfvars.json": "keep",
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name),
			[]byte(content),
			0o600,
		); err != nil {
			t.Fatalf(
				"failed to seed %s: %v",
				name,
				err,
			)
		}
	}

	if err := prepareWorkspace(
		directory,
		Files{
			FileProvider: "new provider",
		},
	); err != nil {
		t.Fatalf(
			"prepareWorkspace() returned unexpected error: %v",
			err,
		)
	}

	if _, err := os.Stat(
		filepath.Join(directory, "old.tf"),
	); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("stale .tf file was not removed")
	}

	for _, name := range []string{
		"terraform.tfstate",
		".terraform.lock.hcl",
		"notes.txt",
		"terraform.tfvars.json",
	} {
		if _, err := os.Stat(
			filepath.Join(directory, name),
		); err != nil {
			t.Fatalf(
				"%s must be preserved: %v",
				name,
				err,
			)
		}
	}

	content, err := os.ReadFile(
		filepath.Join(directory, FileProvider),
	)

	if err != nil ||
		string(content) != "new provider" {
		t.Fatalf(
			"provider.tf was not rewritten: %q (%v)",
			content,
			err,
		)
	}
}

func TestPrepareWorkspaceRejectsUnsafeFileNames(t *testing.T) {
	for _, name := range []string{
		"../escape.tf",
		"nested/file.tf",
		"..",
	} {
		err := prepareWorkspace(
			t.TempDir(),
			Files{
				name: "content",
			},
		)

		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf(
				"%q: expected ErrInvalidInput, got %v",
				name,
				err,
			)
		}
	}
}

func TestRecoverInterruptedFailsUnfinishedRuns(t *testing.T) {
	fixture := newServiceFixture(t)

	running := Run{
		ID:        bson.NewObjectID(),
		OwnerID:   fixture.ownerID,
		ProjectID: fixture.projectID,
		Type:      RunTypeApply,
		Status:    RunStatusRunning,
	}

	finished := Run{
		ID:        bson.NewObjectID(),
		OwnerID:   fixture.ownerID,
		ProjectID: fixture.projectID,
		Type:      RunTypePlan,
		Status:    RunStatusSucceeded,
	}

	fixture.store.seed(running)
	fixture.store.seed(finished)

	if err := fixture.service.RecoverInterrupted(
		context.Background(),
	); err != nil {
		t.Fatalf(
			"RecoverInterrupted() returned unexpected error: %v",
			err,
		)
	}

	recovered, _ := fixture.store.FindByIDAndProject(
		context.Background(),
		running.ID,
		fixture.ownerID,
		fixture.projectID,
	)

	if recovered.Status != RunStatusFailed ||
		recovered.Error == "" {
		t.Fatalf(
			"running run was not failed: %+v",
			recovered,
		)
	}

	untouched, _ := fixture.store.FindByIDAndProject(
		context.Background(),
		finished.ID,
		fixture.ownerID,
		fixture.projectID,
	)

	if untouched.Status != RunStatusSucceeded {
		t.Fatal("finished runs must not be modified")
	}
}

func TestGetAndListAreScopedToOwnerAndProject(t *testing.T) {
	fixture := newServiceFixture(t)

	plan := fixture.successfulPlan(t)

	got, err := fixture.service.Get(
		context.Background(),
		fixture.ownerID,
		fixture.projectID,
		plan.ID,
	)

	if err != nil || got.ID != plan.ID {
		t.Fatalf(
			"Get() failed: %v",
			err,
		)
	}

	_, err = fixture.service.Get(
		context.Background(),
		bson.NewObjectID(),
		fixture.projectID,
		plan.ID,
	)

	if !errors.Is(err, ErrRunNotFound) {
		t.Fatalf(
			"another owner must not read the run, got %v",
			err,
		)
	}

	runs, err := fixture.service.List(
		context.Background(),
		fixture.ownerID,
		fixture.projectID,
	)

	if err != nil || len(runs) != 1 {
		t.Fatalf(
			"List() returned %d runs (%v)",
			len(runs),
			err,
		)
	}

	otherRuns, err := fixture.service.List(
		context.Background(),
		bson.NewObjectID(),
		fixture.projectID,
	)

	if err != nil || len(otherRuns) != 0 {
		t.Fatalf(
			"another owner must see no runs, got %d (%v)",
			len(otherRuns),
			err,
		)
	}
}

func TestGetPropagatesProjectErrors(t *testing.T) {
	fixture := newServiceFixture(t)

	fixture.projects.err = project.ErrProjectNotFound

	_, err := fixture.service.Get(
		context.Background(),
		fixture.ownerID,
		fixture.projectID,
		bson.NewObjectID(),
	)

	if !errors.Is(err, project.ErrProjectNotFound) {
		t.Fatalf(
			"expected ErrProjectNotFound, got %v",
			err,
		)
	}
}

func TestParseChangeSummary(t *testing.T) {
	testCases := []struct {
		name     string
		step     string
		output   string
		expected *ChangeSummary
	}{
		{
			name:     "plan with changes",
			step:     StepPlan,
			output:   "Terraform will perform...\n\nPlan: 5 to add, 2 to change, 1 to destroy.\n",
			expected: &ChangeSummary{Add: 5, Change: 2, Destroy: 1},
		},
		{
			name:     "plan with import",
			step:     StepPlan,
			output:   "Plan: 1 to import, 2 to add, 0 to change, 0 to destroy.",
			expected: &ChangeSummary{Add: 2},
		},
		{
			name:     "plan without changes",
			step:     StepPlan,
			output:   "\nNo changes. Your infrastructure matches the configuration.\n",
			expected: &ChangeSummary{},
		},
		{
			name:     "apply",
			step:     StepApply,
			output:   "Apply complete! Resources: 4 added, 0 changed, 1 destroyed.",
			expected: &ChangeSummary{Add: 4, Destroy: 1},
		},
		{
			name:     "destroy",
			step:     StepDestroy,
			output:   "Destroy complete! Resources: 7 destroyed.",
			expected: &ChangeSummary{Destroy: 7},
		},
		{
			name:     "no summary",
			step:     StepInit,
			output:   "Terraform has been successfully initialized!",
			expected: nil,
		},
		{
			name:     "plan output without a summary",
			step:     StepPlan,
			output:   "Error: something failed",
			expected: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(
			testCase.name,
			func(t *testing.T) {
				actual := ParseChangeSummary(
					testCase.step,
					testCase.output,
				)

				switch {
				case testCase.expected == nil && actual != nil:
					t.Fatalf(
						"expected no summary, got %+v",
						actual,
					)

				case testCase.expected != nil && actual == nil:
					t.Fatalf(
						"expected %+v, got nil",
						testCase.expected,
					)

				case testCase.expected != nil &&
					*actual != *testCase.expected:
					t.Fatalf(
						"expected %+v, got %+v",
						testCase.expected,
						actual,
					)
				}
			},
		)
	}
}

func TestHashFilesIsDeterministicAndContentSensitive(t *testing.T) {
	first := Files{
		"a.tf": "one",
		"b.tf": "two",
	}

	same := Files{
		"b.tf": "two",
		"a.tf": "one",
	}

	changed := Files{
		"a.tf": "one",
		"b.tf": "three",
	}

	if hashFiles(first) != hashFiles(same) {
		t.Fatal("hash must not depend on map order")
	}

	if hashFiles(first) == hashFiles(changed) {
		t.Fatal("hash must change when content changes")
	}
}
