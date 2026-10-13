package usecase

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/StasanDev/avitoLabs/internal/domain"
	"github.com/google/uuid"
)

type TripService struct {
	tripRepository              tripRepository
	tripStatusHistoryRepository tripStatusHistoryRepository
	idempotencyRepository       idempotencyRepository
	txManager                   TxManager
	idempotencyTTL              time.Duration
}

func NewTripService(
	tripRepository tripRepository,
	tripStatusHistoryRepository tripStatusHistoryRepository,
	idempotencyRepository idempotencyRepository,
	txManager TxManager,
	idempotencyTTL time.Duration,
) *TripService {
	return &TripService{
		tripRepository:              tripRepository,
		tripStatusHistoryRepository: tripStatusHistoryRepository,
		idempotencyRepository:       idempotencyRepository,
		txManager:                   txManager,
		idempotencyTTL:              idempotencyTTL,
	}
}

func (s *TripService) CreateTrip(ctx context.Context, tripInp domain.TripInput) (domain.Trip, bool, error) {
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

	resultTrip := trip
	replayed := false
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		if tripInp.IdempotencyKey != nil {
			idempotencyRecord := domain.IdempotencyRecord{
				Key:         *tripInp.IdempotencyKey,
				RequestHash: tripInp.RequestHash,
				TripID:      trip.ID,
				ExpiresAt:   now.Add(s.idempotencyTTL),
			}

			existingRecord, reserved, err := s.idempotencyRepository.ReserveOrGet(txCtx, idempotencyRecord)
			if err != nil {
				return fmt.Errorf("reserve or get idempotency key: %w", err)
			}
			if !reserved {
				if !bytes.Equal(existingRecord.RequestHash, tripInp.RequestHash) {
					return domain.ErrIdempotencyConflict
				}

				resultTrip, err = s.tripRepository.GetTrip(txCtx, existingRecord.TripID)
				if err != nil {
					return fmt.Errorf("get trip by idempotency key: %w", err)
				}
				replayed = true
				return nil
			}
		}
		if err := s.tripRepository.CreateTrip(txCtx, trip); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}
		if err := s.tripStatusHistoryRepository.AddInTripHistory(txCtx, history); err != nil {
			return fmt.Errorf("create initial trip status history: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Trip{}, false, err
	}

	return resultTrip, replayed, nil
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
