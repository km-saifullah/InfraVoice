package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
)

var ErrSpecificationNotFound = errors.New(
	"infrastructure specification not found",
)

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(
	database *mongodb.Database,
) *Repository {
	return &Repository{
		collection: database.InfrastructureSpecs(),
	}
}

func (r *Repository) Create(
	ctx context.Context,
	spec *InfrastructureSpec,
) error {
	if spec == nil {
		return fmt.Errorf(
			"infrastructure specification is required",
		)
	}

	if spec.ID.IsZero() {
		spec.ID = bson.NewObjectID()
	}

	_, err := r.collection.InsertOne(
		ctx,
		spec,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to create infrastructure specification: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) FindByIDAndProject(
	ctx context.Context,
	specID bson.ObjectID,
	projectID bson.ObjectID,
) (*InfrastructureSpec, error) {
	var spec InfrastructureSpec

	err := r.collection.
		FindOne(
			ctx,
			bson.D{
				bson.E{
					Key:   "_id",
					Value: specID,
				},
				bson.E{
					Key:   "project_id",
					Value: projectID,
				},
			},
		).
		Decode(&spec)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSpecificationNotFound
		}

		return nil, fmt.Errorf(
			"failed to find infrastructure specification: %w",
			err,
		)
	}

	return &spec, nil
}

func (r *Repository) ListByProject(
	ctx context.Context,
	projectID bson.ObjectID,
) ([]InfrastructureSpec, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.D{
			bson.E{
				Key:   "project_id",
				Value: projectID,
			},
		},
		options.Find().
			SetSort(
				bson.D{
					bson.E{
						Key:   "updated_at",
						Value: -1,
					},
				},
			),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to list infrastructure specifications: %w",
			err,
		)
	}

	defer cursor.Close(ctx)

	specifications := make(
		[]InfrastructureSpec,
		0,
	)

	if err := cursor.All(
		ctx,
		&specifications,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode infrastructure specifications: %w",
			err,
		)
	}

	return specifications, nil
}

type ResourceCounts struct {
	Specifications int64
	VPC            int64
	EC2            int64
	S3             int64
	SNS            int64
}

type resourceCountsRow struct {
	Specifications int64 `bson:"specifications"`
	VPC            int64 `bson:"vpc"`
	EC2            int64 `bson:"ec2"`
	S3             int64 `bson:"s3"`
	SNS            int64 `bson:"sns"`
}

func (r *Repository) AggregateResourceCounts(
	ctx context.Context,
	projectIDs []bson.ObjectID,
) (ResourceCounts, error) {
	if len(projectIDs) == 0 {
		return ResourceCounts{}, nil
	}

	emptyArray := bson.A{}

	cursor, err := r.collection.Aggregate(
		ctx,
		bson.A{
			bson.D{
				bson.E{
					Key: "$match",
					Value: bson.D{
						bson.E{
							Key: "project_id",
							Value: bson.D{
								bson.E{
									Key:   "$in",
									Value: projectIDs,
								},
							},
						},
					},
				},
			},
			bson.D{
				bson.E{
					Key: "$project",
					Value: bson.D{
						bson.E{
							Key: "vpc_count",
							Value: bson.D{
								bson.E{
									Key: "$size",
									Value: bson.D{
										bson.E{
											Key:   "$ifNull",
											Value: bson.A{"$vpcs", emptyArray},
										},
									},
								},
							},
						},
						bson.E{
							Key: "ec2_count",
							Value: bson.D{
								bson.E{
									Key: "$size",
									Value: bson.D{
										bson.E{
											Key:   "$ifNull",
											Value: bson.A{"$ec2", emptyArray},
										},
									},
								},
							},
						},
						bson.E{
							Key: "s3_count",
							Value: bson.D{
								bson.E{
									Key: "$size",
									Value: bson.D{
										bson.E{
											Key:   "$ifNull",
											Value: bson.A{"$s3", emptyArray},
										},
									},
								},
							},
						},
						bson.E{
							Key: "sns_count",
							Value: bson.D{
								bson.E{
									Key: "$size",
									Value: bson.D{
										bson.E{
											Key:   "$ifNull",
											Value: bson.A{"$sns", emptyArray},
										},
									},
								},
							},
						},
					},
				},
			},
			bson.D{
				bson.E{
					Key: "$group",
					Value: bson.D{
						bson.E{
							Key:   "_id",
							Value: nil,
						},
						bson.E{
							Key: "specifications",
							Value: bson.D{
								bson.E{
									Key:   "$sum",
									Value: 1,
								},
							},
						},
						bson.E{
							Key: "vpc",
							Value: bson.D{
								bson.E{
									Key:   "$sum",
									Value: "$vpc_count",
								},
							},
						},
						bson.E{
							Key: "ec2",
							Value: bson.D{
								bson.E{
									Key:   "$sum",
									Value: "$ec2_count",
								},
							},
						},
						bson.E{
							Key: "s3",
							Value: bson.D{
								bson.E{
									Key:   "$sum",
									Value: "$s3_count",
								},
							},
						},
						bson.E{
							Key: "sns",
							Value: bson.D{
								bson.E{
									Key:   "$sum",
									Value: "$sns_count",
								},
							},
						},
					},
				},
			},
		},
	)

	if err != nil {
		return ResourceCounts{}, fmt.Errorf(
			"failed to aggregate infrastructure resources: %w",
			err,
		)
	}

	defer cursor.Close(ctx)

	var rows []resourceCountsRow

	if err := cursor.All(
		ctx,
		&rows,
	); err != nil {
		return ResourceCounts{}, fmt.Errorf(
			"failed to decode infrastructure resource aggregation: %w",
			err,
		)
	}

	if len(rows) == 0 {
		return ResourceCounts{}, nil
	}

	row := rows[0]

	return ResourceCounts{
		Specifications: row.Specifications,
		VPC:            row.VPC,
		EC2:            row.EC2,
		S3:             row.S3,
		SNS:            row.SNS,
	}, nil
}

func (r *Repository) UpdateByIDAndProject(
	ctx context.Context,
	specID bson.ObjectID,
	projectID bson.ObjectID,
	update bson.D,
) (*InfrastructureSpec, error) {
	var updatedSpec InfrastructureSpec

	err := r.collection.
		FindOneAndUpdate(
			ctx,
			bson.D{
				bson.E{
					Key:   "_id",
					Value: specID,
				},
				bson.E{
					Key:   "project_id",
					Value: projectID,
				},
			},
			bson.D{
				bson.E{
					Key:   "$set",
					Value: update,
				},
			},
			options.FindOneAndUpdate().
				SetReturnDocument(options.After),
		).
		Decode(&updatedSpec)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSpecificationNotFound
		}

		return nil, fmt.Errorf(
			"failed to update infrastructure specification: %w",
			err,
		)
	}

	return &updatedSpec, nil
}

func (r *Repository) DeleteByIDAndProject(
	ctx context.Context,
	specID bson.ObjectID,
	projectID bson.ObjectID,
) error {
	result, err := r.collection.DeleteOne(
		ctx,
		bson.D{
			bson.E{
				Key:   "_id",
				Value: specID,
			},
			bson.E{
				Key:   "project_id",
				Value: projectID,
			},
		},
	)

	if err != nil {
		return fmt.Errorf(
			"failed to delete infrastructure specification: %w",
			err,
		)
	}

	if result.DeletedCount == 0 {
		return ErrSpecificationNotFound
	}

	return nil
}
