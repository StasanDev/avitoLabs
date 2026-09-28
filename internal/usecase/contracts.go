package usecase

import (
	"context"
	"time"

	"github.com/StasanDev/avitoLabs/internal/domain"
	"github.com/google/uuid"
)

type tripRepository interface {
	CreateTrip(ctx context.Context, trip domain.Trip) error
	GetTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error)
	FinishTrip(ctx context.Context, id uuid.UUID, at time.Time) (domain.Trip, error)
}

type tripStatusHistoryRepository interface {
	AddInTripHistory(ctx context.Context, history domain.TripStatusHistory) error
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type idempotencyRepository interface {
	ReserveOrGet(
		ctx context.Context,
		record domain.IdempotencyRecord,
	) (existing domain.IdempotencyRecord, reserved bool, err error)
}
