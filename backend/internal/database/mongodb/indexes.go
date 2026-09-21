package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func EnsureIndexes(
	ctx context.Context,
	database *Database,
) error {
	if database == nil {
		return fmt.Errorf(
			"mongodb database is required",
		)
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

	if _, err := database.Users().
		Indexes().
		CreateMany(ctx, userIndexes); err != nil {
		return fmt.Errorf(
			"failed to create user indexes: %w",
			err,
		)
	}

	projectIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				bson.E{
					Key:   "owner_id",
					Value: 1,
				},
				bson.E{
					Key:   "created_at",
					Value: -1,
				},
			},
			Options: options.Index().
				SetName("projects_owner_created_at"),
		},
	}

	if _, err := database.Projects().
		Indexes().
		CreateMany(ctx, projectIndexes); err != nil {
		return fmt.Errorf(
			"failed to create project indexes: %w",
			err,
		)
	}

	commandIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				bson.E{
					Key:   "project_id",
					Value: 1,
				},
				bson.E{
					Key:   "user_id",
					Value: 1,
				},
				bson.E{
					Key:   "created_at",
					Value: -1,
				},
			},
			Options: options.Index().
				SetName("commands_project_user_created_at"),
		},
	}

	if _, err := database.Commands().
		Indexes().
		CreateMany(ctx, commandIndexes); err != nil {
		return fmt.Errorf(
			"failed to create command indexes: %w",
			err,
		)
	}

	infrastructureIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				bson.E{
					Key:   "project_id",
					Value: 1,
				},
				bson.E{
					Key:   "updated_at",
					Value: -1,
				},
			},
			Options: options.Index().
				SetName(
					"infrastructure_specs_project_updated_at",
				),
		},
	}

	if _, err := database.InfrastructureSpecs().
		Indexes().
		CreateMany(
			ctx,
			infrastructureIndexes,
		); err != nil {
		return fmt.Errorf(
			"failed to create infrastructure specification indexes: %w",
			err,
		)
	}

	return nil
}