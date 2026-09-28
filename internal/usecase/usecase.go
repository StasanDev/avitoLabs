package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/StasanDev/avitoLabs/internal/domain"
	"github.com/google/uuid"
)

type TripService struct {
	tripRepository              tripRepository
	tripStatusHistoryRepository tripStatusHistoryRepository
	txManager                   TxManager
}

func NewTripService(
	tripRepository tripRepository,
	tripStatusHistoryRepository tripStatusHistoryRepository,
	txManager TxManager,
) *TripService {
	return &TripService{
		tripRepository:              tripRepository,
		tripStatusHistoryRepository: tripStatusHistoryRepository,
		txManager:                   txManager,
	}
}

func (s *TripService) CreateTrip(ctx context.Context, tripInp domain.TripInput) (domain.Trip, error) {
	now := time.Now().UTC()
	trip := domain.Trip{
		TripInput:  tripInp,
		ID:         uuid.New(),
		Status:     domain.TripStatusActive,
		StartedAt:  now,
		FinishedAt: nil,
	}

	reason := "trip created"
	history := domain.TripStatusHistory{
		TripID:    trip.ID,
		ToStatus:  domain.TripStatusActive,
		Reason:    &reason,
		ChangedAt: now,
	}

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.tripRepository.CreateTrip(txCtx, trip); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}
		if err := s.tripStatusHistoryRepository.AddInTripHistory(txCtx, history); err != nil {
			return fmt.Errorf("create initial trip status history: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Trip{}, err
	}

	return trip, nil
}

func (s *TripService) GetTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return s.tripRepository.GetTrip(ctx, id)
}

func (s *TripService) FinishTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	finishAt := time.Now().UTC()
	fromStatus := domain.TripStatusActive
	reason := "trip finished"

	history := domain.TripStatusHistory{
		TripID:     id,
		FromStatus: &fromStatus,
		ToStatus:   domain.TripStatusCompleted,
		Reason:     &reason,
		ChangedAt:  finishAt,
	}

	var trip domain.Trip
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		var err error
		trip, err = s.tripRepository.FinishTrip(txCtx, id, finishAt)
		if err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}
		if err := s.tripStatusHistoryRepository.AddInTripHistory(txCtx, history); err != nil {
			return fmt.Errorf("add finished trip status history: %w", err)
		}
		return nil
	})

	if err != nil {
		return domain.Trip{}, err
	}

	return trip, nil
}
