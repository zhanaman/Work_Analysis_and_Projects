package storage

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres wraps a connection pool to PostgreSQL.
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres creates a new PostgreSQL connection pool.
func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres DSN: %w", err)
	}

	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	slog.Info("connected to PostgreSQL")
	return &Postgres{Pool: pool}, nil
}

// Close gracefully closes the connection pool.
func (p *Postgres) Close() {
	p.Pool.Close()
	slog.Info("PostgreSQL connection closed")
}
