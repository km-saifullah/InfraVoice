package terraform

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	RunTypeValidate = "validate"
	RunTypePlan     = "plan"
	RunTypeApply    = "apply"
	RunTypeDestroy  = "destroy"
)

const (
	RunStatusQueued    = "queued"
	RunStatusRunning   = "running"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
)

const (
	StepStatusRunning   = "running"
	StepStatusSucceeded = "succeeded"
	StepStatusFailed    = "failed"
)

const (
	StepInit     = "init"
	StepValidate = "validate"
	StepPlan     = "plan"
	StepApply    = "apply"
	StepDestroy  = "destroy"
)

// PlanFileName is the name of the saved plan inside a workspace.
// An apply run only ever executes a plan file that was reviewed
// through a successful plan run.
const PlanFileName = "tfplan"

var (
	ErrRunInProgress = errors.New(
		"a terraform run is already in progress for this specification",
	)

	ErrConfirmationRequired = errors.New(
		"explicit confirmation is required for this terraform action",
	)

	ErrInvalidPlan = errors.New(
		"invalid terraform plan",
	)

	ErrPlanStale = errors.New(
		"terraform plan is stale",
	)
)

type Run struct {
	ID              bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	OwnerID         bson.ObjectID  `bson:"owner_id" json:"owner_id"`
	ProjectID       bson.ObjectID  `bson:"project_id" json:"project_id"`
	SpecificationID bson.ObjectID  `bson:"specification_id" json:"specification_id"`
	PlanRunID       *bson.ObjectID `bson:"plan_run_id,omitempty" json:"plan_run_id,omitempty"`

	Type   string `bson:"type" json:"type"`
	Status string `bson:"status" json:"status"`
	Region string `bson:"region" json:"region"`

	ConfigHash string `bson:"config_hash" json:"config_hash"`

	// PlanChecksum is the SHA-256 checksum of the saved plan file.
	// It is only set on successful plan runs and is used to prove
	// that the plan being applied is the plan that was reviewed.
	PlanChecksum string `bson:"plan_checksum,omitempty" json:"-"`

	Summary *ChangeSummary `bson:"summary,omitempty" json:"summary,omitempty"`
	Steps   []Step         `bson:"steps" json:"steps"`
	Error   string         `bson:"error,omitempty" json:"error,omitempty"`

	CreatedAt  time.Time  `bson:"created_at" json:"created_at"`
	StartedAt  *time.Time `bson:"started_at,omitempty" json:"started_at,omitempty"`
	FinishedAt *time.Time `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
	UpdatedAt  time.Time  `bson:"updated_at" json:"updated_at"`
}

type Step struct {
	Name       string     `bson:"name" json:"name"`
	Command    string     `bson:"command" json:"command"`
	Status     string     `bson:"status" json:"status"`
	ExitCode   int        `bson:"exit_code" json:"exit_code"`
	Output     string     `bson:"output" json:"output"`
	Truncated  bool       `bson:"truncated" json:"truncated"`
	StartedAt  time.Time  `bson:"started_at" json:"started_at"`
	FinishedAt *time.Time `bson:"finished_at,omitempty" json:"finished_at,omitempty"`
}

type ChangeSummary struct {
	Add     int `bson:"add" json:"add"`
	Change  int `bson:"change" json:"change"`
	Destroy int `bson:"destroy" json:"destroy"`
}

type StartInput struct {
	SpecificationID bson.ObjectID
	Region          string
	Type            string
	PlanRunID       bson.ObjectID
	Confirm         bool
}

func NormalizeRunType(
	value string,
) string {
	return strings.ToLower(
		strings.TrimSpace(value),
	)
}

func IsValidRunType(
	value string,
) bool {
	switch value {
	case RunTypeValidate,
		RunTypePlan,
		RunTypeApply,
		RunTypeDestroy:
		return true

	default:
		return false
	}
}

// RequiresConfirmation reports whether a run type changes real
// infrastructure and therefore needs explicit confirmation.
func RequiresConfirmation(
	runType string,
) bool {
	return runType == RunTypeApply ||
		runType == RunTypeDestroy
}

func (r *Run) IsTerminal() bool {
	return r.Status == RunStatusSucceeded ||
		r.Status == RunStatusFailed
}

var (
	planSummaryPattern = regexp.MustCompile(
		`Plan: (?:\d+ to import, )?(\d+) to add, (\d+) to change, (\d+) to destroy`,
	)

	applySummaryPattern = regexp.MustCompile(
		`Apply complete! Resources: (\d+) added, (\d+) changed, (\d+) destroyed`,
	)

	destroySummaryPattern = regexp.MustCompile(
		`Destroy complete! Resources: (\d+) destroyed`,
	)

	noChangesPattern = regexp.MustCompile(
		`(?m)^No changes\.`,
	)
)

// ParseChangeSummary extracts the resource change counts that
// Terraform prints at the end of a plan, apply or destroy step.
// It returns nil when the output does not contain a summary.
func ParseChangeSummary(
	stepName string,
	output string,
) *ChangeSummary {
	switch stepName {
	case StepPlan:
		if matches := planSummaryPattern.FindStringSubmatch(
			output,
		); matches != nil {
			return summaryFromMatches(
				matches[1],
				matches[2],
				matches[3],
			)
		}

		if noChangesPattern.MatchString(output) {
			return &ChangeSummary{}
		}

	case StepApply:
		if matches := applySummaryPattern.FindStringSubmatch(
			output,
		); matches != nil {
			return summaryFromMatches(
				matches[1],
				matches[2],
				matches[3],
			)
		}

	case StepDestroy:
		if matches := destroySummaryPattern.FindStringSubmatch(
			output,
		); matches != nil {
			return summaryFromMatches(
				"0",
				"0",
				matches[1],
			)
		}
	}

	return nil
}

func summaryFromMatches(
	add string,
	change string,
	destroy string,
) *ChangeSummary {
	addCount, err := strconv.Atoi(add)

	if err != nil {
		return nil
	}

	changeCount, err := strconv.Atoi(change)

	if err != nil {
		return nil
	}

	destroyCount, err := strconv.Atoi(destroy)

	if err != nil {
		return nil
	}

	return &ChangeSummary{
		Add:     addCount,
		Change:  changeCount,
		Destroy: destroyCount,
	}
}
