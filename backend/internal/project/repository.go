package project

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
)

var ErrProjectNotFound = errors.New("project not found")

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongodb.Database) *Repository {
	return &Repository{
		collection: database.Projects(),
	}
}

func (r *Repository) Create(
	ctx context.Context,
	newProject *Project,
) error {
	if newProject == nil {
		return fmt.Errorf("project is required")
	}

	if newProject.ID.IsZero() {
		newProject.ID = bson.NewObjectID()
	}

	_, err := r.collection.InsertOne(
		ctx,
		newProject,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to create project: %w",
			err,
		)
	}

	return nil
}

func (r *Repository) FindByIDAndOwner(
	ctx context.Context,
	projectID bson.ObjectID,
	ownerID bson.ObjectID,
) (*Project, error) {
	var foundProject Project

	err := r.collection.
		FindOne(
			ctx,
			bson.D{
				bson.E{
					Key:   "_id",
					Value: projectID,
				},
				bson.E{
					Key:   "owner_id",
					Value: ownerID,
				},
			},
		).
		Decode(&foundProject)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrProjectNotFound
		}

		return nil, fmt.Errorf(
			"failed to find project: %w",
			err,
		)
	}

	return &foundProject, nil
}

func (r *Repository) ListByOwner(
	ctx context.Context,
	ownerID bson.ObjectID,
) ([]Project, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.D{
			bson.E{
				Key:   "owner_id",
				Value: ownerID,
			},
		},
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to list projects: %w",
			err,
		)
	}

	defer cursor.Close(ctx)

	var projects []Project

	if err := cursor.All(
		ctx,
		&projects,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode projects: %w",
			err,
		)
	}

	if projects == nil {
		projects = []Project{}
	}

	return projects, nil
}

func (r *Repository) UpdateByIDAndOwner(
	ctx context.Context,
	projectID bson.ObjectID,
	ownerID bson.ObjectID,
	update bson.D,
) (*Project, error) {
	var updatedProject Project

	err := r.collection.
		FindOneAndUpdate(
			ctx,
			bson.D{
				bson.E{
					Key:   "_id",
					Value: projectID,
				},
				bson.E{
					Key:   "owner_id",
					Value: ownerID,
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
		Decode(&updatedProject)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrProjectNotFound
		}

		return nil, fmt.Errorf(
			"failed to update project: %w",
			err,
		)
	}

	return &updatedProject, nil
}

func (r *Repository) DeleteByIDAndOwner(
	ctx context.Context,
	projectID bson.ObjectID,
	ownerID bson.ObjectID,
) error {
	result, err := r.collection.DeleteOne(
		ctx,
		bson.D{
			bson.E{
				Key:   "_id",
				Value: projectID,
			},
			bson.E{
				Key:   "owner_id",
				Value: ownerID,
			},
		},
	)

	if err != nil {
		return fmt.Errorf(
			"failed to delete project: %w",
			err,
		)
	}

	if result.DeletedCount == 0 {
		return ErrProjectNotFound
	}

	return nil
}

// StatusCounts is a per-owner rollup used by the overview endpoint.
type StatusCounts struct {
	Total    int64
	Active   int64
	Archived int64
}

func (r *Repository) CountStatusesByOwner(
	ctx context.Context,
	ownerID bson.ObjectID,
) (StatusCounts, error) {
	total, err := r.collection.CountDocuments(
		ctx,
		bson.D{
			bson.E{
				Key:   "owner_id",
				Value: ownerID,
			},
		},
	)

	if err != nil {
		return StatusCounts{}, fmt.Errorf(
			"failed to count projects: %w",
			err,
		)
	}

	active, err := r.collection.CountDocuments(
		ctx,
		bson.D{
			bson.E{
				Key:   "owner_id",
				Value: ownerID,
			},
			bson.E{
				Key:   "status",
				Value: StatusActive,
			},
		},
	)

	if err != nil {
		return StatusCounts{}, fmt.Errorf(
			"failed to count active projects: %w",
			err,
		)
	}

	return StatusCounts{
		Total:    total,
		Active:   active,
		Archived: total - active,
	}, nil
}

func (r *Repository) ListIDsByOwner(
	ctx context.Context,
	ownerID bson.ObjectID,
) ([]bson.ObjectID, error) {
	cursor, err := r.collection.Find(
		ctx,
		bson.D{
			bson.E{
				Key:   "owner_id",
				Value: ownerID,
			},
		},
		options.Find().SetProjection(
			bson.D{
				bson.E{
					Key:   "_id",
					Value: 1,
				},
			},
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to list project ids: %w",
			err,
		)
	}

	defer cursor.Close(ctx)

	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
	}

	if err := cursor.All(
		ctx,
		&rows,
	); err != nil {
		return nil, fmt.Errorf(
			"failed to decode project ids: %w",
			err,
		)
	}

	ids := make([]bson.ObjectID, len(rows))

	for index, row := range rows {
		ids[index] = row.ID
	}

	return ids, nil
}
