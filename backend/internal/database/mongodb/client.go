package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
)

type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

func New(ctx context.Context, cfg config.MongoDBConfig) (*Client, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("mongodb URI is required")
	}

	if cfg.Database == "" {
		return nil, fmt.Errorf("mongodb database name is required")
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create mongodb client: %w", err)
	}

	if err := client.Ping(connectCtx, readpref.Primary()); err != nil {
		disconnectCtx, disconnectCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer disconnectCancel()

		_ = client.Disconnect(disconnectCtx)

		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	database := client.Database(cfg.Database)

	return &Client{
		client:   client,
		database: database,
	}, nil
}

func (c *Client) Database() *mongo.Database {
	return c.database
}

func (c *Client) Client() *mongo.Client {
	return c.client
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("mongodb client is not initialized")
	}

	if err := c.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongodb ping failed: %w", err)
	}

	return nil
}

func (c *Client) Disconnect(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}

	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect mongodb: %w", err)
	}

	return nil
}
