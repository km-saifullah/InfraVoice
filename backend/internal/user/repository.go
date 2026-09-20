package user

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/km-saifullah/infra-voice/backend/internal/database/mongodb"
)

var ErrUserNotFound = errors.New("user not found")

type Repository struct {
	collection *mongo.Collection
}

func NewRepository(database *mongodb.Database) *Repository {
	return &Repository{
		collection: database.Users(),
	}
}

func (r *Repository) Create(
	ctx context.Context,
	newUser *User,
) error {
	if newUser == nil {
		return fmt.Errorf("user is required")
	}

	if newUser.ID.IsZero() {
		newUser.ID = bson.NewObjectID()
	}

	result, err := r.collection.InsertOne(ctx, newUser)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if insertedID, ok := result.InsertedID.(bson.ObjectID); ok {
		newUser.ID = insertedID
	}

	return nil
}

func (r *Repository) FindByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	var foundUser User

	err := r.collection.
		FindOne(
			ctx,
			bson.D{
				{Key: "email", Value: email},
			},
		).
		Decode(&foundUser)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return &foundUser, nil
}

func (r *Repository) FindByID(
	ctx context.Context,
	id bson.ObjectID,
) (*User, error) {
	var foundUser User

	err := r.collection.
		FindOne(
			ctx,
			bson.D{
				{Key: "_id", Value: id},
			},
		).
		Decode(&foundUser)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return &foundUser, nil
}
