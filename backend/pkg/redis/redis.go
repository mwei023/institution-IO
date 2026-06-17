package redis

import (
	"context"
	"fmt"
	"time"

	"institution-os/pkg/config"
	goredis "github.com/redis/go-redis/v9"
)

// Client wraps the go-redis client with helpers used across the application.
type Client struct {
	*goredis.Client
}

// Connect creates and validates a Redis client.
func Connect(cfg *config.Config) (*Client, error) {
	opts, err := goredis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}

	opts.PoolSize        = 20
	opts.MinIdleConns    = 5
	opts.ConnMaxLifetime = 30 * time.Minute
	opts.ConnMaxIdleTime = 10 * time.Minute

	rdb := goredis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Client{rdb}, nil
}

// HealthCheck returns nil if Redis is reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Ping(ctx).Err()
}

// SetWithTTL is a convenience wrapper around Set with a TTL.
func (c *Client) SetWithTTL(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.Set(ctx, key, value, ttl).Err()
}

// GetString retrieves a string value; returns ("", false, nil) if the key does not exist.
func (c *Client) GetString(ctx context.Context, key string) (string, bool, error) {
	val, err := c.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

// BlacklistToken stores a JWT JTI in Redis so it cannot be reused before expiry.
func (c *Client) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	return c.SetWithTTL(ctx, "bj:"+jti, "1", ttl)
}

// IsTokenBlacklisted checks whether a JTI has been revoked.
func (c *Client) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	_, found, err := c.GetString(ctx, "bj:"+jti)
	return found, err
}

// CachePermissions stores serialised permissions for a user.
func (c *Client) CachePermissions(ctx context.Context, userID string, permsJSON string) error {
	return c.SetWithTTL(ctx, "perm:"+userID, permsJSON, 5*time.Minute)
}

// GetCachedPermissions retrieves cached permissions for a user.
func (c *Client) GetCachedPermissions(ctx context.Context, userID string) (string, bool, error) {
	return c.GetString(ctx, "perm:"+userID)
}
