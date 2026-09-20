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
