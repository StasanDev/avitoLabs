package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/StasanDev/avitoLabs/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const idempotencyKeysTable = "trip_idempotency_keys"

var idempotencyKeyColumns = []string{
	"idempotency_key", "trip_id", "request_hash", "expires_at",
}

type IdempotencyRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewIdempotencyRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *IdempotencyRepository {
	return &IdempotencyRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *IdempotencyRepository) ReserveOrGet(
	ctx context.Context,
	record domain.IdempotencyRecord,
) (domain.IdempotencyRecord, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	reserveQuery, reserveArgs, err := sq.
		Insert(idempotencyKeysTable+" AS current_record").Columns(idempotencyKeyColumns...).
		Values(record.Key, record.TripID, record.RequestHash, record.ExpiresAt).
		Suffix(`
		ON CONFLICT (idempotency_key) DO UPDATE
		SET
			trip_id = EXCLUDED.trip_id,
			request_hash = EXCLUDED.request_hash,
			expires_at = EXCLUDED.expires_at
		WHERE current_record.expires_at <= now()
		RETURNING idempotency_key, trip_id, request_hash, expires_at
		`).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("build reserve idempotency key: %w", err)
	}

	var reservedRecord domain.IdempotencyRecord
	err = r.executor(ctx).QueryRow(ctx, reserveQuery, reserveArgs...).Scan(
		&reservedRecord.Key,&reservedRecord.TripID,
		&reservedRecord.RequestHash,&reservedRecord.ExpiresAt,
	)
	if err == nil {
		return reservedRecord, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("reserve idempotency key: %w", err)
	}

	getQuery, getArgs, err := sq.
		Select(idempotencyKeyColumns...).
		From(idempotencyKeysTable).
		Where(sq.Eq{"idempotency_key": record.Key}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("build get idempotency key: %w", err)
	}

	var existing domain.IdempotencyRecord
	if err := r.executor(ctx).QueryRow(ctx, getQuery, getArgs...).Scan(
		&existing.Key,
		&existing.TripID,
		&existing.RequestHash,
		&existing.ExpiresAt,
	); err != nil {
		return domain.IdempotencyRecord{}, false, fmt.Errorf("get idempotency key: %w", err)
	}

	return existing, false, nil
}

func (r *IdempotencyRepository) executor(ctx context.Context) DBTX {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return r.pool
}
