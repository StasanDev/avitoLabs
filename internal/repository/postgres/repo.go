package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/StasanDev/avitoLabs/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var queryTripCols = []string{
	"id", "user_id", "driver_id", "start_latitude", "start_longitude",
	"end_latitude", "end_longitude", "price", "status",
	"started_at", "finished_at",
}

const (
	tripsTable              = "trips"
	tripStatusHistoryTable  = "trip_status_history"
	driverActiveUniqueIndex = "trips_driver_active_uidx"
)

type TripRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

type TripStatusHistoryRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTripRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *TripRepository {
	return &TripRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func NewTripStatusRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *TripStatusHistoryRepository {
	return &TripStatusHistoryRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *TripRepository) CreateTrip(ctx context.Context, trip domain.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.
		Insert(tripsTable).Columns(queryTripCols...).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.StartPoint.Latitude, trip.StartPoint.Longitude,
			trip.EndPoint.Latitude, trip.EndPoint.Longitude,
			trip.Price, trip.Status, trip.StartedAt, trip.FinishedAt,
		).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build create trip: %w", err)
	}

	if _, err := r.executor(ctx).Exec(ctx, query, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == driverActiveUniqueIndex {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}

	return nil
}

func (r *TripStatusHistoryRepository) AddInTripHistory(ctx context.Context, history domain.TripStatusHistory) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Insert(tripStatusHistoryTable).Columns("trip_id", "from_status", "to_status", "reason", "changed_at").
		Values(history.TripID, history.FromStatus, history.ToStatus,
			history.Reason, history.ChangedAt).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build trip status history: %w", err)
	}

	if _, err := r.executor(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("add in trip status history: %w", err)
	}

	return nil
}

func (r *TripRepository) GetTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Select(queryTripCols...).From(tripsTable).Where(sq.Eq{"id": id}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build get trip: %w", err)
	}

	row := r.executor(ctx).QueryRow(ctx, query, args...)
	trip, err := scanTrip(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Trip{}, domain.ErrTripNotFound
		}
		return domain.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return trip, nil
}

func (r *TripRepository) FinishTrip(ctx context.Context, id uuid.UUID, at time.Time) (domain.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := sq.Update(tripsTable).SetMap(map[string]interface{}{
		"status":      domain.TripStatusCompleted,
		"finished_at": at,
		"updated_at":  at,
	}).Where(sq.Eq{
		"id":     id,
		"status": domain.TripStatusActive,
	}).Suffix("RETURNING " + strings.Join(queryTripCols, ", ")).
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return domain.Trip{}, fmt.Errorf("build finish trip: %w", err)
	}

	row := r.executor(ctx).QueryRow(ctx, query, args...)
	trip, err := scanTrip(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, err := r.GetTrip(ctx, id)
			if errors.Is(err, domain.ErrTripNotFound) {
				return domain.Trip{}, domain.ErrTripNotFound
			}
			if err != nil {
				return domain.Trip{}, fmt.Errorf("check trip after unsuccessful finish: %w", err)
			}
			return domain.Trip{}, domain.ErrTripCompleted
		}
		return domain.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return trip, nil
}

func (r *TripRepository) executor(ctx context.Context) DBTX {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *TripStatusHistoryRepository) executor(ctx context.Context) DBTX {
	if tx, ok := txFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func scanTrip(row pgx.Row) (domain.Trip, error) {
	var trip domain.Trip
	err := row.Scan(&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.StartPoint.Latitude, &trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude, &trip.EndPoint.Longitude,
		&trip.Price, &trip.Status, &trip.StartedAt, &trip.FinishedAt)

	if err != nil {
		return domain.Trip{}, fmt.Errorf("scan trip: %w", err)
	}
	return trip, nil
}
