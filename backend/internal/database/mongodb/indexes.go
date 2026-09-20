package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func EnsureIndexes(ctx context.Context, database *Database) error {
	if database == nil {
		return fmt.Errorf("mongodb database is required")
	}

	userIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				bson.E{
					Key:   "email",
					Value: 1,
				},
			},
			Options: options.Index().
				SetUnique(true).
				SetName("users_email_unique"),
		},
	}

	if _, err := database.Users().Indexes().CreateMany(
		ctx,
		userIndexes,
	); err != nil {
		return fmt.Errorf("failed to create user indexes: %w", err)
	}

	return nil
}
