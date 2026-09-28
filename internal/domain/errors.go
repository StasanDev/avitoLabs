package domain

import "errors"

var (
	ErrDriverBusy          = errors.New("driver already has an active trip")
	ErrTripNotFound        = errors.New("trip not found")
	ErrTripCompleted       = errors.New("trip is completed")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
)
