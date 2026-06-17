package db

import (
	"context"
	"fmt"
	"time"

	"institution-os/pkg/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool wraps pgxpool.Pool with institution-OS-specific helpers.
type Pool struct {
	*pgxpool.Pool
}

// Connect creates and validates a PostgreSQL connection pool.
func Connect(ctx context.Context, cfg *config.Config) (*Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolConfig.MaxConns = cfg.DBMaxConns
	poolConfig.MinConns = cfg.DBMinConns
	poolConfig.MaxConnLifetime = cfg.DBMaxConnLife
	poolConfig.MaxConnIdleTime = cfg.DBMaxConnIdle
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Pool{pool}, nil
}

// HealthCheck returns nil if the database is reachable.
func (p *Pool) HealthCheck(ctx context.Context) error {
	return p.Ping(ctx)
}
