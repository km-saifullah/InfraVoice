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
