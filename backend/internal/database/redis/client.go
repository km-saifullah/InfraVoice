package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/km-saifullah/infra-voice/backend/internal/config"
)

type Client struct {
	client *goredis.Client
}

func New(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("redis URL is required")
	}

	options, err := goredis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Explicit application configuration takes precedence over
	// the password and database values parsed from the URL.
	if cfg.Password != "" {
		options.Password = cfg.Password
	}

	options.DB = cfg.DB

	client := goredis.NewClient(options)

	pingContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingContext).Err(); err != nil {
		_ = client.Close()

		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

func (c *Client) Client() *goredis.Client {
	if c == nil {
		return nil
	}

	return c.client
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}

	return nil
}
