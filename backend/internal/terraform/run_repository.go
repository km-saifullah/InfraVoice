package terraform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
)

var ErrRunNotFound = errors.New(
	"terraform run not found",
)

type RunRepository struct {
	collection *mongo.Collection
}

func NewRunRepository(
	database *mongodb.Database,
) *RunRepository {
	return &RunRepository{
		collection: database.TerraformRuns(),
	}
}

func (r *RunRepository) Create(
	ctx context.Context,
	run *Run,
) error {
	if run == nil {
		return fmt.Errorf(
			"terraform run is required",
		)
	}

	if run.ID.IsZero() {
		run.ID = bson.NewObjectID()
	}

	_, err := r.collection.InsertOne(
		ctx,
		run,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to create terraform run: %w",
			err,
		)
	}

	return nil
}

func (r *RunRepository) Update(
	ctx context.Context,
	run *Run,
) error {
	if run == nil || run.ID.IsZero() {
		return fmt.Errorf(
			"terraform run is required",
		)
	}

	result, err := r.collection.ReplaceOne(
		ctx,
		bson.D{
			bson.E{
				Key:   "_id",
				Value: run.ID,
			},
		},
		run,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to update terraform run: %w",
			err,
		)
	}

	if result.MatchedCount == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (r *RunRepository) FindByIDAndProject(
	ctx context.Context,
	runID bson.ObjectID,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
) (*Run, error) {
	var run Run

	err := r.collection.
		FindOne(
			ctx,
			bson.D{
				bson.E{
					Key:   "_id",
					Value: runID,
				},
				bson.E{
					Key:   "owner_id",
					Value: ownerID,
				},
				bson.E{
					Key:   "project_id",
					Value: projectID,
				},
			},
		).
		Decode(&run)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRunNotFound
		}

		return nil, fmt.Errorf(
			"failed to find terraform run: %w",
			err,
		)
	}

	return &run, nil
}

func (r *RunRepository) ListByProject(
	ctx context.Context,
	ownerID bson.ObjectID,
	projectID bson.ObjectID,
	limit int64,
) ([]Run, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.D{
			bson.E{
				Key:   "owner_id",
				Value: ownerID,
			},
			bson.E{
				Key:   "project_id",
				Value: projectID,
			},
		},
		options.Find().
			SetSort(
				bson.D{
					bson.E{
						Key:   "created_at",
						Value: -1,
					},
				},
			).
			SetLimit(limit),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to list terraform runs: %w",
			err,
		)
	}

	defer cursor.Close(ctx)

	runs := make([]Run, 0)

	if err := cursor.All(
		ctx,
		&runs,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode terraform runs: %w",
			err,
		)
	}

	return runs, nil
}

// MarkInterrupted fails every run that is still queued or running.
// It is meant to be called once at startup: a run cannot survive a
// restart of the process that executes it.
func (r *RunRepository) MarkInterrupted(
	ctx context.Context,
	message string,
	at time.Time,
) (int64, error) {
	result, err := r.collection.UpdateMany(
		ctx,
		bson.D{
			bson.E{
				Key: "status",
				Value: bson.D{
					bson.E{
						Key: "$in",
						Value: bson.A{
							RunStatusQueued,
							RunStatusRunning,
						},
					},
				},
			},
		},
		bson.D{
			bson.E{
				Key: "$set",
				Value: bson.D{
					bson.E{
						Key:   "status",
						Value: RunStatusFailed,
					},
					bson.E{
						Key:   "error",
						Value: message,
					},
					bson.E{
						Key:   "finished_at",
						Value: at,
					},
					bson.E{
						Key:   "updated_at",
						Value: at,
					},
					bson.E{
						Key:   "steps.$[running].status",
						Value: StepStatusFailed,
					},
					bson.E{
						Key:   "steps.$[running].finished_at",
						Value: at,
					},
				},
			},
		},
		options.UpdateMany().SetArrayFilters(
			[]any{
				bson.D{
					bson.E{
						Key:   "running.status",
						Value: StepStatusRunning,
					},
				},
			},
		),
	)

	if err != nil {
		return 0, fmt.Errorf(
			"failed to mark interrupted terraform runs: %w",
			err,
		)
	}

	return result.ModifiedCount, nil
}
