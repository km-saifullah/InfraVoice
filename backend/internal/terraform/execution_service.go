package terraform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
	"github.com/km-saifullah/infra-voice/backend/internal/infrastructure"
	"github.com/km-saifullah/infra-voice/backend/internal/project"
)

const (
	// MaxListedRuns bounds the number of runs returned by List.
	MaxListedRuns = 50

	persistTimeout = 10 * time.Second

	defaultExecutionTimeout = 300 * time.Second

	interruptedRunMessage = "terraform run was interrupted by a server restart"
)

// SpecificationReader is satisfied by *infrastructure.Service.
type SpecificationReader interface {
	Get(
		ctx context.Context,
		ownerID bson.ObjectID,
		projectID bson.ObjectID,
		specID bson.ObjectID,
	) (*infrastructure.InfrastructureSpec, error)
}

// ProjectReader is satisfied by *project.Service.
type ProjectReader interface {
	Get(
		ctx context.Context,
		ownerID bson.ObjectID,
		projectID bson.ObjectID,
	) (*project.Project, error)
}

// RunStore is satisfied by *RunRepository.
type RunStore interface {
	Create(
		ctx context.Context,
		run *Run,
	) error

	Update(
		ctx context.Context,
		run *Run,
	) error

	FindByIDAndProject(
		ctx context.Context,
		runID bson.ObjectID,
		ownerID bson.ObjectID,
		projectID bson.ObjectID,
	) (*Run, error)

	ListByProject(
		ctx context.Context,
		ownerID bson.ObjectID,
		projectID bson.ObjectID,
		limit int64,
	) ([]Run, error)

	MarkInterrupted(
		ctx context.Context,
		message string,
		at time.Time,
	) (int64, error)
}

type ExecutionService struct {
	config    config.TerraformConfig
	generator *Generator
	specs     SpecificationReader
	projects  ProjectReader
	store     RunStore
	executor  Executor
	logger    *slog.Logger
	locks     *workspaceLocks
	running   sync.WaitGroup
}

func NewExecutionService(
	cfg config.TerraformConfig,
	generator *Generator,
	specs SpecificationReader,
	projects ProjectReader,
	store RunStore,
	executor Executor,
	logger *slog.Logger,
) *ExecutionService {
	if logger == nil {
		logger = slog.Default()
	}

	return &ExecutionService{
		config:    cfg,
		generator: generator,
		specs:     specs,
		projects:  projects,
		store:     store,
		executor:  executor,
		logger:    logger,
		locks:     newWorkspaceLocks(),
	}
}

// Start validates the request, records a queued run and executes it
// in the background. The returned run is a snapshot; poll Get for
// progress.
func (s *ExecutionService) Start(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	input StartInput,
) (*Run, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}

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

	runType := NormalizeRunType(input.Type)

	if !IsValidRunType(runType) {
		return nil, fmt.Errorf(
			"%w: type must be one of validate, plan, apply or destroy",
			ErrInvalidInput,
		)
	}

	if RequiresConfirmation(runType) &&
		!input.Confirm {
		return nil, fmt.Errorf(
			"%w: set confirm to true to run terraform %s",
			ErrConfirmationRequired,
			runType,
		)
	}

	if err := s.executor.Check(); err != nil {
		return nil, err
	}

	specificationID := input.SpecificationID
	region := strings.TrimSpace(input.Region)

	var planRun *Run

	if runType == RunTypeApply {
		if input.PlanRunID.IsZero() {
			return nil, fmt.Errorf(
				"%w: plan_run_id is required to apply",
				ErrInvalidInput,
			)
		}

		loadedPlan, err := s.loadPlanRun(
			ctx,
			ownerID,
			projectID,
			input.PlanRunID,
		)

		if err != nil {
			return nil, err
		}

		planRun = loadedPlan

		if !specificationID.IsZero() &&
			specificationID != planRun.SpecificationID {
			return nil, fmt.Errorf(
				"%w: specification_id does not match the plan",
				ErrInvalidInput,
			)
		}

		if region != "" &&
			region != planRun.Region {
			return nil, fmt.Errorf(
				"%w: region does not match the plan",
				ErrInvalidInput,
			)
		}

		specificationID = planRun.SpecificationID
		region = planRun.Region
	} else {
		if specificationID.IsZero() {
			return nil, fmt.Errorf(
				"%w: specification_id is required",
				ErrInvalidInput,
			)
		}

		if region == "" {
			return nil, fmt.Errorf(
				"%w: AWS region is required",
				ErrInvalidInput,
			)
		}
	}

	specification, err := s.specs.Get(
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
		GenerateRequest{
			Region: region,
		},
	)

	if err != nil {
		return nil, err
	}

	if runType != RunTypeValidate {
		if err := requireEC2AMIs(
			*specification,
		); err != nil {
			return nil, err
		}
	}

	configHash := hashFiles(files)

	if planRun != nil &&
		planRun.ConfigHash != configHash {
		return nil, fmt.Errorf(
			"%w: the specification changed after the plan was created; run a new plan",
			ErrPlanStale,
		)
	}

	directory, err := s.workspacePath(
		projectID,
		specificationID,
	)

	if err != nil {
		return nil, err
	}

	if !s.locks.TryAcquire(directory) {
		return nil, ErrRunInProgress
	}

	handedOff := false

	defer func() {
		if !handedOff {
			s.locks.Release(directory)
		}
	}()

	if planRun != nil {
		if err := verifyPlanFile(
			directory,
			planRun.PlanChecksum,
		); err != nil {
			return nil, err
		}
	}

	now := time.Now()

	run := &Run{
		ID:              bson.NewObjectID(),
		OwnerID:         ownerID,
		ProjectID:       projectID,
		SpecificationID: specificationID,
		Type:            runType,
		Status:          RunStatusQueued,
		Region:          region,
		ConfigHash:      configHash,
		Steps:           []Step{},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if planRun != nil {
		planRunID := planRun.ID
		run.PlanRunID = &planRunID
	}

	if err := s.store.Create(
		ctx,
		run,
	); err != nil {
		return nil, err
	}

	handedOff = true

	// The snapshot must be taken before the goroutine starts, which
	// takes ownership of run.
	snapshot := *run

	s.running.Add(1)

	go s.execute(
		run,
		files,
		directory,
	)

	return &snapshot, nil
}

func (s *ExecutionService) Get(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	runID bson.ObjectID,
) (*Run, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}

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

	if runID.IsZero() {
		return nil, fmt.Errorf(
			"%w: run is required",
			ErrInvalidInput,
		)
	}

	if _, err := s.projects.Get(
		ctx,
		ownerID,
		projectID,
	); err != nil {
		return nil, err
	}

	return s.store.FindByIDAndProject(
		ctx,
		runID,
		ownerID,
		projectID,
	)
}

func (s *ExecutionService) List(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
) ([]Run, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}

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

	if _, err := s.projects.Get(
		ctx,
		ownerID,
		projectID,
	); err != nil {
		return nil, err
	}

	return s.store.ListByProject(
		ctx,
		ownerID,
		projectID,
		MaxListedRuns,
	)
}

// RecoverInterrupted fails runs left queued or running by a previous
// process. Call it once at startup, before serving requests. It
// assumes a single backend instance executes terraform runs.
func (s *ExecutionService) RecoverInterrupted(
	ctx context.Context,
) error {
	if err := s.ready(); err != nil {
		return err
	}

	count, err := s.store.MarkInterrupted(
		ctx,
		interruptedRunMessage,
		time.Now(),
	)

	if err != nil {
		return err
	}

	if count > 0 {
		s.logger.Warn(
			"marked interrupted terraform runs as failed",
			"count",
			count,
		)
	}

	return nil
}

// Wait blocks until all in-flight runs have finished or ctx ends.
func (s *ExecutionService) Wait(
	ctx context.Context,
) error {
	done := make(chan struct{})

	go func() {
		s.running.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ExecutionService) ready() error {
	if s.generator == nil {
		return errors.New(
			"terraform generator is not initialized",
		)
	}

	if s.specs == nil {
		return errors.New(
			"infrastructure service is not initialized",
		)
	}

	if s.projects == nil {
		return errors.New(
			"project service is not initialized",
		)
	}

	if s.store == nil {
		return errors.New(
			"terraform run store is not initialized",
		)
	}

	if s.executor == nil {
		return errors.New(
			"terraform executor is not initialized",
		)
	}

	return nil
}

func (s *ExecutionService) timeout() time.Duration {
	if s.config.ExecutionTimeoutSec <= 0 {
		return defaultExecutionTimeout
	}

	return time.Duration(
		s.config.ExecutionTimeoutSec,
	) * time.Second
}

func (s *ExecutionService) workspacePath(
	projectID bson.ObjectID,
	specificationID bson.ObjectID,
) (string, error) {
	root := strings.TrimSpace(
		s.config.WorkspaceRoot,
	)

	if root == "" {
		return "", errors.New(
			"terraform workspace root is not configured",
		)
	}

	absoluteRoot, err := filepath.Abs(root)

	if err != nil {
		return "", fmt.Errorf(
			"failed to resolve terraform workspace root: %w",
			err,
		)
	}

	return filepath.Join(
		absoluteRoot,
		projectID.Hex(),
		specificationID.Hex(),
	), nil
}

func (s *ExecutionService) loadPlanRun(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	planRunID bson.ObjectID,
) (*Run, error) {
	planRun, err := s.store.FindByIDAndProject(
		ctx,
		planRunID,
		ownerID,
		projectID,
	)

	if err != nil {
		if errors.Is(err, ErrRunNotFound) {
			return nil, fmt.Errorf(
				"%w: plan run not found",
				ErrInvalidPlan,
			)
		}

		return nil, err
	}

	if planRun.Type != RunTypePlan ||
		planRun.Status != RunStatusSucceeded ||
		planRun.PlanChecksum == "" {
		return nil, fmt.Errorf(
			"%w: run %s is not a successful plan",
			ErrInvalidPlan,
			planRun.ID.Hex(),
		)
	}

	return planRun, nil
}

func (s *ExecutionService) execute(
	run *Run,
	files Files,
	directory string,
) {
	defer s.running.Done()
	defer s.locks.Release(directory)

	defer func() {
		if recovered := recover(); recovered != nil {
			s.logger.Error(
				"terraform run panicked",
				"run_id",
				run.ID.Hex(),
				"panic",
				fmt.Sprint(recovered),
			)

			s.fail(
				run,
				"terraform run terminated unexpectedly",
			)
		}
	}()

	startedAt := time.Now()

	run.Status = RunStatusRunning
	run.StartedAt = &startedAt

	s.persist(run)

	// Apply executes the saved plan against the workspace exactly as
	// it was when the plan was created, so nothing is rewritten.
	if run.Type != RunTypeApply {
		if err := prepareWorkspace(
			directory,
			files,
		); err != nil {
			s.logger.Error(
				"failed to prepare terraform workspace",
				"run_id",
				run.ID.Hex(),
				"error",
				err,
			)

			s.fail(
				run,
				"failed to prepare terraform workspace",
			)

			return
		}
	}

	for _, definition := range stepDefinitions(
		run.Type,
	) {
		if !s.runStep(
			run,
			directory,
			definition,
		) {
			return
		}
	}

	s.finish(
		run,
		directory,
	)
}

func (s *ExecutionService) runStep(
	run *Run,
	directory string,
	definition stepDefinition,
) bool {
	run.Steps = append(
		run.Steps,
		Step{
			Name: definition.Name,
			Command: "terraform " + strings.Join(
				definition.Args,
				" ",
			),
			Status:    StepStatusRunning,
			StartedAt: time.Now(),
		},
	)

	stepIndex := len(run.Steps) - 1

	s.persist(run)

	stepContext, cancel := context.WithTimeout(
		context.Background(),
		s.timeout(),
	)

	result, err := s.executor.Execute(
		stepContext,
		Command{
			Dir:  directory,
			Args: definition.Args,
		},
	)

	cancel()

	finishedAt := time.Now()

	step := &run.Steps[stepIndex]

	step.Output = result.Output
	step.Truncated = result.Truncated
	step.ExitCode = result.ExitCode
	step.FinishedAt = &finishedAt

	if err != nil {
		step.Status = StepStatusFailed

		s.logger.Error(
			"terraform step failed",
			"run_id",
			run.ID.Hex(),
			"step",
			definition.Name,
			"error",
			err,
		)

		s.fail(
			run,
			s.failureMessage(
				definition.Name,
				err,
			),
		)

		return false
	}

	step.Status = StepStatusSucceeded

	if summary := ParseChangeSummary(
		definition.Name,
		result.Output,
	); summary != nil {
		run.Summary = summary
	}

	s.persist(run)

	return true
}

func (s *ExecutionService) finish(
	run *Run,
	directory string,
) {
	planPath := filepath.Join(
		directory,
		PlanFileName,
	)

	switch run.Type {
	case RunTypePlan:
		checksum, err := fileChecksum(planPath)

		if err != nil {
			s.logger.Error(
				"failed to checksum terraform plan",
				"run_id",
				run.ID.Hex(),
				"error",
				err,
			)

			s.fail(
				run,
				"terraform did not produce a plan file",
			)

			return
		}

		run.PlanChecksum = checksum

	case RunTypeApply:
		// A plan is single-use. Removing it means it can never be
		// applied twice, even if the state serial were to match.
		if err := os.Remove(
			planPath,
		); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			s.logger.Warn(
				"failed to remove applied terraform plan",
				"run_id",
				run.ID.Hex(),
				"error",
				err,
			)
		}
	}

	finishedAt := time.Now()

	run.Status = RunStatusSucceeded
	run.FinishedAt = &finishedAt

	s.persist(run)
}

func (s *ExecutionService) fail(
	run *Run,
	message string,
) {
	finishedAt := time.Now()

	run.Status = RunStatusFailed
	run.Error = message
	run.FinishedAt = &finishedAt

	s.persist(run)
}

func (s *ExecutionService) persist(
	run *Run,
) {
	run.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		persistTimeout,
	)

	defer cancel()

	if err := s.store.Update(
		ctx,
		run,
	); err != nil {
		s.logger.Error(
			"failed to persist terraform run",
			"run_id",
			run.ID.Hex(),
			"status",
			run.Status,
			"error",
			err,
		)
	}
}

// failureMessage returns the text stored on a failed run. It is
// shown to the user, so unexpected errors are reduced to a generic
// message and only logged in full.
func (s *ExecutionService) failureMessage(
	stepName string,
	err error,
) string {
	switch {
	case errors.Is(err, ErrExecutionTimeout):
		return fmt.Sprintf(
			"terraform %s timed out after %s",
			stepName,
			s.timeout(),
		)

	case errors.Is(err, context.Canceled):
		return fmt.Sprintf(
			"terraform %s was canceled",
			stepName,
		)

	case errors.Is(err, ErrExecutionFailed):
		return fmt.Sprintf(
			"terraform %s failed: %s",
			stepName,
			strings.TrimPrefix(
				err.Error(),
				ErrExecutionFailed.Error()+": ",
			),
		)

	case errors.Is(err, ErrStartFailed):
		return ErrStartFailed.Error()

	default:
		return fmt.Sprintf(
			"terraform %s failed unexpectedly",
			stepName,
		)
	}
}

type stepDefinition struct {
	Name string
	Args []string
}

func stepDefinitions(
	runType string,
) []stepDefinition {
	switch runType {
	case RunTypeValidate:
		return []stepDefinition{
			{
				Name: StepInit,
				Args: []string{
					"init",
					"-input=false",
					"-no-color",
					"-backend=false",
				},
			},
			{
				Name: StepValidate,
				Args: []string{
					"validate",
					"-no-color",
				},
			},
		}

	case RunTypePlan:
		return []stepDefinition{
			{
				Name: StepInit,
				Args: []string{
					"init",
					"-input=false",
					"-no-color",
				},
			},
			{
				Name: StepValidate,
				Args: []string{
					"validate",
					"-no-color",
				},
			},
			{
				Name: StepPlan,
				Args: []string{
					"plan",
					"-input=false",
					"-no-color",
					"-lock-timeout=60s",
					"-out=" + PlanFileName,
				},
			},
		}

	case RunTypeApply:
		return []stepDefinition{
			{
				Name: StepApply,
				Args: []string{
					"apply",
					"-input=false",
					"-no-color",
					"-lock-timeout=60s",
					PlanFileName,
				},
			},
		}

	case RunTypeDestroy:
		return []stepDefinition{
			{
				Name: StepInit,
				Args: []string{
					"init",
					"-input=false",
					"-no-color",
				},
			},
			{
				Name: StepDestroy,
				Args: []string{
					"destroy",
					"-input=false",
					"-no-color",
					"-lock-timeout=60s",
					"-auto-approve",
				},
			},
		}

	default:
		return nil
	}
}

// requireEC2AMIs rejects specifications that Terraform cannot run
// non-interactively: an EC2 instance without an AMI produces a
// variable that has no default value.
func requireEC2AMIs(
	specification infrastructure.InfrastructureSpec,
) error {
	for index, instance := range specification.EC2 {
		if strings.TrimSpace(instance.AMI) == "" {
			return fmt.Errorf(
				"%w: ec2[%d] (%s) requires an ami before terraform can run",
				ErrInvalidInput,
				index,
				instance.Name,
			)
		}
	}

	return nil
}

func hashFiles(
	files Files,
) string {
	hasher := sha256.New()

	for _, name := range sortedFileNames(files) {
		content := files[name]

		fmt.Fprintf(
			hasher,
			"%s\x00%d\x00",
			name,
			len(content),
		)

		hasher.Write(
			[]byte(content),
		)
	}

	return hex.EncodeToString(
		hasher.Sum(nil),
	)
}

func fileChecksum(
	path string,
) (string, error) {
	file, err := os.Open(path)

	if err != nil {
		return "", err
	}

	defer file.Close()

	hasher := sha256.New()

	if _, err := io.Copy(
		hasher,
		file,
	); err != nil {
		return "", err
	}

	return hex.EncodeToString(
		hasher.Sum(nil),
	), nil
}

// verifyPlanFile proves that the plan file in the workspace is the
// exact plan that was reviewed: not replaced by a newer plan, not
// already applied and not modified.
func verifyPlanFile(
	directory string,
	expectedChecksum string,
) error {
	actualChecksum, err := fileChecksum(
		filepath.Join(
			directory,
			PlanFileName,
		),
	)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf(
				"%w: the saved plan no longer exists (it may already have been applied); run a new plan",
				ErrPlanStale,
			)
		}

		return fmt.Errorf(
			"failed to read terraform plan: %w",
			err,
		)
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf(
			"%w: a newer plan replaced this one; apply the latest plan or run a new plan",
			ErrPlanStale,
		)
	}

	return nil
}

// prepareWorkspace writes the generated files into the workspace and
// removes .tf files that are no longer generated, so a resource
// removed from the specification is not silently kept. State, the
// provider lock file and the plugin directory are left untouched.
func prepareWorkspace(
	directory string,
	files Files,
) error {
	if err := os.MkdirAll(
		directory,
		0o750,
	); err != nil {
		return fmt.Errorf(
			"failed to create terraform workspace: %w",
			err,
		)
	}

	for _, name := range sortedFileNames(files) {
		if name == "" ||
			name == "." ||
			name == ".." ||
			name != filepath.Base(name) {
			return fmt.Errorf(
				"%w: unsafe terraform file name %q",
				ErrInvalidInput,
				name,
			)
		}

		if err := os.WriteFile(
			filepath.Join(
				directory,
				name,
			),
			[]byte(files[name]),
			0o600,
		); err != nil {
			return fmt.Errorf(
				"failed to write terraform file %s: %w",
				name,
				err,
			)
		}
	}

	entries, err := os.ReadDir(directory)

	if err != nil {
		return fmt.Errorf(
			"failed to read terraform workspace: %w",
			err,
		)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() ||
			!strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		if _, generated := files[entry.Name()]; generated {
			continue
		}

		if err := os.Remove(
			filepath.Join(
				directory,
				entry.Name(),
			),
		); err != nil {
			return fmt.Errorf(
				"failed to remove stale terraform file %s: %w",
				entry.Name(),
				err,
			)
		}
	}

	return nil
}

// workspaceLocks serializes runs per workspace inside this process.
type workspaceLocks struct {
	mutex sync.Mutex
	held  map[string]struct{}
}

func newWorkspaceLocks() *workspaceLocks {
	return &workspaceLocks{
		held: make(map[string]struct{}),
	}
}

func (l *workspaceLocks) TryAcquire(
	key string,
) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if _, exists := l.held[key]; exists {
		return false
	}

	l.held[key] = struct{}{}

	return true
}

func (l *workspaceLocks) Release(
	key string,
) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	delete(l.held, key)
}
