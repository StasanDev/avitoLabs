package database

import (
	"context"
	"fmt"

	"github.com/StasanDev/avitoLabs/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(
	ctx context.Context,
	cfg *config.DBConfig,
) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return pool, nil
}
