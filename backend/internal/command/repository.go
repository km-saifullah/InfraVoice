package command

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
)

var ErrCommandNotFound = errors.New("command not found")

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongodb.Database) *Repository {
	return &Repository{
		collection: database.Commands(),
	}
}

func (r *Repository) Create(
	ctx context.Context,
	newCommand *Command,
) error {
	if newCommand == nil {
		return fmt.Errorf("command is required")
	}

	if newCommand.ID.IsZero() {
		newCommand.ID = bson.NewObjectID()
	}

	_, err := r.collection.InsertOne(
		ctx,
		newCommand,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to create command: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) ListByProjectAndUser(
	ctx context.Context,
	projectID bson.ObjectID,
	userID bson.ObjectID,
) ([]Command, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.D{
			bson.E{
				Key:   "project_id",
				Value: projectID,
			},
			bson.E{
				Key:   "user_id",
				Value: userID,
			},
		},
		options.Find().SetSort(
			bson.D{
				bson.E{
					Key:   "created_at",
					Value: -1,
				},
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to list commands: %w",
			err,
		)
	}

	defer cursor.Close(ctx)

	commands := make([]Command, 0)

	if err := cursor.All(
		ctx,
		&commands,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode commands: %w",
			err,
		)
	}

	return commands, nil
}

// Counts is a per-user rollup used by the overview endpoint.
type Counts struct {
	Total    int64
	ByStatus map[string]int64
	BySource map[string]int64
}

type countBucket struct {
	ID    string `bson:"_id"`
	Count int64  `bson:"count"`
}

type countFacetResult struct {
	Total []struct {
		Count int64 `bson:"count"`
	} `bson:"total"`
	ByStatus []countBucket `bson:"by_status"`
	BySource []countBucket `bson:"by_source"`
}

func (r *Repository) CountByUser(
	ctx context.Context,
	userID bson.ObjectID,
) (Counts, error) {
	cursor, err := r.collection.Aggregate(
		ctx,
		bson.A{
			bson.D{
				bson.E{
					Key: "$match",
					Value: bson.D{
						bson.E{
							Key:   "user_id",
							Value: userID,
						},
					},
				},
			},
			bson.D{
				bson.E{
					Key: "$facet",
					Value: bson.D{
						bson.E{
							Key: "total",
							Value: bson.A{
								bson.D{
									bson.E{
										Key:   "$count",
										Value: "count",
									},
								},
							},
						},
						bson.E{
							Key: "by_status",
							Value: bson.A{
								bson.D{
									bson.E{
										Key: "$group",
										Value: bson.D{
											bson.E{
												Key:   "_id",
												Value: "$status",
											},
											bson.E{
												Key: "count",
												Value: bson.D{
													bson.E{
														Key:   "$sum",
														Value: 1,
													},
												},
											},
										},
									},
								},
							},
						},
						bson.E{
							Key: "by_source",
							Value: bson.A{
								bson.D{
									bson.E{
										Key: "$group",
										Value: bson.D{
											bson.E{
												Key:   "_id",
												Value: "$source",
											},
											bson.E{
												Key: "count",
												Value: bson.D{
													bson.E{
														Key:   "$sum",
														Value: 1,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	)

	if err != nil {
		return Counts{}, fmt.Errorf(
			"failed to aggregate commands: %w",
			err,
		)
	}

	defer cursor.Close(ctx)

	var results []countFacetResult

	if err := cursor.All(
		ctx,
		&results,
	); err != nil {
		return Counts{}, fmt.Errorf(
			"failed to decode command aggregation: %w",
			err,
		)
	}

	counts := Counts{
		ByStatus: make(map[string]int64),
		BySource: make(map[string]int64),
	}

	if len(results) == 0 {
		return counts, nil
	}

	result := results[0]

	if len(result.Total) > 0 {
		counts.Total = result.Total[0].Count
	}

	for _, bucket := range result.ByStatus {
		counts.ByStatus[bucket.ID] = bucket.Count
	}

	for _, bucket := range result.BySource {
		counts.BySource[bucket.ID] = bucket.Count
	}

	return counts, nil
}

func (r *Repository) FindByIDAndUser(
	ctx context.Context,
	commandID bson.ObjectID,
	userID bson.ObjectID,
) (*Command, error) {
	var foundCommand Command

	err := r.collection.
		FindOne(
			ctx,
			bson.D{
				bson.E{
					Key:   "_id",
					Value: commandID,
				},
				bson.E{
					Key:   "user_id",
					Value: userID,
				},
			},
		).
		Decode(&foundCommand)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrCommandNotFound
		}

		return nil, fmt.Errorf(
			"failed to find command: %w",
			err,
		)
	}

	return &foundCommand, nil
}
