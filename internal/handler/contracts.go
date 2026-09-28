package handler

import (
	"context"

	"github.com/StasanDev/avitoLabs/internal/domain"
	"github.com/google/uuid"
)

type tripService interface {
	CreateTrip(ctx context.Context, trip domain.TripInput) (domain.Trip, error)
	GetTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error)
	FinishTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error)
}

type pinger interface {
	Ping(ctx context.Context) error
}
